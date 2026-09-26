package sopclient

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	// ErrTaskNotFound is returned when a task id doesn't exist.
	ErrTaskNotFound = errors.New("task not found")
	// ErrProjectNotFound is returned when a project id isn't configured.
	ErrProjectNotFound = errors.New("project not found")
)

// Client is the concrete SOP boundary over one or more project roots.
type Client struct {
	stores  map[string]*Store
	order   []string
	cmd     *Commander
	timeout time.Duration
}

// New opens the SOP state for each root and returns a boundary client.
func New(roots []string, sopBin string, timeout time.Duration) (*Client, error) {
	c := &Client{stores: map[string]*Store{}, cmd: NewCommander(sopBin, timeout), timeout: timeout}
	for _, root := range roots {
		st, err := OpenStore(root)
		if err != nil {
			return nil, err
		}
		id := st.ProjectID()
		if _, dup := c.stores[id]; dup {
			return nil, fmt.Errorf("duplicate project id %q (project roots need distinct directory names)", id)
		}
		c.stores[id] = st
		c.order = append(c.order, id)
	}
	return c, nil
}

// Close releases the underlying state databases.
func (c *Client) Close() error {
	var err error
	for _, st := range c.stores {
		if e := st.Close(); e != nil && err == nil {
			err = e
		}
	}
	return err
}

// Root returns the filesystem root for a project id.
func (c *Client) Root(id string) (string, bool) {
	st, ok := c.stores[id]
	if !ok {
		return "", false
	}
	return st.root, true
}

func (c *Client) Projects(ctx context.Context) ([]ProjectSummary, error) {
	out := make([]ProjectSummary, 0, len(c.order))
	for _, id := range c.order {
		s, err := c.stores[id].Summary(ctx)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

func (c *Client) Project(ctx context.Context, id string) (ProjectDetail, error) {
	st, ok := c.stores[id]
	if !ok {
		return ProjectDetail{}, ErrProjectNotFound
	}
	sum, err := st.Summary(ctx)
	if err != nil {
		return ProjectDetail{}, err
	}
	tasks, err := st.Tasks(ctx)
	if err != nil {
		return ProjectDetail{}, err
	}
	plan, _ := st.PlanSource()
	return ProjectDetail{Summary: sum, Tasks: tasks, PlanSource: plan, Plan: st.Plan()}, nil
}

func (c *Client) Task(ctx context.Context, projectID, taskID string) (TaskDetail, error) {
	st, ok := c.stores[projectID]
	if !ok {
		return TaskDetail{}, ErrProjectNotFound
	}
	return st.Task(ctx, taskID)
}

// Activity returns recent structured SOP activity across a project. It reads
// SOP's persisted activity stream (activity.jsonl) through the same read path
// as TaskDetail.Activity, so both consumers share one ActivityEvent model; a
// project whose runs predate activity reporting has none.
//
// Ordering: events come back in SOP's append order (oldest first), stable across
// repeated reads. When limit > 0 the most recent limit events are returned,
// still oldest first.
func (c *Client) Activity(ctx context.Context, projectID string, limit int) ([]ActivityEvent, error) {
	st, ok := c.stores[projectID]
	if !ok {
		return nil, ErrProjectNotFound
	}
	// Enumerate ids only: no need to build each task's summary or read its run
	// artifacts just to find its activity stream. Ids come back ordered, so the
	// concatenation is deterministic across reads.
	ids, err := st.taskIDs(ctx)
	if err != nil {
		return nil, err
	}
	var events []ActivityEvent
	for _, id := range ids {
		events = append(events, st.activity(id)...)
	}
	if limit > 0 && len(events) > limit {
		events = events[len(events)-limit:]
	}
	return events, nil
}

// PlanSource returns the active plan path SOP recorded for a project, if any.
func (c *Client) PlanSource(projectID string) (string, bool) {
	st, ok := c.stores[projectID]
	if !ok {
		return "", false
	}
	return st.PlanSource()
}

// ChangedTasks is the CTRL012 read of the changed executed tasks awaiting
// reconcile, as SOP reported them. It reads SOP's own reconcile report
// (reconcile.json) read-only, verbatim, through the same read boundary as
// plan.meta.json and the run artifacts; the controller never opens SOP state for
// write and never computes a plan diff of its own. When SOP reports no set, the
// result has Reported=false and no tasks, so a caller shows an explicit
// absence instead of an empty success.
//
// An unknown project yields ErrProjectNotFound, the same sentinel as every other
// boundary operation.
func (c *Client) ChangedTasks(ctx context.Context, projectID string) (ChangedTasks, error) {
	st, ok := c.stores[projectID]
	if !ok {
		return ChangedTasks{}, ErrProjectNotFound
	}
	return st.ChangedTasks(), nil
}

// --- commands (FR-8): each delegates to the SOP CLI ---

func (c *Client) Run(ctx context.Context, projectID string) error {
	_, err := c.exec(ctx, projectID, "run")
	return err
}

func (c *Client) Resume(ctx context.Context, projectID string) error {
	_, err := c.exec(ctx, projectID, "resume")
	return err
}

// Retry requeues a single BLOCKED task through SOP. SOP decides whether the
// task is eligible; the dashboard does not. It never mutates SOP state directly.
func (c *Client) Retry(ctx context.Context, projectID, taskID string) error {
	_, err := c.exec(ctx, projectID, "retry", taskID)
	return err
}

// RetryForce requeues a task whose retry budget is spent, raising max_attempts
// (`sop retry <id> --force`). SOP decides the outcome.
func (c *Client) RetryForce(ctx context.Context, projectID, taskID string) error {
	_, err := c.exec(ctx, projectID, "retry", taskID, "--force")
	return err
}

// RetryAll requeues every BLOCKED task that still has retry budget
// (`sop retry --all`). SOP decides which tasks qualify.
func (c *Client) RetryAll(ctx context.Context, projectID string) error {
	_, err := c.exec(ctx, projectID, "retry", "--all")
	return err
}

// Reconcile applies an intentional plan change through SOP
// (`sop reconcile <PLAN.md>`). SOP preserves unchanged tasks and stops at the
// human boundary when an executed task's definition changed.
func (c *Client) Reconcile(ctx context.Context, projectID, planPath string) (string, error) {
	return c.exec(ctx, projectID, "reconcile", planPath)
}

// ReportTask returns SOP's concise summary of one task's latest run
// (`sop report <run-id>`).
func (c *Client) ReportTask(ctx context.Context, projectID, taskID string) (string, error) {
	return c.exec(ctx, projectID, "report", taskID)
}

// Validate runs the configured build/test/lint commands via SOP (FR-6).
func (c *Client) Validate(ctx context.Context, projectID string) (string, error) {
	return c.exec(ctx, projectID, "validate")
}

// RunReview runs the configured review engine via SOP (FR-5).
func (c *Client) RunReview(ctx context.Context, projectID string) (string, error) {
	return c.exec(ctx, projectID, "review")
}

// RunReport runs SOP's concise project-level run summary (FR-4 supporting detail).
func (c *Client) RunReport(ctx context.Context, projectID string) (string, error) {
	return c.exec(ctx, projectID, "report")
}

func (c *Client) exec(ctx context.Context, projectID, verb string, args ...string) (string, error) {
	st, ok := c.stores[projectID]
	if !ok {
		return "", ErrProjectNotFound
	}
	return c.cmd.Exec(ctx, st.root, verb, args...)
}

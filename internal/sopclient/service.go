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
		changed, _ := c.ChangedTasks(ctx, id)
		// SOP's authoritative approval listing drives every gate projection; a
		// failure surfaces as an explicit absence (no gate), never a fabricated
		// one. The controller still reads only what SOP reports.
		approvals, _ := c.Approvals(ctx, id)
		s, err := c.stores[id].Summary(ctx, changed, approvals)
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
	changed, _ := c.ChangedTasks(ctx, id)
	// SOP's authoritative approval listing (sop approvals --json). A failure is an
	// explicit absence, never a controller-side inference.
	approvals, _ := c.Approvals(ctx, id)
	sum, err := st.Summary(ctx, changed, approvals)
	if err != nil {
		return ProjectDetail{}, err
	}
	tasks, err := st.Tasks(ctx, approvals)
	if err != nil {
		return ProjectDetail{}, err
	}
	plan, _ := st.PlanSource()
	return ProjectDetail{Summary: sum, Tasks: tasks, PlanSource: plan, Plan: st.Plan(), PlanPerformance: st.PlanPerformance()}, nil
}

func (c *Client) Task(ctx context.Context, projectID, taskID string) (TaskDetail, error) {
	st, ok := c.stores[projectID]
	if !ok {
		return TaskDetail{}, ErrProjectNotFound
	}
	// SOP's authoritative approval listing (sop approvals --json).
	approvals, _ := c.Approvals(ctx, projectID)
	return st.Task(ctx, taskID, approvals)
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

// ChangedTasks is the C2-003 read of the changed executed tasks awaiting
// reconcile, from SOP's authoritative listing
// (`sop reconcile <PLAN.md> --list-changed --json`). It resolves the plan path
// from SOP's recorded provenance (plan.meta.json), runs the listing verb through
// the existing Commander boundary, and decodes SOP's document verbatim. The
// listing is a PURE READ: it mutates no task state, graph, active plan,
// acceptance, or provenance, and the controller never computes a plan diff.
//
// When SOP does not record an active plan the read returns ErrNoActivePlan and
// never guesses a PLAN.md path. When the listing verb fails or emits unparsable
// output the result has Reported=false (with the command/decode error surfaced),
// so a caller shows an explicit unreported/failed state instead of an empty
// success. A stale .agent-sdlc/reconcile.json is never read as a fallback.
//
// An unknown project yields ErrProjectNotFound, the same sentinel as every other
// boundary operation.
func (c *Client) ChangedTasks(ctx context.Context, projectID string) (ChangedTasks, error) {
	st, ok := c.stores[projectID]
	if !ok {
		return ChangedTasks{}, ErrProjectNotFound
	}
	planPath, ok := st.PlanSource()
	if !ok {
		return ChangedTasks{}, ErrNoActivePlan
	}
	// `sop reconcile <PLAN.md> --list-changed --json` through the argv-slice
	// Commander (no shell string): the plan path is a discrete argv element.
	out, err := c.exec(ctx, projectID, "reconcile", planPath, "--list-changed", "--json")
	if err != nil {
		return ChangedTasks{}, err
	}
	doc, ok := decodeListing([]byte(out))
	if !ok {
		return ChangedTasks{}, fmt.Errorf("sop reconcile --list-changed --json: unparsable SOP changed-task listing")
	}
	return st.changedTasksFromListing(doc), nil
}

// ErrApprovalsUnavailable is returned by Approvals when SOP's approval listing
// could not be read from ANY source: no persisted listing artifact exists AND
// the `sop approvals --json` verb failed or emitted unparsable output. It is
// distinct from a genuine empty listing (SOP reported a set with no gate
// entries), which returns a Reported listing with no error. A caller MUST treat
// this error as "SOP did not report an approval listing" and surface an explicit
// absence/failure, never as "SOP reported no gates"; a failing SOP must never be
// indistinguishable from a quiet one and silently hide a real gate.
var ErrApprovalsUnavailable = errors.New("SOP did not report an approval listing")

// Approvals is the C2-001 authoritative approval read: it returns SOP's
// structured approval listing for a project. SOP owns gate presence and
// applicability; the controller reports only what the listing says. The
// controller never reconstructs a gate from run classification, run stage,
// BLOCKED status, prose, attempt counts, or inactivity, and never parses
// `sop approval`'s human text.
//
// SOP's listing is read from the verb itself: the controller invokes `sop
// approvals --json` and decodes its stdout. An optional persisted listing
// artifact is consulted first as a cache; SOP does not write one today, so a real
// project always takes the live read.
//
// Error contract (this is the fix for the silent-empty-listing defect):
//   - a persisted, well-formed listing is returned verbatim with a nil error;
//   - a genuine empty listing (SOP reported a set with no entries) is returned
//     with Reported=true and a nil error;
//   - when NO persisted listing exists AND the verb fails to run, times out,
//     or emits unparsable output, the read returns an UNREPORTED listing AND a
//     non-nil error wrapping ErrApprovalsUnavailable. A backend failure is
//     therefore never indistinguishable from "SOP reported no gates": the
//     caller can surface the failure instead of silently hiding a real gate.
//
// An unknown project yields ErrProjectNotFound, the same sentinel as every other
// boundary operation.
func (c *Client) Approvals(ctx context.Context, projectID string) (ApprovalsListing, error) {
	st, ok := c.stores[projectID]
	if !ok {
		return ApprovalsListing{}, ErrProjectNotFound
	}
	if listing := st.Approvals(); listing.Reported {
		return listing, nil
	}
	// No persisted listing artifact: ask SOP directly the same way the refresh
	// path does, so a listing SOP emits only on stdout is read correctly. A failed
	// or unparsable invocation is surfaced as a non-nil error rather than being
	// swallowed into an unreported empty listing, which would make a failing SOP
	// indistinguishable from a quiet one.
	out, err := c.exec(ctx, projectID, "approvals", "--json")
	if err != nil {
		return ApprovalsListing{}, fmt.Errorf("%w: sop approvals --json: %v", ErrApprovalsUnavailable, err)
	}
	listing, ok := decodeApprovals([]byte(out))
	if !ok {
		return ApprovalsListing{}, fmt.Errorf("%w: sop approvals --json: unparsable SOP approval listing", ErrApprovalsUnavailable)
	}
	return listing, nil
}

// RefreshApprovals runs SOP's authoritative listing command
// (`sop approvals --json`) through the CLI boundary and decodes its structured
// output. It exists so a caller that wants SOP to (re)compute the listing can
// drive the verb the same way every other command delegates to SOP; the read
// path (Approvals) then reports the persisted artifact verbatim, falling back to
// this same stdout decode when SOP has not persisted one.
//
// When SOP exposes no such verb the command reports an error and the caller
// surfaces the present-or-absent failure, never controller-side inference.
func (c *Client) RefreshApprovals(ctx context.Context, projectID string) (ApprovalsListing, error) {
	if _, ok := c.stores[projectID]; !ok {
		return ApprovalsListing{}, ErrProjectNotFound
	}
	out, err := c.exec(ctx, projectID, "approvals", "--json")
	if err != nil {
		return ApprovalsListing{}, err
	}
	listing, ok := decodeApprovals([]byte(out))
	if !ok {
		return ApprovalsListing{}, fmt.Errorf("sop approvals --json: unparsable SOP approval listing")
	}
	return listing, nil
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

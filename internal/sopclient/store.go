package sopclient

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"sop-controller/internal/config"

	_ "modernc.org/sqlite"
)

// Store reads SOP's authoritative state for one project. SOP owns the schema;
// this package is the only place that knows it.
type Store struct {
	root string
	db   *sql.DB
	// checkpoints keeps the last SOP-reported checkpoint per task so a refresh
	// that yields no new evidence does not regress known progress (CTRL010). It
	// is an atomically-initialized pointer so a Store value copy shares one
	// mutex+map (never copying the lock), and so lazy initialization from
	// concurrent read paths (including a Store literal constructed outside
	// OpenStore) cannot race on the field. See checkpointStore in
	// checkpoint_cache.go.
	checkpoints checkpointStore
}

// StatePath is the SOP state database for a project root.
func StatePath(root string) string {
	return filepath.Join(root, ".agent-sdlc", "state.db")
}

// OpenStore opens the SOP state database read-only.
func OpenStore(root string) (*Store, error) {
	path := StatePath(root)
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("no SOP state at %s (is this an initialized SOP project?)", path)
	}
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{root: root, db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

// ProjectID returns the stable identifier for this store's project. It is the
// same identity discovery assigns, so configured and discovered projects are
// indistinguishable to the rest of the controller.
func (s *Store) ProjectID() string {
	return config.ProjectID(s.root)
}

// Summary builds the FR-1 project overview from task statuses.
//
// The NeedsAttention count folds the same per-task human-decision signal
// TaskSummary carries (so it can never drift from the list/detail views) with
// any pending changed-executed task SOP reports. When the caller supplies the
// SOP-reported changed set (changedTasks), those pending tasks are counted; when
// SOP reports no set, no changed-task count is added and the absence is NOT
// rendered as zero pending. The changed set itself is read from SOP's listing by
// the caller (Client.ChangedTasks); the Store never reads the retired
// reconcile.json artifact.
//
// The SOP-reported approval listing (approvals) is likewise supplied by the
// caller, which obtained it from SOP's authoritative `sop approvals --json`
// surface (Client.Approvals). The Store never infers a gate from task status,
// run classification, or a controller-side artifact SOP does not write.
func (s *Store) Summary(ctx context.Context, changedTasks ChangedTasks, approvals ApprovalsListing) (ProjectSummary, error) {
	name, branch := s.projectMeta()
	p := ProjectSummary{ID: s.ProjectID(), Name: name, Branch: branch}

	rows, err := s.db.QueryContext(ctx, `SELECT status, COUNT(*) FROM tasks GROUP BY status`)
	if err != nil {
		return p, err
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var n int
		if err := rows.Scan(&status, &n); err != nil {
			return p, err
		}
		p.Total += n
		switch {
		case IsTerminal(status):
			p.Completed += n
		case status == StatusBlocked:
			p.Blocked += n
		case status == StatusFixRequired:
			p.FixRequired += n
		case status == StatusReady:
			p.Ready += n
		case status == StatusPlanned:
			p.Planned += n
		default:
			p.Running += n
		}
	}
	if p.Name == "" {
		p.Name = p.ID
	}
	if err := rows.Err(); err != nil {
		return p, err
	}

	tasks, err := s.Tasks(ctx, approvals)
	if err != nil {
		return p, err
	}
	for _, t := range tasks {
		if t.NeedsHuman {
			p.NeedsAttention++
		}
	}
	if changedTasks.Reported {
		p.NeedsAttention += len(changedTasks.Pending())
	}
	return p, nil
}

// Tasks returns every task with its dependencies resolved (FR-2).
//
// approvals is SOP's authoritative approval listing, supplied by the caller
// (Client, which fetched it via `sop approvals --json`); it is shared with every
// row so the list projection is never reconstructed per task and can never
// disagree with TaskDetail.Approval. The Store does not read an approval
// artifact SOP never writes.
func (s *Store) Tasks(ctx context.Context, approvals ApprovalsListing) ([]TaskSummary, error) {
	deps, err := s.dependencyMap(ctx)
	if err != nil {
		return nil, err
	}
	// Share SOP's authoritative approval listing with every row.
	listing := approvals
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, title, status, COALESCE(blocked_reason, ''), attempt, max_attempts, updated_at
		FROM tasks ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byID := map[string]*TaskSummary{}
	var order []string
	for rows.Next() {
		var t TaskSummary
		var updated string
		if err := rows.Scan(&t.ID, &t.Title, &t.Status, &t.BlockedReason, &t.Attempt, &t.MaxAttempts, &updated); err != nil {
			return nil, err
		}
		t.UpdatedAt = parseTime(updated)
		byID[t.ID] = &t
		order = append(order, t.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// Resolve dependencies and compute which tasks are still blocked by an
	// incomplete dependency (FR-2 "why a task is not eligible to run").
	for id, t := range byID {
		t.Dependencies = deps[id]
		for _, dep := range t.Dependencies {
			if d, ok := byID[dep]; !ok || !IsTerminal(d.Status) {
				t.BlockedBy = append(t.BlockedBy, dep)
			}
		}
		// Enrich with the latest run's stage and SOP's recovery disposition, then
		// derive the human-decision signal from SOP's approval listing - the SAME
		// listing entry TaskDetail.Approval uses - so the list can never disagree
		// with the task detail view about which tasks need a human decision, and no
		// controller-side inference over classification/stage/BLOCKED is performed.
		ri := s.runInfo(id)
		t.Stage = ri.Stage
		if ri.Classification != nil {
			t.Recovery = ri.Classification.Disposition
		}
		t.FixCycles = ri.FixCycles
		t.NeedsHuman, t.ApprovalKind = taskNeedsHuman(listing, id)
	}
	out := make([]TaskSummary, 0, len(order))
	for _, id := range order {
		out = append(out, *byID[id])
	}
	return out, nil
}

// Task returns one task with attempts, validation, review, and handoff (FR-3),
// plus the CTRL003 structured status fields (run status, aggregate validation/
// review/JEV status, and report reference).
//
// approvals is SOP's authoritative approval listing, supplied by the caller
// (Client, from `sop approvals --json`); it is the same listing Store.Tasks
// receives, so the detail and list views cannot disagree.
func (s *Store) Task(ctx context.Context, id string, approvals ApprovalsListing) (TaskDetail, error) {
	var d TaskDetail
	var updated string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, title, status, COALESCE(blocked_reason, ''), attempt, max_attempts,
		       COALESCE(objective, ''), COALESCE(acceptance_criteria, ''), updated_at
		FROM tasks WHERE id = ?`, id).
		Scan(&d.ID, &d.Title, &d.Status, &d.BlockedReason, &d.Attempt, &d.MaxAttempts,
			&d.Objective, &d.AcceptanceCriteria, &updated)
	if err == sql.ErrNoRows {
		return d, ErrTaskNotFound
	}
	if err != nil {
		return d, err
	}
	d.UpdatedAt = parseTime(updated)

	if deps, err := s.dependencyMap(ctx); err == nil {
		d.Dependencies = deps[id]
		if statuses, err := s.statusMap(ctx); err == nil {
			for _, dep := range d.Dependencies {
				if st, ok := statuses[dep]; !ok || !IsTerminal(st) {
					d.BlockedBy = append(d.BlockedBy, dep)
				}
			}
		}
	}
	if attempts, err := s.attempts(ctx, id); err == nil {
		d.Attempts = attempts
		if n := len(attempts); n > 0 {
			last := attempts[n-1]
			if last.Reason != "" {
				d.LatestFailure = last.Reason
			}
		}
	}
	if h, err := s.handoff(ctx, id); err == nil {
		d.Handoff = h
	}
	// Runs artifacts carry the run record, validation/review output, and the
	// structured activity stream when SOP has executed the task.
	d.Run = s.runInfo(id)
	d.Activity = s.activity(id)
	d.Stage, d.Recovery = d.Run.Stage, ""
	if d.Run.Classification != nil {
		d.Recovery = d.Run.Classification.Disposition
	}
	if d.LatestFailure == "" && d.Run.Classification != nil {
		d.LatestFailure = d.Run.Classification.Reason
	}
	d.Validation = s.readValidation(id)
	d.Review = s.readReview(id)

	// CTRL003: render-ready status projections. Each reports absence explicitly
	// and never resolves to PASS without SOP-persisted evidence.
	d.RunStatus = d.Run.Status()
	d.ValidationStatus = aggregateValidation(d.Validation)
	d.ReviewStatus = aggregateReview(d.Review)
	d.QualityStatus = aggregateQuality(d.Run.JEV)
	d.Report = s.ReportRef(id)

	// CTRL010: SOP-reported checkpoint / bounded progress. Only the checkpoint
	// value comes from a SOP artifact/activity read; the merge only prevents a
	// transient re-read from regressing a recently known value, and is bounded in
	// time and size so a retracted/reset checkpoint is not replayed forever.
	d.Checkpoint = s.mergeCheckpoint(id, s.Checkpoint(id))

	// CTRL011 / C2-001: SOP-reported approval gate, projected from SOP's
	// authoritative approval listing - the same listing Store.Tasks reads - so the
	// detail and list views can never disagree. The controller never infers a gate.
	listing := approvals
	d.Approval = s.approval(d, listing)
	// Mirror the list projection onto the embedded summary so TaskDetail carries
	// the same NeedsHuman/ApprovalKind signal as TaskSummary.
	d.TaskSummary.NeedsHuman, d.TaskSummary.ApprovalKind = taskNeedsHuman(listing, id)

	// Performance is SOP's own persisted measurement for the latest run, read
	// read-only. It is diagnostic only: it never feeds a lifecycle decision, and a
	// task with no performance artifact simply carries an absent record.
	d.Performance = s.Performance(id)
	return d, nil
}

// mergeCheckpoint applies the bounded refresh-merge rule for the checkpoint
// cache (CTRL010). It routes through the store's atomically-initialized cache so
// a Store value constructed outside OpenStore still gets a usable cache and
// concurrent read paths never race on the cache pointer.
func (s *Store) mergeCheckpoint(taskID string, fresh CheckpointProgress) CheckpointProgress {
	return s.checkpoints.get().merge(taskID, fresh)
}

func (s *Store) attempts(ctx context.Context, id string) ([]Attempt, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT number, status, COALESCE(reason, ''), duration, timestamp
		FROM task_attempts WHERE task_id = ? ORDER BY number`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Attempt
	for rows.Next() {
		var a Attempt
		var ts string
		var dur int64
		if err := rows.Scan(&a.Number, &a.Status, &a.Reason, &dur, &ts); err != nil {
			return nil, err
		}
		a.Duration = time.Duration(dur)
		a.Timestamp = parseTime(ts)
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) handoff(ctx context.Context, id string) (*Handoff, error) {
	var h Handoff
	var created string
	err := s.db.QueryRowContext(ctx, `
		SELECT status, COALESCE(content, ''), COALESCE(compression_error, ''), created_at
		FROM handoffs WHERE task_id = ?`, id).
		Scan(&h.Status, &h.Content, &h.CompressionError, &created)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	h.CreatedAt = parseTime(created)
	return &h, nil
}

// taskIDs returns every persisted task id, for callers that only need to
// enumerate tasks and not their summary or run artifacts.
func (s *Store) taskIDs(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM tasks ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (s *Store) statusMap(ctx context.Context) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, status FROM tasks`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := map[string]string{}
	for rows.Next() {
		var id, status string
		if err := rows.Scan(&id, &status); err != nil {
			return nil, err
		}
		m[id] = status
	}
	return m, rows.Err()
}

func (s *Store) dependencyMap(ctx context.Context) (map[string][]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT task_id, dependency_task_id FROM task_dependencies`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := map[string][]string{}
	for rows.Next() {
		var task, dep string
		if err := rows.Scan(&task, &dep); err != nil {
			return nil, err
		}
		m[task] = append(m[task], dep)
	}
	return m, rows.Err()
}

// projectMeta reads the small, non-secret subset of .agent-sdlc/config.yaml we
// display (project name and integration branch) through the shared parser, so the
// store and discovery agree on how SOP configuration is interpreted.
func (s *Store) projectMeta() (name, branch string) {
	cfg, err := config.ReadProjectConfig(s.root)
	if err != nil {
		return "", ""
	}
	return cfg.Name, cfg.Branch
}

func parseTime(s string) time.Time {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

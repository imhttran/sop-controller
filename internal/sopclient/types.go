// Package sopclient is the SOP application/API boundary. The dashboard reads
// and commands SOP through this package and never touches SOP's SQLite schema
// from the web layer.
package sopclient

import "time"

// Task status values are SOP's own workflow vocabulary (agentic-sop
// internal/domain.TaskStatus). The controller mirrors them for display only; it
// never decides a task's status, only reports the one SOP persisted.
const (
	StatusPlanned        = "PLANNED"
	StatusReady          = "READY"
	StatusBranchCreated  = "BRANCH_CREATED"
	StatusTestsWritten   = "TESTS_WRITTEN"
	StatusRedVerified    = "RED_VERIFIED"
	StatusImplementing   = "IMPLEMENTING"
	StatusLocalTestsPass = "LOCAL_TESTS_PASS"
	StatusReview         = "REVIEW"
	StatusReviewPass     = "REVIEW_PASS"
	StatusPROpen         = "PR_OPEN"
	StatusCIRunning      = "CI_RUNNING"
	StatusCIPass         = "CI_PASS"
	StatusFixRequired    = "FIX_REQUIRED"
	StatusMerged         = "MERGED"
	StatusDone           = "DONE"
	StatusLocalDone      = "LOCAL_DONE"
	StatusBlocked        = "BLOCKED"
)

// IsTerminal reports whether a status is a completed end state. A DONE task was
// merged; a LOCAL_DONE task passed the local lifecycle without a merge.
func IsTerminal(status string) bool {
	switch status {
	case StatusDone, StatusLocalDone, StatusMerged:
		return true
	default:
		return false
	}
}

// IsCommitted reports whether SOP's status means the work is integrated into the
// shared branch or completed as a merge. It exists so callers can keep
// LOCAL_DONE distinct from DONE/MERGED: LOCAL_DONE is a completed local
// lifecycle without a commit/merge.
func IsCommitted(status string) bool {
	return status == StatusDone || status == StatusMerged
}

// TaskState is a coarse, display-only grouping of a task's SOP status, used by
// the list and summary when a single word is enough: DONE, RUNNING, BLOCKED,
// READY, or PLANNED. It never changes SOP state.
func TaskState(status string) string {
	switch {
	case IsTerminal(status):
		return "DONE"
	case status == StatusBlocked:
		return "BLOCKED"
	case status == StatusFixRequired:
		return "FAILED"
	case status == StatusReady:
		return "READY"
	case status == StatusPlanned:
		return "PLANNED"
	default:
		return "RUNNING"
	}
}

// ProjectSummary is the FR-1 project overview.
type ProjectSummary struct {
	ID          string
	Name        string
	Branch      string
	Total       int
	Completed   int
	Running     int
	Ready       int
	Planned     int
	Blocked     int
	FixRequired int
	// NeedsAttention is the count of tasks reporting a human decision boundary
	// (TaskSummary.NeedsHuman) plus any pending changed-executed task SOP
	// reported awaiting reconcile. It is never a duplicate lifecycle state: it
	// is a tally over the same SOP-reported signals the other counts use.
	NeedsAttention int
}

// PercentComplete is completed/total as an integer percentage.
func (p ProjectSummary) PercentComplete() int {
	if p.Total == 0 {
		return 0
	}
	return p.Completed * 100 / p.Total
}

// State is a coarse overall execution state for the project.
func (p ProjectSummary) State() string {
	switch {
	case p.FixRequired > 0:
		return "attention"
	case p.Blocked > 0:
		return "blocked"
	case p.Running > 0:
		return "running"
	case p.Total > 0 && p.Completed == p.Total:
		return "complete"
	default:
		return "idle"
	}
}

// ProjectDetail is a project plus its tasks (FR-2).
type ProjectDetail struct {
	Summary ProjectSummary
	Tasks   []TaskSummary
	// PlanSource is the active plan path recorded by SOP, when known.
	PlanSource string
	// Plan is the active plan and its final plan gate as SOP persisted them
	// (CTRL003). Recorded is false when SOP persisted no plan metadata.
	Plan PlanGate
	// PlanPerformance is SOP's plan-level performance aggregate for the active
	// plan, read from SOP's recorded plan identity (plan.meta.json) and its run
	// metrics artifact. It is diagnostic metadata only and Present is false when
	// SOP persisted (or this project has) no aggregate.
	PlanPerformance PlanPerformance
}

// ActiveTask returns the id of the running task the controller should surface
// as "the active task", derived only from SOP-persisted status and stage. A
// task is active when SOP's status groups it as RUNNING and its latest run
// stage is an in-flight stage. It returns ("", false) when nothing is active.
func (p ProjectDetail) ActiveTask() (string, bool) {
	for _, t := range p.Tasks {
		if t.State() == "RUNNING" && activeStage(t.Stage) {
			return t.ID, true
		}
	}
	return "", false
}

// HasRetryableBlocked reports whether any task in the project is retryable
// BLOCKED work.
func (p ProjectDetail) HasRetryableBlocked() bool {
	for _, t := range p.Tasks {
		if t.Retryable() {
			return true
		}
	}
	return false
}

// TaskSummary is one row in the project workflow view (FR-2).
type TaskSummary struct {
	ID            string
	Title         string
	Status        string
	BlockedReason string
	Attempt       int
	MaxAttempts   int
	UpdatedAt     time.Time
	// Dependencies are task ids this task depends on.
	Dependencies []string
	// BlockedBy is the subset of dependencies not yet COMPLETED.
	BlockedBy []string
	// Stage is the latest run's SOP lifecycle stage (e.g. IMPLEMENTING,
	// VALIDATING, PASSED), or "" when the task has not run.
	Stage string
	// Recovery is SOP's recovery disposition for the latest failure
	// (AUTO_FIX/CONTINUE/RETRY/REPLAN/NEEDS_HUMAN), or "" when none.
	Recovery string
	// FixCycles is the latest run's reported auto-fix cycle count, or 0 when
	// the task has no run.
	FixCycles int
	// NeedsHuman reports whether SOP reports an applicable human approval gate
	// for this task, projected from the SAME authoritative approval listing
	// (sop approvals --json) that TaskDetail.Approval uses, so the list and
	// detail view can never disagree. It is true only when SOP's listing has an
	// applicable entry for the task; BLOCKED status alone, run stage, model/
	// recovery prose, attempt counts, and inactivity never set it.
	NeedsHuman bool
	// ApprovalKind is SOP's reported gate kind (kind) when NeedsHuman is true, or
	// "" otherwise. It is carried verbatim from the listing entry.
	ApprovalKind string
}

// State is the coarse display state for this task.
func (t TaskSummary) State() string { return TaskState(t.Status) }

// Retries returns how many attempts SOP has already spent: attempt-1 when
// attempt > 0, and (0, false) when SOP recorded no attempt yet, so a genuine
// zero is distinguishable from an absence.
func (t TaskSummary) Retries() (int, bool) {
	if t.Attempt <= 0 {
		return 0, false
	}
	return t.Attempt - 1, true
}

// Eligible reports whether SOP would let this task run now.
func (t TaskSummary) Eligible() bool {
	return len(t.BlockedBy) == 0 &&
		(t.Status == StatusReady || t.Status == StatusPlanned)
}

// Retryable reports whether this is BLOCKED work SOP still has retry budget
// for.
func (t TaskSummary) Retryable() bool {
	return t.Status == StatusBlocked && t.Attempt < t.MaxAttempts
}

// TaskDetail is the FR-3 task view.
type TaskDetail struct {
	TaskSummary
	Objective          string
	AcceptanceCriteria string
	Attempts           []Attempt
	// LatestFailure is the most recent failure/review diagnostic, if any.
	LatestFailure string
	Validation    []ValidationCheck
	Review        Review
	Handoff       *Handoff
	// Run is the latest run's structured record (stage, decision, classification).
	Run RunInfo
	// Activity is the structured activity SOP persisted for the task.
	Activity []ActivityEvent

	// --- CTRL003 structured status (render-ready, read-only) ---

	// ValidationStatus is the aggregate validation result over Validation.
	ValidationStatus StatusLevel
	// ReviewStatus is the aggregate review result over Review.
	ReviewStatus StatusLevel
	// QualityStatus is the JEV/quality status SOP persisted, or StatusUnknown.
	QualityStatus StatusLevel
	// RunStatus exposes the current run's stage/decision and active/blocked
	// grouping. Missing run artifacts leave Present false, never PASSED.
	RunStatus RunStatus
	// Report is the location/reference of SOP's persisted report artifact for
	// the latest run. Present is false when SOP persisted no report; the path is
	// then empty rather than a synthesized one.
	Report ReportRef

	// --- CTRL010 checkpoint / bounded progress (render-ready, read-only) ---

	// Checkpoint is the most recent checkpoint/bounded-progress line SOP
	// reported for this task (e.g. "11/15 cases analyzed, covered: 11,
	// missing: 4, next: ..."). Every value is SOP-reported verbatim; Present is
	// false when SOP reported no checkpoint, in which case a renderer shows an
	// explicit absence and never a number. Partial progress here is never task
	// completion: completion still comes only from SOP's task status/run decision.
	Checkpoint CheckpointProgress

	// --- CTRL011 / C2-001 human approval boundary (render-ready, read-only) ---

	// Approval is SOP's human approval gate for this task, projected from SOP's
	// authoritative approval listing (sop approvals --json). Present is false when
	// the listing reports no applicable gate for the task; an approval action may
	// only be offered while Present is true. It is never inferred from run
	// classification, run stage, BLOCKED status, prose, attempt counts, or
	// inactivity.
	Approval Approval

	// --- performance (diagnostic, read-only, SOP-measured) ---

	// Performance is the latest performance record SOP persisted for this task
	// (.agent-sdlc/runs/<task>/metrics.json, or report.json's performance field),
	// projected verbatim. It is diagnostic metadata only: it never determines task
	// status, selection, retry, recovery, approval, or any lifecycle decision.
	// Present is false when SOP persisted no record, in which case a renderer
	// shows an explicit absence rather than a 0s measurement.
	Performance Performance
}

// NeedsHuman reports whether SOP reports an applicable human approval gate for
// this task, projected from the authoritative approval listing. It is true only
// when the listing has an applicable entry; it is NOT derived from BLOCKED
// status or a run classification, which no longer produce a gate on their own.
func (t TaskDetail) NeedsHuman() bool {
	return t.Approval.Present
}

// Recovering returns the disposition SOP applied when it is recovering from a
// failure on its own (AUTO_FIX/CONTINUE/RETRY/REPLAN), or "" when SOP is not
// recovering (no classification, or a human boundary).
func (t TaskDetail) Recovering() string {
	if t.Run.Classification == nil {
		return ""
	}
	d := t.Run.Classification.Disposition
	if d == DispositionNeedsHuman {
		return ""
	}
	return d
}

// Attempt is one SOP task attempt (FR-4 event source).
type Attempt struct {
	Number    int
	Status    string
	Reason    string
	Duration  time.Duration
	Timestamp time.Time
}

// ValidationCheck is one build/test/lint result (FR-6 CI view).
type ValidationCheck struct {
	Category string
	Command  string
	Status   string
	ExitCode int
	Stderr   string
}

// ReviewFinding is one review finding (FR-5).
type ReviewFinding struct {
	Severity string
	Title    string
	Detail   string
	Location string
}

// Review is the review result for a task (FR-5).
type Review struct {
	Summary  string
	Findings []ReviewFinding
}

// Handoff is the FR-7 handoff context for a task.
type Handoff struct {
	Status           string
	Content          string
	CompressionError string
	CreatedAt        time.Time
}

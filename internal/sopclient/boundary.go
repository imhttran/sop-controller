package sopclient

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// ErrOperationUnsupported is returned by a boundary operation that has no
// corresponding SOP application operation. The controller must not simulate the
// missing SOP lifecycle operation itself: SOP owns the lifecycle, so a gap is
// reported rather than filled with controller-side orchestration.
//
// Callers test for it with errors.Is; the recorded reason is available from
// Boundary and Lookup.
var ErrOperationUnsupported = errors.New("operation unsupported by SOP")

// ErrDecisionRejected is returned when SOP itself rejected a decision command
// (`sop approve` / `sop decline`) - for example a stale or no-longer-applicable
// gate. SOP is the authority that validates the gate at command time, so the
// controller does not pre-judge eligibility from its own read: it delegates the
// action and reports SOP's rejection truthfully.
//
// A *DecisionRejection wraps ErrDecisionRejected (and the underlying command
// error) so a caller - here the web layer - can classify the outcome as an
// actionable conflict rather than an infrastructure failure, carrying SOP's own
// message plus the affected task id so the operator can act. It is never a
// fabricated success and never a task-failure signal: the controller writes no
// SOP state and marks nothing complete or failed either way.
var ErrDecisionRejected = errors.New("SOP rejected the decision")

// DecisionRejection is the typed error a rejected approve/decline returns. It
// classifies SOP's own non-zero result (stale/not-applicable gate, or any other
// refusal SOP reports) as a decision rejection, distinct from an infrastructure
// failure such as a missing binary or a timeout. It carries SOP's message and the
// affected task id verbatim so a renderer can show an actionable conflict.
type DecisionRejection struct {
	// Verb is the SOP verb that was rejected (`approve` or `decline`).
	Verb string
	// TaskID is the task the decision concerned, as supplied by the caller.
	TaskID string
	// Message is SOP's own output/reason, verbatim. It is never a synthesized
	// explanation of why the gate is stale: SOP owns that verdict.
	Message string
	// Err is the underlying command error (a non-zero exit), retained for
	// errors.Is/Unwrap.
	Err error
}

func (e *DecisionRejection) Error() string {
	msg := e.Message
	if msg == "" {
		msg = "no message"
	}
	return fmt.Sprintf("sop %s %s rejected by SOP: %s", e.Verb, e.TaskID, msg)
}

func (e *DecisionRejection) Unwrap() []error { return []error{ErrDecisionRejected, e.Err} }

// ErrReconcileRejected is returned when SOP itself rejected a reconcile
// mutation - `sop reconcile <PLAN.md> --accept-changed <TASK_ID>` (or a plain
// `sop reconcile <PLAN.md>`). SOP is the authority that validates reconciliation
// at command time, so when it reports a non-zero result the controller surfaces
// SOP's own answer truthfully rather than fabricating a success or simulating the
// operation.
//
// A *ReconcileRejection wraps ErrReconcileRejected (and the underlying command
// error) so a caller can classify the outcome as an actionable conflict, carrying
// SOP's own message verbatim plus the affected task ids. It is distinct from the
// precondition sentinels (ErrTaskNotInChangedSet and friends), which are decided
// by the controller BEFORE any SOP call, and from ErrOperationUnsupported. It is
// never a fabricated success and never a task-failure signal: the controller
// writes no SOP state and marks nothing complete or failed either way.
var ErrReconcileRejected = errors.New("SOP rejected the reconciliation")

// ReconcileRejection is the typed error a rejected reconcile/accept-changed
// returns. It classifies SOP's own non-zero result (a real gap in the external
// SOP binary for the verb/flag, a refused plan change, or any other refusal SOP
// reports) as a reconciliation rejection, distinct from an infrastructure failure
// such as a missing binary or a timeout. It carries SOP's message verbatim and
// the affected task ids so a renderer can show an actionable conflict.
type ReconcileRejection struct {
	// Verb is the SOP verb that was rejected (`reconcile`).
	Verb string
	// TaskIDs are the task ids the rejected invocation concerned, as supplied by
	// the caller (empty for a plain reconcile).
	TaskIDs []string
	// Message is SOP's own output/reason, verbatim. It is never a synthesized
	// explanation of why SOP refused: SOP owns that verdict.
	Message string
	// Err is the underlying command error (a non-zero exit), retained for
	// errors.Is/Unwrap.
	Err error
}

func (e *ReconcileRejection) Error() string {
	msg := e.Message
	if msg == "" {
		msg = "no message"
	}
	if len(e.TaskIDs) > 0 {
		return fmt.Sprintf("sop %s (accept-changed %s) rejected by SOP: %s", e.Verb, strings.Join(e.TaskIDs, ", "), msg)
	}
	return fmt.Sprintf("sop %s rejected by SOP: %s", e.Verb, msg)
}

func (e *ReconcileRejection) Unwrap() []error { return []error{ErrReconcileRejected, e.Err} }

// DecisionOptions carries the optional actor/note metadata for an approval
// decision. Both fields are optional: when empty, no corresponding flag is
// forwarded, so the invocation stays exactly `sop approve <task-id>` /
// `sop decline <task-id>`. They are forwarded through the argv-slice Commander
// (never a shell string), so an untrusted actor or note value cannot be
// interpreted by a shell.
type DecisionOptions struct {
	// By is the actor recorded with the decision (`--by NAME`). Empty means the
	// flag is omitted and SOP records its own default actor.
	By string
	// Note is an optional free-form note recorded with the decision
	// (`--note TEXT`). Empty means the flag is omitted.
	Note string
}

// decisionArgs builds the argv for an approve/decline command: the task id as a
// single argv element, then `--by`/`--note` only when supplied. It never emits
// `--run`: recording a decision is separate from executing work, and the
// controller keeps that separation so approve does not start a run.
func decisionArgs(taskID string, opts DecisionOptions) []string {
	args := []string{taskID}
	if opts.By != "" {
		args = append(args, "--by", opts.By)
	}
	if opts.Note != "" {
		args = append(args, "--note", opts.Note)
	}
	return args
}

// Operation names one conceptual controller-to-SOP operation from the CTRL001
// contract (docs/PLAN-SOP-Controller.md).
type Operation string

const (
	OpGetPlan            Operation = "GetPlan"
	OpGetTasks           Operation = "GetTasks"
	OpGetTask            Operation = "GetTask"
	OpGetTaskActivity    Operation = "GetTaskActivity"
	OpGetTaskProgress    Operation = "GetTaskProgress"
	OpGetTaskReport      Operation = "GetTaskReport"
	OpStartOrContinueRun Operation = "StartOrContinueRun"
	OpRetryTask          Operation = "RetryTask"
	OpCancelRun          Operation = "CancelRun"
	OpApproveTask        Operation = "ApproveTask"
	OpDeclineTask        Operation = "DeclineTask"
	OpReconcilePlan      Operation = "ReconcilePlan"
	// CTRL012 reconcile controls. OpGetChangedExecutedTasks is the read of the
	// set of changed executed tasks awaiting reconcile, from SOP's authoritative
	// listing (`sop reconcile <PLAN.md> --list-changed --json`, C2-003).
	// OpAcceptChangedTask is the per-task accept-changed application operation
	// (C2-004): it delegates to SOP via `sop reconcile <PLAN.md> --accept-changed
	// <TASK_ID>` (the flag repeated once per explicitly selected id, in a single
	// invocation), so SOP's own atomic reconciliation semantics are preserved and
	// the controller never simulates or bulk-applies an acceptance.
	OpGetChangedExecutedTasks Operation = "GetChangedExecutedTasks"
	OpAcceptChangedTask       Operation = "AcceptChangedTask"
	// C2-001 authoritative approval read: OpGetApprovals is the read of SOP's
	// structured approval listing (`sop approvals --json`). It is the sole source
	// of truth for approval presence and applicability; the controller never
	// infers a gate from run classification, run stage, or BLOCKED status.
	OpGetApprovals Operation = "GetApprovals"
	// OpGetTaskPerformance is the read-only performance read: SOP's own
	// persisted measurement of where a task - and, at plan scope, a run - spent
	// its time and how many agent/validation/review/fix operations it cost. It
	// reads SOP's metrics artifacts verbatim; the controller starts no timer and
	// computes no lifecycle duration. Performance is diagnostic metadata only: it
	// never influences task status, selection, retry, recovery, approval, or
	// routing.
	OpGetTaskPerformance Operation = "GetTaskPerformance"
)

// OperationStatus classifies whether the boundary can serve an operation.
type OperationStatus string

const (
	// StatusSupported: the operation maps to an existing SOP application
	// operation and is served by a sopclient entry point.
	StatusSupported OperationStatus = "supported"
	// StatusUnsupported: SOP exposes no application operation for this
	// operation, so the boundary records an explicit gap instead of simulating
	// SOP behaviour in the controller.
	StatusUnsupported OperationStatus = "unsupported"
)

// Descriptor documents one conceptual controller operation: the sopclient entry
// point that serves it and the SOP application operation that entry point uses.
// It is data, not behaviour; the controller derives no orchestration decision
// from it.
type Descriptor struct {
	// Operation is the conceptual PRD operation name.
	Operation Operation
	// EntryPoint is the sopclient method (or methods, comma-separated) that
	// serves the operation. Empty when Status is StatusUnsupported.
	EntryPoint string
	// SOPOperation is the SOP application operation the entry point uses: a read
	// of SOP-persisted state/artifacts, or a sop CLI verb.
	SOPOperation string
	// SOPVerbs lists the sop CLI verbs the operation drives (command operations
	// only); empty for read operations.
	SOPVerbs []string
	// Status is supported or unsupported.
	Status OperationStatus
	// Reason explains an unsupported gap. Empty when supported.
	Reason string
}

// Gap reasons, recorded once so the descriptor (Boundary/Lookup) and the runtime
// error message cannot drift apart.
const (
	reasonCancelRun = "SOP exposes no cancellation application operation (`sop cancel` does not exist); cancelling an active run is a SOP lifecycle decision the controller must not simulate"
)

// The CTRL011 / C2-001 human approval boundary reads SOP's OWN approval listing
// (`sop approvals --json`, OpGetApprovals). SOP owns approval and every lifecycle
// gate; the controller only reports the gates SOP reports in that structured
// listing, and the present-or-absent projection rides on the existing GetTask
// read (TaskDetail.Approval) plus the shared list read (TaskSummary.NeedsHuman).
//
// Approval ACTIONS (OpApproveTask / OpDeclineTask, C2-002) now delegate to SOP's
// own application operations (`sop approve` / `sop decline`). SOP is still the
// authority: it validates the gate at command time, so a stale/not-applicable
// decision is rejected by SOP and reported verbatim rather than manufactured or
// pre-judged by the controller. No `--run` is ever passed, keeping the decision
// separate from execution, and the controller writes no approval state of its
// own, marks no task complete, and never converts a decline into a failure.

// The CTRL012 reconcile controls follow the same rule. Reconciliation stays
// entirely inside SOP (`sop reconcile <PLAN.md>`); the controller never diffs the
// plan, never edits state.db, and never applies a per-task approval itself.
// Reading the changed executed tasks (OpGetChangedExecutedTasks) IS supported and
// is a PURE READ: SOP reports its own changed-executed-task set through its
// authoritative listing (`sop reconcile <PLAN.md> --list-changed --json`, C2-003),
// and ChangedTasks resolves the plan from SOP's recorded provenance and decodes
// that listing verbatim. The controller computes no diff of its own and never
// reads the retired .agent-sdlc/reconcile.json artifact.
//
// Accepting changed tasks (OpAcceptChangedTask, C2-004) is now also supported and
// DELEGATED to SOP: Client.AcceptChangedTask(s) validates every precondition
// against SOP's authoritative listing (unknown project, no active plan,
// unreported set, task not in the reported set) BEFORE any SOP call, then runs a
// SINGLE invocation `sop reconcile <PLAN.md> --accept-changed <id> ...` with only
// the ids the human explicitly selected (the flag repeated once per id, so SOP's
// own atomic reconciliation semantics are preserved rather than a controller loop
// of partial reconciliations). The controller never bulk-accepts, never treats
// viewing a change as accepting it, never simulates the operation, and never
// writes SOP state. The descriptor records the PRD's declared SOP syntax; it does
// not assert that the external binary implements it, so a real SOP-side gap
// surfaces as an honest runtime failure - reported truthfully as a
// *ReconcileRejection - rather than being pre-judged by the controller, and
// availability stays derived from Boundary() so the gap stays discoverable.

// Boundary returns the documented controller-to-SOP operation contract in the
// PRD's order. It is the single source of truth shared by the contract document
// (docs/history/CTRL001/BOUNDARY-CONTRACT.md) and the boundary tests.
//
// Every entry either maps to an existing SOP application operation or records an
// explicit gap. Nothing here lets the controller mutate SOP persistence or make
// a scheduling decision: reads report SOP's own values, and commands delegate to
// sop CLI verbs.
func Boundary() []Descriptor {
	return []Descriptor{
		{
			Operation:    OpGetPlan,
			EntryPoint:   "PlanSource",
			SOPOperation: "read .agent-sdlc/plan.meta.json (SOP's recorded active plan source)",
			Status:       StatusSupported,
		},
		{
			Operation:    OpGetTasks,
			EntryPoint:   "Project",
			SOPOperation: "read tasks + task_dependencies from state.db",
			Status:       StatusSupported,
		},
		{
			Operation:    OpGetTask,
			EntryPoint:   "Task",
			SOPOperation: "read tasks, task_attempts, handoffs from state.db; optional SOP-reported checkpoint/bounded-progress line from .agent-sdlc/runs/<task>/checkpoint.json; the SOP-reported human approval gate (TaskDetail.Approval) is projected from SOP's approval listing (see GetApprovals), not computed here",
			Status:       StatusSupported,
		},
		{
			Operation:    OpGetTaskActivity,
			EntryPoint:   "Activity, Task",
			SOPOperation: "read .agent-sdlc/runs/<task>/activity.jsonl",
			Status:       StatusSupported,
		},
		{
			Operation:    OpGetTaskProgress,
			EntryPoint:   "Project",
			SOPOperation: "read task statuses from state.db (ProjectDetail.Summary/PercentComplete)",
			Status:       StatusSupported,
		},
		{
			Operation:    OpGetApprovals,
			EntryPoint:   "Approvals, Tasks, Task",
			SOPOperation: "sop approvals --json (SOP's structured approval listing; read back verbatim from .agent-sdlc/approvals.json, present-or-absent; the controller never reconstructs a gate)",
			SOPVerbs:     []string{"approvals"},
			Status:       StatusSupported,
		},
		{
			Operation:    OpGetTaskReport,
			EntryPoint:   "ReportTask",
			SOPOperation: "sop report <task>",
			SOPVerbs:     []string{"report"},
			Status:       StatusSupported,
		},
		{
			Operation:    OpStartOrContinueRun,
			EntryPoint:   "Run, Resume",
			SOPOperation: "sop run / sop resume",
			SOPVerbs:     []string{"run", "resume"},
			Status:       StatusSupported,
		},
		{
			Operation:    OpRetryTask,
			EntryPoint:   "Retry, RetryForce, RetryAll",
			SOPOperation: "sop retry <task> / sop retry <task> --force / sop retry --all",
			SOPVerbs:     []string{"retry"},
			Status:       StatusSupported,
		},
		{
			Operation: OpCancelRun,
			Status:    StatusUnsupported,
			Reason:    reasonCancelRun,
		},
		{
			Operation:    OpApproveTask,
			EntryPoint:   "ApproveTask",
			SOPOperation: "sop approve <task-id> [--by NAME] [--note TEXT] (SOP validates the gate at command time; the controller passes no --run and writes no approval state of its own)",
			SOPVerbs:     []string{"approve"},
			Status:       StatusSupported,
		},
		{
			Operation:    OpDeclineTask,
			EntryPoint:   "DeclineTask",
			SOPOperation: "sop decline <task-id> [--by NAME] [--note TEXT] (SOP validates the gate at command time; the controller passes no --run and manufactures no failure)",
			SOPVerbs:     []string{"decline"},
			Status:       StatusSupported,
		},
		{
			Operation:    OpReconcilePlan,
			EntryPoint:   "Reconcile",
			SOPOperation: "sop reconcile <PLAN.md> (SOP validates before it mutates and preserves unchanged tasks; the controller never edits state.db)",
			SOPVerbs:     []string{"reconcile"},
			Status:       StatusSupported,
		},
		{
			Operation:    OpGetChangedExecutedTasks,
			EntryPoint:   "ChangedTasks",
			SOPOperation: "sop reconcile <PLAN.md> --list-changed --json (SOP's authoritative changed-executed-task listing; the plan path comes from SOP's recorded plan.meta.json provenance; a pure read the controller decodes verbatim and never computes a diff for)",
			SOPVerbs:     []string{"reconcile"},
			Status:       StatusSupported,
		},
		{
			Operation:    OpAcceptChangedTask,
			EntryPoint:   "AcceptChangedTasks, AcceptChangedTask",
			SOPOperation: "sop reconcile <PLAN.md> --accept-changed <TASK_ID> (repeated --accept-changed once per explicitly selected task id, in a single invocation, so SOP's own atomic reconciliation semantics are preserved; the plan path comes from SOP's recorded plan.meta.json provenance; the controller never diffs the plan, never bulk-accepts, and writes no SOP state)",
			SOPVerbs:     []string{"reconcile"},
			Status:       StatusSupported,
		},
		{
			Operation:    OpGetTaskPerformance,
			EntryPoint:   "Performance, Task, Project",
			SOPOperation: "read .agent-sdlc/runs/<task>/metrics.json (SOP's per-task performance record; report.json's performance field is the fallback) and .agent-sdlc/runs/<plan-id>/metrics.json (the plan-level aggregate, keyed by SOP's recorded plan.meta.json plan_id). Both are SOP-produced diagnostic metadata read verbatim; the controller starts no timer and computes no lifecycle duration, and performance never influences task status, selection, retry, recovery, approval, or routing",
			Status:       StatusSupported,
		},
	}
}

// Lookup returns the descriptor for one operation and whether it is known.
func Lookup(op Operation) (Descriptor, bool) {
	for _, d := range Boundary() {
		if d.Operation == op {
			return d, true
		}
	}
	return Descriptor{}, false
}

// unsupported builds the error for a recorded gap, reusing the descriptor's
// Reason so the error text and Boundary()/Lookup() cannot drift apart.
func unsupported(op Operation) error {
	reason := "SOP exposes no application operation for this operation"
	if d, ok := Lookup(op); ok && d.Reason != "" {
		reason = d.Reason
	}
	return fmt.Errorf("%s: %w (%s)", op, ErrOperationUnsupported, reason)
}

// CancelRun is the CTRL001 CancelRun boundary operation. SOP exposes no
// cancellation application operation, so this reports ErrOperationUnsupported
// rather than stopping a run from the controller: deciding to stop an active run
// is SOP's lifecycle authority, not the controller's.
//
// It validates the project id first so an unknown project yields the same
// ErrProjectNotFound sentinel as every other boundary operation; the gap is
// reported only for a project that exists.
func (c *Client) CancelRun(ctx context.Context, projectID string) error {
	if _, ok := c.stores[projectID]; !ok {
		return ErrProjectNotFound
	}
	return unsupported(OpCancelRun)
}

// ApproveTask is the CTRL001 ApproveTask boundary operation, and the application
// boundary the controller's approval action invokes (CTRL011, C2-002). It
// delegates the decision to SOP (`sop approve <task-id> [--by NAME] [--note
// TEXT]`) through the argv-slice Commander, so the task id and any actor/note are
// passed as discrete argv elements and are never shell-interpreted.
//
// SOP owns the gate: it validates the decision at command time, so the
// controller does not pre-judge eligibility from its own read. When SOP rejects
// the decision (for example a stale/not-applicable gate) the returned error is a
// *DecisionRejection carrying SOP's message and the task id, which callers
// classify as an actionable conflict rather than an infrastructure 500 or a
// fabricated success. No `--run` is passed: approving records a decision and does
// not start execution. The controller writes no SOP state and marks no task
// complete.
//
// It validates the project id first so an unknown project yields the same
// ErrProjectNotFound sentinel as every other boundary operation.
func (c *Client) ApproveTask(ctx context.Context, projectID, taskID string) error {
	return c.ApproveTaskWithOptions(ctx, projectID, taskID, DecisionOptions{})
}

// ApproveTaskWithOptions is ApproveTask carrying optional actor/note metadata,
// forwarded as `--by` and `--note` only when supplied.
func (c *Client) ApproveTaskWithOptions(ctx context.Context, projectID, taskID string, opts DecisionOptions) error {
	return c.decide(ctx, projectID, "approve", taskID, opts)
}

// DeclineTask is the CTRL011 decline/withhold counterpart of ApproveTask. It is
// the same human approval gate, only answered negatively, but it is a distinct
// boundary operation (OpDeclineTask) so an operator or test can distinguish a
// failed decline from a failed approve. It delegates to SOP's own decline
// operation (`sop decline <task-id> [--by NAME] [--note TEXT]`), which decides
// whether the gate still applies.
//
// Withholding approval must leave SOP's task state exactly as SOP reported it:
// the controller never writes an approval decision of its own, fabricates a task
// as approved, or manufactures a failure. A successful decline records no
// controller-side failure state, and a SOP rejection is surfaced verbatim.
//
// It validates the project id first so an unknown project yields the same
// ErrProjectNotFound sentinel as every other boundary operation.
func (c *Client) DeclineTask(ctx context.Context, projectID, taskID string) error {
	return c.DeclineTaskWithOptions(ctx, projectID, taskID, DecisionOptions{})
}

// DeclineTaskWithOptions is DeclineTask carrying optional actor/note metadata,
// forwarded as `--by` and `--note` only when supplied.
func (c *Client) DeclineTaskWithOptions(ctx context.Context, projectID, taskID string, opts DecisionOptions) error {
	return c.decide(ctx, projectID, "decline", taskID, opts)
}

// AcceptChangedTask is the CTRL012 per-task accept-changed application
// operation (C2-004): the `--accept-changed` equivalent the controller must
// obtain explicitly, one changed task at a time, before reconciliation mutates.
// It is a one-id convenience that delegates to AcceptChangedTasks.
//
// Every precondition is checked, in order, before anything else: an unknown
// project yields ErrProjectNotFound; no active plan yields ErrNoActivePlan; no
// SOP-reported changed set yields ErrChangedTasksNotReported; a task absent
// from SOP's reported set yields ErrTaskNotInChangedSet. The reported set is
// read from SOP's authoritative listing (`sop reconcile <PLAN.md>
// --list-changed --json`, C2-003), the same pure read Client.ChangedTasks
// exposes, so a precondition can never be fed by the retired reconcile.json
// artifact. An approval naming an unknown or unrelated task is therefore
// rejected before any SOP call could ever be attempted. Only once all of those
// hold does it delegate to SOP. The controller never edits state.db and never
// bulk-accepts: each changed task must be named explicitly by the caller.
func (c *Client) AcceptChangedTask(ctx context.Context, projectID, taskID string) error {
	return c.AcceptChangedTasks(ctx, projectID, []string{taskID})
}

// AcceptChangedTasks is the C2-004 batch form of AcceptChangedTask: it accepts
// the explicitly selected changed task ids in a SINGLE SOP invocation,
// `sop reconcile <PLAN.md> --accept-changed <id> --accept-changed <id> ...`, so
// SOP's own atomic reconciliation semantics are preserved rather than being
// replaced by a controller loop of partial reconciliations.
//
// Precondition order (all BEFORE any SOP call): unknown project ->
// ErrProjectNotFound; no active plan -> ErrNoActivePlan; a failed/unreported
// listing -> surfaced / ErrChangedTasksNotReported; any id absent from SOP's
// reported changed set -> ErrTaskNotInChangedSet (checked for every id before the
// invocation, so a batch is all-or-nothing at the precondition stage). Ids are
// normalized first (trimmed, empties dropped, duplicates removed) so the argv
// carries each explicitly selected id exactly once; an empty selection yields
// ErrNoExplicitTaskIDs, so no invocation can be made with nothing selected.
//
// It passes the ids as discrete argv elements through the Commander (never a
// shell string), so an untrusted id cannot be shell-interpreted. On a non-zero
// SOP result it returns a *ReconcileRejection carrying SOP's message verbatim and
// the affected ids: the failure is reported truthfully, never fabricated as
// success and never written to SOP state. The controller never accepts a task
// the caller did not explicitly select, and never treats viewing a change as
// accepting it.
func (c *Client) AcceptChangedTasks(ctx context.Context, projectID string, taskIDs []string) error {
	st, ok := c.stores[projectID]
	if !ok {
		return ErrProjectNotFound
	}
	planPath, ok := c.PlanSource(projectID)
	if !ok {
		return ErrNoActivePlan
	}
	changed, err := c.ChangedTasks(ctx, projectID)
	if err != nil {
		// A failed/unreported listing is surfaced as such, never treated as an
		// empty set that could be silently accepted.
		return err
	}
	if !changed.Reported {
		return ErrChangedTasksNotReported
	}

	ids := normalizeTaskIDs(taskIDs)
	if len(ids) == 0 {
		return ErrNoExplicitTaskIDs
	}
	// Every selected id must be in SOP's reported changed set BEFORE any SOP
	// call: a batch is validated in full, so no partial mutation can occur.
	for _, id := range ids {
		if !changed.Has(id) {
			return ErrTaskNotInChangedSet
		}
	}

	out, err := c.cmd.Exec(ctx, st.root, "reconcile", acceptChangedArgs(planPath, ids)...)
	if err != nil {
		return &ReconcileRejection{Verb: "reconcile", TaskIDs: ids, Message: decisionMessage(out, err), Err: err}
	}
	return nil
}

// acceptChangedArgs builds the argv for a batch accept-changed invocation: the
// plan path, then `--accept-changed <id>` once per explicitly selected id, all as
// discrete argv elements (never a shell string).
func acceptChangedArgs(planPath string, ids []string) []string {
	args := []string{planPath}
	for _, id := range ids {
		args = append(args, "--accept-changed", id)
	}
	return args
}

// normalizeTaskIDs trims, drops empty entries, and de-duplicates the selected
// task ids while preserving first-seen order, so the argv carries each
// explicitly selected id exactly once.
func normalizeTaskIDs(taskIDs []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, raw := range taskIDs {
		id := strings.TrimSpace(raw)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

// ApprovalOperations reports whether SOP exposes an application operation to
// apply (approve) and to decline/withhold a human approval gate. The controller
// uses it to decide whether an approval control may be offered at all: a control
// whose only possible outcome is an unsupported error must never be shown. It
// derives the answer from the single Boundary() source of truth, so it cannot
// drift from the descriptors or the runtime gap errors.
func ApprovalOperations() (approve, decline bool) {
	if d, ok := Lookup(OpApproveTask); ok {
		approve = d.Status == StatusSupported
	}
	if d, ok := Lookup(OpDeclineTask); ok {
		decline = d.Status == StatusSupported
	}
	return approve, decline
}

// CancelOperations reports whether SOP exposes an application operation to
// cancel/stop an active run (CTRL006). The controller uses it to decide
// whether a Stop control may be offered at all: a control whose only possible
// outcome is an unsupported error must never be shown as if it could act. It
// derives the answer from the single Boundary() source of truth, so it cannot
// drift from the descriptor or the runtime gap error CancelRun returns. Today
// that descriptor is StatusUnsupported (SOP has no `sop cancel` verb yet), so
// this returns false; the moment Boundary() records OpCancelRun as
// StatusSupported, this flips with no code change, by construction rather
// than by a hardcoded literal. cancelSupported carries the derivation so it
// can be exercised against both descriptor states, not just today's fixed one.
func CancelOperations() (cancel bool) {
	d, ok := Lookup(OpCancelRun)
	return cancelSupported(d, ok)
}

func cancelSupported(d Descriptor, found bool) bool {
	return found && d.Status == StatusSupported
}

// ReconcileOperations reports whether SOP exposes the CTRL012 reconcile-control
// operations: a structured read of the changed executed tasks
// (OpGetChangedExecutedTasks) and a per-task accept-changed application
// operation (OpAcceptChangedTask). The controller uses it to decide whether to
// offer the changed-task list and per-task approval at all: a control whose only
// possible outcome is an unsupported error must never be shown. Both are true
// today (SOP's authoritative `reconcile <PLAN.md> --list-changed --json` listing
// read works; C2-004 accepts changed tasks via `reconcile <PLAN.md>
// --accept-changed <id>`). Like ApprovalOperations it derives the answer from the
// single Boundary() source of truth, so it cannot drift from the descriptors or
// the runtime behaviour: if SOP later reports a real gap, the descriptor change
// flips this by construction.
func ReconcileOperations() (listChanged, acceptChanged bool) {
	dList, okList := Lookup(OpGetChangedExecutedTasks)
	dAccept, okAccept := Lookup(OpAcceptChangedTask)
	return reconcileOperationsFromDescriptors(dList, dAccept, okList, okAccept)
}

// reconcileOperationsFromDescriptors carries the ReconcileOperations derivation so
// it can be exercised against both descriptor states (supported and unsupported),
// not just today's fixed one; the flip is by construction rather than a hardcoded
// literal.
func reconcileOperationsFromDescriptors(list, accept Descriptor, listFound, acceptFound bool) (listChanged, acceptChanged bool) {
	return listFound && list.Status == StatusSupported, acceptFound && accept.Status == StatusSupported
}

// ApprovalsOperations reports whether SOP exposes the authoritative approval
// listing read (OpGetApprovals). The controller uses it to decide whether to
// offer the approval surface at all: when SOP exposes no listing, the controller
// reports an explicit absence and reconstructs no gate. It derives the answer
// from the single Boundary() source of truth, so it cannot drift from the
// descriptor.
func ApprovalsOperations() (list bool) {
	d, ok := Lookup(OpGetApprovals)
	return ok && d.Status == StatusSupported
}

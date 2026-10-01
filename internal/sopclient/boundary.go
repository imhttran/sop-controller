package sopclient

import (
	"context"
	"errors"
	"fmt"
)

// ErrOperationUnsupported is returned by a boundary operation that has no
// corresponding SOP application operation. The controller must not simulate the
// missing SOP lifecycle operation itself: SOP owns the lifecycle, so a gap is
// reported rather than filled with controller-side orchestration.
//
// Callers test for it with errors.Is; the recorded reason is available from
// Boundary and Lookup.
var ErrOperationUnsupported = errors.New("operation unsupported by SOP")

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
	// set of changed executed tasks awaiting reconcile; OpAcceptChangedTask is
	// the per-task accept-changed application operation (the `--accept-changed`
	// equivalent). Both are recorded as explicit gaps: SOP exposes no structured
	// pre-mutation read of the changed set and no per-task accept-changed
	// application operation, so the controller must not compute or apply either.
	OpGetChangedExecutedTasks Operation = "GetChangedExecutedTasks"
	OpAcceptChangedTask       Operation = "AcceptChangedTask"
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
	reasonCancelRun     = "SOP exposes no cancellation application operation (`sop cancel` does not exist); cancelling an active run is a SOP lifecycle decision the controller must not simulate"
	reasonApproveTask   = "SOP exposes no approval application operation (`sop approve` does not exist); human approval is a SOP lifecycle gate the controller must not bypass"
	reasonDeclineTask   = "SOP exposes no decline/withhold application operation (`sop decline` does not exist); declining a human approval gate is a SOP lifecycle decision the controller must not simulate"
	reasonAcceptChanged = "SOP exposes no per-task accept-changed application operation (`sop reconcile --accept-changed <task>` does not exist); accepting a changed executed task is a SOP human-boundary decision the controller must not simulate or bulk-approve"
)

// The CTRL011 human approval boundary is NOT a separate read operation. SOP owns
// approval and every lifecycle gate; the controller only reports the boundary SOP
// persisted, and that optional present-or-absent projection rides on the existing
// GetTask read (TaskDetail.Approval). Any approval ACTION - approve or decline -
// is a distinct boundary operation (OpApproveTask / OpDeclineTask), each of which
// is an explicit unsupported gap here: SOP exposes no approval application
// operation, so the controller never manufactures one and the UI never offers a
// control that cannot apply.

// The CTRL012 reconcile controls follow the same rule. Reconciliation stays
// entirely inside SOP (`sop reconcile <PLAN.md>`); the controller never diffs the
// plan, never edits state.db, and never applies a per-task approval itself.
// Reading the changed executed tasks (OpGetChangedExecutedTasks) IS supported:
// SOP optionally writes its own changed-set report (.agent-sdlc/reconcile.json),
// and ChangedTasks reads it back verbatim, present-or-absent, same as any other
// SOP-persisted artifact. Accepting one changed task (OpAcceptChangedTask) is
// NOT: SOP exposes no `--accept-changed`-equivalent application operation, so it
// is recorded as an explicit gap until SOP exposes one, and the UI must never
// offer a control whose only outcome is that gap.

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
			SOPOperation: "read tasks, task_attempts, handoffs from state.db; optional SOP-reported checkpoint/bounded-progress line from .agent-sdlc/runs/<task>/checkpoint.json and optional SOP-reported human approval boundary (TaskDetail.Approval) from SOP-persisted task status/classification/run stage (present-or-absent; the controller never computes either)",
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
			Operation: OpApproveTask,
			Status:    StatusUnsupported,
			Reason:    reasonApproveTask,
		},
		{
			Operation: OpDeclineTask,
			Status:    StatusUnsupported,
			Reason:    reasonDeclineTask,
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
			SOPOperation: "read optional .agent-sdlc/reconcile.json (SOP's own changed-executed-task report, present-or-absent; the controller computes no diff of its own)",
			Status:       StatusSupported,
		},
		{
			Operation: OpAcceptChangedTask,
			Status:    StatusUnsupported,
			Reason:    reasonAcceptChanged,
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
// boundary the controller's approval action invokes (CTRL011). SOP exposes no
// approval application operation, so this reports ErrOperationUnsupported rather
// than approving work from the controller: human approval is a SOP lifecycle
// gate, and the controller must not manufacture it.
//
// It validates the project id first so an unknown project yields the same
// ErrProjectNotFound sentinel as every other boundary operation; the gap is
// reported for any existing project, since SOP has no approval operation to
// check eligibility against. The controller pre-gates this ACTION on whether SOP
// reports an approval boundary (so it is only ever invoked where SOP requests
// it), but the decision itself is delegated to SOP: it delegates the ACTION and
// reports SOP's answer (here, that no such operation exists), so no
// controller-side decision can substitute for SOP's.
func (c *Client) ApproveTask(ctx context.Context, projectID, taskID string) error {
	if _, ok := c.stores[projectID]; !ok {
		return ErrProjectNotFound
	}
	return unsupported(OpApproveTask)
}

// DeclineTask is the CTRL011 decline/withhold counterpart of ApproveTask. It is
// the same human approval gate, only answered negatively, but it is a distinct
// boundary operation (OpDeclineTask) so an operator or test can distinguish a
// failed decline from a failed approve. SOP exposes no decline application
// operation either, so this reports ErrOperationUnsupported rather than
// recording a decline itself. Withholding approval must leave SOP's task state
// exactly as SOP reported it, and the controller never writes an approval
// decision of its own nor fabricates a task as approved.
//
// It validates the project id first so an unknown project yields the same
// ErrProjectNotFound sentinel as every other boundary operation.
func (c *Client) DeclineTask(ctx context.Context, projectID, taskID string) error {
	if _, ok := c.stores[projectID]; !ok {
		return ErrProjectNotFound
	}
	return unsupported(OpDeclineTask)
}

// AcceptChangedTask is the CTRL012 per-task accept-changed application
// operation: the `--accept-changed` equivalent the controller must obtain
// explicitly, one changed task at a time, before reconciliation mutates.
//
// Every precondition is checked, in order, before anything else: an unknown
// project yields ErrProjectNotFound; no active plan yields ErrNoActivePlan; no
// SOP-reported changed set yields ErrChangedTasksNotReported; a task absent
// from SOP's reported set yields ErrTaskNotInChangedSet. An approval naming an
// unknown or unrelated task is therefore rejected before any mutation could
// ever be attempted. Only once all of those hold does it reach the action
// itself - and SOP exposes no such per-task application operation yet, so it
// reports ErrOperationUnsupported rather than applying or simulating an
// approval. The controller never edits state.db and never bulk-approves: each
// changed task needs its own call here, and none can ever apply until SOP
// exposes the operation.
func (c *Client) AcceptChangedTask(ctx context.Context, projectID, taskID string) error {
	st, ok := c.stores[projectID]
	if !ok {
		return ErrProjectNotFound
	}
	if _, ok := st.PlanSource(); !ok {
		return ErrNoActivePlan
	}
	changed := st.ChangedTasks()
	if !changed.Reported {
		return ErrChangedTasksNotReported
	}
	if !changed.Has(taskID) {
		return ErrTaskNotInChangedSet
	}
	return unsupported(OpAcceptChangedTask)
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
// possible outcome is an unsupported error must never be shown. listChanged is
// true today (SOP's optional reconcile.json read works); acceptChanged is false
// until SOP exposes a real per-task accept-changed application operation. Like
// ApprovalOperations it derives the answer from the single Boundary() source of
// truth, so it cannot drift from the descriptors or the runtime gap errors.
func ReconcileOperations() (listChanged, acceptChanged bool) {
	if d, ok := Lookup(OpGetChangedExecutedTasks); ok {
		listChanged = d.Status == StatusSupported
	}
	if d, ok := Lookup(OpAcceptChangedTask); ok {
		acceptChanged = d.Status == StatusSupported
	}
	return listChanged, acceptChanged
}

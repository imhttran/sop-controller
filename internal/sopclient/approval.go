package sopclient

import "strings"

// This file defines the controller-facing approval boundary (CTRL011): the
// optional, present-or-absent signal by which SOP reports that it is waiting at
// an actual human approval boundary for a task.
//
// Ownership rule: SOP owns approval and every lifecycle gate. The controller
// never decides that approval is required, never fabricates an approval, and
// never approves work itself. It only reports the boundary SOP already recorded
// and delegates any approval ACTION to SOP's own application boundary. When SOP
// records no approval boundary, this model reports Present=false and no approval
// action may be taken.
//
// The boundary is derived ONLY from SOP-persisted evidence, and every branch
// requires an explicit SOP-recorded HUMAN signal:
//   - SOP's run classification disposition is NEEDS_HUMAN, or
//   - SOP's run classification Kind is a human kind (CategoryHuman), or
//   - SOP's run stage is WAITING_FOR_HUMAN, or
//   - SOP's task status is BLOCKED *and* SOP recorded a human classification
//     (a human Kind or the NEEDS_HUMAN disposition).
//
// A BLOCKED task is NOT treated as an approval boundary merely because it
// carries a non-empty blocked_reason: SOP's blocked_reason is frequently a
// non-human reason (a dependency waiting, an environment/tooling problem), so
// inferring a human gate from it would let the controller surface an approval
// boundary SOP never requested. The dependency-only BLOCKED case is ordinary
// scheduling that SOP resolves by running the dependency, and is therefore
// excluded. The boundary is never derived from retries, attempt counts, activity
// wording, the plan gate, or any other controller-side heuristic.

// Approval kinds name the SOP-reported source of a human approval boundary.
// They are display labels for the evidence SOP persisted; the controller does
// not choose among them.
const (
	// ApprovalKindBlocked: SOP's task status is BLOCKED with a recorded human
	// classification (a human Kind or the NEEDS_HUMAN disposition).
	ApprovalKindBlocked = "BLOCKED"
	// ApprovalKindNeedsHuman: SOP's run classification disposition is
	// NEEDS_HUMAN.
	ApprovalKindNeedsHuman = "NEEDS_HUMAN"
	// ApprovalKindWaitingForHuman: SOP parked the run at the WAITING_FOR_HUMAN
	// lifecycle stage.
	ApprovalKindWaitingForHuman = "WAITING_FOR_HUMAN"
)

// Approval is the controller-facing projection of a SOP-reported human approval
// boundary for one task. It is read-only and render-ready.
//
// Present is the source of truth for whether SOP reports a human boundary at
// all; it is true only when SOP-persisted state reports one. Whether a control
// may be offered is tracked PER ACTION by ApproveApplicable and
// DeclineApplicable: approve and decline are two independent boundary
// operations, so a future state where SOP exposes one but not the other must
// still offer the operation that exists rather than suppressing both. A control
// is shown only when its own operation is exposed; the gate itself is always
// shown as a read-only report. When SOP reports a boundary but exposes no
// operation (the CTRL011 unsupported gap), no control is offered, so the UI
// never advertises an action whose only possible outcome is an unsupported
// error. Every free-form text field is passed through sanitizeDetail so no
// consumer can surface a prompt, secret, or dump. FinalGate reports SOP's
// recorded final plan gate verbatim when SOP persisted plan metadata; it is
// never inferred.
type Approval struct {
	// Present is true when SOP reports an actual approval boundary for this task.
	// It is false when SOP reports none. Kind/Evidence are empty when Present is
	// false; Target and Stage may still carry the task id and SOP's persisted run
	// stage so a renderer can show context, but they are never an absence signal.
	// Key an absence check off Present, not off Target being empty.
	Present bool
	// ApproveApplicable is true when an APPROVE control may be offered: the
	// boundary is Present and SOP exposes the approve application operation. It is
	// derived from Boundary() (via ApprovalOperations), never inferred from the
	// gate's shape, so it can never drift from the recorded operation gap.
	ApproveApplicable bool
	// DeclineApplicable is the independent counterpart for the decline/withhold
	// control. It is tracked separately from ApproveApplicable because approve and
	// decline are distinct boundary operations: exposing one must not suppress the
	// other.
	DeclineApplicable bool
	// Applicable is true when at least one approval control may be offered (the
	// boundary is Present and SOP exposes at least one of the approve/decline
	// operations). It is a convenience for callers that only need "is any action
	// available"; per-action rendering must use ApproveApplicable/DeclineApplicable
	// so one existing operation is never hidden behind a missing counterpart.
	Applicable bool
	// ActionReason explains, when a boundary is Present but not fully Applicable,
	// why a control is (or operations are) unavailable. It is built from the
	// recorded unsupported-operation reasons so the explanation cannot drift from
	// the boundary contract. It names every missing operation (approve and/or
	// decline), so it states exactly which action is missing. Empty when the
	// boundary is not Present.
	ActionReason string
	// Kind is the SOP-reported boundary source (one of the ApprovalKind*
	// constants), or "" when Present is false.
	Kind string
	// Target is SOP's own identifier for what the boundary concerns (the task id
	// SOP persisted). It is set whether or not Present is true; use Present (not
	// an empty Target) to decide whether an approval boundary is reported.
	Target string
	// Evidence is the SOP-reported reason/detail behind the boundary (SOP's own
	// classification reason), sanitized for safe display.
	Evidence string
	// Stage is SOP's persisted run lifecycle stage for the task, verbatim.
	Stage string
	// Disposition is SOP's persisted classification disposition, verbatim.
	Disposition string
	// FinalGate is SOP's recorded final plan gate value when SOP persisted plan
	// metadata, or StatusUnknown. It is never inferred.
	FinalGate string
	// FinalGateRecorded is true when SOP persisted plan metadata carrying the
	// gate value.
	FinalGateRecorded bool
}

// humanBoundary derives whether SOP reports a human decision boundary for a
// task, and which ApprovalKind evidences it, from the task's status and run
// info alone (SOP's run classification disposition/Kind, run stage, and task
// status). It is the single derivation both Store.approval() (TaskDetail) and
// Store.Tasks() (TaskSummary) call, so the task-detail approval panel and the
// task-list/project-hero projections can never disagree about which tasks need
// a human decision. See the file doc above for the evidence rule itself; this
// function only implements it.
func humanBoundary(status string, run RunInfo) (present bool, kind string) {
	switch {
	case run.Classification.HumanRequired():
		return true, ApprovalKindNeedsHuman
	case run.Stage == StageWaitingForHuman:
		return true, ApprovalKindWaitingForHuman
	case status == StatusBlocked && run.Classification.Category() == CategoryHuman:
		return true, ApprovalKindBlocked
	default:
		return false, ""
	}
}

// approval derives the approval boundary for a task from SOP-persisted evidence
// only. It returns Present=false when SOP reports no human boundary, so an
// approval action can only ever appear when SOP itself requested one. It also
// records, per action, whether the controller can actually apply the gate
// (ApproveApplicable / DeclineApplicable), derived from the boundary contract's
// operation statuses, so a reported-but-unsupported gate is never rendered as a
// control that can only fail and a supported action is never suppressed by a
// missing counterpart.
func (s *Store) approval(task TaskDetail) Approval {
	a := Approval{Target: task.ID, Stage: task.Stage}
	plan := s.Plan()
	a.FinalGate = plan.FinalGate
	a.FinalGateRecorded = plan.Recorded

	// Applicability comes from the boundary contract, not from the gate shape.
	// Approve and decline are independent operations, so each control is gated on
	// its OWN operation; one being missing must not suppress the other.
	approveOK, declineOK := ApprovalOperations()
	a.ApproveApplicable = approveOK
	a.DeclineApplicable = declineOK
	a.Applicable = approveOK || declineOK
	a.ActionReason = approvalActionReason(approveOK, declineOK)

	present, kind := humanBoundary(task.Status, task.Run)
	if !present {
		// No boundary: there is nothing to act on, so no action reason is shown.
		a.ActionReason = ""
		return a
	}
	a.Present = true
	a.Kind = kind
	if task.Run.Classification != nil {
		a.Disposition = task.Run.Classification.Disposition
		a.Evidence = sanitizeDetail(task.Run.Classification.Reason)
	}
	return a
}

// approvalActionReason builds the read-only explanation shown when SOP reports a
// boundary but the controller cannot offer one or both actions. It names every
// missing application operation (approve and/or decline) using the recorded
// unsupported reasons, so it cannot drift from Boundary() and does not silently
// omit the decline gap. It returns "" only when both operations exist.
func approvalActionReason(approveOK, declineOK bool) string {
	if approveOK && declineOK {
		return ""
	}
	var missing []string
	if !approveOK {
		if d, ok := Lookup(OpApproveTask); ok {
			missing = append(missing, d.Reason)
		} else {
			missing = append(missing, "SOP exposes no approval application operation")
		}
	}
	if !declineOK {
		if d, ok := Lookup(OpDeclineTask); ok {
			missing = append(missing, d.Reason)
		} else {
			missing = append(missing, "SOP exposes no decline application operation")
		}
	}
	return strings.Join(missing, "; ")
}

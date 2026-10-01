package sopclient

import "strings"

// This file defines the controller-facing approval boundary (CTRL011, C2-001):
// the projection of SOP's own approval gate for one task, read from SOP's
// authoritative approval listing.
//
// Ownership rule: SOP owns approval and every lifecycle gate. The controller
// never decides that approval is required, never fabricates an approval, and
// never approves work itself. It only reports the gate SOP already recorded in
// its `sop approvals --json` listing and delegates any approval ACTION to SOP's
// own application boundary. When SOP reports no applicable gate for a task,
// this model reports Present=false and no approval action may be taken.
//
// Presence and applicability come ONLY from SOP's structured approval listing
// (see approvals.go). The retired controller-side inference - run classification
// disposition/Kind, run stage WAITING_FOR_HUMAN, or BLOCKED status with a human
// classification - is gone. BLOCKED alone, model/recovery prose, attempt counts,
// and inactivity never produce a gate: the listing is the sole source of truth
// for whether a gate exists and whether it is currently applicable.

// Approval is the controller-facing projection of a SOP-reported approval gate
// for one task. It is read-only and render-ready, and every field is carried
// verbatim from the selected SOP listing entry.
//
// Present is the source of truth for whether SOP reports an applicable gate at
// all; it is true only when the listing contains an applicable entry for the
// task. Whether a control may be offered is tracked PER ACTION by
// ApproveApplicable and DeclineApplicable: approve and decline are two
// independent boundary operations, so a future state where SOP exposes one but
// not the other must still offer the operation that exists rather than
// suppressing both. A control is shown only when its own operation is exposed;
// the gate itself is always shown as a read-only report. When SOP reports a gate
// but exposes no operation (the CTRL011 unsupported gap), no control is offered,
// so the UI never advertises an action whose only possible outcome is an
// unsupported error. Every free-form text field is passed through sanitizeDetail
// so no consumer can surface a prompt, secret, or dump. FinalGate reports SOP's
// recorded final plan gate verbatim when SOP persisted plan metadata; it is
// never inferred.
type Approval struct {
	// Present is true when SOP reports an applicable approval gate for this task.
	// It is false when SOP reports none. Kind/Evidence are empty when Present is
	// false; Target and Stage may still carry the task id and SOP's persisted run
	// stage so a renderer can show context, but they are never an absence signal.
	// Key an absence check off Present, not off Target being empty.
	Present bool
	// ApproveApplicable is true when an APPROVE control may be offered: a gate is
	// Present and SOP exposes the approve application operation. It is derived
	// from Boundary() (via ApprovalOperations), never inferred from the gate's
	// shape, so it can never drift from the recorded operation gap.
	ApproveApplicable bool
	// DeclineApplicable is the independent counterpart for the decline/withhold
	// control. It is tracked separately from ApproveApplicable because approve and
	// decline are distinct boundary operations: exposing one must not suppress the
	// other.
	DeclineApplicable bool
	// Applicable is true when at least one approval control may be offered (a gate
	// is Present and SOP exposes at least one of the approve/decline operations).
	// It is a convenience for callers that only need "is any action available";
	// per-action rendering must use ApproveApplicable/DeclineApplicable so one
	// existing operation is never hidden behind a missing counterpart.
	Applicable bool
	// ActionReason explains, when a gate is Present but not fully Applicable, why
	// a control is (or operations are) unavailable. It is built from the recorded
	// unsupported-operation reasons so the explanation cannot drift from the
	// boundary contract. It names every missing operation (approve and/or
	// decline), so it states exactly which action is missing. Empty when the gate
	// is not Present.
	ActionReason string
	// Kind is SOP's reported gate kind (kind), verbatim, or "" when Present is
	// false.
	Kind string
	// Target is SOP's own identifier for what the gate concerns (target),
	// verbatim. It is set whether or not Present is true; use Present (not an
	// empty Target) to decide whether a gate is reported.
	Target string
	// Evidence is SOP's free-form reason/evidence behind the gate (SOP's `reason`
	// field), sanitized for safe display. DISPLAY-ONLY: it never drives presence
	// or applicability.
	Evidence string
	// Stage is SOP's persisted gate/run lifecycle stage (stage), verbatim.
	Stage string
	// Disposition is SOP's persisted gate disposition (disposition), verbatim.
	Disposition string
	// Status is SOP's persisted gate status (status), verbatim. It is the
	// structured field the applicability rule reads.
	Status string
	// RequestedAt is SOP's recorded gate request time (requested_at), verbatim.
	RequestedAt string
	// TaskStatus is SOP's task status at the time of the gate (task_status),
	// verbatim.
	TaskStatus string
	// FinalGate is SOP's recorded final plan gate value when SOP persisted plan
	// metadata, or StatusUnknown. It is never inferred.
	FinalGate string
	// FinalGateRecorded is true when SOP persisted plan metadata carrying the
	// gate value.
	FinalGateRecorded bool
}

// approval derives the approval boundary for a task from SOP's authoritative
// approval listing only. It returns Present=false when the listing reports no
// applicable gate for the task, so an approval action can only ever appear when
// SOP itself requested one. Every reported field is carried through verbatim
// from the selected entry. It also records, per action, whether the controller
// can actually apply the gate (ApproveApplicable / DeclineApplicable), derived
// from the boundary contract's operation statuses, so a reported-but-unsupported
// gate is never rendered as a control that can only fail and a supported action
// is never suppressed by a missing counterpart.
func (s *Store) approval(task TaskDetail, listing ApprovalsListing) Approval {
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

	entry, present := listing.Lookup(task.ID)
	if !present {
		// No applicable gate: there is nothing to act on, so no action reason is
		// shown. Presence comes only from the listing.
		a.ActionReason = ""
		return a
	}
	a.Present = true
	// Carry every SOP-reported field through verbatim. Free-form Reason/Evidence
	// are display-only and were sanitized at decode time.
	a.Kind = entry.Kind
	if entry.Target != "" {
		a.Target = entry.Target
	}
	if entry.Stage != "" {
		a.Stage = entry.Stage
	}
	a.Evidence = entry.Reason
	a.Disposition = entry.Disposition
	a.Status = entry.Status
	a.RequestedAt = entry.RequestedAt
	a.TaskStatus = entry.TaskStatus
	a.ActionReason = approvalActionReason(approveOK, declineOK)
	return a
}

// taskNeedsHuman projects the list/hero signal from the same listing entry the
// detail view uses: a task needs a human when SOP reports an applicable gate for
// it. It returns the gate kind for display, or "" when there is none. It never
// consults run classification, run stage, BLOCKED status, prose, attempt counts,
// or inactivity.
func taskNeedsHuman(listing ApprovalsListing, taskID string) (bool, string) {
	entry, ok := listing.Lookup(taskID)
	if !ok {
		return false, ""
	}
	return true, entry.Kind
}

// approvalActionReason builds the read-only explanation shown when SOP reports a
// gate but the controller cannot offer one or both actions. It names every
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

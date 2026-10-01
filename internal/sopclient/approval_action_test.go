package sopclient

// This test proves the approval action applicability is derived from the single
// Boundary() source of truth (ApprovalOperations), so flipping the descriptors
// flips the projection with no duplicated logic. With C2-002 the approve/decline
// descriptors are supported, so a present gate offers both actions and no action
// reason is shown.

import (
	"testing"
)

func TestApprovalGateApplicabilityTracksBoundary(t *testing.T) {
	listing, ok := decodeApprovals([]byte(approvalsFixture))
	if !ok {
		t.Fatal("decode failed")
	}
	approveOK, declineOK := ApprovalOperations()

	st := &Store{}
	detail := TaskDetail{TaskSummary: TaskSummary{ID: "t-open", Status: StatusBlocked}}
	detail.Approval = st.approval(detail, listing)
	if !detail.Approval.Present {
		t.Fatal("Approval.Present = false, want true")
	}
	if detail.Approval.ApproveApplicable != approveOK {
		t.Fatalf("ApproveApplicable = %v, want %v (derived from Boundary)", detail.Approval.ApproveApplicable, approveOK)
	}
	if detail.Approval.DeclineApplicable != declineOK {
		t.Fatalf("DeclineApplicable = %v, want %v (derived from Boundary)", detail.Approval.DeclineApplicable, declineOK)
	}
	if detail.Approval.Applicable != (approveOK || declineOK) {
		t.Fatalf("Applicable = %v, want %v", detail.Approval.Applicable, approveOK || declineOK)
	}
	// When both actions are supported the action reason is empty (nothing is
	// missing); when one is missing it names the missing operation's reason.
	if approveOK && declineOK {
		if detail.Approval.ActionReason != "" {
			t.Fatalf("ActionReason = %q, want empty when both actions are supported", detail.Approval.ActionReason)
		}
	} else if detail.Approval.ActionReason == "" {
		t.Fatal("ActionReason is empty while an action is unsupported")
	}
}

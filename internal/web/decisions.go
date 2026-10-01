package web

import (
	"context"
	"net/http"

	"sop-controller/internal/sopclient"
)

// This file wires the C2-006 project-level human-decision surface: a single
// read-only projection of the decisions SOP reports for a project, rendered by
// the decisions.html partial. It is an aggregation layer ONLY: it never decides
// that a decision is required, never computes eligibility, and never mutates
// anything. Every control it renders is a POST form that delegates to the
// already-existing approve / decline / accept-changed command routes, and every
// value it shows is SOP's own reported value.
//
// The two decision sources are the SAME authoritative reads the rest of the
// dashboard uses:
//
//   - Approve/Decline: SOP's approval listing, surfaced per task as
//     TaskSummary.NeedsHuman/ApprovalKind (the same listing TaskDetail.Approval
//     uses). A gate entry is included only when SOP reports an applicable gate
//     AND exposes at least one of the approve/decline operations.
//   - Accept: SOP's changed-executed-task listing, surfaced as ChangedTasks. An
//     accept entry is included only for a task SOP reports changed-executed and
//     still pending (ChangedTasks.Pending()).
//
// No eligibility is computed locally beyond these SOP-reported flags.

// decisionGate is one SOP-reported approval gate awaiting an approve/decline
// decision. Every field is carried verbatim from SOP's approval projection; the
// controller adds only the task display title from SOP's own task store.
type decisionGate struct {
	TaskID string
	Title  string
	// ApproveApplicable / DeclineApplicable mirror SOP's per-action operation
	// availability. A control is rendered only for an action SOP exposes.
	ApproveApplicable bool
	DeclineApplicable bool
	// Kind is SOP's reported gate kind, carried verbatim.
	Kind string
}

// decisionAccept is one SOP-reported changed-executed task still pending the
// human's explicit accept. Every field is SOP's own value.
type decisionAccept struct {
	TaskID        string
	Title         string
	Stage         string
	ChangeSummary string
}

// decisionsData is the shared shape for the decisions partial and the
// GET /projects/{project}/decisions route. Gates and Accepts are SOP's reported
// decisions; the error field records when a read failed so a failure is shown as
// an explicit error state rather than an empty success (mirroring
// projectPage.ChangedTasksError).
type decisionsData struct {
	baseData
	Project sopclient.ProjectDetail
	// Gates are SOP-reported applicable approval gates (approve/decline).
	Gates []decisionGate
	// Accepts are SOP-reported changed-executed tasks still pending accept.
	Accepts []decisionAccept
	// ListChanged / AcceptChanged mirror sopclient.ReconcileOperations(): whether
	// SOP exposes the changed-task read and the per-task accept-changed operation.
	// The accept controls are gated on AcceptChanged, never offered when SOP
	// cannot apply them.
	ListChanged   bool
	AcceptChanged bool
	// ChangedTasksError, when non-empty, records that SOP's changed-task listing
	// could not be read; the panel then states the failure rather than showing an
	// empty success.
	ChangedTasksError string
}

// decisions projects SOP's reported decision state for a project from reads the
// caller already has: the project detail (carrying each task's SOP-reported
// NeedsHuman signal) and SOP's changed-executed-task listing. It performs no SOP
// calls of its own and computes no eligibility beyond SOP's reported flags.
func decisions(base baseData, project sopclient.ProjectDetail, changed sopclient.ChangedTasks, changedErr string) decisionsData {
	listChanged, acceptChanged := sopclient.ReconcileOperations()
	d := decisionsData{
		baseData:          base,
		Project:           project,
		ListChanged:       listChanged,
		AcceptChanged:     acceptChanged,
		ChangedTasksError: changedErr,
	}

	// Approve/Decline gates: SOP's own approval boundary per task. A task appears
	// only when SOP reports an applicable gate for it (NeedsHuman, projected from
	// the authoritative approval listing). Whether the approve and decline
	// controls are offered is gated per action on SOP's exposed operations, so one
	// existing operation is never hidden behind a missing counterpart.
	approveExposed := approvalOperationExposed(approveVerb)
	declineExposed := approvalOperationExposed(declineVerb)
	for _, t := range project.Tasks {
		if !t.NeedsHuman {
			continue
		}
		d.Gates = append(d.Gates, decisionGate{
			TaskID:            t.ID,
			Title:             t.Title,
			ApproveApplicable: approveExposed,
			DeclineApplicable: declineExposed,
			Kind:              t.ApprovalKind,
		})
	}

	// Accept: SOP's changed-executed tasks still pending. Only included when SOP
	// reports a set (not an unreported/empty-success) and a matching read
	// succeeded; each entry is one still-pending task SOP reported.
	if listChanged && changedErr == "" {
		for _, c := range changed.Pending() {
			d.Accepts = append(d.Accepts, decisionAccept{
				TaskID:        c.TaskID,
				Title:         c.Title,
				Stage:         c.Stage,
				ChangeSummary: c.ChangeSummary,
			})
		}
	}
	return d
}

// projectDecisions renders the read-only decisions surface for GET
// /projects/{project}/decisions. It reads SOP state and renders it; it starts no
// command and performs no SOP mutation, so a GET is always side-effect free. An
// unknown project is a 404 through renderError, never a phantom empty panel.
func (h *Handlers) projectDecisions(w http.ResponseWriter, r *http.Request) {
	detail, err := h.sop.Project(r.Context(), r.PathValue("project"))
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	changed, changedErr := h.readChangedTasks(r.Context(), detail)
	h.render(w, http.StatusOK, "decisions.html", decisions(
		h.base(r, "projects"), detail, changed, changedErr))
}

// readChangedTasks reads SOP's authoritative changed-executed-task listing for
// the project, returning an explicit error string (not an empty success) when
// the read is unavailable or fails. It is shared by the project page and the
// decisions route so the two surfaces cannot disagree, and it avoids a duplicate
// SOP read when the project page already needs the listing.
func (h *Handlers) readChangedTasks(ctx context.Context, detail sopclient.ProjectDetail) (sopclient.ChangedTasks, string) {
	listChanged, _ := sopclient.ReconcileOperations()
	if !listChanged {
		return sopclient.ChangedTasks{}, ""
	}
	changed, err := h.sop.ChangedTasks(ctx, detail.Summary.ID)
	if err != nil {
		return sopclient.ChangedTasks{}, err.Error()
	}
	return changed, ""
}

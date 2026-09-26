package web

import (
	"context"
	"net/http"

	"sop-controller/internal/sopclient"
)

// This file wires the CTRL011 human approval controls. The controller never
// decides that approval is required and never approves work itself: it reads
// SOP's reported boundary first and refuses (409) when SOP reports none, then
// delegates the approve/decline ACTION to SOP's application boundary. No path
// here bypasses validation/review/JEV/quality gates, because the only action
// offered is SOP's own approval operation.
//
// It also refuses when SOP reports a boundary but exposes no application
// operation for the requested action (the recorded operation gap). In that case
// the gate is reported read-only in the UI and the action endpoint returns an
// explicit conflict rather than starting a command whose only outcome would be
// an unsupported error. That keeps the rendered state truthful: no control is
// ever advertised that cannot have the effect it describes.

// approveVerb names the approve action for the URL.
const approveVerb = "approve"

// declineVerb names the decline action for the URL.
const declineVerb = "decline"

// approvalCommandKey is the per-(task, verb) command key for a task's approval
// decision. Approve and decline are two independent answers to a SOP gate, so
// each carries its own command key (mirroring the taskCommand pattern): a failed
// decline is distinguishable from a failed approve by the recorded command state
// itself, not only by URL, and a completed approve can never be read back from
// the decline status route.
func approvalCommandKey(verb, taskID string) string { return verb + ":" + taskID }

// approve delegates approval of a SOP-reported human gate to SOP's application
// boundary. It first reads SOP's reported boundary and refuses when SOP reports
// none or when SOP exposes no approve operation, so approval is only ever
// invoked where SOP actually requests it and the operation exists.
func (h *Handlers) approve(w http.ResponseWriter, r *http.Request) {
	h.approvalCommand(w, r, approveVerb, func(ctx context.Context, project, taskID string) (string, error) {
		if err := h.sop.ApproveTask(ctx, project, taskID); err != nil {
			return "", err
		}
		return "SOP reported the approval decision.", nil
	})
}

func (h *Handlers) approveStatus(w http.ResponseWriter, r *http.Request) {
	h.approvalCommandStatus(w, r, approveVerb)
}

// decline delegates declining/withholding approval. It is available on the same
// terms as approve (only where SOP reports a boundary and exposes the decline
// operation) and delegates only to SOP's own decline operation, or performs no
// state change when SOP has none, so a declined task is never left claiming
// approval.
//
// The controller never claims a decline was recorded when it was not: it reports
// whatever SOP's application boundary answers, verbatim. When SOP exposes no
// decline operation the command surfaces that unsupported answer rather than a
// fabricated success, and no local task state is written either way.
func (h *Handlers) decline(w http.ResponseWriter, r *http.Request) {
	h.approvalCommand(w, r, declineVerb, func(ctx context.Context, project, taskID string) (string, error) {
		if err := h.sop.DeclineTask(ctx, project, taskID); err != nil {
			return "", err
		}
		return "SOP reported the decline/withhold decision.", nil
	})
}

func (h *Handlers) declineStatus(w http.ResponseWriter, r *http.Request) {
	h.approvalCommandStatus(w, r, declineVerb)
}

// approvalApplicable reports whether the requested verb's own operation is
// available for this boundary. Approve and decline are independent operations,
// so each action is gated on its own applicability rather than on a shared flag,
// and an approve is never refused merely because decline is unsupported (or vice
// versa).
func approvalApplicable(a sopclient.Approval, verb string) bool {
	switch verb {
	case approveVerb:
		return a.ApproveApplicable
	case declineVerb:
		return a.DeclineApplicable
	default:
		return false
	}
}

// approvalCommand reads SOP's reported approval boundary and refuses the action
// with an explicit conflict when SOP does not request approval, or when SOP
// reports a boundary but exposes no operation for the requested verb. Only when
// SOP both requests approval and exposes that verb's operation does it start the
// background command. It reuses the existing command + polling status fragment
// pattern for duplicate-click safety, and keys the command per (verb, task) so
// approve and decline never collide and a completed action is never confused with
// its counterpart.
//
// The task is read through the project-scoped Task read, which returns
// ErrTaskNotFound for an id that does not belong to the project; the handler
// never trusts the URL's (project, task) pairing on its own, so it cannot act on
// a mismatched pair.
func (h *Handlers) approvalCommand(w http.ResponseWriter, r *http.Request, verb string, fn func(ctx context.Context, project, taskID string) (string, error)) {
	project, taskID := r.PathValue("project"), r.PathValue("task")
	detail, err := h.sop.Task(r.Context(), project, taskID)
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	if detail.ID != taskID {
		// Defensive: Task must return the requested task. If it ever returned a
		// different detail, refuse rather than act on a mismatched pairing.
		h.notFound(w, r, "Task not found")
		return
	}
	if !detail.Approval.Present {
		h.render(w, http.StatusConflict, "error.html", errorPage{
			baseData: h.base(r, ""),
			Status:   http.StatusConflict,
			Message:  "SOP reports no human approval boundary for this task; there is nothing to approve or decline.",
		})
		return
	}
	if !approvalApplicable(detail.Approval, verb) {
		h.render(w, http.StatusConflict, "error.html", errorPage{
			baseData: h.base(r, ""),
			Status:   http.StatusConflict,
			Message:  "SOP reports a human approval boundary for this task but exposes no " + verb + " application operation to apply it, so no approval action is available: " + detail.Approval.ActionReason,
		})
		return
	}
	key := approvalCommandKey(verb, taskID)
	_, _ = h.runner.Start(project, key, func(ctx context.Context) (string, error) {
		return fn(ctx, project, taskID)
	})
	h.renderCommand(w, r, project, key, taskCommandURL(project, taskID, verb))
}

// approvalCommandStatus renders the polling status fragment for an approval
// command. A command that was legitimately started while the gate was
// Present+applicable for that verb must keep reporting its status while it is
// still running (and once, on completion, so the operator sees the outcome): the
// operator started a real command, so a 404 there would look like a failure.
//
// It is truthful about retracted boundaries. A decision command's key is keyed
// per (verb, task) and is not cleared by the runner, so a status fragment could
// otherwise be resolvable indefinitely after SOP retracts the gate, implying a
// command the current SOP state would now reject. It therefore only reports a
// command that is still running regardless of the boundary, and otherwise
// requires the boundary to still be Present and this verb's operation to be
// applicable. When no command is running and the verb is not currently
// applicable, the URL is treated as unknown: with no boundary, or a boundary SOP
// exposes no operation for, no command could have been (re)started, so a 404 is
// truthful where a status fragment would imply an accepted command.
func (h *Handlers) approvalCommandStatus(w http.ResponseWriter, r *http.Request, verb string) {
	project, taskID := r.PathValue("project"), r.PathValue("task")
	key := approvalCommandKey(verb, taskID)
	if st, ok := h.runner.Status(project, key); ok && st.Running() {
		// A decision the operator legitimately started is still in flight; report
		// it regardless of any boundary change so a started action is never
		// hidden mid-flight.
		h.renderCommand(w, r, project, key, taskCommandURL(project, taskID, verb))
		return
	}
	detail, err := h.sop.Task(r.Context(), project, taskID)
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	if detail.ID != taskID {
		h.notFound(w, r, "Task not found")
		return
	}
	if !detail.Approval.Present || !approvalApplicable(detail.Approval, verb) {
		// No command can be (re)started for this task right now, so there is no
		// command status to report; a 404 is truthful where a status fragment
		// would imply an accepted command the current SOP state would reject.
		h.notFound(w, r, "No approval command for this task")
		return
	}
	h.renderCommand(w, r, project, key, taskCommandURL(project, taskID, verb))
}

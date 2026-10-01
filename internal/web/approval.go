package web

import (
	"context"
	"net/http"

	"sop-controller/internal/sopclient"
)

// This file wires the CTRL011 human approval controls (C2-002). The controller
// never decides that approval is required and never approves work itself: it
// delegates the approve/decline ACTION to SOP's application operation
// (`sop approve` / `sop decline`), which validates the gate at command time.
// No path here bypasses validation/review/JEV/quality gates, because the only
// action offered is SOP's own approval operation.
//
// SOP is the authority that validates the gate at command time. The controller
// does NOT pre-judge staleness from its own read of the approval listing: it
// delegates the decision and surfaces SOP's answer. When SOP rejects the
// decision (for example a stale/not-applicable gate) the outcome is classified as
// an actionable conflict carrying SOP's message and the task id, never a 500 and
// never a fabricated success. The controller writes no SOP state, marks no task
// complete on approve, and manufactures no failure on decline.

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
// operation. Any actor/note supplied by the operator is forwarded to SOP as
// `--by`/`--note`; no `--run` is passed, so recording the decision does not start
// execution. It refuses only when SOP exposes no approve operation at all; when
// SOP exposes the operation, the decision is delegated and SOP's own answer
// (accept or reject) is surfaced rather than pre-judged by the controller.
func (h *Handlers) approve(w http.ResponseWriter, r *http.Request) {
	h.approvalCommand(w, r, approveVerb, func(ctx context.Context, project, taskID string, opts sopclient.DecisionOptions) (string, error) {
		if err := h.sop.ApproveTaskWithOptions(ctx, project, taskID, opts); err != nil {
			return "", err
		}
		return "SOP recorded the approval decision.", nil
	})
}

func (h *Handlers) approveStatus(w http.ResponseWriter, r *http.Request) {
	h.approvalCommandStatus(w, r, approveVerb)
}

// decline delegates declining/withholding approval. It is available on the same
// terms as approve and delegates only to SOP's own decline operation, so a
// declined task is never left claiming approval.
//
// The controller never claims a decline was recorded when it was not: it reports
// whatever SOP's application operation answers, verbatim. A rejection is
// surfaced as an actionable conflict rather than a fabricated success, and no
// local task state is written either way - a decline never manufactures a
// task failure.
func (h *Handlers) decline(w http.ResponseWriter, r *http.Request) {
	h.approvalCommand(w, r, declineVerb, func(ctx context.Context, project, taskID string, opts sopclient.DecisionOptions) (string, error) {
		if err := h.sop.DeclineTaskWithOptions(ctx, project, taskID, opts); err != nil {
			return "", err
		}
		return "SOP recorded the decline/withhold decision.", nil
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

// approvalOperationExposed reports whether SOP exposes the requested decision
// operation at all, independent of whether the controller's own read currently
// reports an applicable gate. It is used to refuse only a control whose only
// possible outcome is an unsupported error (SOP exposes no such verb); when SOP
// exposes the verb the controller delegates and lets SOP validate the gate at
// command time, so a stale gate is rejected by SOP and surfaced truthfully rather
// than pre-judged here.
func approvalOperationExposed(verb string) bool {
	approve, decline := sopclient.ApprovalOperations()
	switch verb {
	case approveVerb:
		return approve
	case declineVerb:
		return decline
	default:
		return false
	}
}

// approvalOptions reads the optional actor/note metadata from the request form.
// Values are treated as untrusted input and passed through to SOP as discrete
// argv elements (never a shell string); an empty value simply omits the flag.
func approvalOptions(r *http.Request) sopclient.DecisionOptions {
	return sopclient.DecisionOptions{
		By:   r.FormValue("by"),
		Note: r.FormValue("note"),
	}
}

// approvalCommand delegates the approve/decline action to SOP's application
// operation. It refuses with an explicit conflict only when SOP exposes no
// operation for the requested verb (a control whose only outcome would be an
// unsupported error). Otherwise it starts the background command and lets SOP
// validate the gate at command time: a rejected (stale/not-applicable) decision
// is surfaced as an actionable conflict by the status fragment, never a 500 and
// never a fabricated success.
//
// It deliberately does NOT pre-refuse on the controller's own read of the
// approval listing: that read is a projection, not a decision source, and
// pre-gating on it can mask a gate SOP would accept (for example when the
// listing has not yet been persisted). SOP is the authority that validates the
// gate.
//
// The task is read through the project-scoped Task read, which returns
// ErrTaskNotFound for an id that does not belong to the project; the handler
// never trusts the URL's (project, task) pairing on its own, so it cannot act on
// a mismatched pair.
func (h *Handlers) approvalCommand(w http.ResponseWriter, r *http.Request, verb string, fn func(ctx context.Context, project, taskID string, opts sopclient.DecisionOptions) (string, error)) {
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
	if !approvalOperationExposed(verb) {
		h.render(w, http.StatusConflict, "error.html", errorPage{
			baseData: h.base(r, ""),
			Status:   http.StatusConflict,
			Message:  "SOP exposes no " + verb + " application operation to apply an approval decision, so no approval action is available.",
		})
		return
	}
	opts := approvalOptions(r)
	key := approvalCommandKey(verb, taskID)
	_, _ = h.runner.Start(project, key, func(ctx context.Context) (string, error) {
		return fn(ctx, project, taskID, opts)
	})
	h.renderCommand(w, r, project, key, taskCommandURL(project, taskID, verb))
}

// approvalCommandStatus renders the polling status fragment for an approval
// command. A command that was legitimately started must keep reporting its
// status while it is still running (and once, on completion, so the operator
// sees the outcome): the operator started a real command, so a 404 there would
// look like a failure.
//
// A completed command is reported on its own merits regardless of the boundary
// the controller currently reads, because SOP is the authority on the outcome
// and the controller must not hide a real result behind its own projection: the
// operator asked SOP to decide, and SOP's answer (accept or reject) is reported.
// When there is no recorded command at all, the URL is treated as unknown (404).
func (h *Handlers) approvalCommandStatus(w http.ResponseWriter, r *http.Request, verb string) {
	project, taskID := r.PathValue("project"), r.PathValue("task")
	key := approvalCommandKey(verb, taskID)
	if _, ok := h.runner.Status(project, key); ok {
		// A decision the operator legitimately started is reported regardless of any
		// boundary change: a started action is never hidden mid-flight or after the
		// fact, so SOP's own outcome (accept or reject) reaches the operator.
		h.renderCommand(w, r, project, key, taskCommandURL(project, taskID, verb))
		return
	}
	// No command was ever recorded for this key right now, so there is no command
	// status to report; a 404 is truthful where a status fragment would imply an
	// accepted command.
	h.notFound(w, r, "No approval command for this task")
}

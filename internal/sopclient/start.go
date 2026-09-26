package sopclient

import "context"

// StartOrContinueResult reports how the controller delegated a start-or-continue
// request through the SOP boundary. The controller never selects a task: it only
// chooses which SOP verb to invoke from SOP's own persisted plan state.
type StartOrContinueResult struct {
	// Verb is the SOP verb the request delegated to ("run" or "resume"), or ""
	// when SOP reported the plan already complete.
	Verb string
	// Complete is true when SOP's authoritative state reports the plan finished,
	// so no `sop run`/`sop resume` was dispatched and no phantom work started.
	Complete bool
	// Message is a human-readable result for the status fragment.
	Message string
}

// StartOrContinue starts an idle plan via `sop run` or continues an incomplete
// plan via `sop resume`, deciding only from SOP's persisted state whether the
// plan is idle, incomplete, or already complete. It never chooses a task, never
// reorders or skips tasks, and never mutates SOP state: SOP remains the workflow
// authority that selects the runnable task, applies dependency ordering, and
// owns verify-first/recovery behaviour.
//
// Behaviour, derived solely from SOP's recorded tasks and statuses:
//   - complete: SOP records every task terminal; no verb is dispatched and the
//     plan is reported complete instead of starting phantom work.
//   - idle: SOP records no started work yet; the plan is started with `sop run`.
//   - otherwise: the plan is incomplete (work in flight or stopped); it is
//     continued with `sop resume`.
func (c *Client) StartOrContinue(ctx context.Context, projectID string) (StartOrContinueResult, error) {
	detail, err := c.Project(ctx, projectID)
	if err != nil {
		return StartOrContinueResult{}, err
	}
	if planComplete(detail) {
		return StartOrContinueResult{
			Complete: true,
			Message:  "Plan complete: SOP reports every task finished; nothing to run.",
		}, nil
	}
	if !planStarted(detail) {
		if err := c.Run(ctx, projectID); err != nil {
			return StartOrContinueResult{Verb: "run"}, err
		}
		return StartOrContinueResult{
			Verb:    "run",
			Message: "Started idle plan (sop run); SOP selected the runnable task.",
		}, nil
	}
	if err := c.Resume(ctx, projectID); err != nil {
		return StartOrContinueResult{Verb: "resume"}, err
	}
	return StartOrContinueResult{
		Verb:    "resume",
		Message: "Continued incomplete plan (sop resume); SOP selected the runnable task.",
	}, nil
}

// planComplete reports whether SOP's own state says every task is finished, so a
// request must report completion rather than dispatch a run. It reads only
// SOP-persisted counts, never a controller-owned copy of task state.
func planComplete(detail ProjectDetail) bool {
	return detail.Summary.Total > 0 && detail.Summary.Completed == detail.Summary.Total
}

// planStarted reports whether SOP records any work already begun: a task that
// left the PLANNED/READY states, or an attempt was spent. A plan whose tasks are
// all still PLANNED/READY is idle and is started with `sop run`; anything else
// is incomplete and is continued with `sop resume`.
func planStarted(detail ProjectDetail) bool {
	for _, t := range detail.Tasks {
		if t.Status != StatusPlanned && t.Status != StatusReady {
			return true
		}
		if t.Attempt > 0 {
			return true
		}
	}
	return false
}

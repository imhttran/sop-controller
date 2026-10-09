package sopclient

import "context"

// StartOrContinueResult reports which SOP verb a start request delegated to.
type StartOrContinueResult struct {
	// Verb is "run" or "resume", or "" when the plan was already complete.
	Verb     string
	Complete bool
	Message  string
}

// StartOrContinue chooses only the SOP verb, from SOP's persisted state:
// complete (every task terminal) dispatches nothing; idle (nothing started)
// runs `sop run`; anything else runs `sop resume`. SOP still picks the task.
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

// planComplete reports whether every task is terminal.
func planComplete(detail ProjectDetail) bool {
	return detail.Summary.Total > 0 && detail.Summary.Completed == detail.Summary.Total
}

// planStarted reports whether any task left PLANNED/READY or spent an attempt.
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

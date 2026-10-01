package sopclient

import (
	"context"
	"strings"
)

// decide is the shared delegation for approve/decline: it validates the project,
// runs exactly `sop <verb> <task-id> [--by NAME] [--note TEXT]` through the
// Commander (argv slice, no shell), and classifies a non-zero result as SOP's
// own decision rejection. It never passes `--run`, never writes SOP state, and
// never converts the outcome into a task-completion or task-failure signal.
//
// SOP owns gate validation: this does not pre-judge eligibility from the
// controller's own read of the approval listing. It delegates the action and
// reports SOP's answer - success, or a *DecisionRejection carrying SOP's own
// message.
func (c *Client) decide(ctx context.Context, projectID, verb, taskID string, opts DecisionOptions) error {
	st, ok := c.stores[projectID]
	if !ok {
		return ErrProjectNotFound
	}
	out, err := c.cmd.Exec(ctx, st.root, verb, decisionArgs(taskID, opts)...)
	if err != nil {
		return &DecisionRejection{Verb: verb, TaskID: taskID, Message: decisionMessage(out, err), Err: err}
	}
	return nil
}

// decisionMessage chooses the message carried on a DecisionRejection. SOP's own
// output (stdout and stderr are captured together by the Commander) is preferred
// verbatim; when SOP produced no output at all the underlying command error is
// used, so a rejection always carries an actionable message even when SOP only
// signalled failure through its exit status.
func decisionMessage(out string, err error) string {
	if msg := strings.TrimSpace(out); msg != "" {
		return msg
	}
	if err != nil {
		return err.Error()
	}
	return ""
}

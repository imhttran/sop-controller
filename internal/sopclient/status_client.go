package sopclient

import "context"

// This file defines the structured plan/status/report accessors on *Client
// (CTRL003). Everything here reads SOP-persisted state and artifacts through
// Store; the controller never opens SOP state storage directly.

// Plan returns the active plan and its final plan gate for a project, as SOP
// persisted them in plan.meta.json. A project with no plan metadata returns a
// PlanGate with Recorded=false and FinalGate=StatusUnknown: absence is explicit
// and never inferred to be a completed (gated) plan.
func (c *Client) Plan(projectID string) (PlanGate, bool) {
	st, ok := c.stores[projectID]
	if !ok {
		return PlanGate{}, false
	}
	return st.Plan(), true
}

// TaskStatus returns the render-ready execution status for one task: run stage,
// aggregate validation/review/JEV status, retry/fix counts, provider/model, and
// the report reference. Every field reports absence explicitly; a missing
// artifact never resolves to PASS. Unknown projects and tasks yield the boundary
// sentinels.
func (c *Client) TaskStatus(ctx context.Context, projectID, taskID string) (TaskDetail, error) {
	return c.Task(ctx, projectID, taskID)
}

// ReportRef is the location/reference of a task's persisted report artifact. It
// identifies SOP's own report under .agent-sdlc/runs/<task-id>/; the controller
// only reads it and never writes it.
type ReportRef struct {
	// Present is true when SOP persisted a report artifact for the task.
	Present bool
	// TaskID is the task whose run directory holds the report.
	TaskID string
	// Path is the absolute path to the report artifact, or "" when absent. It is
	// empty rather than synthesized when no report exists.
	Path string
	// Name is the artifact filename (e.g. report.json) when present.
	Name string
}

// Report returns the report location/reference for a task's latest run. A
// missing report yields Present=false and an empty Path, never a synthesized
// path that could imply a result.
func (c *Client) Report(ctx context.Context, projectID, taskID string) (ReportRef, error) {
	st, ok := c.stores[projectID]
	if !ok {
		return ReportRef{}, ErrProjectNotFound
	}
	if _, err := st.Task(ctx, taskID); err != nil {
		return ReportRef{}, err
	}
	return st.ReportRef(taskID), nil
}

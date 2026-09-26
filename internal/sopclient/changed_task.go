package sopclient

// This file defines the controller-facing projection for the CTRL012 reconcile
// controls: the set of changed executed tasks awaiting reconciliation, and the
// per-task approval the controller must obtain before SOP mutates anything.
//
// Ownership rule: SOP owns reconciliation. `sop reconcile <PLAN.md>` validates
// before it mutates, preserves unchanged tasks, and stops at the human boundary
// when an executed task's definition changed. The controller never diffs the
// plan itself, never edits state.db, and never applies an approval. It only ever
// presents what SOP reports, so this projection is a verbatim view of
// SOP-reported values and nothing derived from controller-side heuristics.

import (
	"errors"
	"path/filepath"
	"strings"
)

// reconcileArtifact is the SOP-owned report of the changed executed tasks
// awaiting reconcile, read from <root>/.agent-sdlc/reconcile.json exactly like
// plan.meta.json and the per-run artifacts. It is optional and
// present-or-absent: SOP writes it when it has computed a changed set, and its
// absence is normal and never an error. The controller reads it verbatim and
// NEVER computes a plan diff of its own to fill the gap.
const reconcileArtifact = "reconcile.json"

// ErrTaskNotInChangedSet is returned by AcceptChangedTask when the per-task
// approval names a task that SOP did not report in its changed executed task
// set. An approval for an unknown or unrelated task is rejected before any
// mutation, so the controller can never approve work SOP did not flag.
var ErrTaskNotInChangedSet = errors.New("task is not in SOP's reported changed executed task set")

// ErrChangedTasksNotReported is returned by AcceptChangedTask when SOP reports
// no changed-executed-task set at all. With nothing for SOP to accept there is
// nothing to approve, so the controller reports the absence rather than
// pretending a task was accepted.
var ErrChangedTasksNotReported = errors.New("SOP reports no changed executed task set to approve")

// ErrNoActivePlan is returned by AcceptChangedTask when SOP records no active
// plan source. Per-task accept-changed reconciliation names the plan, so without
// one there is nothing to reconcile and the controller refuses rather than
// guessing a plan.
var ErrNoActivePlan = errors.New("no active plan recorded by SOP")

// ChangedExecutedTask is one SOP-reported changed executed task awaiting
// reconciliation. Every field is SOP's own value; the controller neither
// computes the changed set nor ranks or filters it.
type ChangedExecutedTask struct {
	// TaskID is SOP's identifier for the changed executed task.
	TaskID string
	// Title is SOP's recorded task title, sanitized for safe display.
	Title string
	// Stage is SOP's persisted run lifecycle stage for the task, verbatim.
	Stage string
	// ChangeSummary is SOP's own description of how the task definition changed,
	// sanitized for safe display. Empty when SOP reported no summary.
	ChangeSummary string
	// Approved is true when SOP reports this changed task already accepted. The
	// controller never sets this; it mirrors SOP's recorded decision so a task
	// that still needs explicit approval is visibly unapproved.
	Approved bool
}

// ChangedTasks is the result of the CTRL012 read of the changed executed tasks
// awaiting reconcile. Reported distinguishes "SOP reported a structured set"
// from "SOP reported none": a caller must never render an unreported set as an
// empty success. Every task in Tasks is SOP's own value.
type ChangedTasks struct {
	// Reported is true only when SOP reported a structured changed-executed-task
	// set. When false, Tasks is empty and the caller shows an explicit
	// "not reported by SOP" state rather than an empty success.
	Reported bool
	// Source names where SOP reported the set (the artifact filename), so a
	// reader can trace the value. Empty when Reported is false.
	Source string
	// Tasks are the changed executed tasks SOP reported, in SOP's own order.
	Tasks []ChangedExecutedTask
}

// Pending returns the changed tasks that still require explicit approval: those
// SOP reported as not yet accepted. It returns nil when SOP reported no set, so
// a caller never treats "unreported" as "nothing pending".
func (r ChangedTasks) Pending() []ChangedExecutedTask {
	if !r.Reported {
		return nil
	}
	var out []ChangedExecutedTask
	for _, t := range r.Tasks {
		if !t.Approved {
			out = append(out, t)
		}
	}
	return out
}

// Has reports whether SOP's reported changed set contains taskID. It is false
// when SOP reported no set, so an approval for any task against an unreported
// set is rejected.
func (r ChangedTasks) Has(taskID string) bool {
	for _, t := range r.Tasks {
		if t.TaskID == taskID {
			return true
		}
	}
	return false
}

// reconcileDoc is the persisted shape of SOP's changed-executed-task report.
// Only SOP-owned fields are decoded; the controller adds nothing.
type reconcileDoc struct {
	Changed []reconcileTaskDoc `json:"changed"`
}

type reconcileTaskDoc struct {
	TaskID   string `json:"task_id"`
	Title    string `json:"title"`
	Stage    string `json:"stage"`
	Summary  string `json:"change_summary"`
	Approved bool   `json:"approved"`
}

// ChangedTasks reads SOP's reported changed executed tasks for this project.
//
// It reads SOP's own reconcile report (reconcile.json) read-only, through the
// same read boundary as plan.meta.json and the run artifacts; the controller
// never opens SOP state storage for write and never computes a plan diff. When
// SOP reports no set, the result has Reported=false and no tasks, so a caller
// shows an explicit absence instead of an empty success. When SOP reports a set,
// every field is reported verbatim (title/summary sanitized for safe display).
func (s *Store) ChangedTasks() ChangedTasks {
	var doc reconcileDoc
	if !readJSON(filepath.Join(s.root, ".agent-sdlc", reconcileArtifact), &doc) {
		return ChangedTasks{}
	}
	out := ChangedTasks{Reported: true, Source: reconcileArtifact}
	for _, t := range doc.Changed {
		out.Tasks = append(out.Tasks, ChangedExecutedTask{
			TaskID:        strings.TrimSpace(t.TaskID),
			Title:         sanitizeDetail(t.Title),
			Stage:         strings.TrimSpace(t.Stage),
			ChangeSummary: sanitizeDetail(t.Summary),
			Approved:      t.Approved,
		})
	}
	return out
}

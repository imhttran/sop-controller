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
//
// C2-003: the changed-executed-task source is SOP's authoritative listing,
// `sop reconcile <PLAN.md> --list-changed --json`, resolved from SOP's recorded
// plan metadata (.agent-sdlc/plan.meta.json) rather than guessed. The listing is
// a pure read: it must not mutate task state, the task graph, the active plan,
// acceptance, or provenance. The retired present-or-absent artifact path
// (.agent-sdlc/reconcile.json) is no longer read here; no consumer can fall back
// to that stale artifact for the changed set.
//
// `changed_executed` carries task ids only. Display Title/Stage are enriched
// from SOP's own task store (state.db), never computed from a plan diff.

import (
	"encoding/json"
	"errors"
	"strings"
)

// listingSource names where the changed-executed set came from: SOP's
// authoritative listing verb. It replaces the retired reconcile.json artifact
// filename so a reader can trace the value to SOP rather than a stale file.
const listingSource = "sop reconcile --list-changed --json"

// ErrTaskNotInChangedSet is returned by AcceptChangedTask(s) when the per-task
// approval names a task that SOP did not report in its changed executed task
// set. An approval for an unknown or unrelated task is rejected before any
// mutation, so the controller can never approve work SOP did not flag.
var ErrTaskNotInChangedSet = errors.New("task is not in SOP's reported changed executed task set")

// ErrChangedTasksNotReported is returned by AcceptChangedTask(s) when SOP reports
// no changed-executed-task set at all. With nothing for SOP to accept there is
// nothing to approve, so the controller reports the absence rather than
// pretending a task was accepted.
var ErrChangedTasksNotReported = errors.New("SOP reports no changed executed task set to approve")

// ErrNoExplicitTaskIDs is returned by AcceptChangedTasks when the caller
// supplied no usable task id (an empty list, or only blank entries). Nothing was
// explicitly selected, so there is nothing to accept and no SOP call is made:
// acceptance can only ever name ids the human explicitly selected.
var ErrNoExplicitTaskIDs = errors.New("no explicitly selected task ids to accept")

// ErrNoActivePlan is returned by AcceptChangedTask(s) when SOP records no active
// plan source. Per-task accept-changed reconciliation names the plan, so without
// one there is nothing to reconcile and the controller refuses rather than
// guessing a plan.
var ErrNoActivePlan = errors.New("no active plan recorded by SOP")

// ChangedExecutedTask is one SOP-reported changed executed task awaiting
// reconciliation. Every field is SOP's own value; the controller neither
// computes the changed set nor ranks or filters it.
type ChangedExecutedTask struct {
	// TaskID is SOP's identifier for the changed executed task, from the
	// listing's changed_executed set.
	TaskID string
	// Title is SOP's recorded task title, enriched from SOP's task store and
	// sanitized for safe display. Empty when the task store has no matching row.
	Title string
	// Stage is SOP's persisted run lifecycle stage for the task, verbatim;
	// enriched from SOP's task store. Empty when SOP has no run stage for it.
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
	// set (including an observed empty changed_executed list). When false, Tasks
	// is empty and the caller shows an explicit "not reported by SOP" state
	// rather than an empty success.
	Reported bool
	// Source names where SOP reported the set (the listing verb), so a reader can
	// trace the value. Empty when Reported is false.
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

// listingDoc is the persisted shape of SOP's changed-task listing document, as
// emitted by `sop reconcile <PLAN.md> --list-changed --json`. Only SOP-owned
// field names are decoded; the controller adds nothing, renames nothing, and
// infers nothing. The keys (`version`, `source`, `plan_id`, `plan_changed`,
// `unchanged`, `updated`, `added`, `removed`, `changed_executed`,
// `removed_executed`, `auto_reconciled`) are decoded verbatim.
type listingDoc struct {
	Version         int      `json:"version"`
	Source          string   `json:"source"`
	PlanID          string   `json:"plan_id"`
	PlanChanged     bool     `json:"plan_changed"`
	Unchanged       []string `json:"unchanged"`
	Updated         []string `json:"updated"`
	Added           []string `json:"added"`
	Removed         []string `json:"removed"`
	ChangedExecuted []string `json:"changed_executed"`
	RemovedExecuted []string `json:"removed_executed"`
	AutoReconciled  []string `json:"auto_reconciled"`
}

// decodeListing parses SOP's `--list-changed --json` document. It returns the
// decoded listing and whether the bytes were a well-formed SOP listing. Decoding
// never derives or renames a field; a malformed document yields ok=false so the
// caller shows an explicit absence/failure rather than a partial set. Blank
// output (SOP reported nothing) is treated as not-reported, not an empty
// success, since an observed listing always carries at least the document
// envelope.
func decodeListing(data []byte) (listingDoc, bool) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return listingDoc{}, false
	}
	var doc listingDoc
	if err := json.Unmarshal([]byte(trimmed), &doc); err != nil {
		return listingDoc{}, false
	}
	return doc, true
}

// changedTasksFromListing projects SOP's decoded listing onto the
// controller-facing ChangedTasks model, enriching each listed task id with the
// display Title/Stage from SOP's own task store. The changed-executed set is
// exactly the listing's `changed_executed` ids, in SOP's order; a listed id the
// task store does not know still appears with an empty title rather than being
// dropped, and no plan diff is ever computed.
func (s *Store) changedTasksFromListing(doc listingDoc) ChangedTasks {
	out := ChangedTasks{Reported: true, Source: listingSource}
	for _, rawID := range doc.ChangedExecuted {
		id := strings.TrimSpace(rawID)
		if id == "" {
			continue
		}
		task := ChangedExecutedTask{TaskID: id}
		// Enrichment is a separate read of SOP's task store: title comes from the
		// tasks table and stage from the task's persisted run stage. Neither value
		// is synthesized, ranked, or diffed by the controller.
		if title, ok := s.taskTitle(id); ok {
			task.Title = sanitizeDetail(title)
		}
		task.Stage = strings.TrimSpace(s.runInfo(id).Stage)
		out.Tasks = append(out.Tasks, task)
	}
	return out
}

// taskTitle reads one task's title from SOP's task store. It returns ok=false
// when SOP has no row for the id (SOP's task store is the only source; the
// controller never invents a title).
func (s *Store) taskTitle(taskID string) (string, bool) {
	var title string
	err := s.db.QueryRow(`SELECT title FROM tasks WHERE id = ?`, taskID).Scan(&title)
	if err != nil {
		return "", false
	}
	return title, true
}

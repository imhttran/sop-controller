package sopclient

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
)

// Live activity delivery (CTRL007) - stable ordering + cursor contract.
//
// Discovery (S1) pinned the persisted activity model: Store.activity reads
// <root>/.agent-sdlc/runs/<task>/activity.jsonl in append order (oldest first),
// capped to the most recent maxActivityEvents, applying sanitizeDetail to every
// Detail. Each ActivityEvent carries TaskID, Stage, Action, Detail, Timestamp.
//
// What that read does NOT provide, and what this file adds:
//   - a per-event identity stable across reads (for idempotent rendering),
//   - a total order across ALL of a project's tasks (Store.activity is per
//     task; Client.Activity concatenates per-task streams by task id),
//   - a cursor so a consumer can re-request only events after the last one it
//     already saw (reconnection / poll refresh), and
//   - a bounded recent-window recovery query.
//
// Contract (published by S1, implemented here):
//
//	Identity: Cursor, a stable opaque string derived deterministically from the
//	          event's persisted fields (task id, append index within its task,
//	          timestamp, stage, action, sanitized detail). The same persisted
//	          event yields the same Cursor on every read.
//	Order:    per-task append order, tasks in ascending task id (exactly the
//	          existing read order); Cursor is a function of the same fields, so
//	          ordering and identity cannot diverge.
//	Cursor:   the Cursor of the last event a consumer has seen; a window
//	          request returns only events strictly after it. An unknown cursor
//	          yields the bounded recent window (recovery) rather than an error,
//	          so a consumer that lost its cursor still recovers a timeline.
//
// The read is read-only: it opens no command operation and cannot change SOP
// state. Sanitization remains applied at the existing Store.activity boundary,
// so this window shares the same safe-summary guarantee as TaskDetail.Activity.

// ActivityView is one delivered activity event: the sanitized event plus the
// identity/order key a consumer uses for idempotent rendering and as the next
// cursor. It embeds the existing ActivityEvent so consumers keep the same shape
// they already render.
type ActivityView struct {
	ActivityEvent
	// Cursor is the stable identity of this event. It is also the value a
	// consumer passes as "after" to fetch only later events.
	Cursor string `json:"cursor"`
	// Seq is the event's position in the project-wide total order, for a
	// readable timeline and stable sorting. It is derived, not persisted.
	Seq int `json:"seq"`
}

// DefaultActivityWindow bounds a recovery window when no limit is supplied.
const DefaultActivityWindow = 100

// MaxActivityWindow bounds any window request, regardless of what a caller asks.
const MaxActivityWindow = 500

// activityCursor derives the stable identity of one persisted event at its
// position in its task's append stream. It is deterministic over the event's
// own persisted fields, so re-reading activity.jsonl yields identical cursors.
func activityCursor(projectID, taskID string, index int, e ActivityEvent) string {
	h := sha256.New()
	for _, part := range []string{
		projectID, taskID, strconv.Itoa(index),
		e.Timestamp.UTC().Format("2006-01-02T15:04:05.000000000Z07:00"),
		e.Stage, e.Action, e.Detail,
	} {
		h.Write([]byte(part))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))[:32]
}

// ActivityWindow is a bounded, ordered window of recent activity for a project.
// It is the single read used by both live delivery and post-refresh recovery.
type ActivityWindow struct {
	// Events are the delivered events in the stable total order (oldest first).
	Events []ActivityView `json:"events"`
	// Cursor is the identity of the last delivered event, or the supplied cursor
	// when nothing new was delivered. A consumer stores it and passes it back as
	// "after" to resume with no duplicates.
	Cursor string `json:"cursor"`
	// More reports whether the source has at least one event beyond the returned
	// window (i.e. the window was truncated by the limit), so a poll client can
	// immediately issue another request instead of waiting.
	More bool `json:"more"`
	// Reset reports that a non-empty "after" cursor was supplied but could not
	// be found in the current source (for example after a truncation/rotation or
	// a stale cursor from another source). In that case Events is the bounded
	// recent window rather than the (empty) set of later events, so the caller
	// can distinguish "I lost my cursor, my timeline may have gaps" from "nothing
	// is new" and decide whether to clear/replace its rendered timeline. Reset
	// is false when after was empty (a fresh load) or when it was honoured.
	Reset bool `json:"reset"`
}

// ActivityWindow returns a bounded window of activity for a project, in the
// stable total order. When after is non-empty, only events strictly after that
// cursor are returned (dedup by identity across reconnection/poll refresh).
// When after is empty the most recent limit events are returned so a refreshed
// consumer recovers the recent timeline. When after is non-empty but unknown,
// the bounded recent window is returned and Reset is set so the caller knows
// the timeline may need replacing. limit <= 0 uses DefaultActivityWindow;
// limit > MaxActivityWindow is clamped.
//
// It is read-only: it delegates to the same Store.activity read (and therefore
// the same sanitizeDetail guarantee) and never invokes a command.
func (c *Client) ActivityWindow(ctx context.Context, projectID, after string, limit int) (ActivityWindow, error) {
	st, ok := c.stores[projectID]
	if !ok {
		return ActivityWindow{}, ErrProjectNotFound
	}
	if limit <= 0 {
		limit = DefaultActivityWindow
	}
	if limit > MaxActivityWindow {
		limit = MaxActivityWindow
	}

	all, err := st.activityViews(ctx, projectID)
	if err != nil {
		return ActivityWindow{}, err
	}

	win := ActivityWindow{Cursor: after, Events: []ActivityView{}}
	start := 0
	if after != "" {
		// Resume: drop everything at or before the last-seen event. A cursor
		// that is not found is reported explicitly via Reset: we fall through
		// to the bounded recent-window recovery path, but the caller can tell
		// that its timeline may have gaps rather than silently skipping events.
		if idx := indexOfCursor(all, after); idx >= 0 {
			start = idx + 1
		} else {
			win.Reset = true
		}
	}
	remaining := all[start:]
	if start == 0 {
		// Initial load, unknown cursor, or explicit reset: bound to the most
		// recent limit events, still oldest first.
		if len(remaining) > limit {
			remaining = remaining[len(remaining)-limit:]
		}
	} else if len(remaining) > limit {
		// A resume window is bounded too, so one poll cannot return an unbounded
		// backlog; More reports that another request will yield more.
		remaining = remaining[:limit]
	}
	win.More = len(all)-start > len(remaining)
	win.Events = append([]ActivityView{}, remaining...)
	if n := len(win.Events); n > 0 {
		win.Cursor = win.Events[n-1].Cursor
	}
	return win, nil
}

// TaskActivityView returns the cursor/seq-keyed activity views for one task,
// directly from that task's own persisted activity (sanitizeDetail applied). It
// is used by the task-scoped fragment so rendering a single task never depends
// on the task's events falling inside a project-wide bounded window: a task
// whose events are older than the most recent MaxActivityWindow project events
// still renders its own activity.
//
// Seq is assigned from the project-wide total order (per-task append order,
// tasks in ascending task id) so a task fragment sorts consistently with the
// project-wide live stream; the offset is the number of events in tasks that
// precede this one in task-id order. It is read-only and never invokes a
// command. ErrProjectNotFound / ErrTaskNotFound are returned when the project
// or task is unknown.
func (c *Client) TaskActivityView(ctx context.Context, projectID, taskID string) ([]ActivityView, error) {
	st, ok := c.stores[projectID]
	if !ok {
		return nil, ErrProjectNotFound
	}
	ids, err := st.taskIDs(ctx)
	if err != nil {
		return nil, err
	}
	seq := 0
	for _, id := range ids {
		events := st.activity(id)
		if id == taskID {
			out := make([]ActivityView, 0, len(events))
			for i, e := range events {
				out = append(out, ActivityView{
					ActivityEvent: e,
					Cursor:        activityCursor(projectID, id, i, e),
					Seq:           seq + i,
				})
			}
			return out, nil
		}
		seq += len(events)
	}
	return nil, ErrTaskNotFound
}

// activityViews returns every activity event for a project, in the stable total
// order, each carrying its cursor and project-wide sequence. It reuses the
// existing per-task Store.activity read (sanitization applied) and the same
// task-id enumeration as Client.Activity, so no consumer bypasses the read
// boundary.
func (s *Store) activityViews(ctx context.Context, projectID string) ([]ActivityView, error) {
	ids, err := s.taskIDs(ctx)
	if err != nil {
		return nil, err
	}
	var out []ActivityView
	seq := 0
	for _, id := range ids {
		events := s.activity(id)
		for i, e := range events {
			out = append(out, ActivityView{
				ActivityEvent: e,
				Cursor:        activityCursor(projectID, id, i, e),
				Seq:           seq,
			})
			seq++
		}
	}
	return out, nil
}

// indexOfCursor returns the position of the event with the given cursor, or -1.
func indexOfCursor(views []ActivityView, cursor string) int {
	for i := range views {
		if views[i].Cursor == cursor {
			return i
		}
	}
	return -1
}

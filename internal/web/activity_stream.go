package web

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"sop-controller/internal/sopclient"
)

// Live activity delivery (CTRL007), transport stage (S3).
//
// Both endpoints are backed solely by sopclient.ActivityWindow (the S2 bounded,
// cursor-addressable read over the existing Store.activity path), so:
//   - every event is already sanitized at the read boundary (no bypass),
//   - events carry the stable Cursor identity/order key from S1/S2, and
//   - neither handler opens a command operation or writes SOP state.
//
// SSE (/activity/stream) streams new events while a run is active; the plain
// window endpoint (/activity/window) is the bounded-poll fallback for clients
// without SSE. A poll refresh and a stream reconnection are the same operation:
// both send the last-seen cursor as "after" and receive only later events, so a
// reconnect cannot duplicate already-delivered lifecycle entries.

// activityPayload is the shared serialization of an activity event used by both
// the SSE and the poll-window endpoint. It is the ActivityView plus a
// render-ready "since" label, so the browser appends identical markup on either
// path.
type activityPayload struct {
	Cursor    string `json:"cursor"`
	Seq       int    `json:"seq"`
	TaskID    string `json:"taskId"`
	Stage     string `json:"stage"`
	Action    string `json:"action"`
	Detail    string `json:"detail"`
	Timestamp string `json:"timestamp"`
	Since     string `json:"since"`
	StageCSS  string `json:"stageClass"`
}

func newActivityPayload(v sopclient.ActivityView) activityPayload {
	ts := ""
	if !v.Timestamp.IsZero() {
		ts = v.Timestamp.UTC().Format(time.RFC3339Nano)
	}
	return activityPayload{
		Cursor:    v.Cursor,
		Seq:       v.Seq,
		TaskID:    v.TaskID,
		Stage:     v.Stage,
		Action:    v.Action,
		Detail:    v.Detail,
		Timestamp: ts,
		Since:     since(v.Timestamp),
		StageCSS:  stageClass(v.Stage),
	}
}

// activityStreamWindow bounds how many events one streamed window delivers.
const activityStreamWindow = sopclient.DefaultActivityWindow

// pollInterval returns the SSE re-check interval: the controller's configured
// poll cadence, with a conservative default so a zero option still polls.
func (h *Handlers) pollInterval() time.Duration {
	if h.poll <= 0 {
		return time.Second
	}
	return h.poll
}

// resumeCursor reads the resume cursor for an activity read. It prefers the
// explicit ?after= query parameter (a poll refresh, or an initial stream
// connection that already knows its last-seen cursor) and falls back to the SSE
// Last-Event-ID request header, which is what a browser sends automatically when
// an EventSource reconnects after the server has emitted `id: <cursor>` frames.
// Without the Last-Event-ID fallback a reconnect would resend the original
// (usually empty) query cursor and the whole recent window would be re-emitted,
// weakening the no-duplicate-lifecycle guarantee. An empty result means recovery.
func resumeCursor(r *http.Request) string {
	if after := r.URL.Query().Get("after"); after != "" {
		return after
	}
	return r.Header.Get("Last-Event-ID")
}

// activityLimit reads ?limit=, falling back to the default window.
func activityLimit(r *http.Request) int {
	n, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || n <= 0 {
		return sopclient.DefaultActivityWindow
	}
	return n
}

// activityWindow is the bounded-poll fallback endpoint. It returns the same
// window shape the SSE stream announces, so a poll-only client keeps the same
// dedup/order contract. A cursor-less request recovers the recent timeline; a
// request with the last-seen cursor returns only later events. When the cursor
// cannot be found in the source, Reset is set and the bounded recent window is
// returned so the consumer can rebase its rendered timeline instead of keeping
// a silently gapped one.
func (h *Handlers) activityWindow(w http.ResponseWriter, r *http.Request) {
	project := r.PathValue("project")
	win, err := h.sop.ActivityWindow(r.Context(), project, resumeCursor(r), activityLimit(r))
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	out := struct {
		Cursor string            `json:"cursor"`
		More   bool              `json:"more"`
		Reset  bool              `json:"reset"`
		Events []activityPayload `json:"events"`
	}{
		Cursor: win.Cursor,
		More:   win.More,
		Reset:  win.Reset,
		Events: make([]activityPayload, 0, len(win.Events)),
	}
	for _, v := range win.Events {
		out.Events = append(out.Events, newActivityPayload(v))
	}
	writeJSON(w, http.StatusOK, out)
}

// activityStream is the local-first SSE endpoint. It holds the connection open
// and emits each newly persisted activity event while a run is active. It never
// issues a command and never writes SOP state: a transport failure (client
// disconnect, write error, context cancellation) simply ends the stream.
func (h *Handlers) activityStream(w http.ResponseWriter, r *http.Request) {
	// Transport capability is checked before any project lookup: a client that
	// cannot consume a stream is rejected on its own merits, whether or not the
	// project exists, and the check does not touch SOP state.
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	project := r.PathValue("project")
	if _, ok := h.sop.Root(project); !ok {
		h.notFound(w, r, "Project not found")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	ctx := r.Context()
	cursor := resumeCursor(r)
	ticker := time.NewTicker(h.pollInterval())
	defer ticker.Stop()

	// Emit the current window immediately so a fresh connection recovers the
	// recent timeline without waiting for the first tick. If the very first read
	// fails we cannot deliver a window, so we end the stream rather than leaving
	// the client on an open connection that will never produce a frame (a hung
	// stream). A transport-level read failure never mutates SOP state.
	if !h.streamActivity(ctx, w, flusher, project, &cursor, true) {
		return
	}

	for {
		select {
		case <-ctx.Done():
			// Transport ended (client disconnected or request cancelled). The
			// read path holds no SOP state to unwind.
			return
		case <-ticker.C:
			if !h.streamActivity(ctx, w, flusher, project, &cursor, false) {
				// A read or write failure on a later tick ends the stream so the
				// client can reconnect (with Last-Event-ID) or fall back to
				// polling, instead of hanging on a dead connection.
				return
			}
		}
	}
}

// streamActivity delivers one window of new events as SSE frames and advances
// the connection cursor. It is deliberately read-only. It returns false when the
// stream should end (a read or write failure, or a cancelled context); on any
// read error it emits nothing rather than a fabricated event, and never mutates
// SOP state.
func (h *Handlers) streamActivity(ctx context.Context, w http.ResponseWriter, flusher http.Flusher, project string, cursor *string, initial bool) bool {
	if ctx.Err() != nil {
		return false
	}
	win, err := h.sop.ActivityWindow(ctx, project, *cursor, activityStreamWindow)
	if err != nil {
		// A transport-level read failure must not alter SOP state and must not
		// abort the connection with a fabricated event; emit nothing and end the
		// stream so the client learns the transport failed.
		return false
	}
	if len(win.Events) == 0 {
		if initial {
			if _, err := w.Write([]byte(": ready\n\n")); err != nil {
				return false
			}
			flusher.Flush()
		}
		return true
	}
	for _, v := range win.Events {
		payload := newActivityPayload(v)
		raw, err := json.Marshal(payload)
		if err != nil {
			continue
		}
		frame := "event: activity\nid: " + payload.Cursor + "\ndata: "
		if _, err := w.Write([]byte(frame)); err != nil {
			return false
		}
		if _, err := w.Write(raw); err != nil {
			return false
		}
		if _, err := w.Write([]byte("\n\n")); err != nil {
			return false
		}
	}
	*cursor = win.Cursor
	flusher.Flush()
	return true
}

package web

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"sop-controller/internal/config"
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
//
// C2-007 adds a short, configurable attention cadence (attentionInterval) to the
// SAME transports: the ticker that re-reads activity also re-reads SOP's
// reported human-decision gate at that cadence, so a newly recorded gate becomes
// visible shortly after SOP records it. No second event system is introduced and
// inactivity is never used to infer a gate.

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

// attentionPayload is the C2-007 projection of one SOP-reported human-decision
// gate for the live-progress consumer. It carries only SOP's reported values
// (never a controller-computed eligibility) and the relevant SOP-exposed
// controls. It is a READ projection over the same decision reads the decisions
// partial uses; no new SOP call and no new event system are introduced.
type attentionPayload struct {
	TaskID string `json:"taskId"`
	Title  string `json:"title,omitempty"`
	Kind   string `json:"kind,omitempty"`
	// ApproveApplicable / DeclineApplicable mirror SOP's per-action operation
	// availability, so the control set matches what SOP exposes.
	ApproveApplicable bool `json:"approveApplicable"`
	DeclineApplicable bool `json:"declineApplicable"`
}

// activityStreamWindow bounds how many events one streamed window delivers.
const activityStreamWindow = sopclient.DefaultActivityWindow

// pollInterval returns the general SSE re-check interval: the controller's
// configured poll cadence, with a conservative default so a zero option still
// polls.
func (h *Handlers) pollInterval() time.Duration {
	if h.poll <= 0 {
		return time.Second
	}
	return h.poll
}

// attentionInterval returns the resolved short attention-poll cadence: the
// configurable cadence at which the live-progress transports re-read SOP's
// reported gate so a newly recorded gate becomes visible promptly. A zero/unset
// value resolves to the documented short default (config.AttentionIntervalDefault)
// rather than an arbitrary long wait, and a configured value is never shorter
// than config.AttentionIntervalFloor. It never yields a fixed multi-minute wait.
func (h *Handlers) attentionInterval() time.Duration {
	if h.attention <= 0 {
		return config.AttentionIntervalDefault
	}
	if h.attention < config.AttentionIntervalFloor {
		return config.AttentionIntervalFloor
	}
	return h.attention
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
//
// C2-007: the response also carries the attention projection (Attention) and the
// resolved attention cadence (AttentionInterval), so a poll-only live-progress
// consumer surfaces a newly recorded gate within a bounded number of attention
// cadence intervals without a second event system. The projection is derived
// from the same SOP reads this endpoint already performs; it is never inferred
// from inactivity.
func (h *Handlers) activityWindow(w http.ResponseWriter, r *http.Request) {
	project := r.PathValue("project")
	win, err := h.sop.ActivityWindow(r.Context(), project, resumeCursor(r), activityLimit(r))
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	out := struct {
		Cursor            string             `json:"cursor"`
		More              bool               `json:"more"`
		Reset             bool               `json:"reset"`
		Events            []activityPayload  `json:"events"`
		Attention         []attentionPayload `json:"attention"`
		AttentionInterval string             `json:"attentionInterval"`
	}{
		Cursor:            win.Cursor,
		More:              win.More,
		Reset:             win.Reset,
		Events:            make([]activityPayload, 0, len(win.Events)),
		Attention:         h.projectAttention(r.Context(), project),
		AttentionInterval: h.attentionInterval().String(),
	}
	for _, v := range win.Events {
		out.Events = append(out.Events, newActivityPayload(v))
	}
	writeJSON(w, http.StatusOK, out)
}

// projectAttention derives the C2-007 'Needs your attention' projection for one
// project from the SAME SOP reads the decisions surface uses: the project detail
// (carrying each task's SOP-reported NeedsHuman signal) and SOP's
// changed-executed-task listing. It is read-only, performs no new SOP call beyond
// those reads, computes no eligibility beyond SOP's reported flags, and never
// derives attention from inactivity. A read failure yields an empty projection
// (no attention) rather than a fabricated gate.
func (h *Handlers) projectAttention(ctx context.Context, project string) []attentionPayload {
	detail, err := h.sop.Project(ctx, project)
	if err != nil {
		return nil
	}
	changed, changedErr := h.readChangedTasks(ctx, detail)
	d := decisions(h.baseForProject(project), detail, changed, changedErr)
	return attentionFromDecisions(d)
}

// attentionFromDecisions projects the attention payloads from an already-built
// decisionsData. Attention is reported if and only if SOP reports an applicable
// gate (an approval gate, or a changed-executed task still pending accept that
// SOP exposes an accept operation for) — never for a quiet project. The elapsed
// time stays in the activity events; no lifecycle state is derived here.
func attentionFromDecisions(d decisionsData) []attentionPayload {
	out := make([]attentionPayload, 0, len(d.Gates)+len(d.Accepts))
	for _, g := range d.Gates {
		out = append(out, attentionPayload{
			TaskID:            g.TaskID,
			Title:             g.Title,
			Kind:              g.Kind,
			ApproveApplicable: g.ApproveApplicable,
			DeclineApplicable: g.DeclineApplicable,
		})
	}
	if d.AcceptChanged {
		for _, a := range d.Accepts {
			out = append(out, attentionPayload{
				TaskID: a.TaskID,
				Title:  a.Title,
				Kind:   "accept-changed",
			})
		}
	}
	return out
}

// activityStream is the local-first SSE endpoint. It holds the connection open
// and emits each newly persisted activity event while a run is active. It never
// issues a command and never writes SOP state: a transport failure (client
// disconnect, write error, context cancellation) simply ends the stream.
//
// C2-007: after each window the stream also emits the current attention
// projection (event: attention) when SOP reports a gate, so a newly recorded gate
// surfaces promptly through the SAME stream — no second event system.
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
	// The SSE ticker runs at the short attention cadence so a newly recorded gate
	// is surfaced promptly. It is never lengthened into a multi-minute wait, and
	// inactivity is never interpreted as a gate.
	ticker := time.NewTicker(h.attentionInterval())
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

// streamActivity delivers one window of new events as SSE frames, advances the
// connection cursor, emits (for an empty initial window) the ': ready' keepalive
// frame, and then emits the current attention projection (if any) as an
// `event: attention` frame. It is deliberately read-only. It returns false when
// the stream should end (a read or write failure, or a cancelled context); on any
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
			// An initial empty window still emits the ': ready' keepalive so a fresh
			// connection is acknowledged; the attention frame may follow on the SAME
			// stream when SOP reports a gate.
			if _, err := w.Write([]byte(": ready\n\n")); err != nil {
				return false
			}
		}
		*cursor = win.Cursor
		if !h.streamAttention(ctx, w, project) {
			return false
		}
		if initial {
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

	// Emit the current attention projection through the SAME stream when SOP
	// reports a gate. This is a read of the decisions projection, never an
	// inference from inactivity, and it introduces no second event system.
	if !h.streamAttention(ctx, w, project) {
		return false
	}
	flusher.Flush()
	return true
}

// streamAttention emits the current attention projection as a single `event:
// attention` frame when a gate exists, and emits nothing when it does not (a
// quiet project must never produce an attention frame). It returns false only on
// a write failure. It performs no mutation and writes no SOP state.
func (h *Handlers) streamAttention(ctx context.Context, w http.ResponseWriter, project string) bool {
	attention := h.projectAttention(ctx, project)
	if len(attention) == 0 {
		return true
	}
	raw, err := json.Marshal(attention)
	if err != nil {
		return true
	}
	if _, err := w.Write([]byte("event: attention\ndata: ")); err != nil {
		return false
	}
	if _, err := w.Write(raw); err != nil {
		return false
	}
	if _, err := w.Write([]byte("\n\n")); err != nil {
		return false
	}
	return true
}

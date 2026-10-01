package web

import (
	"io/fs"
	"net/http"
	"time"

	"sop-controller/internal/config"
	"sop-controller/internal/sopclient"
)

// Options wires the dashboard server.
type Options struct {
	SOP      *sopclient.Client
	Views    *Views
	StaticFS fs.FS
	Poll     time.Duration
	// Attention is the short, configurable cadence at which the live-progress
	// transports re-read SOP's reported human-decision gate so a newly recorded
	// gate is surfaced promptly. A zero value resolves to the documented short
	// default; it is never a fixed multi-minute wait and is never used to infer a
	// gate from inactivity.
	Attention      time.Duration
	CommandTimeout time.Duration
	AllowNetwork   bool
	AccessToken    string
	// Discovery is the read-only project-discovery report for the /discovery
	// diagnostics view. A zero value renders an empty report, not an error.
	Discovery config.DiscoveryReport
}

// NewServer builds the HTTP handler for the dashboard.
func NewServer(opts Options) http.Handler {
	h := &Handlers{sop: opts.SOP, views: opts.Views, poll: opts.Poll, attention: opts.Attention, runner: NewCommandRunner(opts.CommandTimeout), discoveryReport: opts.Discovery}

	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(opts.StaticFS))))

	mux.HandleFunc("GET /healthz", h.health)
	mux.HandleFunc("GET /{$}", h.home)
	mux.HandleFunc("GET /projects", h.projects)
	mux.HandleFunc("GET /discovery", h.discovery)
	mux.HandleFunc("GET /projects/{project}", h.project)
	mux.HandleFunc("GET /projects/{project}/tasks", h.projectTasks)
	mux.HandleFunc("GET /projects/{project}/activity", h.projectActivity)
	// C2-006 project-level human-decision surface: a read-only aggregation of
	// SOP-reported gates (approve/decline) and SOP-reported changed-executed
	// tasks pending accept. The GET renders state only; every mutation it exposes
	// is a POST to the existing task-scoped command routes below.
	mux.HandleFunc("GET /projects/{project}/decisions", h.projectDecisions)
	// Live activity delivery (CTRL007): an SSE stream and a bounded-poll
	// fallback, both read-only windows over the persisted activity read.
	mux.HandleFunc("GET /projects/{project}/activity/stream", h.activityStream)
	mux.HandleFunc("GET /projects/{project}/activity/window", h.activityWindow)
	mux.HandleFunc("GET /projects/{project}/tasks/{task}", h.task)
	mux.HandleFunc("GET /projects/{project}/tasks/{task}/activity", h.taskActivity)
	mux.HandleFunc("GET /projects/{project}/tasks/{task}/recovery", h.taskRecovery)
	mux.HandleFunc("GET /projects/{project}/tasks/{task}/review", h.taskReview)
	mux.HandleFunc("GET /projects/{project}/tasks/{task}/ci", h.taskCI)
	mux.HandleFunc("GET /projects/{project}/tasks/{task}/handoff", h.taskHandoff)
	mux.HandleFunc("POST /projects/{project}/commands/{verb}", h.command)
	mux.HandleFunc("GET /projects/{project}/commands/{verb}", h.commandStatus)
	mux.HandleFunc("POST /projects/{project}/tasks/{task}/commands/retry", h.retry)
	mux.HandleFunc("GET /projects/{project}/tasks/{task}/commands/retry", h.retryStatus)
	mux.HandleFunc("POST /projects/{project}/tasks/{task}/commands/retry-force", h.retryForce)
	mux.HandleFunc("GET /projects/{project}/tasks/{task}/commands/retry-force", h.retryForceStatus)
	mux.HandleFunc("POST /projects/{project}/tasks/{task}/commands/report", h.reportTask)
	mux.HandleFunc("GET /projects/{project}/tasks/{task}/commands/report", h.reportTaskStatus)
	// CTRL011 human approval controls. These routes are the ONLY task-scoped
	// approval surface, and they are reachable but inert unless SOP reports an
	// approval boundary the controller can apply: the handlers pre-gate on
	// Approval.Present && Approval.Applicable and refuse with 409 otherwise. The
	// controller never approves or declines work itself; it delegates the action
	// to SOP's application boundary (or, where SOP exposes none, starts no
	// command at all). The GET status routes mirror the emitted polling URL so a
	// rendered form's status fragment always resolves to a registered route.
	mux.HandleFunc("POST /projects/{project}/tasks/{task}/commands/approve", h.approve)
	mux.HandleFunc("GET /projects/{project}/tasks/{task}/commands/approve", h.approveStatus)
	mux.HandleFunc("POST /projects/{project}/tasks/{task}/commands/decline", h.decline)
	mux.HandleFunc("GET /projects/{project}/tasks/{task}/commands/decline", h.declineStatus)
	// CTRL012 per-task accept-changed approval. Delegates validation and the
	// action to Client.AcceptChangedTask; see internal/web/handlers.go.
	mux.HandleFunc("POST /projects/{project}/tasks/{task}/commands/accept-changed", h.acceptChangedTask)
	mux.HandleFunc("GET /projects/{project}/tasks/{task}/commands/accept-changed", h.acceptChangedTaskStatus)

	var handler http.Handler = mux
	handler = csrfProtect(handler)
	if opts.AllowNetwork {
		handler = accessToken(opts.AccessToken)(handler)
	}
	handler = logRequests(handler)
	return handler
}

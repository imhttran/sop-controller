package web

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"sop-controller/internal/config"
	"sop-controller/internal/sopclient"
)

// Handlers holds the dashboard's HTTP dependencies.
type Handlers struct {
	sop             *sopclient.Client
	views           *Views
	poll            time.Duration
	attention       time.Duration
	runner          *CommandRunner
	discoveryReport config.DiscoveryReport
}

type baseData struct {
	CSRF string
	Poll time.Duration
	Nav  string
}

type projectsPage struct {
	baseData
	Projects []sopclient.ProjectSummary
}

type discoveryPage struct {
	baseData
	Discovery config.DiscoveryReport
}

type projectPage struct {
	baseData
	Project sopclient.ProjectDetail
	// CancelApplicable is SOP's own boundary answer (CTRL006), never a
	// controller guess: the Stop control is offered only when this is true.
	CancelApplicable bool
	// ChangedTasks is the CTRL012 read of the changed executed tasks SOP
	// reported awaiting reconcile, verbatim. Populated only when
	// ReconcileListChanged is true; zero value otherwise.
	ChangedTasks sopclient.ChangedTasks
	// ChangedTasksError, when non-empty, records that SOP's authoritative
	// changed-task listing could not be read (unsupported verb, non-zero exit, or
	// unparsable output). The panel then shows an explicit failure/absence state
	// rather than an empty success; the page itself still renders.
	ChangedTasksError string
	// ReconcileListChanged/ReconcileAcceptChanged mirror
	// sopclient.ReconcileOperations(): whether SOP's changed-task read and
	// per-task accept-changed operation are each available. The changed-task
	// panel and its per-task approval control are gated on these, never shown
	// as if they could act when they cannot.
	ReconcileListChanged   bool
	ReconcileAcceptChanged bool
	// Decisions is the C2-006 human-decision surface projection, built from the
	// same SOP reads this page already performs (no duplicate SOP call). It is
	// rendered inline by the decisions partial.
	Decisions decisionsData
}

type taskPage struct {
	baseData
	ProjectID string
	Task      sopclient.TaskDetail
	Events    []sopclient.ActivityView
	// Approval mirrors Task.Approval so the approval panel partial can be
	// included with this page as its data: it reads .Approval, .ProjectID,
	// .Task.ID and .CSRF, all of which the page now provides. The value is SOP's
	// reported boundary, never computed here.
	Approval sopclient.Approval
}

type errorPage struct {
	baseData
	Status  int
	Message string
}

// activityData is the shared shape for activity.html (project and task scope).
// Events carry the stable Cursor/Seq identity/order keys (CTRL007) so a rendered
// timeline can be deduplicated and the live consumer can resume from the last
// event it already showed.
type activityData struct {
	baseData
	Events []sopclient.ActivityView
}

// cmdStatusData is the shared shape for command_status.html.
type cmdStatusData struct {
	baseData
	Command    CommandState
	CommandURL string
}

func (h *Handlers) base(r *http.Request, nav string) baseData {
	return baseData{CSRF: csrfToken(r), Poll: h.poll, Nav: nav}
}

// baseForProject builds the base template data for a project-scoped fragment.
// It carries the same CSRF token and configured poll cadence as base, so a
// fragment rendered from the live-progress path (the attention projection) can
// share the decisions partial with the full project page. nav is "projects" and
// the request may be nil for an internally-derived projection, in which case no
// per-request token is available and the fragment carries an empty CSRF value
// (it renders state only; a mutation form re-reads the real token on the page).
func (h *Handlers) baseForProject(project string) baseData {
	return baseData{CSRF: "", Poll: h.poll, Nav: "projects"}
}

func (h *Handlers) render(w http.ResponseWriter, status int, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := h.views.Render(w, name, data); err != nil {
		log.Printf("[render] %s: %v", name, err)
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

func (h *Handlers) notFound(w http.ResponseWriter, r *http.Request, msg string) {
	h.render(w, http.StatusNotFound, "error.html",
		errorPage{baseData: h.base(r, ""), Status: http.StatusNotFound, Message: msg})
}

func (h *Handlers) renderError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, sopclient.ErrProjectNotFound):
		h.notFound(w, r, "Project not found")
	case errors.Is(err, sopclient.ErrTaskNotFound):
		h.notFound(w, r, "Task not found")
	default:
		h.render(w, http.StatusInternalServerError, "error.html",
			errorPage{baseData: h.base(r, ""), Status: http.StatusInternalServerError, Message: err.Error()})
	}
}

func (h *Handlers) home(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/projects", http.StatusFound)
}

func (h *Handlers) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (h *Handlers) projects(w http.ResponseWriter, r *http.Request) {
	list, err := h.sop.Projects(r.Context())
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	h.render(w, http.StatusOK, "projects.html", projectsPage{baseData: h.base(r, "projects"), Projects: list})
}

// discovery renders the read-only project-discovery report: the configured
// workspace roots, the depth bound, the registered projects, and the candidates
// discovery skipped. It makes the allowlist boundary visible instead of silent.
func (h *Handlers) discovery(w http.ResponseWriter, r *http.Request) {
	h.render(w, http.StatusOK, "discovery.html", discoveryPage{baseData: h.base(r, "discovery"), Discovery: h.discoveryReport})
}

func (h *Handlers) project(w http.ResponseWriter, r *http.Request) {
	detail, err := h.sop.Project(r.Context(), r.PathValue("project"))
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	// The changed-task listing is a pure read of SOP's authoritative changed set,
	// shared by the changed-task panel and the decisions panel so the two cannot
	// disagree. A failure (unsupported verb, non-zero exit, unparsable output) is
	// surfaced as an explicit error state rather than an empty success; the
	// failure is not allowed to break the rest of the project page.
	changed, changedErr := h.readChangedTasks(r.Context(), detail)
	base := h.base(r, "projects")
	listChanged, acceptChanged := sopclient.ReconcileOperations()
	h.render(w, http.StatusOK, "project.html", projectPage{
		baseData:               base,
		Project:                detail,
		CancelApplicable:       sopclient.CancelOperations(),
		ChangedTasks:           changed,
		ChangedTasksError:      changedErr,
		ReconcileListChanged:   listChanged,
		ReconcileAcceptChanged: acceptChanged,
		Decisions:              decisions(base, detail, changed, changedErr),
	})
}

// projectTasks is the polled task-list fragment (FR-2, FR-10).
func (h *Handlers) projectTasks(w http.ResponseWriter, r *http.Request) {
	detail, err := h.sop.Project(r.Context(), r.PathValue("project"))
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	h.render(w, http.StatusOK, "task_list.html", struct {
		baseData
		Project sopclient.ProjectDetail
	}{h.base(r, ""), detail})
}

// projectActivity is the project-level activity fragment (FR-4). It renders the
// same cursor/seq-keyed views the live stream delivers, so a refresh recovers the
// recent timeline without duplicate rows.
func (h *Handlers) projectActivity(w http.ResponseWriter, r *http.Request) {
	win, err := h.sop.ActivityWindow(r.Context(), r.PathValue("project"), "", 50)
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	h.render(w, http.StatusOK, "activity.html", activityData{h.base(r, ""), win.Events})
}

func (h *Handlers) task(w http.ResponseWriter, r *http.Request) {
	projectID, taskID := r.PathValue("project"), r.PathValue("task")
	detail, err := h.sop.Task(r.Context(), projectID, taskID)
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	h.render(w, http.StatusOK, "task.html", taskPage{
		baseData:  h.base(r, "projects"),
		ProjectID: projectID,
		Task:      detail,
		Events:    h.taskActivityViews(r.Context(), projectID, taskID, detail),
		Approval:  detail.Approval,
	})
}

// taskActivityViews returns the cursor/seq-keyed activity views for one task.
// It reads the task's own persisted activity (bounded per task, not the
// project-wide window), so a task whose events are older than the most recent
// project-wide events still renders its own timeline. When that read fails it
// falls back to the task's own detail.Activity, converted to the same view
// shape, so a render never fails solely because the window read did. It never
// returns a fabricated event.
func (h *Handlers) taskActivityViews(ctx context.Context, projectID, taskID string, detail sopclient.TaskDetail) []sopclient.ActivityView {
	views, err := h.sop.TaskActivityView(ctx, projectID, taskID)
	if err == nil {
		return views
	}
	log.Printf("[activity] task %s/%s window read: %v (falling back to task activity)", projectID, taskID, err)
	out := make([]sopclient.ActivityView, 0, len(detail.Activity))
	for _, e := range detail.Activity {
		out = append(out, sopclient.ActivityView{ActivityEvent: e})
	}
	return out
}

func (h *Handlers) taskActivity(w http.ResponseWriter, r *http.Request) {
	projectID, taskID := r.PathValue("project"), r.PathValue("task")
	detail, err := h.sop.Task(r.Context(), projectID, taskID)
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	h.render(w, http.StatusOK, "activity.html", activityData{h.base(r, ""), h.taskActivityViews(r.Context(), projectID, taskID, detail)})
}

// taskRecovery renders the recovery panel fragment (classification + actions).
func (h *Handlers) taskRecovery(w http.ResponseWriter, r *http.Request) {
	projectID, taskID := r.PathValue("project"), r.PathValue("task")
	detail, err := h.sop.Task(r.Context(), projectID, taskID)
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	h.render(w, http.StatusOK, "recovery.html", taskPage{
		baseData:  h.base(r, ""),
		ProjectID: projectID,
		Task:      detail,
		Approval:  detail.Approval,
	})
}

// taskReview/taskCI/taskHandoff pass the TaskDetail so the same templates can be
// inlined into task.html via {{template "x.html" .Task}}.
func (h *Handlers) taskReview(w http.ResponseWriter, r *http.Request) {
	h.taskFragment(w, r, "review.html")
}

func (h *Handlers) taskCI(w http.ResponseWriter, r *http.Request) {
	h.taskFragment(w, r, "ci.html")
}

func (h *Handlers) taskHandoff(w http.ResponseWriter, r *http.Request) {
	h.taskFragment(w, r, "handoff.html")
}

func (h *Handlers) taskFragment(w http.ResponseWriter, r *http.Request, name string) {
	detail, err := h.sop.Task(r.Context(), r.PathValue("project"), r.PathValue("task"))
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	h.render(w, http.StatusOK, name, detail)
}

// startVerb is the controller command surface (CTRL004) that starts an idle plan
// or continues an incomplete plan. It is one verb regardless of which SOP verb
// (run/resume) SOP's state resolves to, so a duplicate click maps to a single
// per-project command key and cannot launch two runs at once.
const startVerb = "start"

// cancelVerb is the CTRL006 Stop command verb.
const cancelVerb = "cancel"

// reconcileCommandKey is the project-scoped command key for the plain `reconcile`
// command. It serializes reconcile against itself (a duplicate click cannot
// start two reconciles for one project). Per-task accept-changed commands keep
// their OWN per-task key (accept-changed:<taskID>) so the per-task status
// fragment reports the accept-changed result and never conflates it with a
// project-wide reconcile running under this key.
const reconcileCommandKey = "reconcile"

// command starts a SOP command (FR-8) and returns its status fragment.
func (h *Handlers) command(w http.ResponseWriter, r *http.Request) {
	project, verb := r.PathValue("project"), r.PathValue("verb")
	if _, ok := h.sop.Root(project); !ok {
		h.notFound(w, r, "Project not found")
		return
	}
	var fn func(context.Context) (string, error)
	switch verb {
	case startVerb, "run":
		// Start-or-continue (CTRL004): the controller does not choose the task or
		// the next SOP run. StartOrContinue reads SOP's own state and delegates to
		// `sop run` (idle) or `sop resume` (incomplete), or reports completion.
		// The `run` verb keeps the existing FR-8 command surface and now performs
		// start-or-continue, so a plan that is already started is continued with
		// `sop resume` rather than starting phantom work.
		fn = func(ctx context.Context) (string, error) {
			res, err := h.sop.StartOrContinue(ctx, project)
			return res.Message, err
		}
	case "resume":
		fn = func(ctx context.Context) (string, error) { return "", h.sop.Resume(ctx, project) }
	case "validate":
		fn = func(ctx context.Context) (string, error) { return h.sop.Validate(ctx, project) }
	case "review":
		fn = func(ctx context.Context) (string, error) { return h.sop.RunReview(ctx, project) }
	case "report":
		fn = func(ctx context.Context) (string, error) { return h.sop.RunReport(ctx, project) }
	case "retry-all":
		fn = func(ctx context.Context) (string, error) { return "", h.sop.RetryAll(ctx, project) }
	case "reconcile":
		plan, ok := h.sop.PlanSource(project)
		if !ok {
			h.render(w, http.StatusConflict, "error.html", errorPage{
				baseData: h.base(r, ""),
				Status:   http.StatusConflict,
				Message:  "No active plan recorded by SOP; there is nothing to reconcile.",
			})
			return
		}
		// CTRL012: every SOP-reported changed executed task must be reported and
		// explicitly approved before reconcile mutates anything. Reconcile
		// validation and mutation both stay inside SOP; this only refuses to
		// start the command while SOP's own changed-set read shows pending
		// (unapproved) tasks, so the human boundary cannot be stepped around
		// through the dashboard.
		if listChanged, _ := sopclient.ReconcileOperations(); listChanged {
			changed, err := h.sop.ChangedTasks(r.Context(), project)
			if err != nil {
				// A failed/unreported listing is surfaced as such, and reconcile is
				// refused rather than started against an unknown changed set.
				h.render(w, http.StatusConflict, "error.html", errorPage{
					baseData: h.base(r, ""),
					Status:   http.StatusConflict,
					Message:  "SOP did not report a changed-executed-task listing; reconcile is refused: " + err.Error(),
				})
				return
			}
			if pending := changed.Pending(); len(pending) > 0 {
				h.render(w, http.StatusConflict, "error.html", errorPage{
					baseData: h.base(r, ""),
					Status:   http.StatusConflict,
					Message:  "SOP reports changed executed tasks still pending approval: " + pendingTaskIDs(pending) + ". Reconcile is refused until each is explicitly accepted.",
				})
				return
			}
		}
		fn = func(ctx context.Context) (string, error) { return h.sop.Reconcile(ctx, project, plan) }
	case cancelVerb:
		// CTRL006: SOP exposes no cancellation application operation today, so a
		// cancel command whose only possible outcome is an unsupported error is
		// refused up front (409) rather than started: the control is never
		// advertised as able to stop a run it cannot stop. This never manufactures
		// a PASS/LOCAL_DONE or any other success, and it touches no SOP
		// persistence or Git state either way.
		if !sopclient.CancelOperations() {
			// error.html is a full standalone document; rendering it here would nest
			// a second <html>/<head>/<body> inside the #cmd target this form swaps
			// into. command_status.html is the fragment built for that target.
			h.render(w, http.StatusConflict, "command_status.html", cmdStatusData{
				baseData: h.base(r, ""),
				Command: CommandState{
					Verb:  cancelVerb,
					State: "error",
					Error: "SOP exposes no cancellation application operation; cancelling an active run is a SOP lifecycle decision the controller cannot simulate.",
				},
			})
			return
		}
		fn = func(ctx context.Context) (string, error) { return "", h.sop.CancelRun(ctx, project) }
	default:
		http.Error(w, "unknown command", http.StatusBadRequest)
		return
	}
	// start/run are the same start-or-continue surface: share one per-project
	// command key so a duplicate click cannot launch a second SOP command.
	// reconcile uses the project-scoped reconciliation key so duplicate reconcile
	// clicks serialize.
	key := verb
	if verb == startVerb {
		key = "run"
	}
	if verb == "reconcile" {
		key = reconcileCommandKey
	}
	h.runner.Start(project, key, fn)
	h.renderCommand(w, r, project, key, commandURL(project, key))
}

// commandStatus renders the current status of a project-scoped command for
// polling. An unknown project is a 404, not a phantom "idle" status.
func (h *Handlers) commandStatus(w http.ResponseWriter, r *http.Request) {
	project, verb := r.PathValue("project"), r.PathValue("verb")
	if _, ok := h.sop.Root(project); !ok {
		h.notFound(w, r, "Project not found")
		return
	}
	if verb == startVerb {
		verb = "run"
	}
	if verb == "reconcile" {
		verb = reconcileCommandKey
	}
	h.renderCommand(w, r, project, verb, commandURL(project, verb))
}

func commandURL(project, verb string) string {
	return "/projects/" + project + "/commands/" + verb
}

// pendingTaskIDs joins the ids of changed executed tasks still awaiting
// approval, for a human-readable refusal message.
func pendingTaskIDs(pending []sopclient.ChangedExecutedTask) string {
	ids := make([]string, len(pending))
	for i, t := range pending {
		ids[i] = t.TaskID
	}
	return strings.Join(ids, ", ")
}

// acceptChangedTask is the CTRL012 per-task accept-changed approval (C2-004).
// It delegates validation and the action entirely to Client.AcceptChangedTask:
// an unknown/unrelated task, an unreported changed set, or no active plan is
// rejected before any SOP call, and a valid selection is delegated to SOP as
// `sop reconcile <PLAN.md> --accept-changed <id>`. A non-zero SOP result is
// reported truthfully (a *ReconcileRejection, classified as a conflict). This
// handler never approves anything on its own, never bulk-accepts, and viewing a
// change performs no mutation.
func (h *Handlers) acceptChangedTask(w http.ResponseWriter, r *http.Request) {
	h.taskCommand(w, r, "accept-changed", func(ctx context.Context, project, taskID string) (string, error) {
		return "", h.sop.AcceptChangedTask(ctx, project, taskID)
	})
}

func (h *Handlers) acceptChangedTaskStatus(w http.ResponseWriter, r *http.Request) {
	h.taskCommandStatus(w, r, "accept-changed")
}

// retry requeues a single task through SOP (FR-8).
func (h *Handlers) retry(w http.ResponseWriter, r *http.Request) {
	h.taskCommand(w, r, "retry", func(ctx context.Context, project, taskID string) (string, error) {
		return "", h.sop.Retry(ctx, project, taskID)
	})
}

func (h *Handlers) retryStatus(w http.ResponseWriter, r *http.Request) {
	h.taskCommandStatus(w, r, "retry")
}

// retryForce requeues a task whose retry budget is spent (`sop retry --force`).
func (h *Handlers) retryForce(w http.ResponseWriter, r *http.Request) {
	h.taskCommand(w, r, "retry-force", func(ctx context.Context, project, taskID string) (string, error) {
		return "", h.sop.RetryForce(ctx, project, taskID)
	})
}

func (h *Handlers) retryForceStatus(w http.ResponseWriter, r *http.Request) {
	h.taskCommandStatus(w, r, "retry-force")
}

// reportTask returns SOP's concise summary of one task's latest run.
func (h *Handlers) reportTask(w http.ResponseWriter, r *http.Request) {
	h.taskCommand(w, r, "report", h.sop.ReportTask)
}

func (h *Handlers) reportTaskStatus(w http.ResponseWriter, r *http.Request) {
	h.taskCommandStatus(w, r, "report")
}

// taskCommandKey maps a task-scoped command verb to its CommandRunner key. Every
// task command — including accept-changed — keeps its own per-(verb, task) key,
// so the per-task status fragment reports that command's own result and never
// conflates it with a project-wide reconcile running under reconcileCommandKey.
func taskCommandKey(verb, taskID string) string {
	return verb + ":" + taskID
}

// taskCommand runs a task-scoped SOP command in the background and renders its
// status fragment (which polls its own URL until the command finishes). Every
// recovery action goes through here, so no two handlers can diverge.
func (h *Handlers) taskCommand(w http.ResponseWriter, r *http.Request, verb string, fn func(ctx context.Context, project, taskID string) (string, error)) {
	project, taskID := r.PathValue("project"), r.PathValue("task")
	if _, ok := h.sop.Root(project); !ok {
		h.notFound(w, r, "Project not found")
		return
	}
	key := taskCommandKey(verb, taskID)
	_, _ = h.runner.Start(project, key, func(ctx context.Context) (string, error) {
		return fn(ctx, project, taskID)
	})
	h.renderTaskCommand(w, r, project, taskID, key, taskCommandURL(project, taskID, verb))
}

// taskCommandStatus renders the polling status fragment for a task-scoped
// command. An unknown project is a 404, not a phantom "idle" status.
func (h *Handlers) taskCommandStatus(w http.ResponseWriter, r *http.Request, verb string) {
	project, taskID := r.PathValue("project"), r.PathValue("task")
	if _, ok := h.sop.Root(project); !ok {
		h.notFound(w, r, "Project not found")
		return
	}
	h.renderTaskCommand(w, r, project, taskID, taskCommandKey(verb, taskID), taskCommandURL(project, taskID, verb))
}

// renderTaskCommand renders a task-scoped command's status, best-effort
// enriched with that task's current Stage/Attempt from SOP. A failed or
// not-found Task() read leaves Stage/Attempt empty rather than erroring the
// command-status response.
func (h *Handlers) renderTaskCommand(w http.ResponseWriter, r *http.Request, project, taskID, key, url string) {
	st := h.commandState(project, key)
	if detail, err := h.sop.Task(r.Context(), project, taskID); err == nil {
		st.TaskID = taskID
		st.Stage = detail.Stage
		st.Attempt = detail.Attempt
	}
	h.render(w, http.StatusOK, "command_status.html", cmdStatusData{
		baseData:   h.base(r, ""),
		Command:    st,
		CommandURL: url,
	})
}

func taskCommandURL(project, taskID, verb string) string {
	return "/projects/" + project + "/tasks/" + taskID + "/commands/" + verb
}

func (h *Handlers) renderCommand(w http.ResponseWriter, r *http.Request, project, key, url string) {
	st := h.commandState(project, key)
	if detail, err := h.sop.Project(r.Context(), project); err == nil {
		if taskID, ok := detail.ActiveTask(); ok {
			for _, t := range detail.Tasks {
				if t.ID == taskID {
					st.TaskID, st.Stage, st.Attempt = t.ID, t.Stage, t.Attempt
					break
				}
			}
		}
	}
	h.render(w, http.StatusOK, "command_status.html", cmdStatusData{
		baseData:   h.base(r, ""),
		Command:    st,
		CommandURL: url,
	})
}

// commandState returns the recorded status for project/key, or an idle
// placeholder when no command has run yet.
func (h *Handlers) commandState(project, key string) CommandState {
	st, ok := h.runner.Status(project, key)
	if !ok {
		return CommandState{Verb: key, State: "idle"}
	}
	return st
}

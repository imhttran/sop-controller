package web

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"sop-controller/internal/config"
	"sop-controller/internal/sopclient"
)

// Handlers holds the dashboard's HTTP dependencies.
type Handlers struct {
	sop             *sopclient.Client
	views           *Views
	poll            time.Duration
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
	h.render(w, http.StatusOK, "project.html", projectPage{baseData: h.base(r, "projects"), Project: detail})
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
		fn = func(ctx context.Context) (string, error) { return h.sop.Reconcile(ctx, project, plan) }
	default:
		http.Error(w, "unknown command", http.StatusBadRequest)
		return
	}
	// start/run are the same start-or-continue surface: share one per-project
	// command key so a duplicate click cannot launch a second SOP command.
	key := verb
	if verb == startVerb {
		key = "run"
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
	h.renderCommand(w, r, project, verb, commandURL(project, verb))
}

func commandURL(project, verb string) string {
	return "/projects/" + project + "/commands/" + verb
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

// taskCommand runs a task-scoped SOP command in the background and renders its
// status fragment (which polls its own URL until the command finishes). Every
// recovery action goes through here, so no two handlers can diverge.
func (h *Handlers) taskCommand(w http.ResponseWriter, r *http.Request, verb string, fn func(ctx context.Context, project, taskID string) (string, error)) {
	project, taskID := r.PathValue("project"), r.PathValue("task")
	if _, ok := h.sop.Root(project); !ok {
		h.notFound(w, r, "Project not found")
		return
	}
	key := verb + ":" + taskID
	_, _ = h.runner.Start(project, key, func(ctx context.Context) (string, error) {
		return fn(ctx, project, taskID)
	})
	h.renderCommand(w, r, project, key, taskCommandURL(project, taskID, verb))
}

// taskCommandStatus renders the polling status fragment for a task-scoped
// command. An unknown project is a 404, not a phantom "idle" status.
func (h *Handlers) taskCommandStatus(w http.ResponseWriter, r *http.Request, verb string) {
	project, taskID := r.PathValue("project"), r.PathValue("task")
	if _, ok := h.sop.Root(project); !ok {
		h.notFound(w, r, "Project not found")
		return
	}
	h.renderCommand(w, r, project, verb+":"+taskID, taskCommandURL(project, taskID, verb))
}

func taskCommandURL(project, taskID, verb string) string {
	return "/projects/" + project + "/tasks/" + taskID + "/commands/" + verb
}

func (h *Handlers) renderCommand(w http.ResponseWriter, r *http.Request, project, key, url string) {
	st, ok := h.runner.Status(project, key)
	if !ok {
		st = CommandState{Verb: key, State: "idle"}
	}
	h.render(w, http.StatusOK, "command_status.html", cmdStatusData{
		baseData:   h.base(r, ""),
		Command:    st,
		CommandURL: url,
	})
}

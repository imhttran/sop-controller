package web

import (
	"net/http"
	"os"
	"strings"
	"testing"

	"sop-controller/internal/sopclient"
)

// boundaryVerbs returns every sop CLI verb the documented boundary can drive,
// including the FR-5/FR-6 supporting commands (validate, review) that also
// delegate to SOP through internal/sopclient.
func boundaryVerbs() map[string]bool {
	verbs := map[string]bool{}
	for _, d := range sopclient.Boundary() {
		for _, v := range d.SOPVerbs {
			verbs[v] = true
		}
	}
	for _, v := range []string{"validate", "review"} {
		verbs[v] = true
	}
	return verbs
}

// dashboardCommandRoutes is the command route table registered by NewServer
// (internal/web/server.go). Every entry is a state-changing command the
// dashboard exposes; the test proves each reaches the router and drives only a
// documented boundary verb. A new or renamed command route changes server.go and
// must be reflected here, so the boundary cannot silently fall out of sync with
// the router. The {id} and {task} placeholders are substituted with the fixture's
// project id and task id at test time.
func dashboardCommandRoutes() []string {
	return []string{
		"/projects/{id}/commands/run",
		"/projects/{id}/commands/resume",
		"/projects/{id}/commands/validate",
		"/projects/{id}/commands/review",
		"/projects/{id}/commands/report",
		"/projects/{id}/commands/retry-all",
		"/projects/{id}/commands/reconcile",
		"/projects/{id}/tasks/{task}/commands/retry",
		"/projects/{id}/tasks/{task}/commands/retry-force",
		"/projects/{id}/tasks/{task}/commands/report",
	}
}

// verbForRoute maps each registered command route to the sop verb it drives, so
// the test can assert the router reaches the boundary verb and no other.
func verbForRoute(path string) string {
	switch {
	case strings.HasSuffix(path, "/commands/run"):
		return "run"
	case strings.HasSuffix(path, "/commands/resume"):
		return "resume"
	case strings.HasSuffix(path, "/commands/validate"):
		return "validate"
	case strings.HasSuffix(path, "/commands/review"):
		return "review"
	case strings.HasSuffix(path, "/commands/retry-all"):
		return "retry"
	case strings.HasSuffix(path, "/commands/reconcile"):
		return "reconcile"
	case strings.HasSuffix(path, "/commands/report"):
		return "report"
	case strings.HasSuffix(path, "/commands/retry"), strings.HasSuffix(path, "/commands/retry-force"):
		return "retry"
	}
	return ""
}

// TestDashboardCommandsResolveToBoundary drives every dashboard command route
// through the real router (NewServer) and asserts it maps to a documented
// boundary operation. The routes come from dashboardCommandRoutes/verbForRoute,
// which mirror server.go; the test fails if a route stops reaching its handler
// (404/400) or drives a verb the boundary does not document.
func TestDashboardCommandsResolveToBoundary(t *testing.T) {
	verbs := boundaryVerbs()
	srv, id, root := newRunServer(t, "PLANNED", nil)
	// Record a plan so reconcile is not rejected by its precondition.
	if err := os.WriteFile(root+"/.agent-sdlc/plan.meta.json", []byte(`{"source":"docs/PLAN.md"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, tmpl := range dashboardCommandRoutes() {
		path := strings.ReplaceAll(tmpl, "{id}", id)
		path = strings.ReplaceAll(path, "{task}", "x1")
		verb := verbForRoute(tmpl)
		if verb == "" {
			t.Fatalf("route %q has no mapped sop verb; update verbForRoute", tmpl)
		}
		if !verbs[verb] {
			t.Errorf("route %q drives sop verb %q, which is not a documented boundary operation", tmpl, verb)
		}
		code := postWithCSRF(t, srv, path)
		switch code {
		case http.StatusNotFound, http.StatusBadRequest:
			t.Errorf("POST %s = %d; route is not a registered dashboard command", path, code)
		}
	}
}

// TestDashboardHasNoUndocumentedCommandRoute asserts the router does not serve
// a phantom approve command: approval is a SOP application operation that
// does not exist on this generic project-command route, so the dashboard must
// not expose it there (it has its own dedicated, always-gated route instead).
func TestDashboardHasNoUndocumentedCommandRoute(t *testing.T) {
	srv, id, _ := newRunServer(t, "PLANNED", nil)
	for _, verb := range []string{"approve"} {
		path := "/projects/" + id + "/commands/" + verb
		if code := postWithCSRF(t, srv, path); code != http.StatusBadRequest {
			t.Errorf("POST %s = %d, want 400 (unknown command)", path, code)
		}
	}
}

// TestDashboardCancelCommandRefusedWhenUnsupported asserts the CTRL006 cancel
// route exists (unlike the undocumented approve verb above) but is refused
// with an explicit 409 rather than started, because SOP exposes no
// cancellation application operation (sopclient.CancelOperations() is false).
// A 404/400 here would say the control does not exist; a 200 would manufacture
// a fake stop. 409 is the only truthful answer until SOP exposes the verb.
func TestDashboardCancelCommandRefusedWhenUnsupported(t *testing.T) {
	srv, id, _ := newRunServer(t, "PLANNED", nil)
	path := "/projects/" + id + "/commands/cancel"
	if code := postWithCSRF(t, srv, path); code != http.StatusConflict {
		t.Errorf("POST %s = %d, want 409 (cancellation unsupported by SOP)", path, code)
	}
}

// TestDashboardDoesNotReadSOPPersistenceDirectly asserts no production web file
// reads SOP persistence (state.db or run artifacts) itself; those reads belong to
// internal/sopclient only. Direct reads in this package appear only in tests,
// which build fixtures.
func TestDashboardDoesNotReadSOPPersistenceDirectly(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		b, err := os.ReadFile(e.Name())
		if err != nil {
			t.Fatal(err)
		}
		for _, banned := range []string{"state.db", "activity.jsonl", ".agent-sdlc"} {
			if strings.Contains(string(b), banned) {
				t.Errorf("%s references %q; direct SOP persistence reads belong in internal/sopclient", e.Name(), banned)
			}
		}
	}
}

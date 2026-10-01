package web

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// writeReconcileReport writes SOP's optional reconcile.json changed-executed-task
// report for the fixture project, the artifact Client.ChangedTasks reads.
func writeReconcileReport(t *testing.T, root, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, ".agent-sdlc", "reconcile.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writePlanSource(t *testing.T, root string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, ".agent-sdlc", "plan.meta.json"), []byte(`{"source":"docs/PLAN.md"}`), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The project page reports a SOP-reported changed executed task before any
// reconcile mutation can happen, verbatim and as pending until accepted.
func TestProjectPageReportsChangedTasksBeforeMutation(t *testing.T) {
	srv, id, root := newRunServer(t, "PLANNED", nil)
	writePlanSource(t, root)
	writeReconcileReport(t, root, `{"changed":[{"task_id":"x1","title":"Task X","stage":"IMPLEMENT","change_summary":"acceptance criteria changed"}]}`)

	code, body := get(t, srv.URL+"/projects/"+id)
	if code != 200 {
		t.Fatalf("GET project = %d", code)
	}
	if !strings.Contains(body, "x1") || !strings.Contains(body, "acceptance criteria changed") {
		t.Errorf("project page does not report the SOP changed task verbatim:\n%s", body)
	}
	if !strings.Contains(body, "pending") {
		t.Errorf("unapproved changed task should render as pending:\n%s", body)
	}
}

// When SOP reports no changed-executed-task set, the project page shows an
// explicit absence, never an empty list presented as success.
func TestProjectPageShowsAbsentChangedTaskSet(t *testing.T) {
	srv, id, root := newRunServer(t, "PLANNED", nil)
	writePlanSource(t, root)

	code, body := get(t, srv.URL+"/projects/"+id)
	if code != 200 {
		t.Fatalf("GET project = %d", code)
	}
	if !strings.Contains(body, "has not reported a changed-executed-task set") {
		t.Errorf("project page should show an explicit absence state when SOP reports no set:\n%s", body)
	}
	if strings.Contains(body, "Changed executed tasks</h2></div>\n  \n  <table") {
		t.Errorf("project page must not render an empty table as success")
	}
}

// Reconcile is refused (409), not started, while SOP reports a changed
// executed task still pending approval.
func TestReconcileRefusedWhilePending(t *testing.T) {
	srv, id, root := newRunServer(t, "PLANNED", nil)
	writePlanSource(t, root)
	writeReconcileReport(t, root, `{"changed":[{"task_id":"x1","title":"Task X"}]}`)

	if code := postWithCSRF(t, srv, "/projects/"+id+"/commands/reconcile"); code != 409 {
		t.Fatalf("reconcile with a pending changed task = %d, want 409", code)
	}
}

// Once SOP reports every changed executed task already accepted, reconcile
// proceeds to delegate to SOP as before.
func TestReconcileProceedsWhenNothingPending(t *testing.T) {
	srv, id, root := newRunServer(t, "PLANNED", nil)
	writePlanSource(t, root)
	writeReconcileReport(t, root, `{"changed":[{"task_id":"x1","title":"Task X","approved":true}]}`)

	if code := postWithCSRF(t, srv, "/projects/"+id+"/commands/reconcile"); code != 200 {
		t.Fatalf("reconcile with nothing pending = %d, want 200", code)
	}
}

// pollCommandDone polls a command status URL until it stops running (or the
// deadline passes) and returns the final fragment body.
func pollCommandDone(t *testing.T, srv string, path string) string {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		_, body := get(t, srv+path)
		if !strings.Contains(body, "cmd-running") {
			return body
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("command did not finish before deadline")
	return ""
}

// An accept-changed approval naming a task SOP did not report in its changed
// set is rejected with ErrTaskNotInChangedSet before any mutation, never a
// fabricated success.
func TestAcceptChangedTaskRejectsUnknownTask(t *testing.T) {
	srv, id, root := newRunServer(t, "PLANNED", nil)
	writePlanSource(t, root)
	writeReconcileReport(t, root, `{"changed":[{"task_id":"x1","title":"Task X"}]}`)

	code := postWithCSRF(t, srv, "/projects/"+id+"/tasks/unknown-task/commands/accept-changed")
	if code != 200 {
		// taskCommand always answers 200 with a "running"/terminal status fragment;
		// the rejection shows up in the polled command state, asserted below.
		t.Fatalf("POST accept-changed = %d", code)
	}
	body := pollCommandDone(t, srv.URL, "/projects/"+id+"/tasks/unknown-task/commands/accept-changed")
	if !strings.Contains(body, "cmd-error") || !strings.Contains(body, "reported changed executed task set") {
		t.Errorf("accept-changed for an unknown task should fail with ErrTaskNotInChangedSet:\n%s", body)
	}
}

// An accept-changed approval submitted when SOP reports no changed set at all
// is rejected with ErrChangedTasksNotReported before any mutation.
func TestAcceptChangedTaskRejectsWhenSetNotReported(t *testing.T) {
	srv, id, root := newRunServer(t, "PLANNED", nil)
	writePlanSource(t, root)

	postWithCSRF(t, srv, "/projects/"+id+"/tasks/x1/commands/accept-changed")
	body := pollCommandDone(t, srv.URL, "/projects/"+id+"/tasks/x1/commands/accept-changed")
	if !strings.Contains(body, "cmd-error") || !strings.Contains(body, "no changed executed task set") {
		t.Errorf("accept-changed with no reported set should fail with ErrChangedTasksNotReported:\n%s", body)
	}
}

// An accept-changed approval submitted when no active plan is recorded is
// rejected with ErrNoActivePlan before any mutation.
func TestAcceptChangedTaskRejectsWithNoActivePlan(t *testing.T) {
	srv, id, _ := newRunServer(t, "PLANNED", nil)

	postWithCSRF(t, srv, "/projects/"+id+"/tasks/x1/commands/accept-changed")
	body := pollCommandDone(t, srv.URL, "/projects/"+id+"/tasks/x1/commands/accept-changed")
	if !strings.Contains(body, "cmd-error") || !strings.Contains(body, "no active plan recorded") {
		t.Errorf("accept-changed with no active plan should fail with ErrNoActivePlan:\n%s", body)
	}
}

// A valid accept-changed approval (active plan, reported set, task present)
// surfaces SOP's own unsupported gap rather than a fabricated success, since
// SOP exposes no per-task accept-changed application operation yet.
func TestAcceptChangedTaskSurfacesUnsupportedGapWhenValid(t *testing.T) {
	srv, id, root := newRunServer(t, "PLANNED", nil)
	writePlanSource(t, root)
	writeReconcileReport(t, root, `{"changed":[{"task_id":"x1","title":"Task X"}]}`)

	postWithCSRF(t, srv, "/projects/"+id+"/tasks/x1/commands/accept-changed")
	body := pollCommandDone(t, srv.URL, "/projects/"+id+"/tasks/x1/commands/accept-changed")
	if !strings.Contains(body, "cmd-error") || !strings.Contains(body, "operation unsupported by SOP") {
		t.Errorf("a valid accept-changed approval should surface SOP's own unsupported gap, not a success:\n%s", body)
	}
}

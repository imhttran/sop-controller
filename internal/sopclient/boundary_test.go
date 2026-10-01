package sopclient

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"sop-controller/internal/config"
)

// The contract names exactly the PRD's eleven conceptual operations (plus the
// CTRL011 decline counterpart of ApproveTask and the CTRL012 reconcile-control
// operations GetChangedExecutedTasks/AcceptChangedTask), once each. A supported
// operation cites an entry point and a SOP application operation; an unsupported
// one records a reason. Nothing else may enter the boundary.
func TestBoundaryContractMatchesPRD(t *testing.T) {
	want := []Operation{
		OpGetPlan, OpGetTasks, OpGetTask, OpGetTaskActivity, OpGetTaskProgress,
		OpGetTaskReport, OpStartOrContinueRun, OpRetryTask, OpCancelRun,
		OpApproveTask, OpDeclineTask, OpReconcilePlan,
		OpGetChangedExecutedTasks, OpAcceptChangedTask,
	}
	got := Boundary()
	if len(got) != len(want) {
		t.Fatalf("boundary has %d operations, want %d", len(got), len(want))
	}
	seen := map[Operation]int{}
	for i, d := range got {
		if d.Operation != want[i] {
			t.Fatalf("operation %d = %q, want %q", i, d.Operation, want[i])
		}
		seen[d.Operation]++
		switch d.Status {
		case StatusSupported:
			if d.EntryPoint == "" || d.SOPOperation == "" {
				t.Errorf("%s: supported operation is missing an entry point or SOP operation", d.Operation)
			}
			if d.Reason != "" {
				t.Errorf("%s: supported operation must not carry a gap reason", d.Operation)
			}
		case StatusUnsupported:
			if d.EntryPoint != "" {
				t.Errorf("%s: unsupported operation must not name an entry point", d.Operation)
			}
			if d.Reason == "" {
				t.Errorf("%s: unsupported operation must record a reason", d.Operation)
			}
		default:
			t.Errorf("%s: unknown status %q", d.Operation, d.Status)
		}
	}
	for _, op := range want {
		if seen[op] != 1 {
			t.Errorf("operation %s appears %d times, want exactly once", op, seen[op])
		}
	}
}

// The boundary exposes no scheduler operation: the controller never decides
// which task runs next. Those decisions stay inside SOP.
func TestBoundaryExposesNoSchedulerOperation(t *testing.T) {
	for _, d := range Boundary() {
		name := strings.ToLower(string(d.Operation))
		for _, banned := range []string{"select", "schedule", "next", "choose", "pick"} {
			if strings.Contains(name, banned) {
				t.Errorf("boundary operation %q looks like a scheduler decision; SOP owns scheduling", d.Operation)
			}
		}
	}
}

// CancelRun, ApproveTask, DeclineTask, and AcceptChangedTask (once its
// preconditions are satisfied) have no SOP application operation, so the
// boundary reports ErrOperationUnsupported and records the gap instead of
// simulating it. Each gap must report a DISTINCT operation so one failure is
// never confusable with another.
func TestUnsupportedOperationsReturnErrOperationUnsupported(t *testing.T) {
	root := newProject(t)
	// AcceptChangedTask's own preconditions (active plan, reported changed set,
	// task membership) must all be satisfied before its unsupported gap is what
	// the caller sees; otherwise one of the precondition sentinels would be.
	if err := os.WriteFile(filepath.Join(root, ".agent-sdlc", "plan.meta.json"),
		[]byte(`{"source":"docs/PLAN.md"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".agent-sdlc", "reconcile.json"),
		[]byte(`{"changed":[{"task_id":"t2","title":"Task Two"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	bin, _ := fakeSop(t)
	c, err := New([]string{root}, bin, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	pid := config.ProjectID(root)
	ctx := context.Background()

	if err := c.CancelRun(ctx, pid); !errors.Is(err, ErrOperationUnsupported) {
		t.Errorf("CancelRun err = %v, want ErrOperationUnsupported", err)
	}
	if err := c.ApproveTask(ctx, pid, "t2"); !errors.Is(err, ErrOperationUnsupported) {
		t.Errorf("ApproveTask err = %v, want ErrOperationUnsupported", err)
	}
	if err := c.DeclineTask(ctx, pid, "t2"); !errors.Is(err, ErrOperationUnsupported) {
		t.Errorf("DeclineTask err = %v, want ErrOperationUnsupported", err)
	}
	if err := c.AcceptChangedTask(ctx, pid, "t2"); !errors.Is(err, ErrOperationUnsupported) {
		t.Errorf("AcceptChangedTask err = %v, want ErrOperationUnsupported", err)
	}

	// The decline gap must reference the decline operation, not the approve one.
	if err := c.DeclineTask(ctx, pid, "t2"); err == nil || !strings.Contains(err.Error(), string(OpDeclineTask)) {
		t.Errorf("DeclineTask err = %v, want it to name %s", err, OpDeclineTask)
	}
	if err := c.ApproveTask(ctx, pid, "t2"); err == nil || !strings.Contains(err.Error(), string(OpApproveTask)) {
		t.Errorf("ApproveTask err = %v, want it to name %s", err, OpApproveTask)
	}
	if err := c.AcceptChangedTask(ctx, pid, "t2"); err == nil || !strings.Contains(err.Error(), string(OpAcceptChangedTask)) {
		t.Errorf("AcceptChangedTask err = %v, want it to name %s", err, OpAcceptChangedTask)
	}

	// An unknown project yields the not-found sentinel for every action.
	if err := c.CancelRun(ctx, "nope"); !errors.Is(err, ErrProjectNotFound) {
		t.Errorf("CancelRun(unknown project) err = %v, want ErrProjectNotFound", err)
	}
	if err := c.ApproveTask(ctx, "nope", "t2"); !errors.Is(err, ErrProjectNotFound) {
		t.Errorf("ApproveTask(unknown project) err = %v, want ErrProjectNotFound", err)
	}
	if err := c.DeclineTask(ctx, "nope", "t2"); !errors.Is(err, ErrProjectNotFound) {
		t.Errorf("DeclineTask(unknown project) err = %v, want ErrProjectNotFound", err)
	}
	if err := c.AcceptChangedTask(ctx, "nope", "t2"); !errors.Is(err, ErrProjectNotFound) {
		t.Errorf("AcceptChangedTask(unknown project) err = %v, want ErrProjectNotFound", err)
	}

	for _, op := range []Operation{OpCancelRun, OpApproveTask, OpDeclineTask, OpAcceptChangedTask} {
		d, ok := Lookup(op)
		if !ok || d.Status != StatusUnsupported || d.Reason == "" {
			t.Errorf("%s = %+v, want an unsupported descriptor with a reason", op, d)
		}
	}
}

// AcceptChangedTask rejects an unknown/unrelated task, an unreported changed
// set, and a missing active plan, each with its own distinct sentinel, before
// ever reaching the (currently unsupported) action itself. These checks run in
// a fixed order (plan, then reported-set, then membership) so the first
// violated precondition is always what the caller sees.
func TestAcceptChangedTaskRejectsInvalidApprovals(t *testing.T) {
	ctx := context.Background()
	bin, _ := fakeSop(t)

	t.Run("no active plan", func(t *testing.T) {
		root := newProject(t)
		c, err := New([]string{root}, bin, time.Minute)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		if err := c.AcceptChangedTask(ctx, config.ProjectID(root), "t2"); !errors.Is(err, ErrNoActivePlan) {
			t.Errorf("AcceptChangedTask err = %v, want ErrNoActivePlan", err)
		}
	})

	t.Run("changed set not reported", func(t *testing.T) {
		root := newProject(t)
		if err := os.WriteFile(filepath.Join(root, ".agent-sdlc", "plan.meta.json"),
			[]byte(`{"source":"docs/PLAN.md"}`), 0o644); err != nil {
			t.Fatal(err)
		}
		c, err := New([]string{root}, bin, time.Minute)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		if err := c.AcceptChangedTask(ctx, config.ProjectID(root), "t2"); !errors.Is(err, ErrChangedTasksNotReported) {
			t.Errorf("AcceptChangedTask err = %v, want ErrChangedTasksNotReported", err)
		}
	})

	t.Run("task not in reported set", func(t *testing.T) {
		root := newProject(t)
		if err := os.WriteFile(filepath.Join(root, ".agent-sdlc", "plan.meta.json"),
			[]byte(`{"source":"docs/PLAN.md"}`), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, ".agent-sdlc", "reconcile.json"),
			[]byte(`{"changed":[{"task_id":"t2","title":"Task Two"}]}`), 0o644); err != nil {
			t.Fatal(err)
		}
		c, err := New([]string{root}, bin, time.Minute)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		if err := c.AcceptChangedTask(ctx, config.ProjectID(root), "unrelated-task"); !errors.Is(err, ErrTaskNotInChangedSet) {
			t.Errorf("AcceptChangedTask err = %v, want ErrTaskNotInChangedSet", err)
		}
	})
}

// ApprovalOperations must report that SOP exposes no approval or decline
// application operation, so the UI can refuse to offer a control whose only
// possible outcome is an unsupported error. It derives the answer from the same
// Boundary() source of truth, so it cannot drift from the descriptors.
func TestApprovalOperationsReportsUnsupported(t *testing.T) {
	approve, decline := ApprovalOperations()
	if approve {
		t.Error("ApprovalOperations approve = true, want false (SOP exposes no approval operation)")
	}
	if decline {
		t.Error("ApprovalOperations decline = true, want false (SOP exposes no decline operation)")
	}
}

// CancelOperations must report that SOP exposes no cancellation application
// operation, so the UI can refuse to offer a Stop control whose only possible
// outcome is an unsupported error. It derives the answer from the same
// Boundary() source of truth, so it cannot drift from the descriptor.
func TestCancelOperationsReportsUnsupported(t *testing.T) {
	if CancelOperations() {
		t.Error("CancelOperations() = true, want false (SOP exposes no cancellation operation)")
	}
}

// TestCancelSupportedDerivation proves the gating formula itself is correct in
// both directions, not just against today's fixed unsupported descriptor: if
// Boundary() ever records OpCancelRun as StatusSupported, CancelOperations
// must flip to true with no code change, and an unknown operation must never
// be reported as supported.
func TestCancelSupportedDerivation(t *testing.T) {
	cases := []struct {
		name  string
		d     Descriptor
		found bool
		want  bool
	}{
		{"unsupported", Descriptor{Operation: OpCancelRun, Status: StatusUnsupported}, true, false},
		{"supported", Descriptor{Operation: OpCancelRun, Status: StatusSupported}, true, true},
		{"not found", Descriptor{}, false, false},
	}
	for _, c := range cases {
		if got := cancelSupported(c.d, c.found); got != c.want {
			t.Errorf("%s: cancelSupported() = %v, want %v", c.name, got, c.want)
		}
	}
}

// ReconcileOperations must report that SOP exposes a structured
// changed-executed-task read (its optional reconcile.json artifact) but no
// per-task accept-changed application operation, so the UI offers the
// changed-task list but never a control whose only possible outcome is an
// unsupported error. It derives the answer from the same Boundary() source of
// truth.
func TestReconcileOperationsReportsListSupportedAcceptUnsupported(t *testing.T) {
	listChanged, acceptChanged := ReconcileOperations()
	if !listChanged {
		t.Error("ReconcileOperations listChanged = false, want true (SOP's reconcile.json read works)")
	}
	if acceptChanged {
		t.Error("ReconcileOperations acceptChanged = true, want false (SOP exposes no accept-changed operation)")
	}
}

// Lookup must return the same descriptor Boundary exposes for the CTRL012
// operations, so the two surfaces cannot drift. Descriptors carry a slice
// (SOPVerbs), so they are compared with reflect.DeepEqual rather than ==.
func TestReconcileOperationDescriptorsMatchBoundary(t *testing.T) {
	byOp := map[Operation]Descriptor{}
	for _, d := range Boundary() {
		byOp[d.Operation] = d
	}

	d, ok := Lookup(OpGetChangedExecutedTasks)
	if !ok {
		t.Fatalf("Lookup(%s) not found", OpGetChangedExecutedTasks)
	}
	if !reflect.DeepEqual(d, byOp[OpGetChangedExecutedTasks]) {
		t.Errorf("Lookup(%s) = %+v, Boundary has %+v", OpGetChangedExecutedTasks, d, byOp[OpGetChangedExecutedTasks])
	}
	if d.Status != StatusSupported || d.EntryPoint == "" || d.SOPOperation == "" || d.Reason != "" {
		t.Errorf("%s = %+v, want supported with an entry point and no reason", OpGetChangedExecutedTasks, d)
	}

	d, ok = Lookup(OpAcceptChangedTask)
	if !ok {
		t.Fatalf("Lookup(%s) not found", OpAcceptChangedTask)
	}
	if !reflect.DeepEqual(d, byOp[OpAcceptChangedTask]) {
		t.Errorf("Lookup(%s) = %+v, Boundary has %+v", OpAcceptChangedTask, d, byOp[OpAcceptChangedTask])
	}
	if d.Status != StatusUnsupported || d.Reason == "" {
		t.Errorf("%s = %+v, want unsupported with a reason", OpAcceptChangedTask, d)
	}
}

// The read operations report SOP's own persisted values, verbatim, and map the
// not-found cases to the boundary sentinels.
func TestReadOperationsReportSOPValues(t *testing.T) {
	root := newProject(t)
	now := time.Now().UTC().Format(time.RFC3339)
	seed(t, root,
		`INSERT INTO tasks VALUES ('t1','Task One','obj','ac','LOCAL_DONE',NULL,1,3,'`+now+`','`+now+`')`,
		`INSERT INTO tasks VALUES ('t2','Task Two','obj','ac','BLOCKED','REVIEW_UNRESOLVED',0,3,'`+now+`','`+now+`')`,
		`INSERT INTO tasks VALUES ('t3','Task Three','obj','ac','PLANNED',NULL,0,3,'`+now+`','`+now+`')`,
		`INSERT INTO task_dependencies VALUES ('t3','t2')`,
	)
	writeArtifact(t, root, "t2", "activity.jsonl",
		`{"stage":"IMPLEMENT","action":"implementing","timestamp":"2026-01-01T00:00:01Z"}`+"\n")
	if err := os.WriteFile(filepath.Join(root, ".agent-sdlc", "plan.meta.json"),
		[]byte(`{"source":"docs/PLAN.md","plan_id":"p"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	c, err := New([]string{root}, "sop", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	pid := config.ProjectID(root)
	ctx := context.Background()

	// GetPlan
	if src, ok := c.PlanSource(pid); !ok || src != "docs/PLAN.md" {
		t.Errorf("GetPlan -> PlanSource = %q,%v, want docs/PLAN.md,true", src, ok)
	}

	// GetTasks + GetTaskProgress
	proj, err := c.Project(ctx, pid)
	if err != nil {
		t.Fatal(err)
	}
	if len(proj.Tasks) != 3 {
		t.Fatalf("GetTasks -> %d tasks, want 3", len(proj.Tasks))
	}
	if proj.Summary.Total != 3 || proj.Summary.Completed != 1 || proj.Summary.PercentComplete() != 33 {
		t.Errorf("GetTaskProgress summary wrong: %+v", proj.Summary)
	}

	// GetTask
	task, err := c.Task(ctx, pid, "t2")
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != StatusBlocked || task.BlockedReason != "REVIEW_UNRESOLVED" {
		t.Errorf("GetTask status = %q/%q, want BLOCKED/REVIEW_UNRESOLVED", task.Status, task.BlockedReason)
	}
	// t3 depends on the non-terminal t2, so t3 is blocked by it.
	blocked, err := c.Task(ctx, pid, "t3")
	if err != nil {
		t.Fatal(err)
	}
	if len(blocked.BlockedBy) != 1 || blocked.BlockedBy[0] != "t2" {
		t.Errorf("GetTask(t3) blockedBy = %v, want [t2]", blocked.BlockedBy)
	}

	// GetTaskActivity
	ev, err := c.Activity(ctx, pid, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(ev) != 1 || ev[0].Stage != "IMPLEMENT" {
		t.Errorf("GetTaskActivity = %+v, want one IMPLEMENT event", ev)
	}

	// Not-found cases must use the boundary sentinels, not a controller guess.
	if _, err := c.Project(ctx, "nope"); !errors.Is(err, ErrProjectNotFound) {
		t.Errorf("Project(unknown) err = %v, want ErrProjectNotFound", err)
	}
	if _, err := c.Task(ctx, pid, "missing"); !errors.Is(err, ErrTaskNotFound) {
		t.Errorf("Task(unknown) err = %v, want ErrTaskNotFound", err)
	}
	if _, err := c.Activity(ctx, "nope", 0); !errors.Is(err, ErrProjectNotFound) {
		t.Errorf("Activity(unknown project) err = %v, want ErrProjectNotFound", err)
	}
}

// The command operations delegate to SOP's own CLI verbs; the controller runs no
// command logic itself.
func TestCommandOperationsDelegateToSOP(t *testing.T) {
	root := newProject(t)
	bin, args := fakeSop(t)
	c, err := New([]string{root}, bin, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	pid := config.ProjectID(root)
	ctx := context.Background()

	cases := []struct {
		name string
		verb string
		run  func() error
	}{
		{"start-or-continue-run (run)", "run", func() error { return c.Run(ctx, pid) }},
		{"start-or-continue-run (resume)", "resume", func() error { return c.Resume(ctx, pid) }},
		{"retry-task", "retry", func() error { return c.Retry(ctx, pid, "t2") }},
		{"get-task-report", "report", func() error { _, err := c.ReportTask(ctx, pid, "t2"); return err }},
		{"reconcile-plan", "reconcile", func() error { _, err := c.Reconcile(ctx, pid, "docs/PLAN.md"); return err }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(); err != nil {
				t.Fatalf("run: %v", err)
			}
			got := args()
			if len(got) == 0 || got[0] != tc.verb {
				t.Fatalf("sop argv = %v, want verb %q", got, tc.verb)
			}
		})
	}
}

// No boundary operation writes SOP persistence: reads report SOP state and
// commands delegate to the sop CLI, so the controller-side call itself leaves
// state.db and the run artifacts byte-for-byte unchanged.
func TestBoundaryDoesNotMutateSOPPersistence(t *testing.T) {
	root := newProject(t)
	now := time.Now().UTC().Format(time.RFC3339)
	seed(t, root, `INSERT INTO tasks VALUES ('t2','Task Two','obj','ac','BLOCKED','REVIEW_UNRESOLVED',1,3,'`+now+`','`+now+`')`)
	writeArtifact(t, root, "t2", "state.json", `{"id":"t2","stage":"WAITING_FOR_HUMAN"}`)
	bin, _ := fakeSop(t)
	c, err := New([]string{root}, bin, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	pid := config.ProjectID(root)
	ctx := context.Background()

	dir := filepath.Join(root, ".agent-sdlc")
	before := snapshotTree(t, dir)

	_, _ = c.Projects(ctx)
	_, _ = c.Project(ctx, pid)
	_, _ = c.Task(ctx, pid, "t2")
	_, _ = c.Activity(ctx, pid, 0)
	_, _ = c.PlanSource(pid)
	_ = c.Run(ctx, pid)
	_ = c.Resume(ctx, pid)
	_ = c.Retry(ctx, pid, "t2")
	_, _ = c.ReportTask(ctx, pid, "t2")
	_, _ = c.Reconcile(ctx, pid, "docs/PLAN.md")
	_ = c.CancelRun(ctx, pid)
	_ = c.ApproveTask(ctx, pid, "t2")
	_ = c.DeclineTask(ctx, pid, "t2")
	_, _ = c.ChangedTasks(ctx, pid)
	_ = c.AcceptChangedTask(ctx, pid, "t2")

	after := snapshotTree(t, dir)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("boundary mutated SOP persistence:\nbefore=%v\nafter =%v", before, after)
	}
}

// snapshotTree maps every file under dir to the hex digest of its contents, so a
// mutation (edit, add, or delete) shows as a diff.
func snapshotTree(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		out[rel] = fmt.Sprintf("%x", sha256.Sum256(b))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

package sopclient

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE tasks (
  id TEXT PRIMARY KEY, title TEXT NOT NULL, objective TEXT, acceptance_criteria TEXT,
  status TEXT NOT NULL, blocked_reason TEXT, attempt INTEGER NOT NULL DEFAULT 0,
  max_attempts INTEGER NOT NULL DEFAULT 1, created_at TEXT NOT NULL, updated_at TEXT NOT NULL);
CREATE TABLE task_dependencies (
  task_id TEXT NOT NULL, dependency_task_id TEXT NOT NULL,
  PRIMARY KEY (task_id, dependency_task_id));
CREATE TABLE task_attempts (
  task_id TEXT NOT NULL, number INTEGER NOT NULL, status TEXT NOT NULL, reason TEXT,
  output TEXT, duration INTEGER NOT NULL DEFAULT 0, timestamp TEXT NOT NULL,
  PRIMARY KEY (task_id, number));
CREATE TABLE handoffs (
  task_id TEXT PRIMARY KEY, capsule_json TEXT NOT NULL, status TEXT NOT NULL,
  content TEXT, references_json TEXT, compression_error TEXT, created_at TEXT NOT NULL);
`

// newProject returns a temp project root with an empty SOP state database.
func newProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	initProjectAt(t, root, "demo")
	return root
}

// initProjectAt initializes an SOP project at dir with the given declared name:
// a .agent-sdlc/config.yaml and an empty state database with the SOP schema.
func initProjectAt(t *testing.T, dir, name string) {
	t.Helper()
	sdlc := filepath.Join(dir, ".agent-sdlc")
	if err := os.MkdirAll(filepath.Join(sdlc, "runs", "t2"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := "project:\n  name: \"" + name + "\"\n  integration_branch: main\n"
	if err := os.WriteFile(filepath.Join(sdlc, "config.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", "file:"+filepath.Join(sdlc, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(schema); err != nil {
		t.Fatal(err)
	}
}

func seed(t *testing.T, root string, stmts ...string) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+StatePath(root))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("seed %q: %v", s, err)
		}
	}
}

func TestSummaryTasksAndBlocking(t *testing.T) {
	root := newProject(t)
	now := time.Now().UTC().Format(time.RFC3339)
	seed(t, root,
		`INSERT INTO tasks VALUES ('a','Task A','obj','ac','LOCAL_DONE',NULL,1,3,'`+now+`','`+now+`')`,
		`INSERT INTO tasks VALUES ('b','Task B','obj','ac','PLANNED',NULL,0,3,'`+now+`','`+now+`')`,
		`INSERT INTO tasks VALUES ('c','Task C','obj','ac','BLOCKED','REVIEW_UNRESOLVED',0,3,'`+now+`','`+now+`')`,
		`INSERT INTO task_dependencies VALUES ('b','a')`,
		`INSERT INTO task_dependencies VALUES ('c','b')`,
	)

	st, err := OpenStore(root)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	ctx := context.Background()
	sum, err := st.Summary(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if sum.Name != "demo" || sum.Branch != "main" {
		t.Fatalf("meta = %q/%q, want demo/main", sum.Name, sum.Branch)
	}
	if sum.Total != 3 || sum.Completed != 1 || sum.Blocked != 1 || sum.Planned != 1 {
		t.Fatalf("summary counts wrong: %+v", sum)
	}
	if sum.PercentComplete() != 33 {
		t.Fatalf("percent = %d, want 33", sum.PercentComplete())
	}

	tasks, err := st.Tasks(ctx)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]TaskSummary{}
	for _, task := range tasks {
		byID[task.ID] = task
	}
	if b := byID["b"]; len(b.BlockedBy) != 0 || !b.Eligible() {
		t.Fatalf("b should be eligible (dep a completed): %+v", b)
	}
	if c := byID["c"]; len(c.BlockedBy) != 1 || c.BlockedBy[0] != "b" || c.Eligible() {
		t.Fatalf("c should be blocked by b: %+v", c)
	}
}

func TestTaskDetailReadsArtifacts(t *testing.T) {
	root := newProject(t)
	now := time.Now().UTC().Format(time.RFC3339)
	seed(t, root,
		`INSERT INTO tasks VALUES ('t2','Task Two','do the thing','works; tested','FIX_REQUIRED','build failed',2,3,'`+now+`','`+now+`')`,
		`INSERT INTO task_attempts VALUES ('t2',1,'FIX_REQUIRED','build failed','boom',1500000000,'`+now+`')`,
		`INSERT INTO handoffs VALUES ('t2','{}','COMPRESSED','carry me','[]',NULL,'`+now+`')`,
	)
	run := filepath.Join(root, ".agent-sdlc", "runs", "t2")
	if err := os.WriteFile(filepath.Join(run, "validation.json"),
		[]byte(`{"Status":"FAIL","Results":[{"Category":"BUILD","Command":"go build ./...","ExitCode":1,"Stderr":"nope","Status":"FAIL"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(run, "review.json"),
		[]byte(`{"Summary":"ok","Findings":[{"Severity":"high","Title":"thing","File":"x.go"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	st, err := OpenStore(root)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	d, err := st.Task(context.Background(), "t2")
	if err != nil {
		t.Fatal(err)
	}
	if d.Objective != "do the thing" || d.LatestFailure != "build failed" {
		t.Fatalf("task detail wrong: %+v", d)
	}
	if len(d.Validation) != 1 || d.Validation[0].Category != "BUILD" {
		t.Fatalf("validation not read: %+v", d.Validation)
	}
	if len(d.Review.Findings) != 1 || d.Review.Findings[0].Severity != "high" {
		t.Fatalf("review not read: %+v", d.Review)
	}
	if d.Handoff == nil || d.Handoff.Status != "COMPRESSED" {
		t.Fatalf("handoff not read: %+v", d.Handoff)
	}
}

func TestTaskNotFound(t *testing.T) {
	root := newProject(t)
	st, err := OpenStore(root)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if _, err := st.Task(context.Background(), "missing"); err != ErrTaskNotFound {
		t.Fatalf("err = %v, want ErrTaskNotFound", err)
	}
}

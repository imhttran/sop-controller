package web

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	assets "sop-controller"

	"sop-controller/internal/config"
	"sop-controller/internal/sopclient"

	_ "modernc.org/sqlite"
)

// CTRL015 dogfood scenario, modeled on the completed JEV task workflow.
//
// Fixture contract (S1): SOP's state.db is opened read-only for the lifetime of
// the server (internal/sopclient.OpenStore), so a scripted step may not replace
// that file wholesale — a swapped file is invisible to the already-open
// connection (observed directly: a `rm`+`cp` leaves the open handle reading the
// old inode). Every scripted mutation below is therefore a real SQL UPDATE/
// INSERT against the *same* state.db file, executed by a second, short-lived
// read-write connection the fake `sop` binary opens for that one call, plus
// plain file writes under runs/<task>/ and plan.meta.json. activity.jsonl is
// always written through the fixture's "append" map, never "write", so a
// failure-stage line recorded in an earlier step survives a later step. No
// production (non-test) file changes to support this: the fake binary's source
// lives only in this test file, compiled to a temp binary per test run.
//
// Each scripted call is numbered by invocation order (0, 1, 2, ...), tracked by
// a counter file under <project-root>/.dogfood-fixture/ (never under
// .agent-sdlc, so it is invisible to the controller's own reads). Call N's
// mutation comes from .dogfood-fixture/steps/N.json, written by the test before
// the server starts; a call with no matching step file is a no-op.

// fakeStatefulSopSrc is the fake `sop` binary used by the dogfood tests. Unlike
// commands_test.go's fakeSop (which only records argv), this one actually
// mutates the project's state.db and run artifacts per invocation, because the
// dogfood scenario needs to observe real lifecycle transitions (LOCAL_DONE,
// plan COMPLETE, BLOCKED -> PASSED) across several sequential commands.
const fakeStatefulSopSrc = `package main

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"

	_ "modernc.org/sqlite"
)

type step struct {
	SQL    []string          ` + "`json:\"sql\"`" + `
	Write  map[string]string ` + "`json:\"write\"`" + `
	Append map[string]string ` + "`json:\"append\"`" + `
}

func main() {
	fixDir := ".dogfood-fixture"
	callsDir := filepath.Join(fixDir, "calls")
	os.MkdirAll(callsDir, 0o755)

	counterPath := filepath.Join(fixDir, "counter")
	n := 0
	if b, err := os.ReadFile(counterPath); err == nil {
		n, _ = strconv.Atoi(string(b))
	}
	var argv string
	for _, a := range os.Args[1:] {
		argv += a + "\n"
	}
	os.WriteFile(filepath.Join(callsDir, strconv.Itoa(n)+".argv"), []byte(argv), 0o644)
	os.WriteFile(counterPath, []byte(strconv.Itoa(n+1)), 0o644)

	raw, err := os.ReadFile(filepath.Join(fixDir, "steps", strconv.Itoa(n)+".json"))
	if err != nil {
		return
	}
	var st step
	if err := json.Unmarshal(raw, &st); err != nil {
		os.Exit(1)
	}
	if len(st.SQL) > 0 {
		db, err := sql.Open("sqlite", "file:.agent-sdlc/state.db")
		if err != nil {
			os.Exit(1)
		}
		defer db.Close()
		for _, q := range st.SQL {
			if _, err := db.Exec(q); err != nil {
				os.Exit(1)
			}
		}
	}
	for rel, content := range st.Write {
		p := filepath.Join(".agent-sdlc", rel)
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(content), 0o644)
	}
	for rel, content := range st.Append {
		p := filepath.Join(".agent-sdlc", rel)
		os.MkdirAll(filepath.Dir(p), 0o755)
		f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			os.Exit(1)
		}
		f.WriteString(content)
		f.Close()
	}
}
`

var (
	fakeStatefulSopBinOnce sync.Once
	fakeStatefulSopBinPath string
	fakeStatefulSopBinErr  error
)

// fakeStatefulSopBin compiles fakeStatefulSopSrc once per test run and returns
// the path to the resulting binary. Every dogfood test shares the compiled
// binary; each gets its own fixture via its own project root (the binary reads
// fixture state from its cwd, which Commander.Exec sets to the project root).
func fakeStatefulSopBin(t *testing.T) string {
	t.Helper()
	fakeStatefulSopBinOnce.Do(func() {
		dir, err := os.MkdirTemp("", "dogfood-sop-src")
		if err != nil {
			fakeStatefulSopBinErr = err
			return
		}
		src := filepath.Join(dir, "main.go")
		if err := os.WriteFile(src, []byte(fakeStatefulSopSrc), 0o644); err != nil {
			fakeStatefulSopBinErr = err
			return
		}
		bin := filepath.Join(dir, "sop")
		cmd := exec.Command("go", "build", "-o", bin, src)
		cmd.Dir = moduleRoot(t)
		if out, err := cmd.CombinedOutput(); err != nil {
			fakeStatefulSopBinErr = fmt.Errorf("%w: %s", err, out)
			return
		}
		fakeStatefulSopBinPath = bin
	})
	if fakeStatefulSopBinErr != nil {
		t.Fatalf("build fake sop: %v", fakeStatefulSopBinErr)
	}
	return fakeStatefulSopBinPath
}

// moduleRoot locates this module's root (the directory containing go.mod) so
// `go build` for the fake binary runs with this module's dependencies
// (modernc.org/sqlite) resolvable, without a separate go.mod.
func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above " + dir)
		}
		dir = parent
	}
}

// writeFixtureStep records the Nth scripted mutation for a project root's fake
// `sop`, applied the Nth time that binary is invoked against this root.
func writeFixtureStep(t *testing.T, root string, n int, sqlStmts []string, write, appendFiles map[string]string) {
	t.Helper()
	dir := filepath.Join(root, ".dogfood-fixture", "steps")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	payload := struct {
		SQL    []string          `json:"sql"`
		Write  map[string]string `json:"write"`
		Append map[string]string `json:"append"`
	}{sqlStmts, write, appendFiles}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, strconv.Itoa(n)+".json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

// fixtureCall reads the recorded argv for the Nth call the fake `sop` received
// for this project root, oldest call first. It fails the test if that call
// never happened.
func fixtureCall(t *testing.T, root string, n int) []string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, ".dogfood-fixture", "calls", strconv.Itoa(n)+".argv"))
	if err != nil {
		t.Fatalf("fake sop call %d not recorded: %v", n, err)
	}
	return strings.Fields(string(raw))
}

// hashFile returns a hex sha256 of a file's contents, for before/after
// mutation comparisons. A missing file hashes to "".
func hashFile(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// newDogfoodServer builds a server over a fresh project root with the fake
// stateful `sop` wired in, running the given seed SQL against state.db. It
// returns the server, project id, and project root.
func newDogfoodServer(t *testing.T, seed []string) (*httptest.Server, string, string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, ".agent-sdlc")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"),
		[]byte("project:\n  name: \"dogfood\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", "file:"+filepath.Join(dir, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	stmts := append([]string{testSchema}, seed...)
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	db.Close()

	sop, err := sopclient.New([]string{root}, fakeStatefulSopBin(t), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sop.Close() })
	views, err := NewViews(assets.FS)
	if err != nil {
		t.Fatalf("views: %v", err)
	}
	staticFS, err := fs.Sub(assets.FS, "static")
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(NewServer(Options{SOP: sop, Views: views, StaticFS: staticFS, Poll: time.Second, CommandTimeout: time.Minute}))
	t.Cleanup(srv.Close)
	return srv, config.ProjectID(root), root
}

// waitCommandDone polls a project- or task-scoped command status fragment
// until it leaves "running", or fails the test. verbKey matches CommandRunner's
// key (e.g. "run" for the project start-or-continue surface, "retry:jev013" for
// a task-scoped retry), and path is the GET status URL to poll.
func waitCommandDone(t *testing.T, srv *httptest.Server, path string) string {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	var body string
	for time.Now().Before(deadline) {
		_, body = get(t, srv.URL+path)
		if !strings.Contains(body, "cmd-running") {
			return body
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("command at %s did not finish: %s", path, body)
	return ""
}

// startOrContinue issues the CSRF-protected start-or-continue POST and waits
// for it to finish, returning the resulting command-status fragment.
func startOrContinue(t *testing.T, srv *httptest.Server, project string) string {
	t.Helper()
	if code := postWithCSRF(t, srv, "/projects/"+project+"/commands/run"); code != http.StatusOK {
		t.Fatalf("start-or-continue POST: %d", code)
	}
	return waitCommandDone(t, srv, "/projects/"+project+"/commands/run")
}

// retryTask issues the CSRF-protected retry POST for one task and waits for it
// to finish.
func retryTask(t *testing.T, srv *httptest.Server, project, task string) string {
	t.Helper()
	if code := postWithCSRF(t, srv, "/projects/"+project+"/tasks/"+task+"/commands/retry"); code != http.StatusOK {
		t.Fatalf("retry POST: %d", code)
	}
	return waitCommandDone(t, srv, "/projects/"+project+"/tasks/"+task+"/commands/retry")
}

const activityStages = `{"stage":"DISCOVER","action":"reading","detail":"internal/cache/cache.go","timestamp":"2026-01-01T00:00:01Z"}
{"stage":"CHANGE","action":"adding caching layer","timestamp":"2026-01-01T00:00:02Z"}
{"stage":"VALIDATE","action":"go test ./...","timestamp":"2026-01-01T00:00:03Z"}
{"stage":"REVIEW","action":"reviewing changes","timestamp":"2026-01-01T00:00:04Z"}
{"stage":"JEV","action":"analyzing changes","timestamp":"2026-01-01T00:00:05Z"}
{"stage":"QUALITY","action":"PASS","timestamp":"2026-01-01T00:00:06Z"}
`

// TestDogfoodNormalFlow drives the controller's real HTTP surface through the
// completed-JEV-task normal flow: Start selects and runs the first task, the
// activity/validation/review/JEV/quality stages become visible, the task
// reaches LOCAL_DONE, Continue advances the second task, and the project
// reports plan COMPLETE with FinalGate PASS. Every lifecycle transition comes
// from the scripted fake `sop`, never from the controller writing state.db or a
// run artifact directly (S5): GET-only inspection in between never changes the
// state.db hash.
func TestDogfoodNormalFlow(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	srv, id, root := newDogfoodServer(t, []string{
		`INSERT INTO tasks VALUES ('jev013','JEV013: add caching layer','objective','criteria','PLANNED',NULL,0,3,'` + now + `','` + now + `')`,
		`INSERT INTO tasks VALUES ('t2','Wire caching into the handler','objective2','criteria2','PLANNED',NULL,0,3,'` + now + `','` + now + `')`,
		`INSERT INTO task_dependencies VALUES ('t2','jev013')`,
	})
	if err := os.WriteFile(filepath.Join(root, ".agent-sdlc", "plan.meta.json"),
		[]byte(`{"source":"docs/PLAN.md","plan_id":"dogfood"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Step 0 (the "run" call): jev013 completes a full lifecycle to LOCAL_DONE.
	writeFixtureStep(t, root, 0,
		[]string{
			`UPDATE tasks SET status='LOCAL_DONE', attempt=1, updated_at='` + now + `' WHERE id='jev013'`,
			`INSERT INTO task_attempts VALUES ('jev013',1,'PASSED','','',0,'` + now + `')`,
		},
		map[string]string{
			"runs/jev013/state.json":  `{"id":"jev013","stage":"PASSED"}`,
			"runs/jev013/report.json": `{"id":"jev013","stage":"PASSED","decision":"PASS","classification":null,"jev":{"status":"PASS","findings":[]},"fix_cycles":0}`,
		},
		map[string]string{"runs/jev013/activity.jsonl": activityStages},
	)
	// Step 1 (the "resume" call): t2 completes too, and the plan gate closes.
	writeFixtureStep(t, root, 1,
		[]string{
			`UPDATE tasks SET status='LOCAL_DONE', attempt=1, updated_at='` + now + `' WHERE id='t2'`,
			`INSERT INTO task_attempts VALUES ('t2',1,'PASSED','','',0,'` + now + `')`,
		},
		map[string]string{
			"runs/t2/state.json":  `{"id":"t2","stage":"PASSED"}`,
			"runs/t2/report.json": `{"id":"t2","stage":"PASSED","decision":"PASS","classification":null,"jev":{"status":"PASS","findings":[]},"fix_cycles":0}`,
			"plan.meta.json":      `{"source":"docs/PLAN.md","plan_id":"dogfood","final_gate":"PASS"}`,
		},
		nil,
	)

	statePath := filepath.Join(root, ".agent-sdlc", "state.db")

	// Before Start: the project page shows jev013 as not yet run.
	_, body := get(t, srv.URL+"/projects/"+id)
	if !strings.Contains(body, "jev013") {
		t.Fatalf("project page missing task: %q", body)
	}
	if strings.Contains(body, "LOCAL_DONE") {
		t.Fatalf("task should not show LOCAL_DONE before Start: %q", body)
	}

	// Start: dispatches `sop run`, SOP selects jev013, runs it to LOCAL_DONE.
	beforeHash := hashFile(statePath)
	startOrContinue(t, srv, id)
	if got := fixtureCall(t, root, 0); !sliceEqual(got, []string{"run"}) {
		t.Fatalf("first start-or-continue argv = %v, want [run]", got)
	}
	afterHash := hashFile(statePath)
	if beforeHash == afterHash {
		t.Fatalf("state.db unchanged after the scripted run; fixture mutation did not take effect")
	}

	// Activity, validation/review/JEV/quality, and LOCAL_DONE are now visible.
	_, activity := get(t, srv.URL+"/projects/"+id+"/tasks/jev013/activity")
	for _, want := range []string{"DISCOVER", "CHANGE", "VALIDATE", "REVIEW", "JEV", "QUALITY"} {
		if !strings.Contains(activity, want) {
			t.Errorf("activity missing stage %q: %q", want, activity)
		}
	}
	_, task := get(t, srv.URL+"/projects/"+id+"/tasks/jev013")
	if !strings.Contains(task, "LOCAL_DONE") {
		t.Fatalf("task not LOCAL_DONE: %q", task)
	}
	if !strings.Contains(task, "JEV PASS") {
		t.Fatalf("JEV status not PASS: %q", task)
	}

	// Read-only inspection between commands must never mutate state.db.
	readOnlyHash := hashFile(statePath)
	get(t, srv.URL+"/projects/"+id)
	get(t, srv.URL+"/projects/"+id+"/tasks/jev013")
	get(t, srv.URL+"/projects/"+id+"/tasks/jev013/activity")
	if hashFile(statePath) != readOnlyHash {
		t.Fatalf("GET-only inspection mutated state.db")
	}

	// Continue: dispatches `sop resume`, SOP selects t2, plan reaches COMPLETE.
	startOrContinue(t, srv, id)
	if got := fixtureCall(t, root, 1); !sliceEqual(got, []string{"resume"}) {
		t.Fatalf("second start-or-continue argv = %v, want [resume]", got)
	}

	_, project := get(t, srv.URL+"/projects/"+id)
	if !strings.Contains(project, "plan gate PASS") {
		t.Fatalf("project page missing plan gate PASS: %q", project)
	}
	if !strings.Contains(project, "2/2") {
		t.Fatalf("project page does not show Completed==Total: %q", project)
	}
}

// TestDogfoodRecoveryFlow reproduces the JEV013 provider-failure incident: the
// task starts BLOCKED with a transient-provider classification and retry budget
// remaining, the controller shows the reason without a human-attributed
// AUTO_FIX disposition or an elevated FixCycles count, a human-triggered Retry
// drives `sop retry`, the scripted fake CLI advances the task to LOCAL_DONE
// while preserving the pre-retry failure evidence, and the plan completes.
func TestDogfoodRecoveryFlow(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	srv, id, root := newDogfoodServer(t, []string{
		`INSERT INTO tasks VALUES ('jev013','JEV013: add caching layer','objective','criteria','BLOCKED','Ollama provider returned HTTP 500 during generation',1,3,'` + now + `','` + now + `')`,
		`INSERT INTO tasks VALUES ('t2','Wire caching into the handler','objective2','criteria2','PLANNED',NULL,0,3,'` + now + `','` + now + `')`,
		`INSERT INTO task_dependencies VALUES ('t2','jev013')`,
		`INSERT INTO task_attempts VALUES ('jev013',1,'FAILED','provider failure: Ollama HTTP 500','',0,'` + now + `')`,
	})
	if err := os.WriteFile(filepath.Join(root, ".agent-sdlc", "plan.meta.json"),
		[]byte(`{"source":"docs/PLAN.md","plan_id":"dogfood"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	// Pre-existing failure evidence, as if written by the earlier BLOCKED run:
	// a provider/transient classification (RETRY, never AUTO_FIX) and the
	// activity line recording the HTTP 500.
	for name, content := range map[string]string{
		"activity.jsonl":      `{"stage":"VALIDATE","action":"command","detail":"provider request failed: ollama http 500","timestamp":"2026-01-01T00:00:01Z"}` + "\n",
		"state.json":          `{"id":"jev013","stage":"FAILED"}`,
		"classification.json": `{"kind":"TRANSIENT_PROVIDER","disposition":"RETRY","confidence":"HIGH","reason":"Ollama provider returned HTTP 500 during generation"}`,
		"report.json":         `{"id":"jev013","stage":"FAILED","decision":"FAIL","classification":{"kind":"TRANSIENT_PROVIDER","disposition":"RETRY","confidence":"HIGH","reason":"Ollama provider returned HTTP 500 during generation"},"fix_cycles":0}`,
	} {
		writeRun(t, root, "jev013", name, content)
	}

	// Step 0 (the "retry jev013" call): the provider succeeds this time; jev013
	// completes to LOCAL_DONE. The activity line is appended, never replacing
	// the earlier failure line, and a second attempt row is added alongside the
	// first rather than replacing it.
	writeFixtureStep(t, root, 0,
		[]string{
			`UPDATE tasks SET status='LOCAL_DONE', blocked_reason=NULL, attempt=2, updated_at='` + now + `' WHERE id='jev013'`,
			`INSERT INTO task_attempts VALUES ('jev013',2,'PASSED','','',0,'` + now + `')`,
		},
		map[string]string{
			"runs/jev013/state.json":          `{"id":"jev013","stage":"PASSED"}`,
			"runs/jev013/report.json":         `{"id":"jev013","stage":"PASSED","decision":"PASS","classification":null,"jev":{"status":"PASS","findings":[]},"fix_cycles":0}`,
			"runs/jev013/classification.json": `{}`,
		},
		map[string]string{"runs/jev013/activity.jsonl": activityStages},
	)
	// Step 1 (the subsequent "resume" call continuing the plan): t2 completes
	// and the plan gate closes.
	writeFixtureStep(t, root, 1,
		[]string{
			`UPDATE tasks SET status='LOCAL_DONE', attempt=1, updated_at='` + now + `' WHERE id='t2'`,
			`INSERT INTO task_attempts VALUES ('t2',1,'PASSED','','',0,'` + now + `')`,
		},
		map[string]string{
			"runs/t2/state.json":  `{"id":"t2","stage":"PASSED"}`,
			"runs/t2/report.json": `{"id":"t2","stage":"PASSED","decision":"PASS","classification":null,"jev":{"status":"PASS","findings":[]},"fix_cycles":0}`,
			"plan.meta.json":      `{"source":"docs/PLAN.md","plan_id":"dogfood","final_gate":"PASS"}`,
		},
		nil,
	)

	statePath := filepath.Join(root, ".agent-sdlc", "state.db")

	// Before Retry: BLOCKED, retryable, reason shown, RETRY disposition (never
	// AUTO_FIX), FixCycles not elevated by the provider failure itself.
	_, before := get(t, srv.URL+"/projects/"+id+"/tasks/jev013")
	if !strings.Contains(before, "BLOCKED") {
		t.Fatalf("task not shown BLOCKED: %q", before)
	}
	if !strings.Contains(before, "Ollama provider returned HTTP 500") {
		t.Fatalf("provider-failure reason not shown: %q", before)
	}
	if !strings.Contains(before, "RETRY") || strings.Contains(before, "AUTO_FIX") {
		t.Fatalf("expected RETRY disposition, never AUTO_FIX, before retry: %q", before)
	}
	if !strings.Contains(before, "callout-human") {
		t.Fatalf("a BLOCKED task must still show the human boundary callout: %q", before)
	}

	beforeHash := hashFile(statePath)
	retryTask(t, srv, id, "jev013")
	if got := fixtureCall(t, root, 0); !sliceEqual(got, []string{"retry", "jev013"}) {
		t.Fatalf("retry argv = %v, want [retry jev013]", got)
	}
	afterHash := hashFile(statePath)
	if beforeHash == afterHash {
		t.Fatalf("state.db unchanged after the scripted retry; fixture mutation did not take effect")
	}

	// After Retry: PASSED/LOCAL_DONE, and the pre-retry failure evidence is
	// still visible (activity line + the original attempt row), not discarded.
	_, after := get(t, srv.URL+"/projects/"+id+"/tasks/jev013")
	if !strings.Contains(after, "LOCAL_DONE") {
		t.Fatalf("task not LOCAL_DONE after retry: %q", after)
	}
	if !strings.Contains(after, "JEV PASS") {
		t.Fatalf("JEV status not PASS after retry: %q", after)
	}
	if !strings.Contains(after, "provider failure: Ollama HTTP 500") {
		t.Fatalf("pre-retry attempt evidence missing after retry: %q", after)
	}
	_, activity := get(t, srv.URL+"/projects/"+id+"/tasks/jev013/activity")
	if !strings.Contains(activity, "provider request failed: ollama http 500") {
		t.Fatalf("pre-retry failure activity line missing after retry (not append-only): %q", activity)
	}
	for _, want := range []string{"DISCOVER", "CHANGE", "VALIDATE", "REVIEW", "JEV", "QUALITY"} {
		if !strings.Contains(activity, want) {
			t.Errorf("activity missing post-retry stage %q: %q", want, activity)
		}
	}

	// SOP continues the remaining task, and the plan reaches COMPLETE.
	startOrContinue(t, srv, id)
	if got := fixtureCall(t, root, 1); !sliceEqual(got, []string{"resume"}) {
		t.Fatalf("continue argv = %v, want [resume]", got)
	}
	_, project := get(t, srv.URL+"/projects/"+id)
	if !strings.Contains(project, "plan gate PASS") {
		t.Fatalf("project page missing plan gate PASS: %q", project)
	}
	if !strings.Contains(project, "2/2") {
		t.Fatalf("project page does not show Completed==Total: %q", project)
	}
}

func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

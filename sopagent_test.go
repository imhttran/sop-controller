package assets

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Tests for scripts/sop-agent.sh — the SOP command-agent adapter — run against
// stub implementation agents, so the command-agent contract is covered
// deterministically without invoking a real model.

func scriptPath(t *testing.T) string {
	t.Helper()
	p, err := filepath.Abs(filepath.Join("scripts", "sop-agent.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("harness not found at %s: %v", p, err)
	}
	return p
}

func newRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	return dir
}

// stub writes an implementation-agent stub (kept outside the repo) and returns
// its path. The stub is invoked with the prompt as $1 and its cwd is the repo.
func stub(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "impl-stub.sh")
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

// invoke runs the adapter for a capability against the stub and returns stdout.
func invoke(t *testing.T, repo, stubPath, capability string) string {
	t.Helper()
	req := `{"capability":"` + capability + `","task":"t","input":"","output_requirements":""}`
	cmd := exec.Command("sh", scriptPath(t))
	cmd.Dir = repo
	cmd.Env = append(os.Environ(), "SOP_AGENT_IMPL="+stubPath)
	cmd.Stdin = strings.NewReader(req)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("harness run: %v\nstderr: %s", err, stderr.String())
	}
	return strings.TrimSpace(stdout.String())
}

func TestExecutionOutcomes(t *testing.T) {
	cases := []struct {
		name string
		body string
		want map[string]any
	}{
		{
			name: "completed with changes expected",
			body: `printf 'x\n' > artifact.txt
echo 'SOP_OUTCOME:{"status":"completed","summary":"added artifact","changes_expected":true}'`,
			want: map[string]any{"status": "completed", "changes_expected": true},
		},
		{
			name: "completed with no changes expected",
			body: `echo 'SOP_OUTCOME:{"status":"completed","summary":"baseline ok","changes_expected":false}'`,
			want: map[string]any{"status": "completed", "changes_expected": false},
		},
		{
			name: "needs human",
			body: `echo 'SOP_OUTCOME:{"status":"needs_human","reason":"needs auth"}'`,
			want: map[string]any{"status": "needs_human", "reason": "needs auth"},
		},
		{
			name: "failed",
			body: `echo 'SOP_OUTCOME:{"status":"failed","reason":"boom"}'`,
			want: map[string]any{"status": "failed", "reason": "boom"},
		},
		{
			name: "changes_expected omitted defaults to reality (changed)",
			body: `printf 'y\n' > artifact.txt
echo 'SOP_OUTCOME:{"status":"completed","summary":"did it"}'`,
			want: map[string]any{"status": "completed", "changes_expected": true},
		},
		{
			name: "malformed outcome fails conservatively",
			body: `echo 'SOP_OUTCOME:{not json'`,
			want: map[string]any{"status": "failed"},
		},
		{
			name: "no outcome, changes made -> completed true",
			body: `printf 'z\n' > artifact.txt`,
			want: map[string]any{"status": "completed", "changes_expected": true},
		},
		{
			name: "no outcome, no changes -> completed false",
			body: `true`,
			want: map[string]any{"status": "completed", "changes_expected": false},
		},
		{
			name: "no outcome, non-zero exit -> failed",
			body: `exit 3`,
			want: map[string]any{"status": "failed"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := invoke(t, newRepo(t), stub(t, tc.body), "IMPLEMENT")
			var got map[string]any
			if err := json.Unmarshal([]byte(out), &got); err != nil {
				t.Fatalf("stdout must be exactly one JSON object, got %q (%v)", out, err)
			}
			for k, want := range tc.want {
				if got[k] != want {
					t.Fatalf("%s = %#v, want %#v (out %q)", k, got[k], want, out)
				}
			}
		})
	}
}

// PLAN and REVIEW must keep returning exactly the structured JSON SOP parses,
// even when the implementation agent wraps it in prose.
func TestPlanAndReviewStructuredOutput(t *testing.T) {
	plan := invoke(t, newRepo(t), stub(t, `printf 'Here is the plan:\n{"project":"demo","stages":[]}\n'`), "PLAN")
	if !strings.HasPrefix(plan, "{") {
		t.Fatalf("PLAN output must start with JSON, got %q", plan)
	}
	var pm map[string]any
	if err := json.Unmarshal([]byte(plan), &pm); err != nil || pm["project"] != "demo" {
		t.Fatalf("PLAN output not the plan JSON: %q (%v)", plan, err)
	}

	review := invoke(t, newRepo(t), stub(t, `printf 'report:\n{"Summary":"ok","Findings":[]}\n'`), "REVIEW")
	var rm map[string]any
	if err := json.Unmarshal([]byte(review), &rm); err != nil || rm["Summary"] != "ok" {
		t.Fatalf("REVIEW output not the review JSON: %q (%v)", review, err)
	}
}

// The adapter must not carry workflow/state-machine behaviour: it neither writes
// SOP state nor references SOP lifecycle commands.
func TestAdapterCarriesNoWorkflowLogic(t *testing.T) {
	raw, err := os.ReadFile(scriptPath(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{
		"sop run", "sop retry", "sop commit", "sop validate", "sop review",
		"state.db", "BLOCKED", "PLANNED",
	} {
		if strings.Contains(string(raw), token) {
			t.Fatalf("adapter references workflow token %q", token)
		}
	}

	// A real invocation must not create SOP state.
	repo := newRepo(t)
	_ = invoke(t, repo, stub(t, `echo 'SOP_OUTCOME:{"status":"completed","changes_expected":false}'`), "IMPLEMENT")
	if _, err := os.Stat(filepath.Join(repo, ".agent-sdlc")); err == nil {
		t.Fatal("adapter created .agent-sdlc state; it must not touch SOP state")
	}
}

// Change detection must recognize all forms of repository modifications.
func TestChangeDetection(t *testing.T) {
	cases := []struct {
		name               string
		setup              string // commands to run in repo before invoking adapter
		stubBody           string // what the implementation agent does
		expectChangesFound bool
		expectSummary      string
	}{
		{
			name:               "no repository changes",
			setup:              `echo "initial" > file.txt; git add file.txt; git commit -q -m "init"`,
			stubBody:           `echo 'SOP_OUTCOME:{"status":"completed","summary":"did nothing","changes_expected":false}'`,
			expectChangesFound: false,
			expectSummary:      "did nothing",
		},
		{
			name:               "tracked file modified",
			setup:              `echo "initial" > file.txt; git add file.txt; git commit -q -m "init"`,
			stubBody:           `echo "modified" > file.txt; echo 'SOP_OUTCOME:{"status":"completed","summary":"modified tracked file"}'`,
			expectChangesFound: true,
			expectSummary:      "modified tracked file",
		},
		{
			name:               "tracked file deleted",
			setup:              `echo "initial" > file.txt; git add file.txt; git commit -q -m "init"`,
			stubBody:           `rm file.txt; echo 'SOP_OUTCOME:{"status":"completed","summary":"deleted tracked file"}'`,
			expectChangesFound: true,
			expectSummary:      "deleted tracked file",
		},
		{
			name:               "new untracked file created",
			setup:              `echo "initial" > file.txt; git add file.txt; git commit -q -m "init"`,
			stubBody:           `echo "new" > newfile.txt; echo 'SOP_OUTCOME:{"status":"completed","summary":"created new file"}'`,
			expectChangesFound: true,
			expectSummary:      "created new file",
		},
		{
			name:               "multiple new files created",
			setup:              `echo "initial" > file.txt; git add file.txt; git commit -q -m "init"`,
			stubBody:           `echo "new1" > file1.txt; echo "new2" > file2.txt; echo 'SOP_OUTCOME:{"status":"completed","summary":"created multiple files"}'`,
			expectChangesFound: true,
			expectSummary:      "created multiple files",
		},
		{
			name:               "ignored file created",
			setup:              `echo "initial" > file.txt; echo "*.log" > .gitignore; git add file.txt .gitignore; git commit -q -m "init"`,
			stubBody:           `echo "log" > test.log; echo 'SOP_OUTCOME:{"status":"completed","summary":"created ignored file","changes_expected":false}'`,
			expectChangesFound: false,
			expectSummary:      "created ignored file",
		},
		{
			name:               "only .agent-sdlc runtime files changed",
			setup:              `echo "initial" > file.txt; git add file.txt; git commit -q -m "init"`,
			stubBody:           `mkdir -p .agent-sdlc; echo "state" > .agent-sdlc/state.db; echo 'SOP_OUTCOME:{"status":"completed","summary":"runtime only","changes_expected":false}'`,
			expectChangesFound: false,
			expectSummary:      "runtime only",
		},
		{
			name:               "untracked file without explicit changes_expected",
			setup:              `echo "initial" > file.txt; git add file.txt; git commit -q -m "init"`,
			stubBody:           `echo "new" > newfile.txt; echo 'SOP_OUTCOME:{"status":"completed","summary":"created without explicit flag"}'`,
			expectChangesFound: true,
			expectSummary:      "created without explicit flag",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := newRepo(t)
			// Configure git for commits
			cmd := exec.Command("git", "config", "user.email", "test@test.com")
			cmd.Dir = repo
			cmd.Run()
			cmd = exec.Command("git", "config", "user.name", "Test")
			cmd.Dir = repo
			cmd.Run()

			// Setup repository state
			if tc.setup != "" {
				cmd := exec.Command("sh", "-c", tc.setup)
				cmd.Dir = repo
				if err := cmd.Run(); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			}

			// Invoke the adapter
			out := invoke(t, repo, stub(t, tc.stubBody), "IMPLEMENT")

			// Parse the JSON result
			var result map[string]any
			if err := json.Unmarshal([]byte(out), &result); err != nil {
				t.Fatalf("stdout must be JSON, got %q (%v)", out, err)
			}

			// Verify status is completed
			if result["status"] != "completed" {
				t.Fatalf("status = %v, want completed", result["status"])
			}

			// Verify summary
			if result["summary"] != tc.expectSummary {
				t.Fatalf("summary = %q, want %q", result["summary"], tc.expectSummary)
			}

			// Verify changes_expected matches expectation
			changesFound, ok := result["changes_expected"].(bool)
			if !ok {
				t.Fatalf("changes_expected is not a bool: %v", result["changes_expected"])
			}
			if changesFound != tc.expectChangesFound {
				t.Fatalf("changes_expected = %v, want %v (output: %q)", changesFound, tc.expectChangesFound, out)
			}
		})
	}
}

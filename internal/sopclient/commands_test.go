package sopclient

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"sop-controller/internal/config"
)

// fakeSop writes an executable "sop" that records its argv, one arg per line, and
// returns it. It lets a test assert exactly which SOP verb and arguments the
// controller drives, without invoking a real SOP.
func fakeSop(t *testing.T) (bin string, args func() []string) {
	t.Helper()
	dir := t.TempDir()
	record := filepath.Join(dir, "args")
	bin = filepath.Join(dir, "sop")
	script := "#!/bin/sh\n: > " + record + "\nfor a in \"$@\"; do printf '%s\\n' \"$a\" >> " + record + "; done\nexit 0\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return bin, func() []string {
		raw, err := os.ReadFile(record)
		if err != nil {
			return nil
		}
		return strings.Fields(string(raw))
	}
}

// Recovery operations must drive SOP's own commands, never mutate state directly.
func TestRecoveryCommandsDriveSop(t *testing.T) {
	root := newProject(t)
	now := time.Now().UTC().Format(time.RFC3339)
	seed(t, root, `INSERT INTO tasks VALUES ('t1','T','o','a','BLOCKED','X',3,3,'`+now+`','`+now+`')`)

	bin, args := fakeSop(t)
	c, err := New([]string{root}, bin, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	pid := config.ProjectID(root)

	cases := []struct {
		name string
		run  func() error
		want []string
	}{
		{"retry", func() error { return c.Retry(context.Background(), pid, "t1") }, []string{"retry", "t1"}},
		{"retry force", func() error { return c.RetryForce(context.Background(), pid, "t1") }, []string{"retry", "t1", "--force"}},
		{"retry all", func() error { return c.RetryAll(context.Background(), pid) }, []string{"retry", "--all"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(); err != nil {
				t.Fatalf("run: %v", err)
			}
			if got := args(); !equal(got, tc.want) {
				t.Fatalf("sop argv = %v, want %v", got, tc.want)
			}
		})
	}

	out, err := c.Reconcile(context.Background(), pid, "docs/PLAN.md")
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	_ = out
	if got := args(); !equal(got, []string{"reconcile", "docs/PLAN.md"}) {
		t.Fatalf("reconcile argv = %v", got)
	}
}

// SOP_ACTIVITY is set for controller-driven commands so SOP persists the
// activity artifact the dashboard reads; an operator-set value is preserved.
func TestActivityEnvDefault(t *testing.T) {
	got := activityEnv([]string{"PATH=/bin"})
	if !contains(got, "SOP_ACTIVITY=on") {
		t.Fatalf("activityEnv = %v, want SOP_ACTIVITY=on added", got)
	}
	kept := activityEnv([]string{"SOP_ACTIVITY=off"})
	if contains(kept, "SOP_ACTIVITY=on") {
		t.Fatalf("activityEnv overrode an operator value: %v", kept)
	}
}

func equal(a, b []string) bool {
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

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

package sopclient

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Commander runs SOP CLI verbs against a project root (FR-8).
type Commander struct {
	bin     string
	timeout time.Duration
}

// NewCommander returns a Commander that shells out to the SOP CLI.
func NewCommander(bin string, timeout time.Duration) *Commander {
	if bin == "" {
		bin = "sop"
	}
	if timeout <= 0 {
		timeout = 15 * time.Minute
	}
	return &Commander{bin: bin, timeout: timeout}
}

// Exec runs `sop <verb> [args...]` in dir and returns its combined output. It
// defaults SOP_ACTIVITY=on (observer-only) so runs persist activity.jsonl; that
// is the only environment the controller adds.
func (c *Commander) Exec(ctx context.Context, dir, verb string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.bin, append([]string{verb}, args...)...)
	cmd.Dir = dir
	cmd.Env = activityEnv(os.Environ())
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err := cmd.Run()
	out := buf.String()
	if ctx.Err() == context.DeadlineExceeded {
		return out, fmt.Errorf("sop %s timed out after %s", verb, c.timeout)
	}
	if err != nil {
		return out, fmt.Errorf("sop %s: %w", verb, err)
	}
	return out, nil
}

func activityEnv(env []string) []string {
	for _, kv := range env {
		if strings.HasPrefix(kv, "SOP_ACTIVITY=") {
			return env
		}
	}
	return append(env, "SOP_ACTIVITY=on")
}

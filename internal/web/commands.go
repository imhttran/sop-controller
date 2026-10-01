package web

import (
	"context"
	"sync"
	"time"
)

// CommandState is the latest result of a SOP command triggered from the UI.
type CommandState struct {
	Verb    string
	State   string // running | done | error
	Output  string
	Error   string
	Started time.Time
	// TaskID/Stage/Attempt surface the task SOP is actively working, when SOP
	// exposes one. All three stay zero-value (absent, never fabricated) unless
	// populated from a live sopclient read.
	TaskID  string
	Stage   string
	Attempt int
}

// Running reports whether the command is still in flight.
func (c CommandState) Running() bool { return c.State == "running" }

// CommandRunner runs SOP commands in the background and remembers the latest
// status per project/verb, so the UI never blocks on a long SOP lifecycle.
type CommandRunner struct {
	mu      sync.Mutex
	states  map[string]*CommandState
	timeout time.Duration
}

// NewCommandRunner returns an empty runner with the specified timeout.
func NewCommandRunner(timeout time.Duration) *CommandRunner {
	if timeout <= 0 {
		timeout = 15 * time.Minute
	}
	return &CommandRunner{states: map[string]*CommandState{}, timeout: timeout}
}

func (cr *CommandRunner) key(project, verb string) string { return project + "/" + verb }

// Start launches fn in the background and records its status.
// Returns (started=true, err) if command started, or (false, err) if already running.
func (cr *CommandRunner) Start(project, verb string, fn func(ctx context.Context) (string, error)) (bool, error) {
	cr.mu.Lock()
	k := cr.key(project, verb)
	if st, ok := cr.states[k]; ok && st.State == "running" {
		cr.mu.Unlock()
		return false, nil
	}
	st := &CommandState{Verb: verb, State: "running", Started: time.Now()}
	cr.states[k] = st
	cr.mu.Unlock()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), cr.timeout)
		defer cancel()
		out, err := fn(ctx)

		cr.mu.Lock()
		defer cr.mu.Unlock()
		st.Output = bound(out, 8000)
		if err != nil {
			st.State = "error"
			st.Error = err.Error()
			return
		}
		st.State = "done"
	}()
	return true, nil
}

// Status returns the last known status for a project/verb, if any.
func (cr *CommandRunner) Status(project, verb string) (CommandState, bool) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	st, ok := cr.states[cr.key(project, verb)]
	if !ok {
		return CommandState{}, false
	}
	return *st, true
}

// bound caps retained output so a runaway command can't bloat memory or the UI.
func bound(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "\n… (output truncated)"
}

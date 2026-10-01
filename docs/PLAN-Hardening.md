# PLAN --- SOP Controller Hardening: Remove Fixed-Wait Monitoring

## Project

sop-controller

## Summary

Monitoring an active SOP run must not depend on an arbitrary fixed wait such as:

```bash
sleep 240
tail -25 .run/sop-run.log
sop status ...
```

That 240-second delay is a polling delay, not SOP lifecycle semantics. Useful
progress may be available immediately; a human gate may occur long before four
minutes; a failure may occur immediately; a task may finish in seconds; and a
task may legitimately take longer than four minutes. The desired flow is:

```text
start SOP
   |
observe authoritative SOP state/activity
   |
update UI as state changes
   |
stop waiting when terminal/actionable state occurs
```

The controller already separates execution from observation (an asynchronous
command runner) and already exposes SOP activity (an SSE stream plus a
bounded-poll window). This plan verifies that separation, closes any gap where a
fixed wait would be the only way to observe progress, documents execution vs
observation vs polling vs timeout, and adds deterministic tests. It does not add
a second event system and does not introduce any fixed multi-minute wait.

## Objective

Make the controller/SOP observability surface sufficient that an operator never
needs an arbitrary multi-minute sleep to observe an active run, while keeping
`agentic-sop` the sole lifecycle authority and `sop-controller` purely an
observation and human-control surface.

## Architecture

```text
agentic-sop      = workflow/lifecycle authority
sop-controller   = observation + human control
```

Constraints:

- The controller MUST NOT decide that a SOP task is complete, blocked, failed, or
  waiting for a human based on elapsed time. It consumes SOP's authoritative
  state.
- Never implement `if elapsed > 240 seconds: assume finished` or
  `if no log output for N seconds: assume blocked`. Time is not lifecycle state.
- No fixed multi-minute sleep may be the normal observation mechanism.
- Polling, if used, is short, configurable, and secondary to activity streaming;
  reuse the existing poll configuration instead of hard-coding a new value.
- Time is diagnostic information (for example "last activity: 2m 14s ago"), never
  lifecycle authority.
- The command execution timeout is a distinct concept from the observation/
  polling interval.
- Do NOT change SOP routing, SMALL/MEDIUM/LARGE classification, JEV, provider
  selection, model resolution, recovery policy, approval policy, reconciliation
  policy, task scheduling, or retry budgets. This is an observability change only.
- Do not add Kafka, Redis, or distributed infrastructure; do not parse arbitrary
  model prose to infer state; do not introduce duplicate lifecycle states.

# HARD001 --- Locate and Report the Fixed-Wait Monitoring Source

Before changing behavior, inspect the current observation code and determine
where the fixed wait comes from. Inspect `internal/sopclient/run.go`,
`internal/sopclient/status.go`, `internal/sopclient/activity.go`,
`internal/sopclient/activity_window.go`, `internal/web/activity_stream.go`,
`internal/web/commands.go`, `internal/web/handlers.go`, `internal/web/server.go`,
and every `sop-run.log` usage in the repository. Search for `sleep`, `240`,
`time.Sleep`, `poll`, `ticker`, `tail`, `sop-run.log`, and `command timeout`.

Classify the result: production code, test code, documentation, scripts, or only
an operator/debugging workflow. If the fixed wait exists only in an external
manual shell command and not in `sop-controller`, do not invent a code change
merely to remove it; instead identify the observability gap that caused the
operator to resort to the wait.

### Acceptance Criteria

- A repository-wide search for the fixed-wait terms is performed and its results
  are recorded.
- The result states whether an arbitrary fixed multi-minute wait exists in
  production code, tests, documentation, scripts, or only an operator/debugging
  workflow.
- If it exists only externally, no code change is invented merely to remove it,
  and the report identifies the observability gap behind the manual wait.
- No production code, configuration value, or default timeout is changed by this
  task.

# HARD002 --- Immediate, Non-Blocking Status Observation

Starting a command and observing it remain separate operations. Starting SOP must
not sit idle for minutes merely to produce a useful status.

### Acceptance Criteria

- Starting a SOP command returns immediately with a command identity/status; the
  HTTP request that starts a command does not block for the command's duration.
- Current SOP status/progress is available as soon as the command is started
  (running state, and task/stage/attempt when SOP exposes them).
- No progress is fabricated: only fields SOP actually exposes are shown.
- The existing command-runner architecture is used; no new blocking path is added.

# HARD003 --- Activity Streaming as the Primary Observation Path

Use the existing activity stream and activity-window abstractions wherever
practical. Do not create a second logging or event system.

### Acceptance Criteria

- Active SOP activity is observable while the command runs, through the existing
  activity surface (SSE stream), with the bounded-poll window as the fallback.
- No second logging/event system is introduced.
- Per-project isolation is preserved: project A activity can never appear as
  project B progress.
- Fallback polling uses the existing configurable poll cadence; no new arbitrary
  interval is hard-coded.

# HARD004 --- Inactivity and Timeout Are Not Lifecycle States

### Acceptance Criteria

- Elapsed time without new activity is displayed as factual information only; it
  never yields FAILED, BLOCKED, or STUCK unless SOP reports that state.
- No `elapsed > N` => finished and no `no output for N` => blocked style inference
  exists anywhere in the observation path.
- The command execution timeout is clearly distinguished from UI polling, is
  documented, remains configurable, returns a truthful timeout error, does not
  silently mark the SOP task failed, and does not mutate SOP lifecycle state.

# HARD005 --- Surface Terminal and Human-Decision States Promptly

### Acceptance Criteria

- RUNNING, BLOCKED, FAILED, DONE, and SOP's human-decision states are surfaced as
  soon as SOP exposes them; the controller uses the actual states SOP exposes and
  introduces no duplicate lifecycle states.
- When SOP reaches a human decision boundary, the UI stops appearing as though
  the system is simply "still working" and surfaces "Needs your attention" with
  the appropriate approval/reconciliation controls.
- Approval, decline, and reconcile delegate to SOP's application boundary; the
  controller never infers or manufactures an approval.
- Changed-executed tasks reported by SOP are surfaced as actionable human
  decisions rather than being polled for self-resolution.

# HARD006 --- Documentation: Execution vs Observation vs Polling vs Timeout

### Acceptance Criteria

- Controller documentation explains execution vs observation vs polling vs
  timeout and states that no fixed four-minute wait is required.
- The documented developer debugging workflow prefers immediate observation
  (for example `tail -f .run/sop-run.log`) or a short bounded polling loop as an
  explicit fallback, and no longer recommends an arbitrary multi-minute wait.
- The controller's native activity/status surfaces are documented as the primary
  solution.
- Any command names and routes referenced by the documentation are accurate.

# HARD007 --- Add Deterministic Observability Tests

### Acceptance Criteria

- Tests cover: immediate running observation for a long-running fake SOP command
  (without waiting for completion); activity visibility through the existing
  surface; completion reflected without an arbitrary long timer; prompt failure
  surfacing; a SOP human-gate decision surfaced rather than a generic running
  state; no-activity not inferring failure/blockage; and (if `CommandTimeout`
  remains supported) timeout behavior verified separately from polling.
- Tests are deterministic, using bounded short intervals or channel synchronization
  and no real multi-minute sleeps.

# HARD008 --- Validation and Final Report

### Acceptance Criteria

- `gofmt -l .`, `go vet ./...`, `go build ./...`, `go test ./...`, and
  `go test -race ./...` are run and their outcomes reported truthfully.
- A final report states: (1) where the original 240-second wait came from;
  (2) whether it existed in production code, tests, docs, or only an operator
  command; (3) files changed; (4) how active commands are now observed; (5) how
  frequently fallback polling occurs; (6) how activity streaming is used; (7) how
  terminal states are detected; (8) how human-decision states are surfaced;
  (9) whether `CommandTimeout` changed and why; (10) tests added; (11) validation
  results; (12) any remaining observability limitations.

# Run Controls: Start, Continue, Retry, Cancellation

**Type:** Guide

How to start and continue a run, how retry works, and what cancellation means in
SOP Controller. Every control here is a request delegated to SOP; SOP decides
what happens. Background: [OPERATING.md](OPERATING.md).

## Start a Run

From a project page, **Run** (route
`POST /projects/{project}/commands/run`) starts a run. The controller asks SOP
to run:

```bash
sop run
```

**SOP selects which task runs next.** The controller does not choose the task
and does not schedule anything — it forwards the request and reports SOP's
result. If you are not sure which task is runnable, open the project workflow
view (`/projects/{id}`) first; the controller shows SOP's status and why a task
is not eligible.

## Continue a Run

**Resume** (route `POST /projects/{project}/commands/resume`) continues the run
SOP had in progress or paused:

```bash
sop resume
```

Resume lets SOP continue its recorded plan from where it stopped. Use it after
starting the controller, after a restart, or after a gate was cleared. Like Run,
it is SOP that resumes the workflow.

> Restarting the controller never loses workflow state. The authoritative state
> lives in SOP (`<project>/.agent-sdlc/state.db`); the controller reads it back
> on the next poll. See
> [../architecture/SOP-BOUNDARY.md](../architecture/SOP-BOUNDARY.md).

### A minimal first run

1. Open `http://127.0.0.1:8080/projects` and pick your project.
2. Click **Run** (or **Resume** to continue a paused plan).
3. **Observe immediately** — watch the project activity view and the live
   activity timeline as SOP progresses. There is no need to wait a fixed number
   of minutes first; see the debugging workflow below.
4. When SOP reaches a review, CI, or human gate, read
   [FAILURE-DISPLAY.md](FAILURE-DISPLAY.md) and
   [APPROVAL-AND-RECONCILE.md](APPROVAL-AND-RECONCILE.md).

## Watching a Run

Execution, observation, polling, and timeout are four separate things. For the
full explanation, see
[EXECUTION-AND-OBSERVATION.md](EXECUTION-AND-OBSERVATION.md). In short:

- **Observation is immediate — it is the primary solution.** Use the
  controller's native activity/status surfaces: the activity view
  (`/projects/{project}/activity`) and the live activity stream
  (`/projects/{project}/activity/stream`). Do not build a custom watcher.
- **Polling is only an explicit fallback.** If the live stream is unavailable,
  the `/projects/{project}/activity/window` endpoint provides a short, bounded,
  configurable re-read (`SOP_CONTROLLER_POLL`, default `3s`). A short bounded
  loop around it is a fallback, not the default, and never an arbitrary
  multi-minute sleep.
- **No fixed four-minute wait is required.** There is no arbitrary multi-minute
  pause in the workflow; waiting a fixed interval only delays reacting to SOP's
  real state. To watch the controller process itself, use `make logs` (tails
  `.run/sop-controller.log`) or `tail -f .run/sop-controller.log`.
- **Timeout is separate.** `SOP_CONTROLLER_COMMAND_TIMEOUT` (default `15m`)
  bounds one command's execution and returns a truthful timeout error; it does
  not bound observation and does not change SOP state.

> `.run/sop-controller.log` is the controller log written by `make start` and
> tailed by `make logs` (the `LOG` variable in the [Makefile](../../Makefile)).
> `.run/sop-run.log`, if present, is a different SOP-side run artifact — not the
> controller process log.

## Retry

Retry controls exist for tasks that stopped short of a pass. All three are
requests to SOP; **SOP decides retry eligibility** and records a new attempt.
The controller only reports the result.

| Dashboard control | Route | SOP command |
| --- | --- | --- |
| Retry | `POST /projects/{project}/tasks/{task}/commands/retry` | `sop retry <task>` |
| Force retry | `POST /projects/{project}/tasks/{task}/commands/retry-force` | `sop retry <task> --force` |
| Retry all | project command verb `retry-all` | `sop retry --all` |

- **Retry** asks SOP to retry the task under SOP's normal eligibility rules.
- **Force retry** asks SOP to retry even where the normal rules would not select
  it; SOP still owns the decision and the outcome.
- **Retry all** asks SOP to retry every task eligible under `sop retry --all`.

### Attempt semantics

A retry is an **attempt**: SOP increments and persists the task's attempt count
in its state, and the controller displays the attempt number read back from SOP
(FR-3, "retry attempts"). The controller does not track attempts itself; it
reports the count SOP recorded. Recovery dispositions
(`AUTO_FIX`/`CONTINUE`/`RETRY`/`REPLAN`/`NEEDS_HUMAN`) tell you whether SOP
intends to retry automatically or needs you — see
[../reference/STATUS-AND-RECOVERY.md](../reference/STATUS-AND-RECOVERY.md).

Each command is bounded by `SOP_CONTROLLER_COMMAND_TIMEOUT`
(default 15m); exceeding it returns a timeout error in the UI. See
[../reference/CONFIGURATION.md](../reference/CONFIGURATION.md).

## Cancellation

**SOP Controller does not offer cancellation in V1.** There is no dashboard
**Cancel** control and no `sop cancel` verb. This is a recorded, explicit
gap rather than a missing button: stopping an active run is a SOP lifecycle
decision, and SOP exposes no cancellation operation for the controller to
delegate to.

The controller therefore **does not simulate, approximate, or fake** a cancel.
The boundary entry `Client.CancelRun` returns `ErrOperationUnsupported`; see
[../history/CTRL001/BOUNDARY-CONTRACT.md](../history/CTRL001/BOUNDARY-CONTRACT.md)
for the recorded gap and
[../architecture/SOP-BOUNDARY.md](../architecture/SOP-BOUNDARY.md) for why the
controller must not implement lifecycle logic itself.

### Safe operator fallback

Because cancellation is unsupported, the safe way to stop active work is to
control the process, not the workflow:

1. Stop the controller process — `make stop`, or Ctrl-C if it runs in the
   foreground.
2. Stop / interrupt the `sop` run process itself in the terminal where it was
   started, if it is running outside the controller.
3. Restart the controller when you want to observe again: `make start` (or
   `go run ./cmd/sop-controller`), then reopen the project.

Stopping the controller never corrupts SOP state: the controller owns no
workflow state, and restarting it simply re-reads SOP's persisted state on the
next poll. When you are ready to continue, use **Resume** (`sop resume`).

> Do not expect a half-finished task to be "rolled back" by stopping the
> controller. What happens to a SOP task is always SOP's lifecycle decision;
> the controller never transitions a task.

## Related Documentation

- [EXECUTION-AND-OBSERVATION.md](EXECUTION-AND-OBSERVATION.md) — execution vs
  observation vs polling vs timeout, and the debugging workflow.
- [../reference/CLI.md](../reference/CLI.md) — commands and routes.
- [../reference/STATUS-AND-RECOVERY.md](../reference/STATUS-AND-RECOVERY.md) —
  statuses, stages, recovery dispositions and actions.
- [../reference/CONFIGURATION.md](../reference/CONFIGURATION.md) — command
  timeout and poll cadence.
- [OPERATING.md](OPERATING.md) — architecture and ownership overview.

# Execution, Observation, Polling, and Timeout

**Type:** Guide

These four concepts are distinct in SOP Controller. Confusing them is the most
common cause of "the run is stuck" reports: an SOP task can legitimately take a
long time, so *nothing being visibly different right now* is not the same as
*the run having failed*.

| Concept | What it is | Who owns it |
| --- | --- | --- |
| **Execution** | A single SOP command actually running (`sop run`, `sop resume`, `sop retry <task>`, ...). The controller delegates this to SOP and reports the result. | SOP (`sop`), driven by the controller |
| **Observation** | Read-only views over SOP's authoritative state and activity: the dashboard, the activity timeline, and the activity stream. The controller never drives the workflow by observing it. | SOP state, read by the controller |
| **Polling** | A short, bounded, configurable re-read cadence used only as a fallback when a live stream is unavailable. It changes how often the UI *asks*, never what SOP *does*. | The browser / controller UI |
| **Timeout** | The upper bound on **one SOP command's execution** (`SOP_CONTROLLER_COMMAND_TIMEOUT`, default 15m). It limits execution, not observation — you can keep watching a run after a command times out. | The controller command runner |

## Execution

Execution is the SOP command itself. The controller is an asynchronous command
runner: it starts the command, then reports SOP's result when it returns. SOP
owns all lifecycle decisions, so how long execution takes is SOP's business —
a task may progress in seconds or may legitimately take far longer than any
fixed guess.

## Observation

Observation is reading. The controller's native observation surfaces are its
**activity/status views**:

- `GET /projects/{project}/activity` — the activity timeline for a project.
- `GET /projects/{project}/activity/stream` — the **live activity stream** (SSE).
  This is the primary way to see SOP's activity as it happens.
- `GET /projects/{project}/activity/window` — the **bounded-poll window**, a
  read-only fallback used when the stream is not available.
- `GET /projects/{project}/tasks/{task}/activity` — activity for a single task.
- `GET /healthz` — controller liveness, unrelated to SOP workflow state.

Prefer the **activity stream** (or the activity view, which uses it). It is
already the controller's native surface; there is no need to build your own
watching mechanism.

## Performance

The task and project pages show SOP's own performance measurements: how long a
run's stages took and how many agent/validation/review/fix operations it cost,
read from SOP's `metrics.json` artifacts. The controller **measures nothing** —
it starts no timer and derives no duration from request latency or polling
cadence. The values are diagnostic evidence (SOP's `internal/perf` output) and
never a decision: they do not change task status, which task runs next, retries,
or approvals. A run with no performance artifact (an older run) shows an
explicit "unavailable" state rather than fabricated zeros.

## Polling

Polling is a *fallback delivery mode*, not the primary one. When the live
activity stream cannot be used, the UI falls back to re-reading SOP state on a
short, bounded cadence configured by `SOP_CONTROLLER_POLL` (default `3s`). It is
explicitly bounded and configurable: there is no invented, hard-coded interval.
Polling cadence affects only how often the controller re-reads; it has no effect
on SOP's workflow.

Polling is never a reason to wait for a fixed amount of time. If you want to
observe progress, observe it now.

## Timeout

Timeout is the command **execution** limit, `SOP_CONTROLLER_COMMAND_TIMEOUT`
(default `15m`). It bounds a single SOP command and is distinct from observation
and polling:

- Exceeding it returns a truthful **timeout error** in the UI; it does not
  corrupt SOP state, and the workflow itself is unaffected.
- It says nothing about how long you should *watch* a run. Observation has no
  timeout.

See [../reference/CONFIGURATION.md](../reference/CONFIGURATION.md) for the
configuration names and defaults.

## No Fixed Four-Minute Wait

**No fixed four-minute wait is required — or useful.** There is no arbitrary
multi-minute pause anywhere in the workflow. A human gate may appear long before
any fixed interval elapses, and a single task may legitimately run far longer
than one. Waiting a predetermined number of minutes just delays your reaction to
the real state.

Instead: observe immediately using the native activity/status surfaces, and let
SOP tell you when a gate or completion actually occurs.

## Developer Debugging Workflow

When you want to know what a run is doing right now, prefer immediate
observation; a short bounded polling loop is an explicit fallback only.

1. **Observe immediately through the controller's native activity/status
   surfaces (primary).** Open the project's activity view
   (`/projects/{project}` / `/projects/{project}/activity`), which consumes the
   live activity stream (`/projects/{project}/activity/stream`). For the
   controller process itself, watch its log directly:

   ```bash
   make logs                    # tail .run/sop-controller.log
   tail -f .run/sop-controller.log
   ```

   This is immediate observation: the log is the controller's own output file
   (`.run/sop-controller.log`, written by `make start`).
2. **If (and only if) you cannot use the live stream**, fall back to a short,
   bounded polling loop that re-reads the same status surfaces — for example:

   ```bash
   # Short, bounded fallback only; the stream above is preferred.
   for i in $(seq 1 10); do
     curl -fsS "http://127.0.0.1:8080/projects/$PROJECT/activity/window" >/dev/null
     sleep 3            # SOP_CONTROLLER_POLL default cadence
   done
   ```

   This loop is deliberately short and bounded and uses the existing
   configurable cadence (`SOP_CONTROLLER_POLL`, default `3s`). It is an explicit
   fallback, not the recommended default.
3. **Do not** wait a fixed multi-minute interval for something to happen.
   Nothing about SOP progress is scheduled to a wall-clock guess.

> **Log path note.** The controller log written by `make start` and tailed by
> `make logs` is `.run/sop-controller.log` (see the `LOG` variable in the
> [Makefile](../../Makefile)). `.run/sop-run.log` is **not** the controller's
> log: it is an SOP-side run artifact that may exist in a project's `.run`
> directory, and it is only present once such a run has produced output. When
> you mean the controller process, use `.run/sop-controller.log`.

### Related Documentation

- [RUN-CONTROLS.md](RUN-CONTROLS.md) — start, continue, retry, cancellation.
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md) — observed state → disposition →
  action.
- [../reference/CONFIGURATION.md](../reference/CONFIGURATION.md) — poll cadence
  and command timeout.
- [../reference/CLI.md](../reference/CLI.md) — routes and `make` targets.
- [../architecture/SOP-BOUNDARY.md](../architecture/SOP-BOUNDARY.md) — why the
  controller observes rather than decides.

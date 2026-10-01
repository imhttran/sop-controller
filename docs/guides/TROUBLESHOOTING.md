# Troubleshooting and CLI Equivalents

**Type:** Guide

Maps the states you observe in the dashboard to recovery dispositions and
actions, and shows the `sop` CLI and `make` targets that are supported
equivalents of the controller's controls. The authoritative status/stage/
recovery vocabulary is in
[../reference/STATUS-AND-RECOVERY.md](../reference/STATUS-AND-RECOVERY.md).

## The CLI Is a Supported Control Surface

The dashboard and the CLI are **equivalent ways to control the same SOP
workflow**. Every dashboard control delegates to a `sop` command; you may run
that command directly. Neither surface is a fallback for the other.

| Dashboard control | Route | Equivalent CLI |
| --- | --- | --- |
| Run / Start | `POST /projects/{project}/commands/run` | `sop run` |
| Resume | `POST /projects/{project}/commands/resume` | `sop resume` |
| Validate | project verb `validate` | `sop validate` |
| Review | project verb `review` | `sop review` |
| Report | `POST /projects/{project}/tasks/{task}/commands/report` | `sop report <task>` |
| Retry | `POST /projects/{project}/tasks/{task}/commands/retry` | `sop retry <task>` |
| Force retry | `POST /projects/{project}/tasks/{task}/commands/retry-force` | `sop retry <task> --force` |
| Retry all | project verb `retry-all` | `sop retry --all` |
| Reconcile | project verb `reconcile` | `sop reconcile <PLAN.md>` |

Running or stopping the **controller process** is independent of SOP:

| Task | Make target |
| --- | --- |
| Run in the foreground | `make run` |
| Build + run in the background | `make start` |
| Stop the background server | `make stop` |
| Stop, then start | `make restart` |
| Is it running? (also pings `/healthz`) | `make status` |
| Tail the background log | `make logs` |
| Build / test / fmt / vet | `make build` / `make test` / `make fmt` / `make vet` |

> The CLI is a separate program from the running controller. Running `sop ...`
> does not require the controller, and vice versa; both act on SOP's own state.
> See [../reference/CLI.md](../reference/CLI.md) and
> [../reference/CONFIGURATION.md](../reference/CONFIGURATION.md) (`SOP_BIN`).

## Observed State -> Disposition -> Action

Find the state in the dashboard, read SOP's recovery disposition, then choose
an action. The controller already shows the disposition; it never decides it.

| What you see | SOP disposition | Action |
| --- | --- | --- |
| Task running short of a pass, recoverable | `AUTO_FIX` / `CONTINUE` / `RETRY` | Do nothing; SOP continues. Watch activity. |
| Task needs re-planning | `REPLAN` | Reconcile if the plan changed; otherwise resume. |
| Task is a human boundary | `NEEDS_HUMAN` | Read the gate; approve/decline where offered (see below). |
| Task terminally blocked with a human classification | `BLOCKED` (human) | Human gate; approve/decline where SOP exposes the operation. |
| Task blocked, not a human reason | dependency/tooling, not a gate | Wait, or fix the dependency/environment; SOP will continue. |
| Run stopped and you want to go on | — | **Resume** (`sop resume`). |
| A task keeps failing after attempts | latest failure reason on the CI view | Inspect the failure, then **Retry** / **Force retry**. |

Recovery **actions** (`Retry`, `Force retry`, `Retry all`, `Reconcile`, `Open
report`, `Resume`/`Run`) all drive SOP and none mutate `state.db` directly — see
[../reference/STATUS-AND-RECOVERY.md](../reference/STATUS-AND-RECOVERY.md).

## Common Situations

- **"Run" seems to do nothing.** SOP selects the task; if no task is runnable,
  the project view explains why (dependencies, blockers). The controller makes no
  scheduling decision — see [OPERATING.md](OPERATING.md).
- **A command times out.** A single SOP command is bounded by
  `SOP_CONTROLLER_COMMAND_TIMEOUT` (default 15m). The UI reports the timeout; the
  workflow state itself is unaffected. See
  [../reference/CONFIGURATION.md](../reference/CONFIGURATION.md).
- **A gate has no buttons.** SOP reports the boundary but exposes no operation
  for that action; the controller shows it read-only. See
  [APPROVAL-AND-RECONCILE.md](APPROVAL-AND-RECONCILE.md).
- **You want to stop a running task.** Cancellation is unsupported in V1; stop
  the controller/SOP process instead. See [RUN-CONTROLS.md](RUN-CONTROLS.md).
- **Activity is empty.** The artifact is absent for that run; the controller
  reports it as absent, not as an error. See
  [ACTIVITY-AND-PRIVACY.md](ACTIVITY-AND-PRIVACY.md).
- **The dashboard shows a failure as "Needs Human".** SOP classified it as
  `NEEDS_HUMAN` (or a human `BLOCKED`). Most failures are `AUTO_FIX`/`CONTINUE`/
  `RETRY` and are not human gates. See [FAILURE-DISPLAY.md](FAILURE-DISPLAY.md).
- **After restarting the controller.** Nothing is lost: SOP owns the state, and
  the controller re-reads it on the next poll. Use **Resume** to continue.

## Related Documentation

- [../reference/STATUS-AND-RECOVERY.md](../reference/STATUS-AND-RECOVERY.md) —
  vocabulary and recovery actions.
- [../reference/CLI.md](../reference/CLI.md) — commands, routes, make targets.
- [RUN-CONTROLS.md](RUN-CONTROLS.md) — start, continue, retry, cancellation.
- [OPERATING.md](OPERATING.md) — architecture and ownership.

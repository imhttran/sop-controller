# Operating the Controller

**Type:** Guide

Operator guidance for running and controlling SOP Controller: architecture and
ownership, run controls (start, continue, retry, cancellation), activity
semantics and privacy, failure display, human approval/reconcile boundaries, and
troubleshooting. It assumes the controller is already running; see
[GETTING-STARTED.md](GETTING-STARTED.md) for install and first run.

## Ownership: Who Decides What

SOP Controller is two planes:

```text
agentic-sop / SOP = executes and decides lifecycle transitions
the controller    = observes and requests human actions
```

- **SOP owns workflow state and legal lifecycle transitions.** Task status,
run stage, retry eligibility, recovery disposition, review/CI/quality gates,
merge, and human-approval gates are all SOP's decisions. SOP persists them to
its authoritative state (`<project>/.agent-sdlc/state.db` and
`.agent-sdlc/runs/*`).
- **The controller observes.** It reads SOP's persisted state read-only and
reports SOP's own values verbatim. It never invents a status, stage, or
recovery value.
- **The controller requests human actions.** Every control in the dashboard
(Run, Resume, Retry, Report, Reconcile, and the human gates) is a request
*delegated to SOP's own command*, not an action the controller performs itself.
- **The controller implements no SOP policy.** It does not advance, cancel,
approve, schedule, or otherwise transition a SOP task. It does not maintain a
second source of truth for task state.

Read the boundary once and it explains every button in the UI:
[SOP-BOUNDARY.md](../architecture/SOP-BOUNDARY.md). The system structure is in
[architecture/OVERVIEW.md](../architecture/OVERVIEW.md).

### What an operator action really is

```text
You (browser) -> controller HTTP handler -> internal/sopclient -> sop CLI -> SOP decides
```

The browser never reaches SQLite. Each dashboard action resolves to a `sop`
verb; SOP decides what happens and the controller reports the result. See the
command table in [reference/CLI.md](../reference/CLI.md).

## Run Controls: Start, Continue, Retry, Cancellation

See [RUN-CONTROLS.md](RUN-CONTROLS.md) for the full operator walkthrough of
starting and continuing a run, retry semantics, and cancellation behavior.

- **Start** a run: dashboard **Run** (or `sop run`) lets SOP select and execute
  the runnable task. The controller does not choose the task.
- **Continue** a run: dashboard **Resume** (or `sop resume`) lets SOP continue
  the recorded plan from where it stopped.
- **Retry**: **Retry**, **Force retry**, and **Retry all** delegate to
  `sop retry`; SOP decides retry eligibility and records a new attempt.
- **Cancellation**: SOP has no cancellation operation in V1. The controller
  does not simulate one. See [RUN-CONTROLS.md](RUN-CONTROLS.md) for the safe
  operator fallback.

## Activity: Semantics and Privacy

See [ACTIVITY-AND-PRIVACY.md](ACTIVITY-AND-PRIVACY.md). In short: the activity
timeline shows SOP's own safe summaries from
`.agent-sdlc/runs/<task>/activity.jsonl`, raw agent transcripts are secondary,
and secrets, environment variables, and agent credentials are never shown.
Activity is a view of SOP's persisted events, not a replacement for
authoritative state. Normative rules live in
[specs/ACTIVITY.md](../specs/ACTIVITY.md).

## Failures: Review, CI, and Handoff

See [FAILURE-DISPLAY.md](FAILURE-DISPLAY.md). The controller shows review
findings and severity (FR-5), CI/validation checks and their failure reasons
(FR-6), retry attempt context, and handoff status and carry-forward facts (FR-7).
Compressed handoff context never replaces authoritative persisted state.

## Human Approval and Reconcile Boundaries

See [APPROVAL-AND-RECONCILE.md](APPROVAL-AND-RECONCILE.md). The controller shows
a human gate only when SOP reports one, and every approve/decline/reconcile
action is a delegated request to SOP. The controller never decides that approval
is required. Normative rules live in
[specs/HUMAN-APPROVAL.md](../specs/HUMAN-APPROVAL.md).

## Troubleshooting

See [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for mapping observed states to
recovery dispositions and actions, plus the CLI equivalents of controller
actions. The CLI remains a fully supported control surface.

## Related Documentation

- [GETTING-STARTED.md](GETTING-STARTED.md) — install, run, open the dashboard.
- [RUN-CONTROLS.md](RUN-CONTROLS.md) — start, continue, retry, cancellation.
- [ACTIVITY-AND-PRIVACY.md](ACTIVITY-AND-PRIVACY.md) — activity and privacy.
- [FAILURE-DISPLAY.md](FAILURE-DISPLAY.md) — review, CI, and handoff display.
- [APPROVAL-AND-RECONCILE.md](APPROVAL-AND-RECONCILE.md) — human gates.
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md) — recovery and CLI equivalents.
- [../architecture/SOP-BOUNDARY.md](../architecture/SOP-BOUNDARY.md) — the boundary.
- [../reference/CLI.md](../reference/CLI.md) — commands and routes.

# Status, Stage, and Recovery

**Type:** Reference

The dashboard reports SOP's decisions; it never makes them. This page lists the
values the controller reads and displays verbatim.

## Task Statuses

SOP's task status vocabulary, shown verbatim. The controller mirrors these; it
never invents a status.

```text
PLANNED   READY   BRANCH_CREATED   TESTS_WRITTEN   RED_VERIFIED
IMPLEMENTING   LOCAL_TESTS_PASS   REVIEW   REVIEW_PASS
PR_OPEN   CI_RUNNING   CI_PASS   FIX_REQUIRED   MERGED
DONE   LOCAL_DONE   BLOCKED
```

A coarse state is derived for scanning only (`DONE`, `RUNNING`, `BLOCKED`,
`READY`, `PLANNED`); it never changes SOP state.

- `DONE` / `MERGED` — merged into the shared branch.
- `LOCAL_DONE` — the local lifecycle passed without a commit/merge.

## Run Stages

The latest run's SOP lifecycle **stage**, read from
`.agent-sdlc/runs/<task>/state.json`:

```text
CREATED   PLANNING   IMPLEMENTING   VALIDATING   REVIEWING
FIXING   WAITING_FOR_HUMAN   PASSED   FAILED
```

The controller reports these verbatim; it never constructs or invents a stage.

## Recovery Dispositions

SOP's failure disposition, read from
`.agent-sdlc/runs/<task>/classification.json` (or `report.json`):

```text
AUTO_FIX   CONTINUE   RETRY   REPLAN   NEEDS_HUMAN
```

SOP's own classification names why a run stopped short of a pass and what SOP
will do about it. The controller only reads and displays it.

## When "Needs Human" Is Shown

The controller shows a prominent **Needs Human** callout only when SOP reports
`NEEDS_HUMAN`, or when a task is terminally `BLOCKED`.

A tool-budget exhaustion, an incomplete implementation, a failed test, a
compiler/lint error, a transient provider failure, or a resolvable stale-test
conflict is *not* a human boundary: SOP classifies those as
`AUTO_FIX`/`CONTINUE`/`RETRY`, and the dashboard shows that recovery instead of
asking you to act. See [../specs/HUMAN-APPROVAL.md](../specs/HUMAN-APPROVAL.md).

## Human-Decision Domains

Two distinct human decisions can be surfaced; they rely on different SOP-owned
evidence and delegate to different operations, and neither substitutes for the
other. See [../specs/HUMAN-APPROVAL.md](../specs/HUMAN-APPROVAL.md).

- **Approval** (task-level human gate): evidence is SOP-reported human
  classification / gate and stage (`NEEDS_HUMAN`, human classification kind,
  `WAITING_FOR_HUMAN`, human-`BLOCKED`); the delegated operations are
  `sop approve` / `sop decline`.
- **Reconciliation** (changed-executed-task decision): evidence is SOP's
  authoritative `sop reconcile <PLAN.md> --list-changed --json` listing; the
  decision delegates to `sop reconcile <PLAN.md> --accept-changed <TASK_ID>`.

## Recovery Actions

Every action drives a SOP command; none mutates `state.db` directly.

| Action | SOP command |
| --- | --- |
| Retry | `sop retry <task>` |
| Force retry | `sop retry <task> --force` |
| Retry all | `sop retry --all` |
| Reconcile | `sop reconcile <PLAN.md>` |
| Reconcile listing | `sop reconcile <PLAN.md> --list-changed --json` |
| Accept changed task | `sop reconcile <PLAN.md> --accept-changed <TASK_ID>` |
| Open report | `sop report <task>` |
| Approve | `sop approve <task-id> [--by NAME] [--note TEXT]` |
| Decline | `sop decline <task-id> [--by NAME] [--note TEXT]` |
| Resume / Run | `sop resume` / `sop run` |

**Reconcile** is offered only when SOP has recorded an active plan in
`.agent-sdlc/plan.meta.json`; the dashboard passes that recorded path and lets
SOP decide, preserving every unchanged task and stopping at the human boundary
when an executed task's definition changed. The retired
`.agent-sdlc/reconcile.json` artifact is never read.

**Cancel is not offered.** SOP exposes no cancellation application operation, so
cancellation remains unsupported until SOP exposes it; the dashboard
advertises no cancel control.

## Related Documentation

- [../specs/WORKFLOW.md](../specs/WORKFLOW.md) — status handling requirements.
- [../specs/EXECUTION.md](../specs/EXECUTION.md) — stage and execution reporting.
- [../specs/HUMAN-APPROVAL.md](../specs/HUMAN-APPROVAL.md) — the human gate and
  the approval-vs-reconciliation distinction.
- [CLI.md](CLI.md) — the commands above in route form.

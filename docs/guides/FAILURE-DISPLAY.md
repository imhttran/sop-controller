# Failure Display: Review, CI, and Handoff

**Type:** Guide

How failures and diagnostics appear to an operator: review findings and severity,
CI/validation checks and failure reasons, retry attempt context, and handoff
status. Every value shown is SOP's own, read from its run artifacts; the
controller reports it verbatim and invents nothing.

## Task Detail: The Failure Overview

The task detail view (`GET /projects/{project}/tasks/{task}`, FR-3) is where a
failure first shows up. It reports the task's state, run **stage** (from
`.agent-sdlc/runs/<task>/state.json`), **recovery disposition** (from
`classification.json`/`report.json`), retry **attempts**, dependencies, timing,
latest test result, and latest failure diagnostic — all SOP values.

Statuses, stages, and recovery dispositions are listed in
[../reference/STATUS-AND-RECOVERY.md](../reference/STATUS-AND-RECOVERY.md).

## Review Findings (FR-5)

The review view (`GET /projects/{project}/tasks/{task}/review`) shows:

- Ponytail / self-review status,
- Open Code Review **findings**, their **severity**, resolution status, and
  remediation progress,
- JEV status where reported.

All of it comes from SOP's run artifacts (`.agent-sdlc/runs/<task>/report.json`
and related). The controller never grades a finding or decides severity; it
reports what SOP recorded. Normative behavior is in
[specs/REVIEW.md](../specs/REVIEW.md).

## CI / Validation Checks (FR-6)

The CI view (`GET /projects/{project}/tasks/{task}/ci`) shows:

- build/test/lint **checks** and their pass / fail / running state,
- the **current retry attempt**,
- the **latest failure reason**, and
- the **current SOP remediation action**.

The controller reports check outcomes and failure reasons as SOP recorded them.
It does not run checks or reproduce CI logic.

## Retry Attempt Context

Retries appear as **attempts** on the task and CI views. SOP increments and
persists the attempt count; the controller displays the number SOP recorded. A
recovery disposition of `RETRY`, `AUTO_FIX`, or `CONTINUE` means SOP intends to
continue without you; see
[RUN-CONTROLS.md](RUN-CONTROLS.md) and
[../reference/STATUS-AND-RECOVERY.md](../reference/STATUS-AND-RECOVERY.md).

## When a Failure Is Shown as "Needs Human"

A failure is **not** automatically a human boundary. A tool-budget exhaustion,
an incomplete implementation, a failed test, a compiler/lint error, a transient
provider failure, or a resolvable stale-test conflict is classified by SOP as
`AUTO_FIX`/`CONTINUE`/`RETRY`, and the dashboard shows that recovery instead of
asking you to act. The dashboard shows the prominent **Needs Human** callout only
when SOP reports `NEEDS_HUMAN`, or when a task is terminally `BLOCKED` with a SOP
human classification. See
[APPROVAL-AND-RECONCILE.md](APPROVAL-AND-RECONCILE.md) and
[specs/HUMAN-APPROVAL.md](../specs/HUMAN-APPROVAL.md).

## Handoff Context (FR-7)

The handoff view (`GET /projects/{project}/tasks/{task}/handoff`) shows:

- handoff **generation state**,
- **compressor / provider** and **health** (including a compressor error when
  one occurred),
- **compressed size** when available, and
- **carry-forward facts and decisions**.

**Compressed handoff context never replaces authoritative persisted state.** The
handoff is a convenience summary for the next run or the next human; SOP's
persisted state (`state.db`, run artifacts) remains the source of truth.

## Related Documentation

- [../reference/STATUS-AND-RECOVERY.md](../reference/STATUS-AND-RECOVERY.md) —
  statuses, stages, recovery dispositions and actions.
- [specs/REVIEW.md](../specs/REVIEW.md) — the review lifecycle the dashboard
  displays.
- [specs/EXECUTION.md](../specs/EXECUTION.md) — stages and execution outcomes.
- [APPROVAL-AND-RECONCILE.md](APPROVAL-AND-RECONCILE.md) — human gates.
- [../reference/CLI.md](../reference/CLI.md) — review/CI/handoff routes.

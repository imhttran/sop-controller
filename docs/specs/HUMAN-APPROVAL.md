# Human Approval Specification

**Type:** Normative specification

## Purpose

Defines the human-approval boundary: when the controller shows a human gate, how
approval actions delegate to SOP, and what the controller must never do.

## Related Specifications

- [WORKFLOW.md](WORKFLOW.md)
- [EXECUTION.md](EXECUTION.md)
- [REVIEW.md](REVIEW.md)
- [../architecture/SOP-BOUNDARY.md](../architecture/SOP-BOUNDARY.md)

## Normative Language

The terms MUST, MUST NOT, SHOULD, SHOULD NOT, and MAY are normative.

## Ownership

- Human approval boundaries MUST remain controlled by SOP.
- The controller MUST NOT decide that approval is required and MUST NOT
  fabricate, simulate, or bypass an approval.
- The controller MUST NOT take any action that bypasses SOP's validation,
  review, OpenJEV, or quality gates.

## When a Human Gate Is Reported

The controller MUST present a human gate only when SOP reports one, from
SOP-persisted evidence:

- SOP's run classification disposition is `NEEDS_HUMAN`, or
- SOP's run classification kind is a human kind, or
- SOP's run stage is `WAITING_FOR_HUMAN`, or
- the task status is `BLOCKED` **and** SOP recorded a human classification.

A `BLOCKED` task whose blocking reason is not a human reason (for example a
dependency still running, or an environment/tooling problem) MUST NOT be treated
as a human gate.

## Approval Actions

- Approve and decline are two independent SOP operations. Each control MUST be
  offered only when SOP both reports the boundary and exposes that operation.
- When SOP reports a boundary but exposes no operation for an action, the
  controller MUST show the gate read-only, MUST NOT advertise a control that
  cannot succeed, and MUST return an explicit conflict if the action is invoked.
- Every approval action MUST delegate to SOP's own application operation and
  MUST report SOP's answer verbatim; the controller MUST NOT write local task
  state.

## Commit Gate

- `human.approval_before_commit` (SOP configuration) controls whether a human
  must approve before committing. When it is on, `sop run` stops at the human
  gate and never commits; the controller MUST report that state rather than
  proceeding.

## Reconciliation Approval

- When SOP reports changed executed tasks awaiting reconciliation, the
  controller MUST obtain approval only for tasks SOP reported, and MUST NOT
  approve a task SOP did not flag.
- When SOP reports no changed-executed-task set, the controller MUST report the
  absence rather than showing an empty success.

## Related Documentation

- [../reference/STATUS-AND-RECOVERY.md](../reference/STATUS-AND-RECOVERY.md) —
  when "Needs Human" is shown and the recovery actions.
- [../reference/CLI.md](../reference/CLI.md) — approve/decline and retry routes.
- [../architecture/SOP-BOUNDARY.md](../architecture/SOP-BOUNDARY.md) — the
  recorded approve/cancel operation gap.

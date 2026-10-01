# Human Approval and Reconcile

**Type:** Guide

When a human gate appears in the controller, what you are asked to decide, and
how approval and reconcile actions delegate to SOP. The controller shows the
gate and forwards your request; **SOP decides everything else**. Normative rules
are in [specs/HUMAN-APPROVAL.md](../specs/HUMAN-APPROVAL.md).

## The Two Halves of a Human Gate

```text
controller shows the gate   (observation)
        |
        v
operator requests an action (delegated to SOP)
        |
        v
SOP decides the transition   (policy)
```

The controller **decides nothing**. It never decides that approval is required,
never fabricates or bypasses an approval, and never writes local task state.
Every action it offers is a request handed to SOP's own operation.

## When a Human Gate Appears

The controller presents a human gate **only when SOP reports one**, from
SOP-persisted evidence:

- SOP's run classification disposition is `NEEDS_HUMAN`, or
- SOP's run classification kind is a human kind, or
- SOP's run stage is `WAITING_FOR_HUMAN`, or
- the task status is `BLOCKED` **and** SOP recorded a human classification.

A `BLOCKED` task whose blocking reason is *not* a human reason (a dependency
still running, or an environment/tooling problem) is **not** a human gate — the
controller does not treat it as one.

See [../reference/STATUS-AND-RECOVERY.md](../reference/STATUS-AND-RECOVERY.md)
for the "Needs Human" callout rule and the surrounding statuses.

## What You Are Asked to Decide

When a gate appears, the controller shows the SOP-reported boundary and, where
SOP exposes the operation, an approve / decline control. You are deciding
whether to let SOP proceed past the gate; the resulting transition is still
SOP's to make.

- **Approve** — asks SOP to accept the gate (route
  `POST /projects/{project}/tasks/{task}/commands/approve`).
- **Decline** — asks SOP to reject the gate (route
  `POST /projects/{project}/tasks/{task}/commands/decline`).
- **Commit gate** — when SOP's `human.approval_before_commit` is on, `sop run`
  stops at the human gate and never commits; the controller reports that state
  rather than proceeding.

Approve and decline are **two independent SOP operations**. The controller
offers a control only when SOP both reports the boundary *and* exposes the
operation. When SOP reports a boundary but exposes no operation for an action,
the controller shows the gate **read-only**, advertises no control that cannot
succeed, and returns an explicit conflict if the action is invoked.

## The Approval Gap (Read-Only Gates)

Human approval is a SOP lifecycle gate. Where SOP has no application operation
for an approval action, `Client.ApproveTask` returns `ErrOperationUnsupported`
and the controller shows the gate without an action. This is a **recorded gap**,
not a hidden failure: see
[../history/CTRL001/BOUNDARY-CONTRACT.md](../history/CTRL001/BOUNDARY-CONTRACT.md)
and [../architecture/SOP-BOUNDARY.md](../architecture/SOP-BOUNDARY.md). The
controller will expose the control once SOP grows the operation; it will not
simulate the approval in the meantime.

## Reconciliation Approval

**Reconcile** is offered when SOP has recorded an active plan in
`.agent-sdlc/plan.meta.json`. The dashboard passes that recorded path to SOP:

```text
sop reconcile <PLAN.md>
```

and lets SOP decide. SOP preserves every unchanged task and stops at the human
boundary when an executed task's definition changed.

- When SOP reports changed executed tasks awaiting reconciliation, the controller
  obtains approval **only for tasks SOP reported**; it never approves a task SOP
  did not flag.
- When SOP reports **no** changed-executed-task set, the controller reports the
  absence rather than showing an empty success. (SOP records this set in
  `.agent-sdlc/reconcile.json`.)

## Related Documentation

- [specs/HUMAN-APPROVAL.md](../specs/HUMAN-APPROVAL.md) — normative approval
  boundary.
- [../reference/STATUS-AND-RECOVERY.md](../reference/STATUS-AND-RECOVERY.md) —
  when "Needs Human" is shown and the recovery actions.
- [../architecture/SOP-BOUNDARY.md](../architecture/SOP-BOUNDARY.md) — the
  approve/cancel operation gap.
- [../reference/CLI.md](../reference/CLI.md) — approve/decline and reconcile
  routes.

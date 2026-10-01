# SOP Boundary

**Type:** Architecture documentation

## Purpose

Defines the ownership boundary between **SOP** and **SOP Controller**, and the
architectural rules the controller must never break. This is the project's
central constraint.

The operation-by-operation contract (which controller operation maps to which
SOP operation, and which are unsupported gaps) lives in
[../history/CTRL001/BOUNDARY-CONTRACT.md](../history/CTRL001/BOUNDARY-CONTRACT.md). This document
is the authority for *why* that boundary exists and what must remain true.

## The Two Planes

```text
agentic-sop / SOP = orchestration + execution authority
sop-controller    = visibility + human control
```

- **SOP** owns control flow, task state, persistence, validation, review, the
  quality gate, fix-loop budgets, retries, scheduling, and human-approval gates.
- **SOP Controller** observes SOP's persisted state and asks SOP to act through
  SOP's own commands. It is not a second orchestrator.

## Core Rule

**SOP is the workflow authority. The controller must not become a second source
of truth for SOP task state.**

```text
SOP SQLite -> SOP application services -> dashboard handlers -> HTML
```

## Invariants

These rules are architecture-level and MUST hold at all times.

1. **Single source of truth.** The controller MUST NOT maintain a second source
   of truth for SOP task state. All task state, status, blocking reason,
   dependencies, stages, and recovery values are read from SOP.
2. **No direct mutation.** The controller MUST NOT modify SOP state directly. No
   production code writes `.agent-sdlc/state.db`, the run artifacts, or SOP
   metadata.
3. **Commands go through SOP.** Every state-changing operation MUST go through
   the supported SOP application/API/CLI boundary (`internal/sopclient` →
   `sop` CLI verbs). The controller does not reproduce workflow transitions.
4. **Report, never invent.** The controller MUST report SOP's lifecycle, status,
   and recovery information verbatim. It MUST NOT invent its own lifecycle
   semantics, statuses, or stages.
5. **Approval stays with SOP.** Human-approval boundaries MUST remain controlled
   by SOP. The controller never decides that approval is required and never
   fabricates an approval; it delegates the approve/decline decision to SOP's
   own operation (`sop approve` / `sop decline`), which validates the gate at
   command time, and reports SOP's answer verbatim (including a rejection).
6. **No scheduler decisions.** The controller MUST NOT choose which task runs
   next. Task selection lives in SOP (`sop run` / `sop resume`); the boundary
   exposes no selection operation.
7. **Restart safety.** Restarting SOP Controller MUST NOT lose authoritative
   workflow state, because that state lives in SOP. Process-local command state
   is refreshed from SOP on the next poll.

## Operations

The controller's boundary is a set of read operations and command operations.
Reads are pure and surface SOP's own values; commands delegate to SOP and report
its result.

| Kind | Examples |
| --- | --- |
| Reads | plan source, tasks, task detail, activity, project progress, approvals (`sop approvals --json`), performance (`.agent-sdlc/runs/<id>/metrics.json`) |
| Commands | `sop run`, `sop resume`, `sop validate`, `sop review`, `sop report`, `sop retry`, `sop reconcile`, `sop approve`, `sop decline` |

Human approval (C2-002) is now a **supported** command operation: the controller
delegates the decision to `sop approve <task-id> [--by NAME] [--note TEXT]` /
`sop decline <task-id> [--by NAME] [--note TEXT]` through the argv-slice command
boundary. No `--run` is passed, so recording a decision is separate from
execution. SOP validates the gate at command time; a rejected (stale/
not-applicable) decision is surfaced as an actionable conflict, never a `500` or
a fabricated success. The controller writes no approval state of its own, marks
no task complete on approve, and manufactures no failure on decline.

**Performance** is a read-only, SOP-measured projection. The controller reads
the per-task `.agent-sdlc/runs/<task>/metrics.json` (with `report.json`'s
`performance` field as a fallback) and the plan-level
`.agent-sdlc/runs/<plan-id>/metrics.json` aggregate, keyed by SOP's recorded
`plan.meta.json` `plan_id`. SOP owns timing (`internal/perf`); the controller
starts no timer, derives no stage duration from its own clock, an HTTP request,
a polling interval, or a status transition, and stores no performance state.
Performance is diagnostic metadata only: it never determines task status,
selection, retry, recovery, approval, or routing.

Some conceptual operations remain **recorded gaps** because SOP exposes no
application operation for them (for example `CancelRun`; per-task
`AcceptChangedTask` until SOP exposes it). They are kept in the contract and
return `ErrOperationUnsupported` rather than being simulated. See
[../history/CTRL001/BOUNDARY-CONTRACT.md](../history/CTRL001/BOUNDARY-CONTRACT.md) for the full
table, the gap rationale, and the test matrix.

## Why the Boundary Exists

- Implementing lifecycle logic in the controller would either duplicate SOP's
  lifecycle (impossible without mutating SOP state) or require a SOP-side
  operation that does not exist.
- A second state store would drift from SOP and present the wrong picture.
- Keeping SOP authoritative means the dashboard can restart, reconnect, or be
  replaced without affecting the workflow.

## Related Documentation

- [../history/CTRL001/BOUNDARY-CONTRACT.md](../history/CTRL001/BOUNDARY-CONTRACT.md) — the
  operation-level contract and tests.
- [OVERVIEW.md](OVERVIEW.md) — system structure.
- [../specs/WORKFLOW.md](../specs/WORKFLOW.md) — how the controller reflects SOP
  workflow state.
- [../specs/HUMAN-APPROVAL.md](../specs/HUMAN-APPROVAL.md) — the approval boundary.
- [../specs/SECURITY.md](../specs/SECURITY.md) — local-first and network safety.

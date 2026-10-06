# Retired plan: plan-sop-controller

**Type:** Historical record (point-in-time, non-normative)

This record documents the retirement/supersession history of the `plan-sop-controller`
plan. It is **non-normative history**. SOP lifecycle state remains the sole authority
for execution; current behavior is defined by the specifications under `docs/specs/`.

## Authority (which state is authoritative for execution)

- **SOP lifecycle state is authoritative.** The active plan is the one recorded beside
  the machine plan in `.agent-sdlc/plan.meta.json` (plan id, source path, content hash),
  and `sop status` reports it. As of the Phase 8 pre-execution cleanup, `sop-controller`'s
  active plan is `plan-sop-controller` (`state: ACTIVE`): the controller plan was
  **re-installed as the replacement used to supersede/clear the misplaced Phase 8 state**
  in this repository.
- **This document is not lifecycle state.** It cannot activate a plan, cannot make a task
  runnable, and cannot deactivate a plan. A Markdown file's presence or absence never
  changes lifecycle state; a plan becomes active only through the deterministic
  activation path (`sop run <PLAN>` / `sop plan activate <PLAN>`), which records its
  provenance. A retired _document_ therefore cannot accidentally become runnable.

## Plan

```text
docs/PLAN-SOP-Controller.md
plan_id: plan-sop-controller
recorded source sha256: 42f48315b3d567813eefb8b635a76fc80f4b068276b92d90b4a37fadc2ce36f1
```

## Disposition (historical)

```text
RETIRED / SUPERSEDED (as a work item)
```

The plan's **work** was retired: it is stale relative to repository reality, and its
first task could not produce verified acceptance (see Reason). The retirement was
recorded here. The plan that superseded it in intent, **Phase 8, is no longer owned by
`sop-controller`**; it is owned by the `agentic-sop` repository and lives there as
`docs/plans/PHASE-8-CONTEXT-EXECUTION-EFFICIENCY.md`.

## Reason

- The plan predates later WRAP/closeout work.
- CTRL001 detects the required behavior as already satisfied but cannot produce
  mutation-based acceptance evidence: the agent repeatedly reported
  `ALREADY_SATISFIED` and produced no repository mutation, so SOP could not verify
  acceptance (`repository_mutations=0`, `termination=no_changes`).
- Repeated `CONTINUE` attempts exhausted the continuation allowance
  (`continuations.txt = 4`).
- The controller boundary was subsequently implemented and independently verified
  during later closeout/reliability work (see `docs/history/` and
  `.agent-sdlc/WRAP-012-FINAL-REPORT.md`).
- CTRL001–CTRL017 remained unexecuted under this plan.

## CTRL006

```text
Stale NEEDS_HUMAN state belonging to the retired plan
(requested_at 2026-10-01T03:52:02Z).
It was not an actionable APPROVAL_REQUIRED decision and therefore was
not approved or declined.
```

Its dependencies (`CTRL001`, `CTRL004`) were unmet, so it was not schedulable. It was
retired together with the plan's task graph; no approval was recorded for it.

## Operator decision

```text
Explicit authorization was given to retire the stale plan and move to
Phase 8 rather than force retries, fabricate mutations, or enlarge
budgets.
```

## Transition history

1. **Superseded by Phase 8** (initial transition), through SOP's reconciliation:

   ```text
   sop reconcile "docs/plans/PHASE-8-CONTEXT-EXECUTION-EFFICIENCY.md"
     removed: [CTRL001 … CTRL017]   (unexecuted)
     added:   [CTX-001 … CTX-012]
   ```

2. **Phase 8 relocated to `agentic-sop`.** The plan document was moved out of this
   repository; the dangling Phase 8 task state here was then **superseded and archived**
   by the PLAN-001 lifecycle commands:

   ```text
   sop plan supersede "docs/PLAN-SOP-Controller.md"
   ```

   which archived the misplaced Phase 8 state as `SUPERSEDED` (under
   `.agent-sdlc/archive/phase-8-context-execution-efficiency/`) and re-installed this
   controller plan as `sop-controller`'s active plan.

Neither transition marked a task PASS, recorded an approval, fabricated completion, or
hand-edited lifecycle state.

## Preservation

- CTRL001–CTRL017 run histories remain under `.agent-sdlc/runs/`.
- Plan snapshots remain under `.agent-sdlc/archive/` (including
  `phase-8-context-execution-efficiency`, archived `SUPERSEDED`).
- Git history for this repository was not rewritten.

## Historical final states

```text
CTRL001  CONTINUE (requeued)   no repository mutation   continuations 4/4
CTRL006  NEEDS_HUMAN / WAITING_FOR_HUMAN   leftover; not an approval; not acted on
```

_Retirement recorded 2026-10-06; wording reconciled with SOP lifecycle state and the
Phase 8 relocation during the Phase 8 pre-execution cleanup._

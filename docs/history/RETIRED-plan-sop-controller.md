# Retired plan: plan-sop-controller

**Type:** Historical record (point-in-time, non-normative)

This record documents why the previously active SOP plan was retired. It is
non-normative history; current behavior is defined by the specifications under
`docs/specs/`.

## Plan

```text
docs/PLAN-SOP-Controller.md
plan_id: plan-sop-controller
recorded source sha256: 42f48315b3d567813eefb8b635a76fc80f4b068276b92d90b4a37fadc2ce36f1
```

## Disposition

```text
RETIRED / SUPERSEDED
```

Superseded by Phase 8 (`plan_id: phase-8-context-execution-efficiency`). Phase 8 is
**owned by the `agentic-sop` repository** (it requires that repository's harness
runtime, trace writer/schema, evaluation, budgets, recovery/replan and routing);
the plan document lives there as
`docs/plans/PHASE-8-CONTEXT-EXECUTION-EFFICIENCY.md`. It is not implemented in
`sop-controller`.

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

## Transition

- Retired / superseded through SOP's supported reconciliation mechanism:

  ```text
  sop reconcile "docs/plans/PHASE-8-CONTEXT-EXECUTION-EFFICIENCY.md"
    removed: [CTRL001 … CTRL017]   (unexecuted)
    added:   [CTX-001 … CTX-012]
  ```

  No task was marked PASS, no approval was recorded, and no lifecycle state was
  hand-edited.

- Existing run/task artifacts are preserved: CTRL001–CTRL017 histories remain under
  `.agent-sdlc/runs/`, and the earlier plan snapshot remains under
  `.agent-sdlc/archive/plan-sop-controller/`.
- Git history for this repository was not rewritten.

## Historical final states

```text
CTRL001  CONTINUE (requeued)   no repository mutation   continuations 4/4
CTRL006  NEEDS_HUMAN / WAITING_FOR_HUMAN   leftover; not an approval; not acted on
```

_Recorded as part of the authorized plan transition, 2026-10-06._

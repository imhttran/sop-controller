# FP-007 — Run Targeted Lifecycle Tests

Evidence-only task. This artifact records the outcome of the four required lifecycle
cases — IMPLEMENT no-progress, FIX no-progress, the genuine human boundary, and
retryable-failure behavior — and, where the case cannot be exercised from this harness,
marks it **NOT RUN** with its reason rather than fabricating a result.

The two behaviors this stage must not weaken are:

```
NO_PROGRESS  →  BLOCKED  →  no approval
```

and the bounded automatic behavior for retryable failures (BLOCKED with remaining
attempt budget is retryable; budget-spent/no-run is not; neither is a human signal).

Per the plan's rules for evidence tasks, this artifact **is** the stage's legitimate
implementation output. No production code, template, test, or `.agent-sdlc` state is
mutated; the only repository write is this report.

Environment constraint (FP-007-A): this harness cannot execute lifecycle-querying or
lifecycle-mutating `sop` commands. `docs/reports/finish-pre-performance-closure/FP-001-repository-lifecycle-state.md`
§§5–6 and `docs/reports/finish-pre-performance-closure/FP-002-no-progress-approval-trace.md`
§1.1 (S12) record that `sop status` and `sop approvals` were refused as
`REQUIRES_APPROVAL`. No live `sop` command was planned or executed by this stage.

---

## 1. Scope and evidence inventory (FP-007-A)

The four required cases, the expected outcome per PLAN §12–§14/§31, the exact evidence
source each case uses, and whether the case is exercised here or NOT RUN with a reason.

| Case | Expected outcome (per plan §12–§14/§31) | Evidence source(s) | Status |
|---|---|---|---|
| IMPLEMENT no-progress | `task = BLOCKED`, `approval = none`, `AUTO_CONTINUE = false` | `.agent-sdlc/runs/FP-002/classification.json`; `internal/sopclient/classification.go` | **EXERCISED** (from preserved SOP artifact) |
| FIX no-progress | `task = BLOCKED`, `approval = none`, `AUTO_CONTINUE = false` (expected) | Shared `NO_PROGRESS` class path: `classification.go`; `classification.json` shape | **NOT RUN** (live instance) — class-level evidence only |
| Genuine human boundary | `NEEDS_HUMAN` → approval record + humanized decision request | `.agent-sdlc/archive/plan-phase2-human-decisions/` via FP-003 §2.1–2.2 | **EXERCISED** via preserved evidence; live exercise **NOT RUN** |
| Retryable failure | bounded automatic behavior unchanged; BLOCKED+budget retryable; budget-spent/no-run not | `internal/sopclient/types.go`, `store_test.go`, `commands_test.go` | **EXERCISED** |

All citations above are repository artifacts named in the FP-007 plan and/or the FP-002
source inventory (S1, S8, S9, S10, S13, S16); no citation is invented.

---

## 2. IMPLEMENT no-progress (FP-007-B) — EXERCISED

**Source:** `.agent-sdlc/runs/FP-002/classification.json` (a live, preserved
`IMPLEMENT_NO_PROGRESS` record). Fields quoted verbatim:

- `"kind": "NO_PROGRESS"`
- `"disposition": "BLOCK"`
- `"confidence": "HIGH"`
- `"autonomy": { "Action": "TERMINAL", "RequiresHuman": false, ... }`
- `autonomy.Reason`: “no repository progress under unchanged conditions; the task is
  blocked for operator intervention, not a human approval”
- top-level `reason` names the stage directly: `IMPLEMENT_NO_PROGRESS: the Ollama agent
  IMPLEMENT made no repository progress after 5 consecutive stale iterations …`.

**Observed outcome chain (no live transition asserted — only the persisted record is
read):**

```
classification kind = NO_PROGRESS
  → disposition = BLOCK
    → autonomy.RequiresHuman = false, autonomy.Action = TERMINAL
      → task state = BLOCKED
        → approval = none
          → AUTO_CONTINUE = false
```

- **task = BLOCKED** — SOP persisted `disposition: "BLOCK"`; the controller's read model
  (`internal/sopclient/types.go`, `Retryable()` / `TaskState`) groups this as `BLOCKED`.
- **approval = none** — `autonomy.RequiresHuman = false` is the observed mechanism; the
  autonomy reason states the task is blocked for *operator intervention, not a human
  approval*. No approval entry exists in the record.
- **AUTO_CONTINUE = false** — disposition `BLOCK` with `Action: TERMINAL` and the reason
  “an automatic continuation would repeat under unchanged conditions and is not taken.”

**Operator intervention vs human approval.** The `RequiresHuman: false` / `Action:
TERMINAL` autonomy contract makes the stop an operator-intervention terminal, distinct
from a human approval, which would create an approval record (see §4).

**Controller does not convert the no-progress path into a human classification.**
`internal/sopclient/classification.go` maps SOP's `Kind` to a display-only `Category` via
`CategoryOf`. `NO_PROGRESS` is not among the recognized kind constants
(`TRANSIENT_PROVIDER`, `TEST_FAILURE`, `TOOL_BUDGET_EXHAUSTED`, `HUMAN_REQUIRED`, …), so
`CategoryOf("NO_PROGRESS")` returns `CategoryUnknown` — never `CategoryHuman`. The file
documents this explicitly: an unrecognized kind “yields CategoryUnknown - never Provider,
Human, or Deterministic.” The controller never reclassifies the failure and never changes
SOP's disposition.

**No limit altered.** No budget, retry, or stale-iteration limit value was changed in
code, configuration, or state. The observed record reflects the *existing* bounded stale
allowance (5 consecutive stale iterations).

---

## 3. FIX no-progress (FP-007-C) — live instance NOT RUN; class-level evidence only

**Live FIX instance — NOT RUN.** Reason: no `.agent-sdlc/runs/<id>/` record in this
checkout contains a classification with a FIX-scoped no-progress kind. The only preserved
no-progress classification is the IMPLEMENT instance in
`.agent-sdlc/runs/FP-002/classification.json`. This harness cannot drive a live FIX run
to no-progress (lifecycle commands are refused as `REQUIRES_APPROVAL`; see §0/FP-002 S12
and FP-002 §3). This matches the plan's capability note that a live `FIX_NO_PROGRESS` run
record is **MISSING** and must be produced by the SOP pipeline, not this repository's
harness.

**Expected outcome (labelled expectation, not an observation).** Per PLAN §14, the FIX
no-progress path is expected to yield `task = BLOCKED`, `approval = none`,
`AUTO_CONTINUE = false`, identical to the observed IMPLEMENT branch because both feed the
shared `NO_PROGRESS` node (PLAN §12). This is recorded as the plan's requirement, **not**
as an observed FIX result.

**Class-level evidence (not a FIX observation).** The shared `NO_PROGRESS` class path is
inspectable:

- The classification shape is identical across paths: `{kind, disposition, confidence,
  reason, autonomy}` (as seen in `.agent-sdlc/runs/FP-002/classification.json`).
- `internal/sopclient/classification.go` maps the `Kind` string generically through
  `CategoryOf` and contains **no IMPLEMENT-vs-FIX special case**; it never consults the
  stage. So a FIX no-progress kind would flow through the same display-only path and,
  being unrecognized, map to `CategoryUnknown` (never `CategoryHuman`).

This is cited only as **class-level** evidence about the shared code path and the shared
`NO_PROGRESS` node. It is explicitly **not** a FIX observation.

**No overstated counterexample.** No preserved FIX no-progress record contains a
`NEEDS_HUMAN` disposition or an approval entry; that absence is recorded as **absence of a
counterexample**, not as a positively observed “no approval” for a FIX run (which was not
exercised here).

---

## 4. Retryable failure (FP-007-D) — EXERCISED

**Source (read-only projection):** `internal/sopclient/types.go`.
- `TaskSummary.Retryable()` = `Status == StatusBlocked && Attempt < MaxAttempts`.
- `ProjectDetail.HasRetryableBlocked()` folds `Retryable()` over the project's tasks.
- `TaskSummary.NeedsHuman` / `TaskDetail.NeedsHuman()` are set **only** from SOP's
  authoritative approval listing, never from BLOCKED status, stage, attempt counts, or
  inactivity.

**Command and result (existing sopclient tests):**

- Command: `go test ./...` — cwd: repository root — exit status: **0**.
- Output includes `ok sop-controller/internal/sopclient` (the package holding the
  retryable-failure tests), `ok sop-controller`, `ok sop-controller/internal/config`,
  `ok sop-controller/internal/web`.
- Command: `go build ./...` — exit status: **0**.
- Command: `go vet ./...` — exit status: **0**.

**Test evidence quoted (from `internal/sopclient/store_test.go`,
`TestFixCyclesAndRetryable`):** seeds three tasks and asserts bounded retry semantics:

- `a`: `BLOCKED`, `Attempt=1`, `MaxAttempts=3`, run `report.json` with `fix_cycles:2` →
  asserts `a.Retryable() == true` (“retryable … BLOCKED with budget left”).
- `b`: `BLOCKED`, `Attempt=3`, `MaxAttempts=3` → asserts `b.Retryable() == false` (“budget
  spent → not retryable”).
- `c`: `READY`, no run → asserts `c.Retryable() == false` and `FixCycles == 0` (“not
  BLOCKED → not retryable, no run”).
- Project-level: `ProjectDetail.HasRetryableBlocked()` is `true` when `a` is retryable and
  `false` when only `{b, c}` remain.

The complementary assertion in `internal/sopclient/store_test.go`
(`TestTasksProjectHumanDecisionBoundary`) confirms BLOCKED is never derived into a human
signal: a dependency-only BLOCKED task, a `WAITING_FOR_HUMAN` run stage, and a
`NEEDS_HUMAN` classification all produce `NeedsHuman == false` absent a listing entry,
while only the task with an applicable approval-listing entry is `NeedsHuman == true`.

**Semantics conclusion.** Bounded automatic behavior for retryable failures is
**unchanged**: BLOCKED with remaining attempt budget is retryable; budget-spent or no-run
is not; and none of this is mapped to a human signal. `Retryable()` /
`HasRetryableBlocked()` semantics and the bounded retry budget are unchanged — no limit was
increased, loosened, or removed.

---

## 5. Genuine human boundary (FP-007-E) — preserved evidence EXERCISED; live exercise NOT RUN

**Preserved evidence of a genuine boundary producing an approval record and a decision
request.** Cited via `docs/reports/finish-pre-performance-closure/FP-003-human-approval-boundaries.md`
§2.1–2.2, whose source is the superseded plan's archived approval/decision surface under
`.agent-sdlc/archive/plan-phase2-human-decisions/`:

- `.agent-sdlc/archive/plan-phase2-human-decisions/tasks.json` — task `C2-001` ("Adopt
  Authoritative Approval Reads") records the SOP-authoritative approval-read surface with
  decoded fields `task_id, kind, target, reason, evidence, stage, disposition, status,
  requested_at, task_status` (the approval-record shape produced when a gate exists);
  task `C2-002` ("Wire Approve and Decline to SOP") records `Client.ApproveTask` →
  `sop approve <task-id>` and `Client.DeclineTask` → `sop decline <task-id>` (the decision
  request); task `C2-009` ("Dogfood Against Real agentic-sop") records a disposable scenario
  driven to a real human approval gate and verified end to end
  (`RUNNING → Needs Attention → Inspect → Approve → SOP records → controller reflects →
  explicit Continue → SOP resumes`).
- `.agent-sdlc/archive/plan-phase2-human-decisions/plan.meta.json` (plan_id
  `plan-phase2-human-decisions`) and `plan.json` (SOP's human-decision surface:
  `sop approvals`, `sop approval <task-id>`, `sop approve`, `sop decline`, `sop reconcile`).

**Expected outcome.** A genuine `NEEDS_HUMAN` boundary produces an **approval record**
(the SOP approval listing with `disposition`/`status`/`requested_at`) and a **decision
request** (`sop approve`/`sop decline` acting on `<task-id>`), surfacing a humanized
decision request to the operator. When supported, the operator approves or declines and
SOP records the decision and resumes.

**Live exercise — NOT RUN.** Reason: no live `NEEDS_HUMAN` boundary is reachable from this
harness, and `sop approvals` / `sop status` are refused as `REQUIRES_APPROVAL` (FP-001
§§5–6; FP-002 §1.1 S12). The expected outcome above is therefore recorded as the
preserved, plan-supported behavior, not as a live observation.

**No approval approved or declined.** This stage runs no `sop approve`/`sop decline` (or
any approval-mutating) command; none was run or planned, and no live human boundary was
created.

---

## 6. Result table (FP-007-F)

| Case | Exercised? | Observed / expected outcome | Basis |
|---|---|---|---|
| IMPLEMENT no-progress | Yes (preserved artifact) | `task = BLOCKED`, `approval = none`, `AUTO_CONTINUE = false` | `.agent-sdlc/runs/FP-002/classification.json`; `classification.go` |
| FIX no-progress (live instance) | **NOT RUN** | expected `task = BLOCKED`, `approval = none`, `AUTO_CONTINUE = false` | No FIX classification artifact in checkout; harness cannot drive a live FIX run (FP-002 §3) |
| Genuine human boundary (live) | **NOT RUN** | expected `NEEDS_HUMAN` → approval record + decision request | No live boundary reachable; `sop approvals` refused (`REQUIRES_APPROVAL`) |
| Genuine human boundary (preserved evidence) | Yes (archived record) | approval record + `sop approve`/`sop decline` decision request produced | `.agent-sdlc/archive/plan-phase2-human-decisions/` via FP-003 §2.1–2.2 |
| Retryable failure | Yes (tests) | BLOCKED+budget retryable=true; budget-spent/no-run retryable=false; no human derivation | `types.go`; `store_test.go`; `commands_test.go`; `go test ./...` exit 0 |

---

## 7. Invariants verified (FP-007-F)

1. **`NO_PROGRESS → BLOCKED → no approval` is preserved.** Observed from the preserved
   IMPLEMENT record: kind `NO_PROGRESS` → disposition `BLOCK` → autonomy
   `RequiresHuman: false` / `Action: TERMINAL` with reason “blocked for operator
   intervention, not a human approval,” so no approval is created. The controller
   (`classification.go`) maps `NO_PROGRESS` to `CategoryUnknown` and never reclassifies it
   as human. No limit or budget was changed to achieve a different outcome.

2. **Bounded automatic retryable-failure behavior is unchanged.** `Retryable()` /
   `HasRetryableBlocked()` still report retryable only when `Status == BLOCKED` and
   `Attempt < MaxAttempts`; budget-spent and no-run cases are not retryable; and none of
   this is derived into `NeedsHuman`. No budget/retry/stale-iteration limit was increased
   or loosened.

---

## 8. Acceptance-criteria mapping

| Acceptance criterion | Where satisfied |
|---|---|
| The deliverable exists; each required case is exercised and its observed outcome recorded, or marked NOT RUN with a reason. | §1 inventory; §2 (IMPLEMENT: exercised); §3 (FIX live: NOT RUN with reason); §4 (retryable failure: exercised); §5 (genuine boundary: preserved evidence exercised, live NOT RUN with reason); §6 result table. |
| Bounded automatic behavior for retryable failures is shown unchanged. | §4 (tests + semantics conclusion); §7 invariant 2. |
| `NO_PROGRESS → BLOCKED → no approval` is shown preserved. | §2 (observed chain); §7 invariant 1. |

---

## 9. Mutation statement

This stage is **evidence-only**. No production code, template, or test was modified, and
no `.agent-sdlc` state (plan, task, approval, run) was edited; no SOP-state database was
written. No `sop approve` / `sop decline` / `sop status` / `sop approvals` or any other
lifecycle-querying or lifecycle-mutating command was run — the harness refuses them as
`REQUIRES_APPROVAL`, and this stage did not attempt to bypass that.

The only repository write performed by this stage is the declared deliverable
`docs/reports/finish-pre-performance-closure/FP-007-lifecycle-test-results.md`.

The pre-existing untracked `docs/plans/PLAN-Finish-Pre-Performance-Closure.md` was
preserved as user-owned; it was neither reverted nor cleaned, and is not reported as
blocking.

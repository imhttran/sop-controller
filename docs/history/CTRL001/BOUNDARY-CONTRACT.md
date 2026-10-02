# CTRL001 — Controller-to-SOP Boundary Contract

> **Current reference / non-normative.** Maintained rendering of `Boundary()`, retained at its historical CTRL001 path. Requirements, specifications, and architecture define required behavior; this reference describes the implemented boundary. Verification reports are point-in-time evidence. See the [documentation index](../../README.md).

**Scope:** the application/API contract the `sop-controller` dashboard uses to
observe and control SOP. It is defined in code by
`internal/sopclient/boundary.go` (`Boundary`, `Lookup`, `ErrOperationUnsupported`)
and exercised by `internal/sopclient/boundary_test.go` and
`internal/web/boundary_test.go`.

    agentic-sop / SOP = orchestration + execution authority
    sop-controller    = visibility + human control

> Architecture rationale and invariants: [`architecture/SOP-BOUNDARY.md`](../../architecture/SOP-BOUNDARY.md).

This document is a *rendering* of `Boundary()`, not an independent
specification: the descriptor table in code is authoritative. The test
`TestBoundaryContractMatchesPRD` keeps the operation set in sync, and any drift is
corrected from `Boundary()` rather than the other way round.

Here, **supported** means implemented at the controller boundary, not verified
against an external SOP binary. `AcceptChangedTask` delegates to
`sop reconcile <PLAN.md> --accept-changed <TASK_ID>`; `CancelRun` remains
unsupported. External support for `--list-changed` / `--accept-changed` was
**UNVERIFIED** in the recorded [C2-009 sandbox run](../C2-009-REPORT.md), whose
real-binary scenarios were **NOT EXERCISED** (readiness **NOT READY**).

## Invariants

1. **SOP remains the only lifecycle owner.** The controller never advances,
   cancels, approves, or otherwise transitions a SOP task. It reads SOP's
   persisted state and asks SOP to act through SOP's own commands.
2. **Controller operations map to SOP application operations.** Every operation
   below either reads SOP-persisted state/artifacts or invokes a `sop` CLI verb.
3. **The controller never directly mutates SOP persistence.** No production
   controller code writes `.agent-sdlc/state.db`, the run artifacts, or any SOP
   metadata. Writes happen only inside SOP, reached through a command.
4. **The controller makes no scheduler decisions.** It never chooses which task
   should run next; that ordering lives in SOP (`sop run` / `sop resume` selects
   the runnable task). The boundary exposes no selection operation.
5. **Existing CLI behavior is preserved.** The controller still drives SOP with
   the same `sop` commands (`run`, `resume`, `retry`, `reconcile`, `report`,
   `approvals`, `approve`, `decline`); this contract documents that wiring, it
   does not change it.

## Operations

The operations in `Boundary()`, in the PRD's order, with their `internal/sopclient`
entry point, the SOP application operation each uses, the `sop` CLI verbs a command
operation drives, and their status. This table mirrors `Boundary()` exactly; the
test `TestBoundaryContractMatchesPRD` keeps them in sync.

| # | Operation | sopclient entry point | SOP application operation | SOP verbs | Status |
|---|-----------|-----------------------|---------------------------|-----------|--------|
| 1 | `GetPlan` | `PlanSource` | read `.agent-sdlc/plan.meta.json` (SOP's recorded active plan source) | — | supported |
| 2 | `GetTasks` | `Project` | read `tasks` + `task_dependencies` from `state.db` | — | supported |
| 3 | `GetTask` | `Task` | read `tasks`, `task_attempts`, `handoffs` from `state.db`; optional SOP-reported checkpoint/bounded-progress line from `.agent-sdlc/runs/<task>/checkpoint.json`; the SOP-reported approval gate (`TaskDetail.Approval`) is projected from SOP's approval listing (see `GetApprovals`), not computed here | — | supported |
| 4 | `GetTaskActivity` | `Activity`, `Task` | read `.agent-sdlc/runs/<task>/activity.jsonl` | — | supported |
| 5 | `GetTaskProgress` | `Project` | read task statuses from `state.db` (`ProjectDetail.Summary`/`PercentComplete`) | — | supported |
| 6 | `GetApprovals` | `Approvals`, `Tasks`, `Task` | `sop approvals --json` (SOP's structured approval listing; read back verbatim from `.agent-sdlc/approvals.json`, present-or-absent; the controller never reconstructs a gate) | `approvals` | supported |
| 7 | `GetTaskReport` | `ReportTask` | `sop report <task>` | `report` | supported |
| 8 | `StartOrContinueRun` | `Run`, `Resume` | `sop run` / `sop resume` | `run`, `resume` | supported |
| 9 | `RetryTask` | `Retry`, `RetryForce`, `RetryAll` | `sop retry <task>` / `sop retry <task> --force` / `sop retry --all` | `retry` | supported |
| 10 | `CancelRun` | — | — | — | **unsupported (gap)** |
| 11 | `ApproveTask` | `ApproveTask` | `sop approve <task-id> [--by NAME] [--note TEXT]` (SOP validates the gate at command time; the controller passes no `--run` and writes no approval state of its own) | `approve` | supported |
| 12 | `DeclineTask` | `DeclineTask` | `sop decline <task-id> [--by NAME] [--note TEXT]` (SOP validates the gate at command time; the controller passes no `--run` and manufactures no failure) | `decline` | supported |
| 13 | `ReconcilePlan` | `Reconcile` | `sop reconcile <PLAN.md>` (SOP validates before it mutates and preserves unchanged tasks; the controller never edits `state.db`) | `reconcile` | supported |
| 14 | `GetChangedExecutedTasks` | `ChangedTasks` | `sop reconcile <PLAN.md> --list-changed --json` (SOP's authoritative changed-executed-task listing; the plan path comes from SOP's recorded `plan.meta.json` provenance; a pure read the controller decodes verbatim and never computes a diff for) | `reconcile` | supported |
| 15 | `AcceptChangedTask` | `AcceptChangedTasks`, `AcceptChangedTask` | `sop reconcile <PLAN.md> --accept-changed <TASK_ID>` (repeated `--accept-changed` once per explicitly selected task id, in a single invocation, so SOP's own atomic reconciliation semantics are preserved; the plan path comes from SOP's recorded `plan.meta.json` provenance; the controller never diffs the plan, never bulk-accepts, and writes no SOP state) | `reconcile` | supported |
| 16 | `GetTaskPerformance` | `Performance`, `Task`, `Project` | read `.agent-sdlc/runs/<task>/metrics.json` (SOP's per-task performance record; `report.json`'s `performance` field is the fallback) and `.agent-sdlc/runs/<plan-id>/metrics.json` (the plan-level aggregate, keyed by SOP's recorded `plan.meta.json` `plan_id`). Both are SOP-produced diagnostic metadata read verbatim; the controller starts no timer and computes no lifecycle duration, and performance never influences task status, selection, retry, recovery, approval, or routing | — | supported |

Every operation is `StatusSupported` except `OpCancelRun` (item 10), which is
`StatusUnsupported`.

### Read operations

Reads are pure: they never write SOP persistence, and they surface SOP's own
values verbatim (task status, run stage, recovery disposition, activity,
approval listing, performance). The controller derives no second source of truth.
Not-found cases map to the boundary sentinels `ErrProjectNotFound` and
`ErrTaskNotFound`; a failed SOP read is surfaced as an explicit error rather than
a fabricated value.

### Command operations

Commands delegate through `sopclient.Commander` to the `sop` CLI, passing
arguments as discrete argv elements (never a shell string). SOP decides retry
eligibility, reconcile outcomes, and which task runs; the controller only reports
the result. `Commander.Exec` enables SOP's observer-only activity stream
(`SOP_ACTIVITY=on`, unless the operator set it) so the dashboard can read
`activity.jsonl`; it forwards no model or provider secrets. A SOP-side rejection
is surfaced verbatim as a typed error (`*DecisionRejection` /
`*ReconcileRejection`), never as a fabricated success.

## Recorded gap: CancelRun

`CancelRun` remains part of the boundary contract, but SOP exposes no
cancellation application operation — there is no `sop cancel` verb. Per the
CTRL001 decision it is recorded as an explicit unsupported capability rather than
removed or simulated:

- `Client.CancelRun(ctx, projectID)` returns `ErrOperationUnsupported` for a
  known project, and `ErrProjectNotFound` for an unknown one.

The gap reason recorded in `Boundary()`/`Lookup()` (and reused verbatim by the
runtime error, so they cannot drift) is:

> SOP exposes no cancellation application operation (`sop cancel` does not
> exist); cancelling an active run is a SOP lifecycle decision the controller must
> not simulate

Why unsupported:

- Stopping an active run is a SOP lifecycle decision. Implementing it in the
  controller would either (a) duplicate SOP's lifecycle (forbidden, and impossible
  without mutating SOP state) or (b) require a SOP-side cancellation application
  operation that does not yet exist.

Callers test for the gap with `errors.Is(err, sopclient.ErrOperationUnsupported)`.
The controller does **not** implement cancellation logic and does **not** remove
the operation from the contract — it documents a capability SOP must grow before
the controller can expose it. Availability is derived from `Boundary()` by
`CancelOperations()`, so the day SOP records `OpCancelRun` as `StatusSupported`,
the gate flips by construction with no code change and no hardcoded literal.

## Consumers

All production controller reads and commands go through `internal/sopclient`:

- Views (`projects`, `project`, `task`, `taskActivity`, `taskRecovery`,
  `taskReview`, `taskCI`, `taskHandoff`, `projectActivity`) call the `Get*`
  reads: `Projects`, `Project`, `Task`, `Activity`, `PlanSource`, `Approvals`,
  `Performance`.
- Commands (`internal/web/handlers.go`) call `Run`, `Resume`, `Retry`,
  `RetryForce`, `RetryAll`, `Reconcile`, `ReportTask`, `ApproveTask`,
  `DeclineTask`, `AcceptChangedTask`, and the FR-5/FR-6 supporting commands
  `Validate`, `Review`, `Report`.

No production handler opens `state.db` or the run artifacts directly; direct
reads appear only in tests, which build fixtures. The cross-check test
`internal/web/boundary_test.go` asserts every dashboard command resolves to a
documented boundary operation.

Availability helpers (`ApprovalOperations`, `CancelOperations`,
`ReconcileOperations`, `ApprovalsOperations`) all derive from the single
`Boundary()` source of truth, so a consumer never offers a control whose only
possible outcome is `ErrOperationUnsupported`.

Supporting FR commands (`Validate`, `Review`, `Report`) are part of the existing
`sopclient` surface and delegate to SOP the same way; they are not among the
conceptual operations but follow the same "delegate to SOP, never decide" rule.

## Tests

| Concern | Test |
|---------|------|
| Contract completeness / PRD mapping | `internal/sopclient.TestBoundaryContractMatchesPRD` |
| No scheduler operation exposed | `internal/sopclient.TestBoundaryExposesNoSchedulerOperation` |
| Unsupported gap returns `ErrOperationUnsupported` | `internal/sopclient.TestUnsupportedOperationsReturnErrOperationUnsupported` |
| CancelRun derivation flips by construction | `internal/sopclient.TestCancelSupportedDerivation` |
| Approve/decline availability | `internal/sopclient.TestApprovalOperationsReportsSupported` |
| Approval listing availability | `internal/sopclient.TestApprovalsOperationsReportsSupported` |
| Reconcile-control availability | `internal/sopclient.TestReconcileOperationsReportsListSupportedAcceptUnsupported`, `TestReconcileOperationsDerivation` |
| Reads report SOP values + not-found sentinels | `internal/sopclient.TestReadOperationsReportSOPValues` |
| Commands delegate to SOP verbs | `internal/sopclient.TestCommandOperationsDelegateToSOP`, `TestApprovalsRefreshDelegatesToSOP` |
| No direct persistence mutation | `internal/sopclient.TestBoundaryDoesNotMutateSOPPersistence` |
| Consumer commands map to the boundary | `internal/web.TestDashboardCommandsResolveToBoundary` |
| Existing consumer behavior (views/actions/CSRF/concurrency) | `internal/web/server_test.go`, `internal/web/recovery_test.go` |

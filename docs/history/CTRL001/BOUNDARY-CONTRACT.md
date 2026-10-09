# CTRL001 — Controller-to-SOP Boundary Contract

> **Current reference / non-normative.** Maintained operation table for the controller boundary, retained at its historical CTRL001 path. Requirements, specifications, and architecture define required behavior; this reference describes the implemented boundary. Verification reports are point-in-time evidence. See the [documentation index](../../README.md).

**Scope:** the application/API contract the `sop-controller` dashboard uses to
observe and control SOP. It is implemented by the `Client` methods in
`internal/sopclient` and exercised by `internal/sopclient/boundary_test.go` and
`internal/web/boundary_test.go`.

    agentic-sop / SOP = orchestration + execution authority
    sop-controller    = visibility + human control

> Architecture rationale and invariants: [`architecture/SOP-BOUNDARY.md`](../../architecture/SOP-BOUNDARY.md).

This table is maintained by hand against the `Client` methods; when they
change, update it.

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

The conceptual operations, in the PRD's order, with their `internal/sopclient`
entry point, the SOP application operation each uses, the `sop` CLI verbs a command
operation drives, and their status.

| # | Operation | sopclient entry point | SOP application operation | SOP verbs | Status |
|---|-----------|-----------------------|---------------------------|-----------|--------|
| 1 | `GetPlan` | `PlanSource` | read `.agent-sdlc/plan.meta.json` (SOP's recorded active plan source) | — | supported |
| 2 | `GetTasks` | `Project` | read `tasks` + `task_dependencies` from `state.db` | — | supported |
| 3 | `GetTask` | `Task` | read `tasks`, `task_attempts`, `handoffs` from `state.db`; optional SOP-reported checkpoint/bounded-progress line from `.agent-sdlc/runs/<task>/checkpoint.json`; the SOP-reported approval gate (`TaskDetail.Approval`) is projected from SOP's approval listing (see `GetApprovals`), not computed here | — | supported |
| 4 | `GetTaskActivity` | `ActivityWindow`, `TaskActivityView`, `Task` | read `.agent-sdlc/runs/<task>/activity.jsonl` | — | supported |
| 5 | `GetTaskProgress` | `Project` | read task statuses from `state.db` (`ProjectDetail.Summary`/`PercentComplete`) | — | supported |
| 6 | `GetApprovals` | `Approvals`, `Tasks`, `Task` | `sop approvals --json` (SOP's structured approval listing, decoded verbatim from stdout; the controller never reconstructs a gate) | `approvals` | supported |
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

Every operation is supported except `CancelRun` (item 10).

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

`CancelRun` remains part of the conceptual contract, but SOP exposes no
cancellation application operation — there is no `sop cancel` verb. The
controller therefore has no cancel method, route, or control; `POST
.../commands/cancel` is an unknown command (400).

Why unsupported:

- Stopping an active run is a SOP lifecycle decision. Implementing it in the
  controller would either (a) duplicate SOP's lifecycle (forbidden, and impossible
  without mutating SOP state) or (b) require a SOP-side cancellation application
  operation that does not yet exist.

The controller does **not** implement cancellation logic. When SOP grows a
cancellation operation, add a `Client` method delegating to it, a route, and a
control.

## Consumers

All production controller reads and commands go through `internal/sopclient`:

- Views (`projects`, `project`, `task`, `taskActivity`, `taskRecovery`,
  `taskReview`, `taskCI`, `taskHandoff`, `projectActivity`) call the `Get*`
  reads: `Projects`, `Project`, `Task`, `ActivityWindow`, `PlanSource`, `Approvals`,
  `Performance`.
- Commands (`internal/web/handlers.go`) call `Run`, `Resume`, `Retry`,
  `RetryForce`, `RetryAll`, `Reconcile`, `ReportTask`, `ApproveTask`,
  `DeclineTask`, `AcceptChangedTask`, and the FR-5/FR-6 supporting commands
  `Validate`, `Review`, `Report`.

No production handler opens `state.db` or the run artifacts directly; direct
reads appear only in tests, which build fixtures. The cross-check test
`internal/web/boundary_test.go` asserts every dashboard command route reaches
its handler.

Supporting FR commands (`Validate`, `Review`, `Report`) are part of the existing
`sopclient` surface and delegate to SOP the same way; they are not among the
conceptual operations but follow the same "delegate to SOP, never decide" rule.

## Tests

| Concern | Test |
|---------|------|
| Actions reject an unknown project | `internal/sopclient.TestActionsRejectUnknownProject` |
| No cancel or generic approve command route | `internal/web.TestDashboardHasNoUndocumentedCommandRoute` |
| Reads report SOP values + not-found sentinels | `internal/sopclient.TestReadOperationsReportSOPValues` |
| Commands delegate to SOP verbs | `internal/sopclient.TestCommandOperationsDelegateToSOP`, `TestApprovalsDelegatesToSOP` |
| No direct persistence mutation | `internal/sopclient.TestBoundaryDoesNotMutateSOPPersistence` |
| Every dashboard command route reaches its handler | `internal/web.TestDashboardCommandsResolveToBoundary` |
| Existing consumer behavior (views/actions/CSRF/concurrency) | `internal/web/server_test.go`, `internal/web/recovery_test.go` |

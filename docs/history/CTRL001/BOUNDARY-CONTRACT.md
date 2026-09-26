# CTRL001 — Controller-to-SOP Boundary Contract

> **Historical / non-normative.** Point-in-time artifact kept for traceability only; it does not define current behavior. See the [documentation index](../../README.md).

**Scope:** the application/API contract the `sop-controller` dashboard uses to
observe and control SOP. It is defined in code by
`internal/sopclient/boundary.go` (`Boundary`, `Lookup`, `ErrOperationUnsupported`)
and exercised by `internal/sopclient/boundary_test.go` and
`internal/web/boundary_test.go`.

    agentic-sop / SOP = orchestration + execution authority
    sop-controller    = visibility + human control

> Architecture rationale and invariants: [`architecture/SOP-BOUNDARY.md`](../../architecture/SOP-BOUNDARY.md).

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
   the same `sop` commands (`run`, `resume`, `validate`, `review`, `retry`,
   `reconcile`, `report`); this contract documents that wiring, it does not
   change it.

## Conceptual operations

The eleven PRD operations, their `internal/sopclient` entry point, the SOP
application operation each uses, and their status. This table mirrors
`Boundary()`; the test `TestBoundaryContractMatchesPRD` keeps them in sync.

| # | Operation | sopclient entry point | SOP application operation | Status |
|---|-----------|-----------------------|---------------------------|--------|
| 1 | `GetPlan` | `Client.PlanSource` | read `.agent-sdlc/plan.meta.json` (SOP's recorded active plan source) | supported |
| 2 | `GetTasks` | `Client.Project` | read `tasks` + `task_dependencies` from `state.db` | supported |
| 3 | `GetTask` | `Client.Task` | read `tasks`, `task_attempts`, `handoffs` from `state.db` | supported |
| 4 | `GetTaskActivity` | `Client.Activity`, `Client.Task` | read `.agent-sdlc/runs/<task>/activity.jsonl` | supported |
| 5 | `GetTaskProgress` | `Client.Project` | read task statuses from `state.db` (`ProjectDetail.Summary`, `PercentComplete`) | supported |
| 6 | `GetTaskReport` | `Client.ReportTask` | `sop report <task>` | supported |
| 7 | `StartOrContinueRun` | `Client.Run`, `Client.Resume` | `sop run` / `sop resume` | supported |
| 8 | `RetryTask` | `Client.Retry`, `Client.RetryForce`, `Client.RetryAll` | `sop retry <task>` / `sop retry <task> --force` / `sop retry --all` | supported |
| 9 | `CancelRun` | — | — | **unsupported (gap)** |
| 10 | `ApproveTask` | — | — | **unsupported (gap)** |
| 11 | `ReconcilePlan` | `Client.Reconcile` | `sop reconcile <PLAN.md>` | supported |

### Read operations

Reads are pure: they never write SOP persistence, and they surface SOP's own
values verbatim (task status, run stage, recovery disposition, activity). The
controller derives no second source of truth. Not-found cases map to the
boundary sentinels `ErrProjectNotFound` and `ErrTaskNotFound`.

### Command operations

Commands delegate through `sopclient.Commander` to the `sop` CLI. SOP decides
retry eligibility, reconcile outcomes, and which task runs; the controller only
reports the result. `Commander.Exec` enables SOP's observer-only activity stream
(`SOP_ACTIVITY=on`, unless the operator set it) so the dashboard can read
`activity.jsonl`; it forwards no model or provider secrets.

## Recorded gaps: CancelRun and ApproveTask

`CancelRun` and `ApproveTask` remain part of the boundary contract, but SOP has
no application operation for either — there is no `sop cancel` and no
`sop approve` verb. Per the CTRL001 decision they are recorded as explicit
unsupported capabilities rather than removed or simulated:

- `Client.CancelRun(ctx, projectID)` returns `ErrOperationUnsupported`.
- `Client.ApproveTask(ctx, projectID, taskID)` returns `ErrOperationUnsupported`.

Why unsupported:

- **CancelRun** — stopping an active run is a SOP lifecycle decision. Implementing
  it in the controller would either (a) duplicate SOP's lifecycle (forbidden, and
  impossible without mutating SOP state) or (b) require a SOP-side cancellation
  application operation that does not yet exist.
- **ApproveTask** — human approval is a SOP lifecycle *gate*. The controller must
  not manufacture approval; approval requires a SOP-side operation that does not
  yet exist.

Callers test for the gap with `errors.Is(err, sopclient.ErrOperationUnsupported)`.
The controller does **not** implement cancellation or approval logic, and does
**not** remove these operations from the contract — they document a capability SOP
must grow before the controller can expose it.

## Consumers

All production controller reads and commands go through `internal/sopclient`:

- Views (`projects`, `project`, `task`, `taskActivity`, `taskRecovery`,
  `taskReview`, `taskCI`, `taskHandoff`, `projectActivity`) call the `Get*`
  reads: `Projects`, `Project`, `Task`, `Activity`, `PlanSource`.
- Commands (`internal/web/handlers.go`) call `Run`, `Resume`, `Retry`,
  `RetryForce`, `RetryAll`, `Reconcile`, `ReportTask`, and the FR-5/FR-6
  supporting commands `Validate`, `Review`, `Report`.

No production handler opens `state.db` or the run artifacts directly; direct
reads appear only in tests, which build fixtures. The cross-check test
`internal/web/boundary_test.go` asserts every dashboard command resolves to a
documented boundary operation.

Supporting FR commands (`Validate`, `Review`, `Report`) are part of the existing
`sopclient` surface and delegate to SOP the same way; they are not among the
eleven conceptual operations but follow the same "delegate to SOP, never decide"
rule.

## Tests

| Concern | Test |
|---------|------|
| Contract completeness / PRD mapping | `internal/sopclient.TestBoundaryContractMatchesPRD` |
| No scheduler operation exposed | `internal/sopclient.TestBoundaryExposesNoSchedulerOperation` |
| Unsupported gaps return `ErrOperationUnsupported` | `internal/sopclient.TestUnsupportedOperationsReturnErrOperationUnsupported` |
| Reads report SOP values + not-found sentinels | `internal/sopclient.TestReadOperationsReportSOPValues` |
| Commands delegate to SOP verbs | `internal/sopclient.TestCommandOperationsDelegateToSOP` |
| No direct persistence mutation | `internal/sopclient.TestBoundaryDoesNotMutateSOPPersistence` |
| Consumer commands map to the boundary | `internal/web.TestDashboardCommandsResolveToBoundary` |
| Existing consumer behavior (views/actions/CSRF/concurrency) | `internal/web/server_test.go`, `internal/web/recovery_test.go` |

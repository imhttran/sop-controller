# CTRL001 — SOP Boundary Inventory

> **Current reference / non-normative.** Maintained code-derived inventory, retained at its historical CTRL001 path. It describes the implemented boundary; requirements, specifications, and architecture define required behavior. Verification reports remain point-in-time evidence. See the [documentation index](../../README.md).

The existing `internal/sopclient` surface, derived from the code (not invented),
mapped to the SOP application operation each entry uses. This is the inventory
behind the boundary contract in
[`BOUNDARY-CONTRACT.md`](BOUNDARY-CONTRACT.md).

The mapping records controller-boundary support, not external-binary
verification. `AcceptChangedTask` is supported and delegates to
`sop reconcile <PLAN.md> --accept-changed <TASK_ID>`; only `CancelRun` is
unsupported. External support for `--list-changed` / `--accept-changed` was
**UNVERIFIED** in the [C2-009 sandbox run](../C2-009-REPORT.md); its real-binary
scenarios were **NOT EXERCISED** (readiness **NOT READY**).

## Read operations (over SOP-persisted state and artifacts)

| sopclient operation | SOP application operation |
|---------------------|---------------------------|
| `Client.Projects` | read `tasks` grouped by status (`state.db`) |
| `Client.Project` | read `tasks` + `task_dependencies` (`state.db`) and `.agent-sdlc/plan.meta.json` |
| `Client.Task` | read `tasks`, `task_attempts`, `handoffs` (`state.db`) and run artifacts (`state.json`, `report.json`, `classification.json`, `validation.json`, `review.json`, `activity.jsonl`, `checkpoint.json`, `metrics.json`) |
| `Client.Activity` | read `.agent-sdlc/runs/*/activity.jsonl` |
| `Client.PlanSource` | read `.agent-sdlc/plan.meta.json` |
| `Client.Approvals` | read `.agent-sdlc/approvals.json` (via `sop approvals --json`) |
| `Client.Performance` | read `.agent-sdlc/runs/<task>/metrics.json` and the plan-level aggregate |
| `Client.Root` | in-memory project-root lookup (no SOP read) |

All reads use `store.go` (open `mode=ro`) and `artifacts.go`/`run.go`
(`os.ReadFile`); none write.

## Command operations (delegating to the `sop` CLI via `Commander`)

| sopclient operation | SOP command |
|---------------------|-------------|
| `Client.Run` | `sop run` |
| `Client.Resume` | `sop resume` |
| `Client.Retry` | `sop retry <task>` |
| `Client.RetryForce` | `sop retry <task> --force` |
| `Client.RetryAll` | `sop retry --all` |
| `Client.Reconcile` | `sop reconcile <PLAN.md>` |
| `Client.ReportTask` | `sop report <task>` |
| `Client.Approvals` / `Client.RefreshApprovals` | `sop approvals --json` |
| `Client.ApproveTask` | `sop approve <task-id> [--by NAME] [--note TEXT]` |
| `Client.DeclineTask` | `sop decline <task-id> [--by NAME] [--note TEXT]` |
| `Client.AcceptChangedTask(s)` | `sop reconcile <PLAN.md> --accept-changed <TASK_ID> ...` |
| `Client.ChangedTasks` | `sop reconcile <PLAN.md> --list-changed --json` |
| `Client.Validate` | `sop validate` |
| `Client.Review` | `sop review` |
| `Client.Report` | `sop report` |

## Conceptual-operation mapping

| PRD operation | sopclient operation | Status |
|---------------|---------------------|--------|
| `GetPlan` | `PlanSource` | existing |
| `GetTasks` | `Project` | existing |
| `GetTask` | `Task` | existing |
| `GetTaskActivity` | `Activity`, `Task` | existing |
| `GetTaskProgress` | `Project` (`ProjectDetail.Summary`) | existing |
| `GetApprovals` | `Approvals`, `Tasks`, `Task` | existing |
| `GetTaskReport` | `ReportTask` | existing |
| `StartOrContinueRun` | `Run`, `Resume` | existing |
| `RetryTask` | `Retry`, `RetryForce`, `RetryAll` | existing |
| `CancelRun` | — | **missing (recorded gap)** |
| `ApproveTask` | `ApproveTask` | existing |
| `DeclineTask` | `DeclineTask` | existing |
| `ReconcilePlan` | `Reconcile` | existing |
| `GetChangedExecutedTasks` | `ChangedTasks` | existing |
| `AcceptChangedTask` | `AcceptChangedTasks`, `AcceptChangedTask` | existing |
| `GetTaskPerformance` | `Performance`, `Task`, `Project` | existing |

## Gaps

- **`CancelRun`** — no SOP application operation exists (`sop cancel` does not
  exist). Recorded as a gap: `Client.CancelRun` returns
  `ErrOperationUnsupported`. Not implemented in the controller and not removed
  from the contract. Availability (`CancelOperations()`) is derived from
  `Boundary()`, so it flips to supported automatically once SOP exposes the verb.

No new orchestration service is proposed for the gap: filling it requires a
SOP-side lifecycle operation, not controller logic.

# CTRL001 — SOP Boundary Inventory

> **Historical / non-normative.** Point-in-time artifact kept for traceability only; it does not define current behavior. See the [documentation index](../../README.md).

The existing `internal/sopclient` surface, derived from the code (not invented),
mapped to the SOP application operation each entry uses. This is the inventory
behind the boundary contract in
[`BOUNDARY-CONTRACT.md`](BOUNDARY-CONTRACT.md).

## Read operations (over SOP-persisted state and artifacts)

| sopclient operation | SOP application operation |
|---------------------|---------------------------|
| `Client.Projects` | read `tasks` grouped by status (`state.db`) |
| `Client.Project` | read `tasks` + `task_dependencies` (`state.db`) and `.agent-sdlc/plan.meta.json` |
| `Client.Task` | read `tasks`, `task_attempts`, `handoffs` (`state.db`) and run artifacts (`state.json`, `report.json`, `classification.json`, `validation.json`, `review.json`, `activity.jsonl`) |
| `Client.Activity` | read `.agent-sdlc/runs/*/activity.jsonl` |
| `Client.PlanSource` | read `.agent-sdlc/plan.meta.json` |
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
| `GetTaskReport` | `ReportTask` | existing |
| `StartOrContinueRun` | `Run`, `Resume` | existing |
| `RetryTask` | `Retry`, `RetryForce`, `RetryAll` | existing |
| `CancelRun` | — | **missing (recorded gap)** |
| `ApproveTask` | — | **missing (recorded gap)** |
| `ReconcilePlan` | `Reconcile` | existing |

## Gaps

- **`CancelRun`** — no SOP application operation exists (`sop cancel` does not
  exist). Recorded as a gap: `Client.CancelRun` returns
  `ErrOperationUnsupported`. Not implemented in the controller and not removed
  from the contract.
- **`ApproveTask`** — no SOP application operation exists (`sop approve` does not
  exist). Recorded as a gap: `Client.ApproveTask` returns
  `ErrOperationUnsupported`. Not implemented in the controller and not removed
  from the contract.

No new orchestration service is proposed for either gap: filling them requires a
SOP-side lifecycle operation, not controller logic.

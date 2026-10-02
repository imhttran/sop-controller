# PLAN --- SOP Controller Run Control and Live Activity

## Project

sop-controller

## Summary

Add a thin human-control and observability layer to `sop-controller` so
a user can start or continue an `agentic-sop` run, retry blocked work,
cancel an active run, inspect task/run state, and follow structured live
activity without moving SOP orchestration logic into the controller.

The controller is a client of the SOP application/API boundary.

`agentic-sop` remains the scheduler, lifecycle owner, failure/recovery
authority, validation/review/JEV owner, and source of truth for task
state.

The first milestone is intentionally narrow:

```text
SOP ActivityEvent
      |
      v
SOP API/application boundary
      |
      v
sop-controller
      |
      +-- Start / Continue
      +-- Retry
      +-- Stop / Cancel
      +-- Task Details
      +-- Live Activity
```

Later tasks add human approval and reconciliation controls without
duplicating SOP policy in the controller.

## Objective

Allow normal SOP operation and common recovery actions to be performed
from `sop-controller` while preserving the same lifecycle, safety
boundaries, provenance, and truthful state that currently exist in the
CLI.

A controller action must request an operation from SOP. It must not
implement the operation independently.

## Architecture

```text
                 sop-controller
              +--------------------+
              | Start / Continue   |
              | Retry              |
              | Stop / Cancel      |
              | Approve            |
              | Reconcile          |
              | Activity / Reports |
              +---------+----------+
                        |
                 SOP application/API
                        |
                        v
              +--------------------+
              |    agentic-sop     |
              |                    |
              | Scheduler          |
              | Lifecycle          |
              | Failure recovery   |
              | Validation         |
              | Review             |
              | JEV                |
              | Quality policy     |
              +--------------------+
```

Core boundary:

```text
agentic-sop / SOP = orchestration + execution authority
sop-controller    = visibility + human control
```

The controller must not parse terminal output to infer lifecycle state.
It should consume structured SOP state, activity, reports, and actions.

## Capabilities

These findings describe existing repository capabilities, not task completion
or verification against an external SOP binary. Reuse completed work and verify
remaining acceptance criteria; SOP retains lifecycle and mutation authority.

### sop_application_api_boundary — EXISTS

- Evidence: internal/sopclient/boundary.go defines Boundary()/Lookup() and delegated command operations; internal/sopclient/boundary_test.go exercises the contract and read-only persistence boundary.
- Owner: controller presents and delegates; SOP owns workflow effects.
- Gap: CancelRun remains unsupported; AcceptChangedTask is supported at the controller boundary, delegating to sop reconcile <PLAN.md> --accept-changed <TASK_ID>. External reconciliation-flag support was UNVERIFIED / NOT EXERCISED in the recorded C2-009 sandbox run.

### structured_activity_event_model — EXISTS

- Evidence: internal/sopclient/run.go defines ActivityEvent and reads SOP-produced activity.jsonl; internal/sopclient/activity.go sanitizes summaries; internal/sopclient/run_test.go covers parsing and bounded history.
- Owner: SOP produces activity; controller reads and safely presents it.

### Structured run/plan/task status model — EXISTS

- Evidence: internal/sopclient/status.go, internal/sopclient/types.go.
- Owner: sop-controller (sopclient package) over SOP state.db/artifacts.

### Start/Continue run boundary operation — EXISTS

- Evidence: internal/sopclient/boundary.go, internal/sopclient/run.go, internal/sopclient/start.go.
- Owner: sop-controller (delegates to agentic-sop) via sopclient.

### Retry task boundary operation — EXISTS

- Evidence: internal/sopclient/boundary.go, internal/sopclient/commands.go.
- Owner: sop-controller (delegates to agentic-sop) via sopclient.

### Cancel/stop active run boundary operation — MISSING

- Evidence: internal/sopclient/boundary.go.
- Owner: agentic-sop (must expose a cancellation application operation).
- Gap: SOP exposes no cancellation application operation.
- Resolution: Keep CancelRun unsupported and offer no Stop control until SOP supplies the operation; never simulate cancellation in the controller.

### Live activity delivery (SSE/polling) — PARTIAL

- Evidence: internal/web/activity_stream.go, internal/sopclient/activity_window.go.
- Owner: sop-controller (web layer over sopclient).

### Task detail and activity UI — PARTIAL

- Evidence: internal/web/handlers.go, internal/web/render.go, templates/, static/.
- Owner: sop-controller (web layer).

### Failure classification and recovery display — EXISTS

- Evidence: internal/sopclient/classification.go, internal/sopclient/types.go.
- Owner: sop-controller (renders SOP classification) via sopclient.

### Checkpoint / bounded continuation progress — EXISTS

- Evidence: internal/sopclient/checkpoint.go.
- Owner: sop-controller (renders SOP-reported progress) via sopclient.

### Human approval controls — PARTIAL

- Evidence: internal/sopclient/boundary.go, internal/sopclient/approval.go, internal/web/approval.go.
- Owner: sop-controller (offers controls) delegating to agentic-sop approval boundary.

### Plan reconciliation controls — PARTIAL

- Evidence: internal/sopclient/boundary.go, internal/sopclient/changed_task.go, internal/sopclient/accept_changed_test.go, internal/web/reconcile_test.go.
- Owner: sop-controller (presents changes, requires explicit per-task approval) delegating to agentic-sop reconcile.

### Responsive dashboard and run controls — PARTIAL

- Evidence: internal/web/dashboard_controls_test.go, internal/web/render.go.
- Owner: sop-controller (web layer).

## CTRL001 --- Define Controller-to-SOP Boundary

Define the application/API contract the controller uses to observe and
control SOP.

Conceptual operations:

```text
GetPlan
GetTasks
GetTask
GetTaskActivity
GetTaskProgress
GetTaskReport
StartOrContinueRun
RetryTask
CancelRun
ApproveTask
ReconcilePlan
```

Reuse existing SOP application boundaries where they already exist.
Do not create duplicate orchestration services merely for the
controller.

### Requires

- sop_application_api_boundary

### Acceptance Criteria

- SOP remains the only lifecycle owner.
- Controller operations map to SOP application operations.
- Controller does not directly mutate SOP persistence.
- Controller does not implement scheduler decisions.
- Existing CLI behavior remains usable.
- Boundary is documented and testable.

## CTRL002 --- Expose Structured Activity Events

Expose a structured event model for meaningful SOP activity.

Conceptually:

```go
type ActivityEvent struct {
    TaskID    string
    Stage     string
    Action    string
    Detail    string
    Timestamp time.Time
}
```

Events should represent lifecycle transitions and safe summaries of
actions such as file inspection, repository mutation, command
execution, validation, review, JEV, quality decisions, recovery, and
completion.

### Dependencies

- CTRL001

### Requires

- structured_activity_event_model
- sop_application_api_boundary

### Acceptance Criteria

- Events are machine-readable.
- Events identify task and lifecycle stage.
- Events have stable ordering/timestamps.
- Safe action summaries are available to the controller.
- Prompts, secrets, API keys, environment dumps, and unrestricted
  command/file contents are not exposed.
- CLI and controller can consume the same underlying activity model.

## CTRL003 --- Expose Run, Plan, and Task Status

Expose enough structured state for the controller to render the current
plan and execution status without reading `state.db` directly.

At minimum expose:

```text
plan/source
task ID/title
task lifecycle state
current stage
active/running task
blocked reason
fix/retry counts
provider/model when available
validation status
review status
JEV/quality status
report location/reference
final plan gate
```

### Dependencies

- CTRL001

### Requires

- Structured run/plan/task status model
- sop_application_api_boundary

### Acceptance Criteria

- Controller can render current plan status from SOP APIs.
- Running, planned, blocked, and completed states are distinguishable.
- `LOCAL_DONE` remains distinct from committed/merged states.
- Missing diagnostic data does not imply PASS.
- Controller never opens or modifies SOP state storage directly.

## CTRL004 --- Start or Continue a SOP Run

Allow the controller to request the equivalent of normal `sop run`
behavior through the SOP boundary.

The controller must not choose the next task itself.

### Dependencies

- CTRL001
- CTRL003

### Requires

- Start/Continue run boundary operation
- sop_application_api_boundary

### Acceptance Criteria

- User can start an idle plan.
- User can continue an incomplete plan.
- SOP selects the runnable task.
- Existing dependency ordering remains authoritative.
- Existing verify-first/recovery behavior remains authoritative.
- Completed plans report completion rather than starting phantom work.
- Duplicate clicks/requests do not create concurrent duplicate runs.

## CTRL005 --- Retry a Blocked Task

Allow a user to request SOP's existing retry behavior for a blocked
task.

The controller displays SOP's reason and recovery classification but
does not decide whether a failure is code, provider, plan, or human
failure.

### Dependencies

- CTRL001
- CTRL003

### Requires

- Retry task boundary operation
- sop_application_api_boundary

### Acceptance Criteria

- Retry invokes SOP's retry operation.
- Existing retry budgets are enforced by SOP.
- `--force`-equivalent behavior requires an explicit separate human
  action if supported.
- Retry preserves prior run history and artifacts.
- Retry does not reset task state by direct persistence mutation.
- Failure reasons remain visible after retry.

## CTRL006 --- Stop or Cancel an Active Run

Add a bounded cancellation path for an active controller-started run.

Cancellation must stop future agent/lifecycle work safely without
pretending the current task passed or completed.

### Dependencies

- CTRL001
- CTRL004

### Requires

- Cancel/stop active run boundary operation
- sop_application_api_boundary

### Acceptance Criteria

- Active run can receive a cancellation request.
- Cancellation is propagated through SOP's execution boundary.
- No PASS/LOCAL_DONE is manufactured by cancellation.
- Completed validation/review evidence remains preserved.
- Working-tree changes are preserved.
- Cancellation does not use destructive Git or state deletion.
- UI clearly distinguishes cancelled/stopped from failed/completed.

## CTRL007 --- Add Live Activity Delivery

Deliver structured activity to the controller using the simplest
local-first mechanism supported by the current architecture.

Prefer existing subscription support, SSE, or bounded polling. Do not
introduce Kafka, Redis, or distributed infrastructure solely for this
feature.

### Dependencies

- CTRL002
- CTRL003

### Requires

- Live activity delivery (SSE/polling)
- structured_activity_event_model
- sop_application_api_boundary

### Acceptance Criteria

- Activity updates appear while a run is active.
- Reconnection/poll refresh does not duplicate lifecycle actions.
- Event ordering is stable enough for a readable timeline.
- Controller can recover the recent timeline after refresh.
- Transport failure does not alter SOP execution state.
- Tests do not depend on timing-sensitive sleeps.

## CTRL008 --- Add Task Detail and Activity UI

Create a task detail view suitable for desktop and small devices.

Show, when available:

```text
Task ID and title
State
Current stage
Started / elapsed
Provider / model
Fix cycles
Retry attempts
Recent activity
Validation
Review
JEV
Quality decision
Blocked reason
Failure classification
Recovery disposition
Report
```

### Dependencies

- CTRL003
- CTRL007
- CTRL009
- CTRL010

### Requires

- Task detail and activity UI
- Structured run/plan/task status model
- Live activity delivery (SSE/polling)

### Acceptance Criteria

- Current task is obvious.
- Recent activity is readable without terminal access.
- Validation, review, JEV, and quality are visually distinct concepts.
- Missing data is shown as unavailable/not run, not PASS.
- UI remains usable on a small screen.
- No raw prompt/secrets are displayed.

## CTRL009 --- Display Failure Classification and Recovery

Render failure information produced by SOP.

Supported disposition vocabulary should follow SOP's authoritative
contract, conceptually:

```text
AUTO_FIX
CONTINUE
RETRY
REPLAN
NEEDS_HUMAN
```

The controller must not independently classify failures.

### Dependencies

- CTRL003

### Requires

- Failure classification and recovery display
- Structured run/plan/task status model

### Acceptance Criteria

- SOP classification/disposition is displayed when available.
- Reason/evidence is available to the user.
- Provider failures are distinguishable from code/validation failures.
- Tool/iteration-budget exhaustion is not automatically labeled human.
- Deterministic failures are not automatically labeled provider.
- Unknown classification remains unknown rather than guessed.

## CTRL010 --- Display Continuation and Checkpoint Progress

Expose useful bounded-progress information for work that spans multiple
agent invocations.

Examples include test coverage analysis, remaining acceptance criteria,
or other checkpoints explicitly reported by SOP.

Example:

```text
JEV012
11 / 15 cases analyzed
covered: 11
missing: 4
next: add deterministic coverage
```

### Dependencies

- CTRL003

### Requires

- Checkpoint / bounded continuation progress
- Structured run/plan/task status model

### Acceptance Criteria

- Progress comes from SOP/run artifacts or structured activity.
- Controller does not invent completion percentages.
- Refresh preserves known progress.
- Partial progress is not treated as task completion.
- Continuation remains an SOP lifecycle decision.

## CTRL011 --- Add Human Approval Controls

Expose approval actions only where SOP reports an actual approval
boundary.

Examples may include commit approval or other explicit human gates
already supported by SOP.

### Dependencies

- CTRL003

### Requires

- Human approval controls
- sop_application_api_boundary

### Acceptance Criteria

- Approval is available only when SOP requests it.
- Approval invokes SOP's application boundary.
- Controller cannot bypass validation/review/JEV/quality gates.
- Approval is recorded with existing SOP provenance.
- Declining/withholding approval preserves truthful task state.

## CTRL012 --- Add Plan Reconciliation Controls

Expose safe reconciliation through SOP's existing reconcile operation.

The controller should present changed executed tasks and require
explicit per-task approval equivalent to `--accept-changed`.

### Dependencies

- CTRL003
- CTRL011

### Requires

- Plan reconciliation controls
- sop_application_api_boundary

### Acceptance Criteria

- Reconcile never edits `state.db` directly.
- All changed executed tasks are reported before mutation.
- Each changed executed task requires explicit approval.
- Validation occurs before reconciliation mutation.
- Reconciliation is atomic/failure-safe.
- Existing run history and lifecycle state are preserved.
- Unknown/unrelated task approvals are rejected.

## CTRL013 --- Add Responsive Dashboard and Run Controls

Integrate controls into the existing controller dashboard.

Desktop may use a table; small devices should use compact task cards.

Suggested desktop columns:

```text
TASK | STATE | STAGE | RECOVERY | FIX
```

Primary controls should be contextual rather than always enabled.

### Dependencies

- CTRL004
- CTRL005
- CTRL006
- CTRL011
- CTRL012

### Requires

- Responsive dashboard and run controls
- sop_application_api_boundary

### Acceptance Criteria

- Start/Continue is visible when appropriate.
- Retry appears for retryable blocked work.
- Stop appears only for an active run.
- Approval appears only for real approval boundaries.
- Reconcile appears only when plan reconciliation is required.
- Controls have clear disabled/loading states.
- Layout works on narrow/mobile screens.
- Repeated clicks cannot launch duplicate actions.

## CTRL014 --- Add Deterministic Unit and Integration Tests

Required cases include:

1. Controller reads plan/task state through SOP boundary.
2. Start/Continue delegates scheduling to SOP.
3. Duplicate Start does not create duplicate runs.
4. Retry delegates to SOP and preserves retry budget.
5. Provider failure is displayed without being reclassified by UI.
6. Cancel stops execution without manufacturing PASS.
7. Activity ordering is deterministic.
8. Refresh/reconnect preserves recent activity.
9. Validation/review/JEV/quality statuses remain distinct.
10. Missing diagnostic data never implies PASS.
11. Human approval remains explicit.
12. Reconcile requires all changed executed-task approvals.
13. Controller never mutates SOP persistence directly.
14. Sensitive prompts/secrets are not exposed in activity.
15. Small-screen task detail/control rendering remains usable.

### Dependencies

- CTRL002
- CTRL003
- CTRL004
- CTRL005
- CTRL006
- CTRL007
- CTRL008
- CTRL009
- CTRL010
- CTRL011
- CTRL012
- CTRL013

### Requires

- sop_application_api_boundary
- structured_activity_event_model
- Structured run/plan/task status model
- Live activity delivery (SSE/polling)
- Responsive dashboard and run controls
- Task detail and activity UI

### Acceptance Criteria

- Deterministic.
- No live Ollama required.
- No network dependency outside in-process/local test fixtures.
- No timing-sensitive sleeps when a deterministic synchronization
  primitive can be used.
- Existing controller and SOP tests continue passing.

## CTRL015 --- Add End-to-End Dogfood Scenario

Use the completed JEV workflow as the primary controller dogfood
scenario.

Prove the normal flow:

```text
controller Start/Continue
 -> SOP selects task
 -> activity streams
 -> validation/review/JEV/quality visible
 -> task LOCAL_DONE
 -> SOP continues
 -> plan COMPLETE
 -> final gate PASS
```

Also prove the recovery flow based on the observed JEV013 provider
failure:

```text
JEV013 running
 -> Ollama/provider HTTP 500
 -> SOP records provider failure
 -> task BLOCKED/retryable according to SOP policy
 -> controller displays reason
 -> human requests Retry
 -> SOP retries
 -> JEV013 PASS
 -> SOP continues remaining tasks
 -> plan COMPLETE
```

Use deterministic fake/provider fixtures for automated tests. The real
historical HTTP 500 is an acceptance scenario, not a required live
failure.

### Dependencies

- CTRL013
- CTRL014

### Requires

- sop_application_api_boundary
- Live activity delivery (SSE/polling)
- Responsive dashboard and run controls
- Failure classification and recovery display

### Acceptance Criteria

- Normal and recovery flows demonstrated.
- Controller never directly changes lifecycle state.
- Provider failure does not consume a FIX cycle unless SOP explicitly
  determines code repair is required.
- Retry preserves earlier failure evidence.
- No external model required by automated tests.
- Final state remains truthful.

## CTRL016 --- Documentation and Operator Guidance

Document architecture, run controls, activity semantics, failure
display, approval/reconcile boundaries, cancellation, and
troubleshooting.

Explicitly document:

```text
SOP executes and decides lifecycle transitions.
Controller observes and requests human actions.
Controller does not implement SOP policy.
```

### Dependencies

- CTRL013
- CTRL015

### Requires

- Responsive dashboard and run controls
- sop_application_api_boundary

### Acceptance Criteria

- New user can start/continue a run from the controller.
- Retry and cancellation behavior are documented.
- Human approval/reconcile boundaries are clear.
- Activity privacy/safety rules are clear.
- CLI remains a supported equivalent control surface.

## CTRL017 --- Final Regression and Compatibility Gate

Verify controller integration does not change SOP semantics.

### Dependencies

- CTRL014
- CTRL015
- CTRL016

### Requires

- sop_application_api_boundary
- Structured run/plan/task status model
- Responsive dashboard and run controls

### Validation

Use actual repository/package ownership discovered during
implementation. At minimum run the relevant focused tests followed by
the repository-wide checks available to each project.

For Go components, conceptually:

```bash
go test ./...
go vet ./...
go test -race ./...
go build ./...
```

If `sop-controller` uses a different frontend build/test toolchain,
run its existing lint, type-check, unit/integration, and production
build commands as well.

### Acceptance Criteria

- Full relevant tests pass.
- Race detector passes for applicable Go packages.
- Vet/static checks pass.
- Controller production build passes.
- Existing CLI run/retry/reconcile behavior remains compatible.
- JEV lifecycle remains compatible.
- Automatic blocked recovery remains compatible.
- Completed-plan handoff remains compatible.
- Human approval remains intact.
- Dirty working trees are preserved.
- No destructive recovery is introduced.
- No safety gate is bypassed.

## Safety Invariants

Never require or perform:

```bash
rm .agent-sdlc/state.db
git reset --hard
git clean
```

The controller must never directly modify SOP's database, machine plan,
task graph, retry counters, lifecycle state, validation evidence,
review evidence, JEV evidence, or quality decision.

Preserve:

- working-tree changes
- task/run history
- plan provenance
- validation/review/JEV evidence
- failure/recovery evidence
- human approval boundaries

Do not expose:

- hidden prompts
- credentials or API keys
- environment dumps containing secrets
- unrestricted command arguments when sensitive
- arbitrary file contents merely because a file was inspected

The controller must not manufacture validation PASS, review PASS, JEV
PASS, quality PASS, CI PASS, task completion, commit, PR creation, or
merge completion.

## Failure and Recovery Boundary

SOP owns failure classification and recovery decisions.

Conceptual SOP dispositions:

```text
failure
  |
  +-- AUTO_FIX    -> SOP FIX lifecycle
  +-- CONTINUE    -> SOP bounded continuation
  +-- RETRY       -> SOP/provider/task retry
  +-- REPLAN      -> SOP plan/reconcile path
  +-- NEEDS_HUMAN -> controller exposes human action
```

The controller renders this state and invokes permitted actions. It does
not independently transform one disposition into another.

## Local-First Strategy

V1 should remain local-first and simple.

Prefer:

```text
sop-controller
      |
local SOP API/application boundary
      |
agentic-sop
```

Do not introduce distributed infrastructure unless an existing
repository requirement already demands it.

## Out of Scope for V1

- duplicating SOP scheduler logic in the controller
- controller-owned FIX loops
- controller-owned failure classification
- controller-owned JEV analysis
- direct state.db mutation
- automatic commits, pushes, PRs, or merges
- unrestricted remote execution
- multi-user remote orchestration
- cloud control plane
- Slack/SMS/community controls
- Kafka/Redis solely for activity delivery
- parsing CLI text as the primary integration contract

## Definition of Done

SOP Controller run control is complete when a user can observe the
current SOP plan, task, stage, progress, diagnostics, and safe live
activity; start or continue execution; retry blocked work; safely cancel
an active run; perform explicit approval/reconciliation actions through
SOP; and complete the JEV013-style recovery scenario without using the
terminal for normal operation.

The controller must remain a thin control and observability surface over
SOP rather than a second orchestration engine.

Preserve the core rule:

```text
Provider != Agent Harness != Model

agentic-sop / SOP = orchestration and execution authority
sop-controller    = visibility and human control
JEV               = engineering analysis and quality signal
Model             = replaceable reasoning engine
```

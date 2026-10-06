# SOP Controller --- `sop run` Hardening Plan

> **Document class:** plan · **Lifecycle:** complete · **Authority:** historical — a record of completed work, not current planning authority.

**Status:** Ready for execution\
**Repository:** `imhttran/sop-controller`\
**Execution command:** `sop run`

## Goal

Harden the existing SOP Controller so `sop run` is the normal end-to-end
execution path and the dashboard remains a thin, reliable human control
plane over SOP.

## Execution Contract

Place this file at the repository root as `PLAN.md`, then run:

``` bash
sop run
```

SOP should execute the plan end-to-end with minimum manual intervention.

### Rules

1.  Preserve the current Go + `html/template` + HTMX architecture.
2.  Do not introduce React, Next.js, or a Node runtime.
3.  SOP remains the workflow authority.
4.  The controller must not reproduce SOP state transitions.
5.  Existing SOP state is authoritative after controller restart.
6.  `sop run` is the primary lifecycle command.
7.  `validate`, `review`, `retry`, `resume`, and `report` remain
    secondary operational commands.
8.  Preserve unrelated existing user changes.
9.  Run deterministic validation before considering work complete.
10. Do not push, merge, or deploy automatically unless repository policy
    explicitly permits it.
11. If work cannot be completed safely, mark it blocked with a concrete
    reason rather than guessing.

## Target Architecture

``` text
Browser
   |
   v
Go + html/template + HTMX
   |
   v
sopclient.Service
   |
   +------------------+
   |                  |
   v                  v
SOP state reads     SOP CLI
                       |
                       v
                    sop run
                       |
       implementation -> validation -> review
                              |
                              v
                       remediation/retry
                              |
                              v
                            report
                              |
                              v
                           complete
```

The dashboard observes and safely commands SOP. It does not become
another orchestrator.

## Definition of Success

After this plan is complete, `sop run` is sufficient to execute the
project end-to-end. The dashboard can answer:

-   What is running?
-   What completed?
-   What is blocked?
-   What failed?
-   Why did it fail?
-   What is SOP doing now?
-   What needs human attention?

Restarting SOP Controller must not lose authoritative workflow
visibility.

------------------------------------------------------------------------

# SC-001 --- Establish Baseline

## Objective

Capture repository state before changing implementation.

## Requirements

Record current branch, commit SHA, working-tree status, Go version,
project structure, and SOP state availability.

Run:

``` bash
git status --short
go build ./...
go vet ./...
go test ./...
```

Do not modify files before the baseline is recorded. Preserve and
distinguish pre-existing changes.

## Acceptance Criteria

-   baseline commit recorded
-   pre-existing changes recorded
-   build, vet, and test results recorded
-   no pre-existing work silently overwritten

------------------------------------------------------------------------

# SC-002 --- Complete SOP Service Boundary

**Depends on:** SC-001

## Objective

Make `sopclient.Service` represent the complete dashboard-facing SOP
contract.

It should cover project/task/activity reads plus Run, Resume, Retry,
Validate, Review, and Report where currently supported.

Example:

``` go
type Service interface {
    Projects(ctx context.Context) ([]ProjectSummary, error)
    Project(ctx context.Context, id string) (ProjectDetail, error)
    Task(ctx context.Context, projectID, taskID string) (TaskDetail, error)
    Activity(ctx context.Context, projectID string, limit int) ([]Event, error)

    Run(ctx context.Context, projectID string) error
    Resume(ctx context.Context, projectID string) error
    Retry(ctx context.Context, projectID, taskID string) error

    Validate(ctx context.Context, projectID string) (string, error)
    Review(ctx context.Context, projectID string) (string, error)
    Report(ctx context.Context, projectID string) (string, error)
}
```

Adapt signatures to the existing domain model if needed.

## Acceptance Criteria

-   web layer depends on the service abstraction where practical
-   concrete client satisfies the interface
-   SQLite details do not escape through the interface
-   workflow transitions remain owned by SOP

## Validation

``` bash
go test ./internal/sopclient/...
go test ./internal/web/...
go vet ./...
go build ./...
```

------------------------------------------------------------------------

# SC-003 --- Make `sop run` the Primary Lifecycle

**Depends on:** SC-002

## Objective

Make the normal dashboard Run action invoke one SOP-owned end-to-end
lifecycle.

``` text
Run button
   |
   v
sop run
   |
   v
SOP owns implementation, validation, review,
remediation, reporting, and completion
```

Keep Resume, Retry, Validate, Review, and Report as delegated secondary
operations.

## Acceptance Criteria

-   Run delegates to `sop run`
-   controller does not manually chain lifecycle stages
-   Validate and Review are not required manual steps after normal Run
-   Resume and Retry remain available where SOP permits them

## Validation

``` bash
go test ./internal/sopclient/...
go test ./internal/web/...
go test ./...
```

------------------------------------------------------------------------

# SC-004 --- Project Command Concurrency Protection

**Depends on:** SC-003

## Objective

Prevent duplicate/conflicting lifecycle commands from the dashboard.

Protect at minimum:

``` text
Run + Run
Run + Resume
Run + Retry
Resume + Resume
```

Use project-scoped atomic check/start behavior. Release protection after
success, failure, or timeout. Read-only diagnostics remain available.

## Acceptance Criteria

-   double-click cannot launch two runs
-   multiple browser sessions cannot accidentally duplicate runs
-   failures/timeouts do not leave permanent locks
-   useful conflict feedback appears in UI
-   concurrency behavior has tests

## Validation

``` bash
go test -race ./internal/web/...
go test ./...
go vet ./...
```

------------------------------------------------------------------------

# SC-005 --- Unify Command Timeout Policy

**Depends on:** SC-004

## Objective

Use one configured timeout for SOP command execution.

``` text
SOP_CONTROLLER_COMMAND_TIMEOUT
              |
              v
       command execution
```

Remove independent hard-coded lifecycle timeouts from the web runner.
Long autonomous runs must support values such as:

``` bash
SOP_CONTROLLER_COMMAND_TIMEOUT=2h
```

## Acceptance Criteria

-   one configuration source controls command timeout
-   no independent hard-coded 30-minute command timeout remains
-   timeout errors are visible
-   timeout behavior has tests

## Validation

``` bash
grep -R "30.*time.Minute" internal || true
go test ./...
go vet ./...
```

------------------------------------------------------------------------

# SC-006 --- Make SOP State Authoritative After Restart

**Depends on:** SC-002

## Objective

Treat in-memory command state only as transient UI convenience.

``` text
in-memory CommandState = transient
SOP persisted state    = authoritative
```

After startup/restart, reconstruct visible workflow state from SOP.

## Acceptance Criteria

-   dashboard status derives from persisted SOP state
-   missing process-local command history does not imply an idle project
-   stale command state cannot override SOP state
-   restart scenario is covered by tests or controlled validation

## Validation

``` bash
go test ./internal/sopclient/...
go test ./internal/web/...
go test ./...
```

------------------------------------------------------------------------

# SC-007 --- Improve End-to-End Run UX

**Depends on:** SC-003, SC-006

## Objective

Make project pages emphasize lifecycle status instead of independent
commands.

Prioritize:

``` text
Project
Overall status
Progress
Current task
Current stage
Provider when available
Elapsed time
Blocker/failure
Human action required
```

Primary action: Run.

Contextual actions: Resume, Retry.

Secondary/advanced actions: Validate, Review, Report.

Do not add source editing.

## Acceptance Criteria

-   active project state is immediately visible
-   Run is clearly the normal action
-   diagnostic commands do not dominate
-   blocked/failing work is obvious
-   phone and desktop layouts remain usable

## Validation

``` bash
go test ./internal/web/...
go test ./...
```

------------------------------------------------------------------------

# SC-008 --- Improve Failure and Blocker Visibility

**Depends on:** SC-007

## Objective

Make the dashboard explain why SOP stopped and what happens next.

Display where available:

``` text
task
stage
provider
attempt
validation gate
failure reason
review findings
last meaningful action
next eligible action
```

Do not invent remediation guidance that SOP did not provide.

## Acceptance Criteria

-   validation failure differs from review failure
-   blocked state includes available reason
-   Retry/Resume appears only when appropriate
-   structured diagnostics are primary; raw logs are secondary

## Validation

``` bash
go test ./internal/web/...
go test ./internal/sopclient/...
go test ./...
```

------------------------------------------------------------------------

# SC-009 --- Preserve Future API/MCP Compatibility

**Depends on:** SC-002

## Objective

Keep the local implementation while avoiding unnecessary persistence
coupling.

``` text
             sopclient.Service
                    |
          +---------+---------+
          |                   |
          v                   v
   Local Provider       Future Provider
   SQLite + CLI          API / MCP
```

Do not implement the remote provider now.

## Acceptance Criteria

-   web handlers do not depend on SQLite schema
-   persistence and command execution remain behind `sopclient`
-   service interfaces remain transport-neutral where practical
-   a future provider could implement the contract without rewriting
    handlers

## Validation

``` bash
go test ./...
go vet ./...
go build ./...
```

------------------------------------------------------------------------

# SC-010 --- Add Hardening Tests

**Depends on:** SC-004, SC-005, SC-006, SC-008, SC-009

## Objective

Add regression coverage for the new guarantees.

Required coverage:

``` text
SOP boundary:
known/unknown project
known/unknown task
activity
run/resume/retry/validate/review/report

Concurrency:
duplicate run
run + resume
run + retry
release after success/failure/timeout

Recovery:
transient state not authoritative
persisted SOP state drives UI after restart

Security:
state-changing endpoints use POST
CSRF remains enabled
network mode requires token

UI:
active run
blocked task
failed validation
review findings
human-attention state
```

## Validation

``` bash
go test -race ./internal/web/...
go test ./...
go vet ./...
go build ./...
```

------------------------------------------------------------------------

# SC-011 --- Synchronize Documentation

**Depends on:** SC-003, SC-008, SC-009

## Objective

Update `README.md`, `docs/requirements/PRD.md`, and `docs/history/PLAN.md` to match actual
behavior.

Document prominently:

> `sop run` is the primary end-to-end execution path. SOP owns
> orchestration, validation, review, remediation, and completion. SOP
> Controller observes that lifecycle and exposes safe controls without
> reproducing workflow logic.

Also document Run/Resume/Retry semantics, diagnostic commands, timeout
configuration, restart behavior, state ownership, local-first operation,
and network security.

## Acceptance Criteria

-   docs match implementation
-   docs do not imply controller-owned orchestration
-   startup/run instructions are current

------------------------------------------------------------------------

# SC-012 --- Final Deterministic Gate

**Depends on:** SC-010, SC-011

## Objective

Run the complete repository validation.

``` bash
gofmt -w .
go vet ./...
go test -race ./...
go test ./...
go build ./...
git status --short
```

If formatting changes files, rerun tests.

## Acceptance Criteria

-   formatting clean
-   vet passes
-   race tests pass
-   tests pass
-   build passes
-   final changed files understood
-   unrelated user changes preserved

------------------------------------------------------------------------

# SC-013 --- Dogfood SOP Controller

**Depends on:** SC-012

## Objective

Validate the hardened controller against a real SOP-managed project,
preferably `sop-controller` itself.

Start with:

``` bash
sop run
```

Verify:

``` text
run starts
current task/stage visible
validation visible
review visible
failure/blocker visible if encountered
browser may close/reopen
controller may restart
authoritative state remains correct
duplicate Run is protected
SOP resumes/remediates as allowed
final report is accessible
run completes
phone-sized layout remains usable
```

## Acceptance Criteria

The dashboard can answer without routine CLI inspection:

``` text
What is running?
What completed?
What failed?
Why?
What is SOP doing next?
Does SOP need me?
```

------------------------------------------------------------------------

# SC-014 --- Final Run Report

**Depends on:** SC-013

## Objective

Produce a final report containing:

``` text
baseline commit
tasks completed
files changed
validation commands/results
review results
dogfood results
remaining warnings
blocked work
follow-up recommendations
```

Return one final gate:

``` text
PASS
NEEDS_HUMAN
FAIL
```

Use PASS only when deterministic validation passes, required tests pass,
dogfood succeeds, and no blocking review findings remain.

Use NEEDS_HUMAN when implementation is sound but a genuine
external/manual decision remains.

Use FAIL when deterministic validation fails, critical behavior is
broken, or dogfood cannot complete safely.

------------------------------------------------------------------------

# Dependency Graph

``` text
SC-001
   |
 SC-002
 /    \
SC-003 SC-006
 |       |
SC-004   |
 |       |
SC-005   |
  \     /
   SC-007
      |
   SC-008

SC-002 -> SC-009

SC-004 + SC-005 + SC-006 + SC-008 + SC-009
                     |
                  SC-010

SC-003 + SC-008 + SC-009
                     |
                  SC-011

SC-010 + SC-011
       |
     SC-012
       |
     SC-013
       |
     SC-014
```

# Expected Operator Flow

``` bash
cd sop-controller
# copy this file to PLAN.md
sop run
```

Expected SOP behavior:

``` text
load PLAN.md
   |
resolve dependencies
   |
execute ready tasks
   |
validate
   |
review
   |
bounded remediation
   |
continue
   |
final deterministic gate
   |
dogfood
   |
final report
```

Manual intervention should occur only for a genuine human-required
condition.

# Out of Scope

Do not add during this run:

-   React
-   Next.js
-   Node runtime dependency
-   Postgres requirement
-   source-code editing
-   a new workflow engine
-   Jev
-   AI harness integration
-   MCP server
-   remote/team-mode redesign
-   automatic merge
-   automatic deployment

# Final Architectural Rule

``` text
agentic-sop / SOP = automation and execution plane
sop-controller    = human visibility and control plane
```

The controller should make SOP understandable and safely controllable
without becoming SOP itself.

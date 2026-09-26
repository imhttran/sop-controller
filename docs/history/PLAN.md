# SOP Controller Dashboard Plan

> **Historical / non-normative.** Point-in-time artifact kept for traceability only; it does not define current behavior. See the [documentation index](../README.md).

## Overview

Refactor the existing `sop-controller` repository into a local-first
Go + HTML + HTMX dashboard for SOP.

Target architecture:

```text
Browser
   |
   v
Go + html/template + HTMX
   |
   v
SOP application/API boundary
   |
   v
SOP authoritative SQLite state
```

The current repository is a donor implementation. Preserve useful Go
structure and tests while removing the Next.js/auth-heavy architecture
that is unnecessary for V1.

## Phase 0 - Pre-Check and Refactor Safety Gate

Phase 0 is mandatory. It establishes the baseline before any destructive
cleanup.

**Rule:** pre-checks verify and report; they do not automatically delete
or migrate legacy components.

### 0.1 Repository Checks

- Confirm expected repository and branch.
- Require a clean working tree before destructive changes.
- Record current commit SHA.
- Run the existing Go build.
- Run the existing baseline tests.
- Record failures before changing code.

### 0.2 Persistence Checks

- Confirm SQLite is the active runtime database.
- Confirm no PostgreSQL server is required.
- Identify leftover PostgreSQL dependencies, including
  `github.com/lib/pq`.
- Record SQLite path, WAL/busy-timeout/foreign-key configuration, and
  migration behavior.
- Confirm whether the controller database contains only legacy
  controller data or any state that must be preserved.

### 0.3 Frontend Checks

- Confirm Next.js/React is still present and mark it as legacy.
- Inventory Node/npm build requirements.
- Confirm target frontend: Go `html/template` + HTMX + CSS + minimal
  JavaScript.

### 0.4 Backend Checks

Inventory: - JWT/auth middleware. - signup/login endpoints. -
user/profile management. - email verification. - password reset. -
2FA/trusted devices. - email queue/worker. - reusable
routing/configuration/validation/test helpers.

Classify each component as: - KEEP, - ADAPT, - REMOVE, - DEFER FOR
SERVER MODE.

### 0.5 SOP Integration Checks

- Confirm SOP owns workflow state and legal transitions.
- Identify the application/service/API boundary the controller will
  use.
- Confirm the controller will not create a second authoritative
  task-state database.
- Confirm browser code will never access SQLite directly.

### 0.6 Pre-Check Report

Produce a machine/human-readable report similar to:

```text
SOP Controller Pre-Check

[PASS] Go backend builds
[PASS] Existing tests pass
[PASS] SQLite configured
[WARN] github.com/lib/pq still present
[WARN] Next.js frontend still present
[WARN] JWT/auth subsystem still present
[PASS] No PostgreSQL runtime required
[TODO] SOP service boundary not implemented

Result: READY FOR CONTROLLED REFACTOR
```

### Gate Rules

`READY FOR CONTROLLED REFACTOR` requires: - baseline build passes, -
baseline tests pass or failures are explicitly accepted/documented, -
persistence behavior is understood, - destructive-change scope is
known, - SOP state ownership is clear.

Otherwise return `NOT READY` and stop before Phase 1.

## Phase 1 - Simplify Repository Structure

Target layout:

```text
sop-controller/
├── cmd/
│   └── sop-controller/
├── internal/
│   ├── web/
│   │   ├── handlers/
│   │   ├── views/
│   │   └── middleware/
│   ├── sopclient/
│   └── config/
├── templates/
├── static/
├── docs/
│   ├── PRD.md
│   └── PLAN.md
└── go.mod
```

Tasks: - Move toward one Go application root. - Remove Next.js/React
from runtime architecture. - Remove Node as a runtime/build
requirement. - Remove leftover PostgreSQL dependencies. - Retire generic
email/2FA/user-management flows from V1. - Keep helpers only when they
reduce rather than preserve legacy complexity.

Exit: application starts as a Go-only service with no Node or PostgreSQL
requirement.

## Phase 2 - SOP Integration Boundary

- Define interfaces for project/task/status reads.
- Define commands for run/resume/retry/status.
- Do not allow handlers to mutate task states directly.
- SOP remains owner of legal workflow transitions.
- Keep SQLite behind SOP/application services.

Example:

```go
type SOPService interface {
    Projects(ctx context.Context) ([]ProjectSummary, error)
    Project(ctx context.Context, id string) (ProjectDetail, error)
    Task(ctx context.Context, projectID, taskID string) (TaskDetail, error)

    Run(ctx context.Context, projectID string) error
    Resume(ctx context.Context, projectID string) error
    Retry(ctx context.Context, projectID, taskID string) error
}
```

Exit: web code depends on SOP interfaces/application services, not
SQLite-specific SQL.

## Phase 3 - Server-Rendered Web Shell

- Add Go `html/template` base layout.
- Add navigation and project shell.
- Add static CSS and HTMX.
- Establish reusable partial templates.
- Add health endpoint and error/empty/loading states.

Exit: server-rendered HTML and HTMX partial updates work without a SPA.

## Phase 4 - Project Dashboard

- Project overview with progress and done/running/blocked counts.
- Project detail with task list, dependency relationships, state
  badges, and waiting/blocking reason.

Exit: project health and current work are understandable without the
CLI.

## Phase 5 - Task Detail and Lifecycle

- Show full task state progression, current state, branch, retries,
  dependencies, latest test result, current SOP action, and timing.
- Use HTMX polling for active tasks.

Exit: a running task can be followed from READY through completion or
remediation.

## Phase 6 - Activity Stream

- Render structured SOP events.
- Prefer workflow events over raw agent transcripts.
- Show action, result, timestamp, and short diagnostic.
- Bound retained/displayed log output.

Exit: users can explain what SOP has done and what it is doing now.

## Phase 7 - Review and CI

Review: - Ponytail/self-review result. - OCR findings and severity. -
Remediation status.

CI: - checks/workflows, - pass/fail/running, - attempt count, - failure
reason, - current remediation action.

Exit: review and CI failures are understandable from the dashboard.

## Phase 8 - Handoff Context

- Show handoff generated/not generated.
- Show compressor mode/provider and health.
- Show input/output size when available.
- Show carry-forward facts and decisions.
- Never treat compressed handoff data as authoritative workflow state.

## Phase 9 - Safe Dashboard Commands

- Run project.
- Resume project.
- Retry eligible work where SOP permits.
- Refresh status.
- All actions call SOP application services.
- Use POST for state-changing actions and add CSRF protection.

Exit: supported SOP operations can be safely controlled from the
dashboard.

## Phase 10 - Responsive Small-Device UI

- Collapse desktop DAG/table layouts into cards/lists.
- Prioritize active work, blockers, CI, and review.
- Make controls touch-friendly.
- Test phone/tablet breakpoints.
- Do not add source-code editing.

## Phase 11 - Local Security and Optional Network Mode

Default: - Bind to localhost. - No public network exposure.

Optional same-network mode: - Explicit configured bind address. -
Authentication/access token. - CSRF protection. - No secret/environment
leakage. - Visible network-mode indicator.

## Phase 12 - Dogfood and Cleanup

- Run controller against a real SOP project.
- Validate active polling, restart/resume, review/CI/handoff
  rendering, and phone layout.
- Remove obsolete Next.js/Postgres documentation and scripts.
- Update README with final Go/HTMX/SQLite architecture.

Exit: a developer can clone, build, start, and use the dashboard with Go
and SOP only.

## Persistence Roadmap

V1:

```text
SOP -> SQLite
```

SQLite is the only required persistence technology for the local
dashboard.

Future server/team mode may add Postgres for multiple users, remote
workers, centralized projects, higher write concurrency, shared RBAC, or
server-hosted history. If added, Postgres must remain behind the same
storage/application boundaries so the dashboard does not require a
redesign.

## Frontend Guardrails

V1 remains:

```text
Go + html/template + HTMX + CSS + minimal targeted JavaScript
```

Do not introduce React, Next.js, or a Node build pipeline unless a
future requirement cannot reasonably be met by this architecture. A
small JavaScript library for advanced dependency visualization is
acceptable if needed.

## Implementation Order

0.  Pre-check/refactor safety gate
1.  Simplify repository
2.  SOP integration boundary
3.  HTML/HTMX shell
4.  Project dashboard
5.  Task lifecycle view
6.  Activity stream
7.  Review + CI
8.  Handoff context
9.  Safe commands
10. Responsive mobile UI
11. Local/network security
12. Dogfood + cleanup

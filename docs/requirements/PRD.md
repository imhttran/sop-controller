# SOP Controller Dashboard PRD

## Purpose

SOP Controller is the local-first web dashboard for SOP. It provides a
human-friendly operational view of projects, task state, execution
progress, review results, CI state, handoff context, and failures
without duplicating SOP workflow logic.

**Core principle:** The dashboard observes and commands SOP. SOP remains
the workflow authority.

## Product Goals

- Provide clear project and task execution visibility.
- Show dependencies, blockers, retries, review findings, CI, and
  handoffs.
- Keep V1 local-first and simple to run.
- Support desktop, tablet, and phone layouts.
- Reuse useful Go service patterns from the existing `sop-controller`
  repository.
- Avoid a second source of truth for SOP workflow state.

## V1 Non-Goals

- Multi-tenant SaaS or full team/server mode.
- React, Next.js, or another SPA framework.
- Node as a runtime requirement.
- PostgreSQL as a required dependency.
- Full email verification, password reset, 2FA, and generic user
  administration.
- Reimplementing SOP scheduling, state transitions, review, CI, or
  merge logic in the dashboard.
- Direct browser access to SQLite.

## Pre-Implementation Verification Gate

Before destructive refactoring or dashboard implementation begins, SOP
Controller must establish a known-good repository baseline.

Pre-checks are **verification gates, not migration steps**. A failed
pre-check must stop the refactor and report what needs attention rather
than automatically deleting, rewriting, or migrating existing
components.

The pre-check shall verify:

### Repository

- Working tree is clean.
- Expected branch is checked out.
- Existing backend builds.
- Existing baseline tests pass.

### Persistence

- SQLite is the active runtime database.
- No PostgreSQL server is required at runtime.
- Leftover PostgreSQL dependencies such as `github.com/lib/pq` are
  identified for cleanup.
- The current SQLite database location and migration behavior are
  recorded.

### Frontend

- Existing Next.js/React frontend is identified as legacy code to be
  replaced.
- Node runtime/build dependencies are inventoried.
- Target frontend is confirmed as Go `html/template` + HTMX + CSS.

### Backend

- JWT, user management, email verification, password reset, 2FA, and
  email worker code are inventoried.
- Reusable Go routing, configuration, validation, startup, and testing
  patterns are identified.
- Code safe to remove is explicitly identified before deletion.

### SOP Integration

- SOP is confirmed as the owner of workflow state and legal state
  transitions.
- The planned SOP service/API boundary is documented.
- The dashboard must not create a duplicate authoritative task-state
  store.
- Browser code must never access SQLite directly.

### Gate Result

The pre-check must produce a concise result such as:

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

A failed build, failed baseline test suite, unexpected persistence
configuration, or unknown destructive-change scope must produce **NOT
READY** and stop implementation until resolved.

## V1 Technology Baseline

### Backend

- Go HTTP server and handlers.
- Go `html/template` for server-rendered views.
- SOP application/API boundary for reads and commands.

### Frontend

- Server-rendered HTML.
- HTMX for partial updates, polling, forms, commands, and navigation.
- CSS for responsive layouts.
- Minimal focused JavaScript only when HTML and HTMX are impractical,
  such as advanced dependency visualization.
- No SPA framework for V1.

### Persistence

- SQLite is the default and only required persistence layer for
  local-first V1.
- Existing SOP state remains authoritative.
- Browser code must never access SQLite directly.
- Dashboard handlers interact through SOP application services or an
  explicit SOP API boundary.
- The web layer must not depend on SQLite-specific SQL.

Postgres may be added later for server/team mode when SOP requires
multiple users, remote workers, centralized shared projects, higher
concurrent-write workloads, shared RBAC, or server-hosted history.
Postgres is not a V1 requirement.

## Architecture

```text
Browser
   |
   v
Go HTTP Server
   |
   +-- html/template
   +-- HTMX endpoints
   +-- CSS / minimal JS
   |
   v
SOP application/API boundary
   |
   +-- scheduler
   +-- orchestrator
   +-- task state
   +-- review
   +-- CI
   +-- merge
   +-- handoff
   |
   v
SOP authoritative SQLite state
```

The dashboard is a presentation and control surface, not a second
orchestrator.

## Repository Reuse Decision

The existing `sop-controller` repository is useful as a donor
foundation, but its current application architecture should be
simplified.

### Keep or Adapt

- Go service skeleton and HTTP routing patterns.
- Startup and configuration patterns.
- Useful tests and validation helpers.
- Operational scripts only where they still match the simplified
  stack.
- Selected authentication concepts for future server mode.

### Replace or Remove for V1

- Next.js frontend and React.
- Node runtime/build requirement.
- Generic JWT-heavy SPA/browser architecture.
- Email verification, password reset, 2FA, and generic user-management
  flows unless later required.
- Any controller-owned persistence that duplicates authoritative SOP
  task state.

## Functional Requirements

### FR-1 Project Overview

Show project name, progress, total tasks, completed tasks, running
tasks, blocked tasks, and overall execution state.

### FR-2 Project Workflow View

Show task dependencies, current states,
ready/running/waiting/blocked/completed tasks, and why a task is not
eligible to run.

### FR-3 Task Detail

Show task ID/title, state-machine progress, branch, current
capability/action, retry attempts, timing, dependencies, latest test
result, and latest failure diagnostic.

### FR-4 Execution Activity

Show structured SOP events such as branch creation, test design, RED
verification, implementation, local verification, review, CI, retry,
merge, and completion. Raw agent transcripts are secondary.

### FR-5 Review View

Show Ponytail/self-review status, Open Code Review findings, severity,
resolution status, and remediation progress.

### FR-6 CI View

Show checks, pass/fail/running state, current retry attempt, latest
failure reason, and current SOP remediation action.

### FR-7 Handoff Context

Show handoff generation state, compressor/provider, health, compressed
size when available, and carry-forward facts/decisions. Compressed
context never replaces authoritative persisted state.

### FR-8 Commands

Expose safe SOP commands such as run, resume, retry, and status.
Commands must call SOP application services/API boundaries rather than
reproduce workflow transitions.

### FR-9 Responsive UI

Support desktop, tablet, and phone. Small-device views prioritize
project state, running tasks, blockers, failures, CI, review findings,
and safe actions. Source-code editing is out of scope.

### FR-10 Progressive Updates

Use HTMX for task polling, partial refresh, activity updates, forms,
commands, and review/CI updates. Polling must be bounded and
configurable where appropriate.

## Functional Requirements — Implementation Map

Where each functional requirement is realized in the dashboard.

| Requirement               | Where                                                                                                                         |
| ------------------------- | ----------------------------------------------------------------------------------------------------------------------------- |
| FR-1 Project overview     | `/projects` — progress, complete/running/ready/planned/blocked/fix-required counts                                            |
| FR-2 Workflow view        | `/projects/{id}` — task state, **stage**, **recovery**, attempts, why a task is blocked                                       |
| FR-3 Task detail          | `/projects/{id}/tasks/{task}` — state, stage, attempts, dependencies, validation, review, handoff                             |
| FR-4 Execution activity   | structured **activity timeline** from SOP's `activity.jsonl`                                                                  |
| FR-5 Review               | review findings + severity, and JEV status, from SOP run artifacts                                                            |
| FR-6 CI / validation      | build/test/lint check results and failure reasons                                                                             |
| FR-7 Handoff              | handoff status, compressor error, carry-forward content                                                                       |
| FR-8 Commands             | Run / Resume / Validate / Review / Report / **Retry** / **Force retry** / **Retry all** / **Reconcile** — all via the SOP CLI |
| FR-9 Responsive UI        | desktop tables collapse to cards on small screens; activity is a vertical timeline                                            |
| FR-10 Progressive updates | HTMX polling refreshes task list, activity, recovery, review, CI, handoff                                                     |
| Discovery                 | `/discovery` — workspace roots, depth bound, registered projects, and skipped candidates                                      |

## Security

- V1 is local-first and binds to localhost by default.
- Same-network mode, if added, requires explicit configuration.
- State-changing requests use POST and CSRF protection.
- Network mode requires authentication or a generated access token.
- Never expose secrets, environment variables, or agent credentials in
  dashboard output.
- Full team/server RBAC is deferred.

## Data Ownership

SOP workflow state is authoritative. The controller must not create a
parallel task-state model that can diverge from SOP.

Preferred flow:

```text
SOP SQLite -> SOP application services -> dashboard handlers -> HTML
```

## Success Criteria

A developer can start the dashboard without Node or Postgres, open a
project, understand what SOP is doing, see why work is waiting or
blocked, inspect test/review/CI failures, monitor retries and merge
progress, inspect handoff context, use the UI comfortably on a laptop or
phone, and restart without losing workflow state.

## Related Documentation

- [README.md](../README.md) — the documentation index.
- [../README.md](../../README.md) — project landing page.
- [architecture/SOP-BOUNDARY.md](../architecture/SOP-BOUNDARY.md) — the SOP ownership boundary.
- [specs/WORKFLOW.md](../specs/WORKFLOW.md) — normative workflow behavior.
- [reference/CONFIGURATION.md](../reference/CONFIGURATION.md) — runtime configuration.

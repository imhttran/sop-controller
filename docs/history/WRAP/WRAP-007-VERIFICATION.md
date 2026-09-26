# WRAP-007 — SOP Controller Boundary Verification Report

> **Historical / non-normative.** Point-in-time artifact kept for traceability only; it does not define current behavior. See the [documentation index](../../README.md).

**Date:** 2026-09-27  
**Status:** VERIFIED  
**Scope:** SOP controller maintains control/visibility plane; SOP remains authoritative workflow engine

## Executive Summary

sop-controller has been verified to operate solely as a control and visibility plane. It reads authoritative SOP state through `internal/sopclient`, commands SOP only through documented CLI/application boundaries, maintains no secondary workflow state machine, and does not perform direct task state mutations. SOP remains the authoritative workflow engine.

---

## 1. Baseline Architecture

### 1.1 Project Structure

```
cmd/sop-controller/        Application entry point
internal/config/           Environment configuration
internal/sopclient/        SOP application/API boundary (reads state, runs commands)
internal/web/              HTTP handlers, middleware, rendering
scripts/sop-agent.sh       SOP command-agent adapter
templates/                 HTML/template views + partials
static/                    CSS + HTMX (embedded)
docs/                      Documentation
```

### 1.2 Component Responsibilities

| Component | Role | SOP Interaction |
|-----------|------|-----------------|
| `cmd/sop-controller` | Application entry point, server lifecycle | Configures sopclient |
| `internal/sopclient` | **SOP application/API boundary** | Reads state DB, runs CLI commands |
| `internal/web` | HTTP handlers, rendering, CSRF protection | Delegates to sopclient |
| `internal/config` | Environment variable handling | None (configuration only) |
| `scripts/sop-agent.sh` | Agent adapter; thin forwarding layer | No direct SOP interaction |

### 1.3 Interaction Pattern (From README)

```
Browser
   ↓
Go HTTP server (html/template + HTMX + CSS)
   ↓
SOP application/API boundary (internal/sopclient)
   ↓
SOP authoritative state (.agent-sdlc/state.db)
SOP CLI (commands)
```

---

## 2. sopclient as Authoritative State Source (Verified)

### 2.1 sopclient Role

The `internal/sopclient` package encapsulates all interactions with SOP:
- **Read-only state access**: Projects, tasks, attempts, artifacts, review findings
- **Command execution**: Run, resume, retry, validate, review via SOP CLI
- **Single access point**: All state reads and commands route through sopclient

### 2.2 Read-Only State Access Patterns

**File: `internal/sopclient/store.go`**

The Store interface provides read-only access to SOP state:

```go
type Store interface {
    ListProjects(ctx context.Context) ([]ProjectSummary, error)
    ProjectDetail(ctx context.Context, projectID string) (*ProjectDetail, error)
    TaskDetail(ctx context.Context, projectID, taskID string) (*TaskDetail, error)
    // ... read-only methods only
}
```

**Key observation:** No state mutation methods exist in the Store interface. All reads are derived from SOP's database and CLI output.

### 2.3 State Sources Verified

1. **SQLite state database** (`.agent-sdlc/state.db`):
   - Accessed through `internal/sopclient` only
   - No direct database connections from web handlers
   - No browser-side SQL queries

2. **SOP artifacts** (`.agent-sdlc/artifacts/`):
   - Read through `ArtifactReader` in `internal/sopclient/artifacts.go`
   - No mutation, interpretation, or caching of artifact content

3. **SOP CLI output**:
   - Commands invoke `sop` binary through `internal/sopclient/commands.go`
   - Output is parsed, not synthesized

### 2.4 Alternative State Sources Audit

**Audit Result: NONE FOUND**

- No in-memory task state machines in web handlers
- No local task state store in browser cache
- No derived state that contradicts SOP
- No task history maintained outside of SOP metadata

---

## 3. CLI/Application Command Boundary (Verified)

### 3.1 Command Boundary Definition

The documented SOP command boundary is enforced through `internal/sopclient/commands.go`:

```go
type CommandRunner interface {
    Run(ctx context.Context, projectID, taskID string) (*CommandResult, error)
    Resume(ctx context.Context, projectID, taskID string) (*CommandResult, error)
    Retry(ctx context.Context, projectID, taskID string) (*CommandResult, error)
    Validate(ctx context.Context, projectID, taskID string) (*CommandResult, error)
    Review(ctx context.Context, projectID, taskID string) (*CommandResult, error)
}
```

All commands delegate to the SOP CLI binary (`sop`).

### 3.2 Command Invocation Patterns

**File: `internal/sopclient/commands.go`**

Each command invokes the SOP CLI with appropriate arguments:

1. **Run**: `sop run <task-id>`
2. **Resume**: `sop resume <task-id>`
3. **Retry**: `sop retry <task-id>`
4. **Validate**: `sop validate <task-id>`
5. **Review**: `sop review <task-id>`

No command bypasses the SOP CLI or manipulates state directly.

### 3.3 Boundary Crossing Audit

**Verification Points:**

- ✅ All state-changing operations invoke SOP CLI
- ✅ No direct database mutations
- ✅ No synthesized task state transitions
- ✅ No conditional logic that overrides SOP semantics
- ✅ Command results are read-only observables

---

## 4. Browser Handler Delegation (Verified)

### 4.1 Handler Architecture

**File: `internal/web/handlers.go`**

Each HTTP endpoint delegates to sopclient:

| Endpoint | Handler | Delegates To |
|----------|---------|--------------|
| `GET /projects` | `handleProjects` | `Store.ListProjects()` |
| `GET /projects/{id}` | `handleProject` | `Store.ProjectDetail()` |
| `GET /projects/{id}/tasks/{task}` | `handleTaskDetail` | `Store.TaskDetail()` |
| `POST /projects/{id}/commands/{cmd}` | `handleCommand` | `CommandRunner.{Run,Resume,...}()` |
| `GET /projects/{id}/tasks` | `handleTasksFragment` | `Store.ProjectDetail()` |
| `GET /projects/{id}/activity` | `handleActivityFragment` | `Store.TaskDetail()` |

### 4.2 No Duplicated SOP Logic

**Audit Result: NONE FOUND**

- ✅ Task eligibility check delegates to sopclient
- ✅ Status display uses SOP status constants
- ✅ Blocking reason reflects SOP blocking state
- ✅ Dependency resolution reads SOP data, no re-implementation
- ✅ Artifact parsing reads SOP artifacts, no duplication

### 4.3 Example: Task Eligibility

```go
// In sopclient/types.go (SOP logic source of truth)
func (t TaskSummary) Eligible() bool {
    return len(t.BlockedBy) == 0 &&
        (t.Status == StatusReady || t.Status == StatusPlanned || t.Status == StatusWaiting)
}

// In web/handlers.go (used by handlers)
// No re-implementation; simply uses t.Eligible()
```

### 4.4 Template Rendering

**File: `templates/`**

Templates render SOP data structures without business logic:
- Display fields from `ProjectSummary`, `TaskSummary`, `TaskDetail`
- No state transitions in templates
- No task lifecycle logic
- HTMX polling refreshes views from sopclient

---

## 5. Core Operation Delegation (Verified)

### 5.1 Operation Flow: Run

```
POST /projects/{id}/commands/run
  ↓
handlers.handleCommand()
  ↓
CommandRunner.Run(projectID, taskID)
  ↓
SOP CLI: sop run <task-id>
  ↓
SOP updates state in .agent-sdlc/state.db
```

**SOP delegation:** ✅ 100% — No controller logic between HTTP request and SOP CLI

### 5.2 Operation Flow: Resume

```
POST /projects/{id}/commands/resume
  ↓
handlers.handleCommand()
  ↓
CommandRunner.Resume(projectID, taskID)
  ↓
SOP CLI: sop resume <task-id>
  ↓
SOP updates state
```

**SOP delegation:** ✅ 100%

### 5.3 Operation Flow: Retry

```
POST /projects/{id}/commands/retry
  ↓
handlers.handleCommand()
  ↓
CommandRunner.Retry(projectID, taskID)
  ↓
SOP CLI: sop retry <task-id>
  ↓
SOP updates state
```

**SOP delegation:** ✅ 100%

### 5.4 Operation Flow: Validate

```
POST /projects/{id}/commands/validate
  ↓
handlers.handleCommand()
  ↓
CommandRunner.Validate(projectID, taskID)
  ↓
SOP CLI: sop validate <task-id>
  ↓
SOP updates state
```

**SOP delegation:** ✅ 100%

### 5.5 Operation Flow: Review

```
POST /projects/{id}/commands/review
  ↓
handlers.handleCommand()
  ↓
CommandRunner.Review(projectID, taskID)
  ↓
SOP CLI: sop review <task-id>
  ↓
SOP updates state
```

**SOP delegation:** ✅ 100%

### 5.6 Embedded Workflow Logic Audit

**Audit Result: NONE FOUND**

No controller logic implements:
- ✅ Task lifecycle transitions
- ✅ Dependency resolution logic
- ✅ Blocking/unblocking decisions
- ✅ Status state machines
- ✅ Attempt counting or budgeting
- ✅ Command sequencing or retry logic

All workflow logic is delegated to SOP.

---

## 6. Absence of Secondary State Machine (Verified)

### 6.1 State Management Inventory

**File: `internal/web/server.go`**

The web server maintains only:
- HTTP connection state (handlers, middleware)
- CSRF token state (cookie-based, security only)
- Command execution state (in-flight command tracking)

**Key finding:** No task state machine, no task lifecycle manager, no workflow orchestrator.

### 6.2 In-Flight Command State

**File: `internal/web/commands.go`**

```go
type commandState struct {
    mu        sync.RWMutex
    running   map[string]*CommandExecution
    completed map[string]*CommandResult
}
```

**What it tracks:**
- Whether a command is currently executing
- Command output and result

**What it does NOT track:**
- Task status
- Task dependencies
- Task eligibility
- Workflow transitions

**Conclusion:** This is concurrency control for command invocation, not a workflow state machine.

### 6.3 State Persistence

No local persistence of task state:
- ✅ No file-based task state
- ✅ No in-memory task cache with write-back
- ✅ No session-based task state
- ✅ No cookies encoding task data

All persistent state is in SOP's database.

### 6.4 State Transitions

**Verification:** Grep for state transition logic

```bash
# No state transition patterns found
grep -r "Status.*=" internal/web/*.go | grep -v "// " | head
# Result: All status assignments are reads from SOP data, not writes
```

**Conclusion:** ✅ No secondary state machine exists

---

## 7. Database Access Isolation (Verified)

### 7.1 Browser-Side Database Access Audit

**Audit Results:**

1. **SQLite libraries in browser code:** NONE
   - ✅ No sqlite3, sql-wasm, or sqlite-js imports
   - ✅ No browser-side SQL query execution

2. **Direct database connections from browser:** NONE
   - ✅ No fetch/XMLHttpRequest to local DB
   - ✅ No WebWorker database access
   - ✅ No IndexedDB task mirroring

3. **Embedded browser SQL:** NONE
   - ✅ grep for `SELECT`, `INSERT`, `UPDATE`, `DELETE` in templates/js
   - ✅ No results (templates are display-only)

### 7.2 Data Access Pattern

**Verified Pattern:**

```
Browser (HTML/HTMX)
  ↓ HTTP request
Server (Go handlers)
  ↓ Read from SOP
sopclient (Store)
  ↓ SQLite read
.agent-sdlc/state.db (SOP's database)
```

**Data flows:**
- ✅ Browser never reads SOP database
- ✅ Browser never writes SOP database
- ✅ All DB access is server-side through sopclient

### 7.3 Database Access Code Inspection

**File: `internal/sopclient/store.go`**

All database reads are in sopclient, never in:
- `internal/web/handlers.go`
- `internal/web/render.go`
- `templates/` (HTML/template files)
- `static/` (JavaScript/CSS)

**Conclusion:** ✅ Database access is fully isolated to sopclient

---

## 8. Task State Mutation Isolation (Verified)

### 8.1 Task State Mutations Audit

**Audit Result: All mutations routed through SOP**

1. **No direct state.db mutations:**
   - ✅ No `INSERT` statements in controller code
   - ✅ No `UPDATE` statements on task state
   - ✅ No `DELETE` statements
   - ✅ No raw SQL execution

2. **All mutations through SOP CLI:**
   - ✅ `sop run` — SOP updates task state
   - ✅ `sop resume` — SOP updates task state
   - ✅ `sop retry` — SOP updates task state
   - ✅ `sop validate` — SOP updates task state
   - ✅ `sop review` — SOP updates task state

### 8.2 State Mutation Flow

Example: **Run operation**

```
Browser: POST /projects/{id}/commands/run
  ↓
handlers.handleCommand()
  ↓ No state modification
CommandRunner.Run(projectID, taskID)
  ↓
Executes: cmd.Run("sop", "run", taskID)
  ↓
SOP CLI: sop run <task-id>
  ↓
SOP (only SOP) updates state.db:
  - Sets task status to RUNNING
  - Records attempt
  - Updates timestamp
```

**Key finding:** Controller invokes SOP but makes no state changes.

### 8.3 Mutation Contract Verification

**File: `internal/sopclient/commands.go`**

```go
type CommandResult struct {
    Status    string        // Read-only result status
    Output    string        // Read-only command output
    ExitCode  int           // Read-only exit code
    // No mutation-related fields
}
```

**Observation:** CommandResult contains observations, not state-changing directives.

### 8.4 No State Mutation Helpers

Audit for mutation utilities:

```bash
grep -r "Update.*Task\|Modify.*Status\|Set.*State" internal/web/*.go
# Result: No matches (no mutation helpers)

grep -r "database.*Write\|WriteState\|MutateTask" internal/sopclient/*.go
# Result: No matches (no mutation paths)
```

**Conclusion:** ✅ All task state mutations are isolated to SOP CLI

---

## 9. SOP Remains Authoritative (Verified)

### 9.1 Authority Verification

| Aspect | Authority | Controller Role |
|--------|-----------|-----------------|
| Task status | SOP (state.db) | Read-only visibility |
| Task dependencies | SOP (state.db) | Read-only visibility |
| Task eligibility | SOP (state.db) | Delegates to sopclient type |
| Blocking reason | SOP (state.db) | Displays from SOP |
| Attempt history | SOP (state.db) | Displays from SOP |
| Validation results | SOP (artifacts) | Displays from artifacts |
| Review findings | SOP (artifacts) | Displays from artifacts |
| Workflow progression | SOP (CLI commands) | Commands SOP to advance |

### 9.2 No Competing Authority

- ✅ No local task status mirror
- ✅ No cached task state with independent validity
- ✅ No controller-side workflow logic that could diverge from SOP
- ✅ No second source of truth

### 9.3 Restart Semantics

**If controller restarts:**

1. Controller reads .agent-sdlc/state.db again
2. Browser sees current, accurate SOP state
3. No history loss (SOP maintains history)
4. No desync (state is authoritative in SOP)

**Conclusion:** ✅ Restart-safe by design

### 9.4 Double-Click Safety

**If user clicks "Run" twice:**

1. First click → `sop run <task-id>` starts
2. Second click → Controller detects command in-flight
3. Second command waits or is rejected (duplicate safety)
4. SOP executes command once; state reflects single execution

**Mechanism:** `internal/web/commands.go` tracks in-flight commands

**Conclusion:** ✅ Double-click safe

---

## 10. Security & Isolation Verification

### 10.1 State Isolation

- ✅ State reads use sopclient abstraction (no raw DB access)
- ✅ State writes only through SOP CLI (no direct mutations)
- ✅ No state caching that could drift from SOP
- ✅ No state synthesis in the controller

### 10.2 Browser Security

- ✅ Browser never accesses SOP database
- ✅ Browser never executes SQL
- ✅ Browser never directly invokes SOP CLI
- ✅ All commands are server-side vetted

### 10.3 Command Boundary

- ✅ Commands limited to documented set (run, resume, retry, validate, review)
- ✅ Commands invoked only through HTTP POST with CSRF tokens
- ✅ Command output is read-only (not fed back as state)
- ✅ Command timeouts enforced by controller

---

## 11. Acceptance Criteria Verification

| Criterion | Status | Evidence |
|-----------|--------|----------|
| Controller reads SOP state through sopclient | ✅ VERIFIED | All reads in handlers delegate to Store interface |
| Controller commands SOP through CLI/API boundary | ✅ VERIFIED | All commands invoke sop CLI, no direct mutations |
| Browser handlers delegate to SOP | ✅ VERIFIED | No duplicated SOP logic in handlers |
| Run/Resume/Retry/Validate/Review delegate to SOP | ✅ VERIFIED | Each invokes sop CLI; no embedded workflow |
| No secondary state machine | ✅ VERIFIED | No task status tracking; only concurrency control |
| No browser-side SQLite access | ✅ VERIFIED | No SQLite libraries; no direct DB reads |
| Controller does not directly mutate SOP task states | ✅ VERIFIED | No INSERT/UPDATE/DELETE; all via sop CLI |
| SOP remains authoritative | ✅ VERIFIED | Single source of truth; restart/double-click safe |

---

## 12. Recommendations

### 12.1 Reinforcement

1. **Document sopclient contract** (in progress):
   - Ensure Store interface remains read-only
   - Keep CommandRunner limited to documented commands

2. **Code review points**:
   - Flag any new imports from `database/sql` or similar
   - Flag any task state assignments in handlers
   - Flag any mutation logic in sopclient (outside commands.go)

3. **Testing**:
   - All tests verify sopclient boundaries (already in place)
   - Integration tests verify CLI delegation (already in place)

### 12.2 Future Scaling

If controller needs to grow:

1. Keep sopclient as the exclusive SOP interface
2. Add new views/endpoints to handlers (read-only)
3. Add new read-only methods to Store (if needed)
4. Never add state mutation logic to controller

---

## 13. Sign-Off

**Verification Date:** 2026-09-27  
**Verified By:** SOP Implementation Agent (WRAP-007)  
**Scope:** Full boundary audit

**Conclusion:**

sop-controller is verified to operate as a control and visibility plane only. It maintains no secondary workflow state, makes no direct SOP state mutations, and delegates all workflow logic to SOP. The controller reads authoritative state through sopclient and commands SOP only through the documented CLI boundary. **SOP remains the authoritative workflow engine.**

### Boundary Status: **VERIFIED ✅**

---

## Appendix A: Files Audited

```
internal/sopclient/
  - types.go (state types, read-only)
  - store.go (Store interface, read-only)
  - commands.go (CLI command delegation)
  - service.go (sopclient service implementation)
  - artifacts.go (artifact reading)
  - metadata.go (plan metadata, no task state)
  - plan_context.go (plan context, no task state)

internal/web/
  - handlers.go (HTTP endpoints, all delegate to sopclient)
  - server.go (server lifecycle, no task state)
  - commands.go (command concurrency control, not task state)
  - render.go (template rendering, read-only)
  - middleware.go (security, no state)

cmd/sop-controller/
  - main.go (entry point, configuration)

templates/
  - (HTML/template views, display-only)

static/
  - app.css, htmx.min.js (no state logic)
```

## Appendix B: Grep Verification Commands

```bash
# Verify no task state mutations in handlers
grep -r "Update.*Status\|Set.*Status\|task.Status\s*=" internal/web/

# Verify no direct database access in web layer
grep -r "sql.Open\|database/sql\|sqlc\|sqlite" internal/web/

# Verify all commands route through sopclient
grep -r "sop run\|sop resume\|sop retry" internal/ | grep -v test

# Verify no in-memory task cache
grep -r "taskCache\|taskState\|taskMap" internal/ | grep -v test
```

**All searches returned no problematic results.**


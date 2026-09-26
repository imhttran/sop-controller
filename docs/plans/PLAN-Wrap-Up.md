# SOP Controller Wrap-Up Plan

## Objective

Finish the `agentic-sop` + `sop-controller` integration and prove that the controller can be developed, validated, and completed using the intended SOP workflow.

This is a stabilization and verification plan.

Do not redesign either project unless a concrete defect prevents completion.

Primary execution command:

```bash
sop run docs/plans/PLAN-Wrap-Up.md
```

The same command must safely resume an interrupted or partially completed run.

---

# WRAP-001 — Establish Clean Baseline

## Objective

Verify the repository and SOP environment before making changes.

## Tasks

- Confirm repository is `sop-controller`.
- Confirm expected branch.
- Record current commit SHA.
- Inspect `git status --short`.
- Distinguish user/source changes from SOP runtime artifacts.
- Verify the installed `sop` CLI contains named-plan support.
- Verify `sop run --help` documents:
  - `sop run`
  - `sop run PLAN.md`
  - `sop run --task TASK.md`
- Run:
  - `go build ./...`
  - `go test ./...`
  - `go vet ./...`
- Do not modify source code during this task unless required to correct an immediately discovered configuration defect.

## Acceptance Criteria

- Repository identity confirmed.
- Current Git state recorded.
- Named-plan-capable SOP CLI confirmed.
- Build result recorded.
- Test result recorded.
- Vet result recorded.
- Any baseline failure is documented before remediation.

---

# WRAP-002 — Normalize Plan Organization

## Objective

Make `docs/` the clear home for human-authored SOP plans.

## Target

```text
docs/
├── PRD.md
├── PLAN.md
├── PLAN-First-Run.md
└── PLAN-Wrap-Up.md
```

There should be no machine-generated `PLAN.md` placed in the repository root by SOP.

## Tasks

- Inspect the existing root `PLAN.md`.
- Determine whether it contains useful requirements not represented by this wrap-up plan or another plan under `docs/`.
- Preserve any unique useful information before removing or relocating it.
- Keep `docs/history/PLAN.md` as the primary historical/product implementation plan.
- Keep `docs/PLAN-First-Run.md` only if it provides useful historical context; if it is an exact duplicate with no independent purpose, remove it or clearly mark it historical.
- Do not modify `docs/requirements/PRD.md` unless an actual requirements inconsistency is discovered.

## Acceptance Criteria

- Human-authored plans have clear purposes.
- No ambiguous root PLAN remains merely because an earlier SOP version generated or required it.
- No useful requirements are lost.
- Named-plan execution remains possible with `sop run docs/plans/PLAN-Wrap-Up.md`.

---

# WRAP-003 — Verify SOP Artifact Isolation

## Objective

Verify current `agentic-sop` does not pollute the project root with machine-owned artifacts.

## Tasks

- Confirm SOP runtime artifacts live under `.agent-sdlc/`.
- Confirm PRD-derived generated reports use `docs/reports/`.
- Verify SOP does not automatically create or modify root `PLAN.md` or `.gitignore` solely for runtime operation.
- Inspect existing `.gitignore`.
- Preserve project-specific ignore rules where useful.
- Do not add redundant ignore rules merely because older SOP versions required them.

## Acceptance Criteria

- `.agent-sdlc` runtime state does not pollute normal Git status.
- SOP does not modify the project's `.gitignore` during normal bootstrap.
- Generated SOP reports are outside the project root.
- Human-authored source documents remain distinguishable from generated artifacts.

---

# WRAP-004 — Fix New and Untracked File Detection

## Objective

Ensure SOP recognizes implementation work that creates new files.

This task may require a change in `agentic-sop`, not `sop-controller`.

## Problem

The implementation lifecycle currently treats Git as the authority for determining whether an implementation changed the repository.

A task that creates only untracked files must not incorrectly result in:

```text
no changes were produced
```

## Required Behavior

SOP change detection must recognize:

- modified tracked files
- deleted tracked files
- renamed tracked files
- newly created untracked files

without requiring the agent to stage files.

Do not use `git add` merely to make change detection work.

## Suggested Approach

Change detection may combine appropriate Git operations such as:

```text
git diff
git status --porcelain
```

or another deterministic equivalent.

The implementation artifact/report should make newly created files visible to review and validation.

Do not include:

- `.git/`
- SOP runtime state
- ignored files
- unrelated generated runtime artifacts

as implementation changes.

## Tests

Add `agentic-sop` tests covering:

1. tracked file modified
2. tracked file deleted
3. new untracked file created
4. multiple new files created
5. ignored file created
6. only `.agent-sdlc` runtime files changed
7. no repository changes

Expected result for #3 and #4: implementation changes detected.

Expected result for #7: `no changes were produced`.

## Acceptance Criteria

- New files count as implementation changes.
- Files do not need to be staged.
- Ignored/runtime files do not produce false positives.
- Existing tracked-file behavior remains correct.
- Relevant `agentic-sop` tests pass.

If this defect is already fixed in the installed/current `agentic-sop`, prove it with tests rather than modifying the implementation unnecessarily.

---

# WRAP-005 — Verify Named Plan Identity

## Objective

Prove that named plans cannot accidentally reuse the wrong machine plan or task graph.

## Tasks

Run/inspect execution for:

```bash
sop run docs/plans/PLAN-Wrap-Up.md
```

Verify persisted metadata identifies at minimum:

- source path
- source content fingerprint
- plan identity

Verify SOP can distinguish:

- same plan + same content
- same plan + changed content
- different plan

Do not use modification timestamps as the authoritative freshness mechanism.

## Acceptance Criteria

- Re-running unchanged `PLAN-Wrap-Up.md` resumes the same execution.
- Editing the plan makes the machine representation stale.
- Running another named plan cannot silently reuse this plan's task graph.
- Completed task history is not silently destroyed when reconciliation requires human intervention.

---

# WRAP-006 — Verify Idempotent Resume

## Objective

Prove the normal recovery mechanism is simply rerunning the same command.

## Required Command

```bash
sop run docs/plans/PLAN-Wrap-Up.md
```

## Tasks

Verify repeated invocation does not:

- duplicate tasks
- reset DONE tasks
- recreate already-current plans
- discard failure information
- silently reset BLOCKED tasks
- create duplicate run state

Where execution can safely continue, SOP should resume from the next eligible task.

## Acceptance Criteria

Given:

```text
WRAP-001 DONE
WRAP-002 DONE
WRAP-003 DONE
WRAP-004 DONE
WRAP-005 DONE
WRAP-006 RUNNING
```

rerunning the named plan does not restart WRAP-001.

---

# WRAP-007 — Verify Controller SOP Boundary

## Objective

Confirm `sop-controller` remains a control/visibility plane and does not become another workflow engine.

## Architectural Rule

```text
agentic-sop / SOP = automation + execution authority
sop-controller     = visibility + human control
```

## Verify

- Controller reads authoritative SOP state through `internal/sopclient`.
- Controller commands SOP through the supported CLI/application boundary.
- Browser handlers must not directly implement workflow state transitions.
- Verify operations such as `run`, `resume`, `retry`, `validate`, and `review` delegate to SOP rather than duplicating SOP logic.

## Acceptance Criteria

- No second authoritative workflow state machine exists.
- No browser-side SQLite access exists.
- Controller does not directly mutate SOP task states.
- SOP remains authoritative.

---

# WRAP-008 — Validate Controller UI and Operations

## Objective

Perform final functional verification of the controller.

## Verify

### Project view
- task counts
- progress
- DONE
- READY/RUNNING
- BLOCKED

### Task detail
- dependencies
- attempts
- latest failure
- execution state
- validation state

### Execution activity
- useful structured events
- understandable failure information

### Review
- findings
- severity
- remediation state

### Validation / CI
- build status
- test status
- lint status
- failure reason where applicable

### Handoff
- available handoff information
- failure/degraded state where applicable

### Commands
Verify supported commands reach SOP safely.

### Responsive UI
Verify important views remain usable at phone/tablet widths.

Do not redesign the UI unless an actual usability defect prevents the requirements above.

## Acceptance Criteria

The controller provides enough information to answer:

- What is running?
- What is done?
- What is blocked?
- Why did something fail?
- What can I safely do next?

without requiring direct inspection of SQLite.

---

# WRAP-009 — Security Verification

## Objective

Verify existing local-first security assumptions.

## Verify

Default mode remains loopback-only (`127.0.0.1`).

Network mode must remain explicit.

When network mode is enabled, verify configured authentication/token behavior.

Verify:

- state-changing browser actions use POST
- CSRF protection remains active
- secrets are not rendered
- environment values are not dumped
- arbitrary shell commands cannot be submitted from browser input
- project/path input remains constrained
- SOP command invocation uses known operations

## Acceptance Criteria

Local mode remains safe by default and optional network exposure requires explicit configuration.

---

# WRAP-010 — Deterministic Validation Gate

## Objective

Run the complete controller validation suite.

Run:

```bash
gofmt -l .
go vet ./...
go test ./...
go test -race ./...
go build ./...
```

`gofmt -l .` must produce no Go source files.

Also run any repository-specific validation documented by the Makefile or README where it adds coverage rather than duplicating the commands above.

## Gate

All required deterministic validation must pass.

If it does not pass, return `FAIL` or `NEEDS_HUMAN` as appropriate.

Do not declare completion with failing deterministic validation.

---

# WRAP-011 — SOP Dogfood Test

## Objective

Prove the named-plan workflow works on the real controller repository.

The key test is execution of this plan itself:

```bash
sop run docs/plans/PLAN-Wrap-Up.md
```

Verify that SOP:

1. resolves the explicit plan
2. identifies its source and fingerprint
3. initializes runtime state if required
4. creates/reconciles the machine plan
5. creates/reconciles the task DAG
6. executes dependency-ready work
7. validates implementations
8. performs review
9. performs bounded remediation
10. resumes correctly when rerun
11. does not duplicate tasks
12. does not confuse another plan with this plan
13. respects human approval boundaries
14. does not claim Git/PR/CI operations that did not occur

## Acceptance Criteria

The named-plan workflow works without requiring:

```bash
sop init
sop plan
sop tasks
```

as manual prerequisites.

Normal invocation remains:

```bash
sop run docs/plans/PLAN-Wrap-Up.md
```

---

# WRAP-012 — Final Report and Cleanup

## Objective

Produce definitive evidence that the integration is complete or clearly identify what remains.

## Final Report

Generate a report containing:

```text
Project
PLAN source
PLAN fingerprint / identity

Tasks:
  total
  done
  blocked
  remaining

Validation:
  gofmt
  go vet
  go test
  go test -race
  go build

SOP:
  named-plan resolution
  bootstrap
  task creation
  resume
  plan identity
  new-file detection
  quality gate
  human approval boundary

Controller:
  SOP boundary
  project view
  task view
  activity
  review
  validation
  handoff
  commands
  responsive UI
  security

Remaining warnings
Remaining manual actions

Final result:
  PASS | NEEDS_HUMAN | FAIL
```

## Completion Rules

### PASS

Use only when:

- all wrap-up tasks are complete
- deterministic validation passes
- named-plan dogfood succeeds
- no known high-severity integration defect remains

### NEEDS_HUMAN

Use when automated work is complete but a legitimate human decision or configured approval boundary remains.

Clearly state exactly what human action is required.

### FAIL

Use when required deterministic validation or a required integration capability remains broken.

Do not convert a failure into PASS merely because the implementation appears mostly complete.

---

# Non-Goals

Do not add during this plan:

- new frontend framework
- React
- Next.js
- PostgreSQL
- Kubernetes
- new workflow engine
- MCP server
- Jev integration
- new AI provider architecture
- automatic deployment
- automatic merge
- large UI redesign
- unrelated refactoring

Those belong in separate named plans.

---

# Final Definition of Done

The integration is wrapped up when this command is sufficient:

```bash
sop run docs/plans/PLAN-Wrap-Up.md
```

and eventually produces a truthful final result showing that the named plan was resolved, compiled into machine state and a task DAG, executed through implementation/validation/review/remediation, verified against `sop-controller`, and completed with a deterministic PASS.

After a successful run, another invocation of:

```bash
sop run docs/plans/PLAN-Wrap-Up.md
```

must recognize that the plan is already complete rather than repeating completed work.

The project should then be ready for new functionality to be introduced through separate named plans instead of continuing to modify the foundational controller plan.

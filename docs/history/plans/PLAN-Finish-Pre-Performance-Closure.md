# Finish Pre-Performance Closure

> **Document class:** plan · **Lifecycle:** complete · **Authority:** historical — a record of completed work, not current planning authority.

## Project

sop-controller

## Summary

Finish, verify and record the pre-performance closure that precedes any performance
architecture work for the agentic-sop and sop-controller repositories, while reducing
unnecessary human interaction.

This document defines the **bounded first run** only. The first run (stages FP-001 …
FP-010) inspects repository and lifecycle state, verifies the no-progress and approval
boundaries, reviews the human-decision presentation, investigates attempts accounting,
assesses safe parallel opportunities, runs the targeted lifecycle tests and the full
deterministic gates, preserves existing closure evidence, and publishes a readiness
verdict. It does **not** rerun CLOSE-004 and does **not** execute CLOSE-005; the remaining
closure stages are deferred (see the appendix).

### Rules for evidence tasks

These rules apply to every stage in this plan.

1. Each stage's deliverable is a written **evidence artifact** under
   `docs/reports/finish-pre-performance-closure/`. The artifact must contain the actual
   findings and evidence that the stage requires — not a placeholder.
2. Writing the artifact **is** the stage's legitimate implementation deliverable. An
   evidence-only task requires **no** production-code mutation; a run that produces the
   artifact has made the progress the lifecycle requires.
3. Do not invent findings simply to create a file. Record only what the repository
   evidence supports. Preserve raw command output (command, cwd, revision, exit status,
   output) wherever the stage requires it.
4. If the investigation cannot establish the required result, write that truthfully in the
   artifact (for example "NOT OBSERVED" or "UNAVAILABLE" with the reason) and let the
   appropriate gate return BLOCKED/FAIL. Truthful failure is a valid outcome.
5. Creating or accepting these report artifacts must not introduce a human approval gate.
6. Keep sop-controller production behavior unchanged unless a later stage identifies an
   actual defect that requires a code change; a defect fix is its own change, authorized by
   evidence, not by this plan's evidence tasks.
7. Preserve the now-proven lifecycle behavior: a `NO_PROGRESS` outcome produces `BLOCKED`
   for operator intervention and does **not** create a human approval.

Governing principle: pre-performance closure must not introduce new human gates merely
for verification, sequencing, evidence collection, reporting, or task completion.
Retryable technical conditions are handled automatically within existing bounded policy;
human interaction is reserved for decisions that genuinely require human authority or
judgement.

The original governance document is preserved verbatim in the appendix at the end of this
file; the structured sections above it are a faithful, SOP-compilable restatement of its
first-run boundary.

## Capabilities

### SOP plan and task lifecycle CLI — EXISTS

- Evidence: `sop status`, `sop task`, `sop approvals`, `sop approval`, `sop approve`, `sop decline`, `sop run`, `sop retry`, `sop resume`, `sop reconcile`, `sop plan supersede`, `sop report`, `sop validate`, `sop review`, `sop gate`.

### Recorded active plan and machine plan — EXISTS

- Evidence: `.agent-sdlc/plan.meta.json` records the active plan id, source path and source sha256; `.agent-sdlc/plan.json` holds the machine plan; `sop status` prints the active plan.

### Human approval gate (list / inspect / decide) — EXISTS

- Evidence: `sop approvals`, `sop approval <task-id>`, `sop approve`, `sop decline`; a superseded plan's pending approval is preserved as history in the archive.

### Deterministic validation and review lifecycle — EXISTS

- Evidence: `.agent-sdlc/config.yaml` configures build `go build ./...`, test `go test ./...`, lint `go vet ./...`; the quality gate uses `fail_on: [critical, high]` with `max_fix_cycles: 3`; the review engine is `self`.

### SOP-owned run evidence — EXISTS

- Evidence: `.agent-sdlc/runs/<task-id>/` holds report.json, report.md, validation.json, review.json, metrics.json, classification and trace artifacts written by SOP.

### Declared report-deliverable support — EXISTS

- Evidence: the SOP agent harness treats an exact `docs/reports/…​.md` path declared as a task deliverable as this task's legitimate output, keeps file tools available, and withholds command/git tools until it exists.

### agentic-sop sibling checkout (read-only from this harness) — EXISTS

- Evidence: the sibling working tree is present in the workspace; SOP named-plan work for this repository executes with sop-controller as the working directory.

### Cross-root mutation (this harness mutating the sibling repository) — MISSING

- Owner: operator / SOP process boundary
- Gap: a repository's native harness cannot mutate its sibling repository.
- Resolution: every sibling-repository check in this plan is read-only; any required sibling mutation is reported, never attempted.

## Assumptions

### The two repositories are siblings in one workspace; agentic-sop is read from this checkout without mutation.

- Evidence: filesystem layout and `.agent-sdlc/config.yaml`.
- Consequence: gate and evidence checks against agentic-sop are read-only.

### SOP is the sole lifecycle authority and `.agent-sdlc` is never hand-edited.

- Evidence: SOP lifecycle documentation for activation, supersession and external completion.
- Consequence: every lifecycle change goes through a `sop` command; recorded state is never edited directly.

### The proven baseline CLOSE-001 through CLOSE-003 is historical evidence unless current state disproves it.

- Evidence: the recorded closure reports and this repository's history.
- Consequence: completed closure tasks are not rerun merely to generate fresh activity.

## FP-001 — Inspect Current Repository and Lifecycle State

Capture the exact starting state of both repositories and the SOP lifecycle: branch, HEAD
and dirty tree for sop-controller and agentic-sop; the active plan as recorded in
`.agent-sdlc/plan.meta.json`; the task list; and any pending approvals. Confirm whether the
recorded CLOSE-001 through CLOSE-003 baseline is consistent with current evidence. Write
the findings to the declared deliverable. This is an evidence-only task: the deliverable
file is its legitimate implementation output and no production-code change is required.
Record only findings the repository supports; preserve raw command output; if something
cannot be established, say so truthfully.

### Deliverables
- docs/reports/finish-pre-performance-closure/FP-001-repository-lifecycle-state.md — the captured sop-controller and agentic-sop branch/HEAD/dirty state, the active plan id and source, the task list, pending approvals, and the baseline confirmation.

### Acceptance Criteria
- The deliverable exists at the declared path and contains both repositories' branch/HEAD/dirty state, and sop-controller's `sop status` and `sop approvals`, quoted verbatim.
- The active plan is identified from recorded provenance, not by scanning markdown.
- The CLOSE-001 through CLOSE-003 baseline is confirmed or refuted with evidence.

## FP-002 — Verify No-Progress Does Not Create Approvals

Trace the no-progress paths end to end — agent harness, IMPLEMENT/FIX outcome, failure
classification, NoProgress, disposition, autonomy, lifecycle result, task state, approval
creation — for `IMPLEMENT_NO_PROGRESS` and `FIX_NO_PROGRESS`, and verify they yield BLOCKED
with no approval record, distinguishing operator intervention from human approval. Preserve
the proven `NO_PROGRESS → BLOCKED → no approval` behavior; do not weaken or bypass it.
Write the trace to the declared deliverable. Evidence-only task: the deliverable is the
legitimate output and no production-code change is required. Record only what the evidence
supports; if a path cannot be exercised, mark it NOT OBSERVED with the reason.

### Dependencies
- FP-001

### Deliverables
- docs/reports/finish-pre-performance-closure/FP-002-no-progress-approval-trace.md — the traced no-progress paths, the resulting task state, and the approval behavior for each.

### Acceptance Criteria
- The deliverable exists and documents each path with task=BLOCKED and approval=none, or records a counterexample with evidence; a path not exercised is marked NOT OBSERVED with the reason.
- No budget, retry or stale-iteration limit is increased; no state is mutated.
- The `NO_PROGRESS → BLOCKED → no approval` behavior is preserved and demonstrated.

## FP-003 — Verify Genuine Human Approval Boundaries Remain Intact

Confirm that a genuine `NEEDS_HUMAN` boundary still creates an approval record and a
decision request, and is not suppressed by automation. Where the current run has no live
boundary, use preserved evidence (for example a superseded plan's archived approval) and
say so. Write the findings to the declared deliverable. Evidence-only task: the deliverable
is the legitimate output and no production-code change is required. Do not approve or
decline anything.

### Dependencies
- FP-001

### Deliverables
- docs/reports/finish-pre-performance-closure/FP-003-human-approval-boundaries.md — the evidence that a genuine human boundary produces an approval record, citing its source.

### Acceptance Criteria
- The deliverable exists and shows a genuine boundary producing an approval record and a decision request, citing its source; an unobserved boundary is marked NOT OBSERVED.
- No approval is approved or declined by this task.

## FP-004 — Review Humanized Decision Presentation

Review the current decision presentation and identify the smallest path to the humanized
decision brief (what happened, recommendation, why, options, impact, technical details)
and to honest multi-option presentation. Write the gap analysis to the declared deliverable.
Evidence-only task: the deliverable is the legitimate output and no production-code change
is required. A recommendation is given only when evidence supports one.

### Dependencies
- FP-001

### Deliverables
- docs/reports/finish-pre-performance-closure/FP-004-decision-presentation-review.md — the mapping of the required brief fields to current behavior, the gaps, and the smallest change path.

### Acceptance Criteria
- The deliverable exists; each required brief field is mapped to current behavior and gaps are stated.
- A recommendation is given only when evidence supports one; otherwise "no recommendation" is explicit.
- No source code is mutated.

## FP-005 — Investigate Attempts Accounting

Determine from evidence whether a blocked task reporting `Attempts: 0` is intentional
semantics or an accounting defect; if a defect is proven, propose the smallest tested fix
(not applied without authorization). Write the verdict to the declared deliverable.
Evidence-only task: the deliverable is the legitimate output and no production-code change
is required; propose a fix only if a defect is proven.

### Dependencies
- FP-001

### Deliverables
- docs/reports/finish-pre-performance-closure/FP-005-attempts-accounting.md — the verdict, the supporting evidence, and a proposed fix only if a defect is proven.

### Acceptance Criteria
- The deliverable exists; the attempts semantics are explained from evidence, and zero is not treated as a defect merely because it appears suspicious.
- No change is made unless a defect is proven.

## FP-006 — Assess Safe Parallel Execution Opportunities

From the task dependency graph, determine which runnable tasks are genuinely independent
and safe to run concurrently, checking dependencies, write ownership, shared artifacts and
lifecycle/approval/evidence/repository-state dependencies; do not invent or remove
dependencies. Write the assessment to the declared deliverable. Evidence-only task: the
deliverable is the legitimate output and no production-code change is required.

### Dependencies
- FP-001

### Deliverables
- docs/reports/finish-pre-performance-closure/FP-006-parallel-safety-assessment.md — the independence and safety assessment for each candidate parallel group, with the checked conditions.

### Acceptance Criteria
- The deliverable exists; every parallel candidate is justified against the required independence conditions.
- Where safety cannot be proven, sequential execution is recommended.
- No dependency is invented to serialize work, and none is removed to parallelize it.

## FP-007 — Run Targeted Lifecycle Tests

Run the required lifecycle tests — IMPLEMENT no-progress, FIX no-progress, the genuine
human boundary, and retryable-failure behavior — and record the outcome of each case. This
task must not weaken `NO_PROGRESS → BLOCKED → no approval` or the bounded retryable-failure
behavior. Write the results to the declared deliverable. Evidence-only task: the deliverable
is the legitimate output and no production-code change is required. Mark a case NOT RUN with
its reason rather than fabricating a result.

### Dependencies
- FP-002

### Deliverables
- docs/reports/finish-pre-performance-closure/FP-007-lifecycle-test-results.md — the result of each required lifecycle case.

### Acceptance Criteria
- The deliverable exists; each required case is exercised and its observed outcome recorded, or marked NOT RUN with a reason.
- Bounded automatic behavior for retryable failures is shown unchanged.
- `NO_PROGRESS → BLOCKED → no approval` is shown preserved.

## FP-008 — Run Full Deterministic Gates

Run the deterministic gate for both repositories — `gofmt -l .`, `go vet ./...`,
`go test ./...`, `go test -race ./...`, `go build ./...`, `git diff --check` — capturing
raw command, cwd, revision, exit status and output. Write the raw results to the declared
deliverable. Evidence-only task: the deliverable is the legitimate output and no
production-code change is required. Sibling-repository checks are read-only. A failing gate
is reported truthfully and is not converted into a human gate.

### Dependencies
- FP-001

### Deliverables
- docs/reports/finish-pre-performance-closure/FP-008-deterministic-gates.md — the raw gate output for sop-controller and agentic-sop.

### Acceptance Criteria
- The deliverable exists; each command, cwd, revision, exit status and raw output is recorded; model summaries are not substituted for raw evidence.
- Sibling-repository checks are read-only; no sibling mutation is attempted.
- Any failing gate is reported truthfully and is not converted into a human gate.

## FP-009 — Preserve CLOSE-004 Evidence

Preserve existing CLOSE-004 deterministic-baseline evidence; do not delete, overwrite or
fabricate evidence to obtain a clean run. Write the evidence location and integrity check to
the declared deliverable. Evidence-only task: the deliverable is the legitimate output and
no production-code change is required.

### Dependencies
- FP-001

### Deliverables
- docs/reports/finish-pre-performance-closure/FP-009-close-004-evidence-preservation.md — the CLOSE-004 evidence location and integrity confirmation.

### Acceptance Criteria
- The deliverable exists; any existing CLOSE-004 evidence is referenced and left untouched.
- No evidence is fabricated.

## FP-010 — Publish the Readiness Report

Publish the readiness verdict in the required format, ending with exactly one of
"CLOSE-004 READY FOR GOVERNED RERUN" or "CLOSE-004 NOT READY", and including Automation,
Human gates, Humanized decisions, Safe parallel opportunities, and Remaining blocker or
Blocking defects. Write it to the declared deliverable. Evidence-only task: the deliverable
is the legitimate output and no production-code change is required.

### Dependencies
- FP-002
- FP-003
- FP-004
- FP-005
- FP-006
- FP-007
- FP-008
- FP-009

### Deliverables
- docs/reports/finish-pre-performance-closure/READINESS.md — the readiness verdict.

### Acceptance Criteria
- The deliverable exists, ends with exactly one verdict, and contains each required field.
- Each no-new-gates acceptance item is addressed.
- CLOSE-004 is not rerun and CLOSE-005 is not executed.

## Appendix - Original Document (preserved verbatim)

# Finish Pre-Performance Closure

## Objective

Finish the existing:

`docs/plans/PLAN-Pre-Performance-Closure.md`

Do not replace the existing closure plan unless it is structurally
unusable.

The objective is to establish a trustworthy `agentic-sop` +
`sop-controller` baseline while **reducing unnecessary human
interaction**.

The closure process should run autonomously as far as safely possible.

------------------------------------------------------------------------

## 1. Governing Automation Principle

Pre-Performance Closure must **not introduce new human gates** merely
for verification, sequencing, evidence collection, reporting, or task
completion.

The desired lifecycle is:

``` text
normal successful work
        ↓
continue automatically

retryable technical condition
        ↓
bounded automatic handling

independent runnable work
        ↓
run concurrently when safe

NO_PROGRESS / technical BLOCK
        ↓
stop affected path + report
(no approval request)

genuine human decision
        ↓
NEEDS_HUMAN
        ↓
humanized decision request
        ↓
recommendation + alternatives
```

Human interaction is reserved for decisions that actually require human
authority or judgment.

------------------------------------------------------------------------

## 2. Human Interaction Invariant

The following must NOT create human approval merely because they
occurred:

``` text
deterministic test failure
build failure
validation failure
review finding
missing evidence
NO_PROGRESS
IMPLEMENT_NO_PROGRESS
FIX_NO_PROGRESS
retryable provider/tool failure
performance measurement failure
task transition
plan transition
report generation
successful deterministic verification
```

These should either:

``` text
continue automatically
retry within existing bounded policy
take an existing deterministic repair path
BLOCK truthfully
FAIL truthfully
```

They must not be converted into `NEEDS_HUMAN` simply because automation
cannot proceed.

------------------------------------------------------------------------

## 3. Genuine Human Boundaries

`NEEDS_HUMAN` should be reserved for genuine decisions such as:

``` text
approval explicitly required by policy
irreversible or high-impact action
security-sensitive authorization
ambiguous product/design choice requiring owner judgment
conflicting valid alternatives without deterministic resolution
scope expansion requiring owner authorization
external action requiring human consent
```

Existing legitimate approval semantics must remain intact.

Do not remove safety boundaries merely to increase automation.

------------------------------------------------------------------------

## 4. Humanized Decision Requests

Whenever SOP genuinely requires a human decision, the request should be
understandable without reading raw execution logs.

Do not present only:

``` text
TASK-123 NEEDS_HUMAN
approve / decline
```

Instead present a compact decision brief.

Required conceptual format:

``` text
DECISION NEEDED

What happened:
<plain-English explanation>

Why I need you:
<why SOP cannot safely/deterministically decide>

Recommended:
<recommended option>

Why:
<short reasoning>

Other options:
1. <alternative>
2. <alternative>

Impact:
<what happens for each important choice>

Suggested action:
<plain-English recommendation>

Technical details:
<task ID / evidence / report / failure classification>
```

The recommendation is advisory.

The human remains authoritative.

------------------------------------------------------------------------

## 5. Recommendations Must Be Evidence-Based

When requesting a decision, SOP should provide a recommendation whenever
repository evidence supports one.

Example:

``` text
DECISION NEEDED

CLOSE-006 reached the expected approval boundary.

Recommended:
Approve and continue.

Why:
All deterministic tests passed, the repository is clean,
and the only remaining boundary is the intentional human
approval required by this workflow.

If approved:
CLOSE-006 completes and CLOSE-007 becomes runnable.

If declined:
The plan stops at CLOSE-006 without changing downstream work.

Suggested action:
Approve.

Technical details:
CLOSE-006 / NEEDS_HUMAN
```

Do not generate a recommendation when evidence does not support one.

In that situation say:

``` text
Recommendation:
No recommendation — the available evidence does not distinguish
the alternatives reliably.
```

------------------------------------------------------------------------

## 6. Multiple Safe Options

When more than one valid action exists, present the choices explicitly.

Example:

``` text
DECISION NEEDED

Two safe paths are available.

Recommended:
Option A — continue with the verified implementation.

Why:
It satisfies the acceptance criteria and requires no additional
repository mutation.

Options:

A. Continue with verified implementation
   Impact: fastest path; preserves current architecture.

B. Rework implementation
   Impact: additional engineering work; no currently demonstrated
   correctness benefit.

C. Stop this task
   Impact: downstream tasks remain gated.

Suggested action:
A
```

Do not reduce a multi-option decision to arbitrary `approve/decline` if
the actual decision has more meaningful alternatives.

------------------------------------------------------------------------

## 7. Parallel Execution Principle

When multiple tasks are simultaneously runnable and independent, SOP
should be able to execute them concurrently **if doing so is
deterministic and repository-safe**.

Parallelism is an execution optimization, not a relaxation of
governance.

Before parallel execution, verify:

``` text
dependencies satisfied
no conflicting write ownership
no shared mutable artifact conflict
no lifecycle dependency
no approval dependency
no evidence-order dependency
no repository-state dependency
```

If those conditions cannot be proven:

``` text
run sequentially
```

------------------------------------------------------------------------

## 8. Parallel Execution Modes

When several runnable tasks exist, support these conceptual choices:

``` text
AUTO
PARALLEL
SEQUENTIAL
```

### AUTO

Recommended default.

The harness determines which tasks are safely independent and
parallelizes only those.

### PARALLEL

Explicitly request maximum safe concurrency within existing
orchestration limits.

This does not override dependency, scope, ownership, or lifecycle rules.

### SEQUENTIAL

Run one runnable task at a time.

Useful for diagnosis or particularly sensitive work.

The execution mode must never bypass the task DAG.

------------------------------------------------------------------------

## 9. Do Not Ask Humans About Obvious Parallelism

If the harness can deterministically prove that two tasks are
independent and safe to run concurrently, it should not require human
approval merely to parallelize them.

For example:

``` text
Task A ──┐
         ├── independent
Task B ──┘
```

with non-overlapping ownership may run concurrently automatically.

Human input is appropriate only if there is a real tradeoff the harness
cannot resolve.

------------------------------------------------------------------------

## 10. Partial Progress Under Parallel Execution

If independent tasks are running concurrently and one path blocks:

``` text
Task A → PASS
Task B → BLOCKED
Task C → PASS
```

do not discard A and C merely because B blocked.

Record truthful results:

``` text
Task A  LOCAL_DONE
Task B  BLOCKED
Task C  LOCAL_DONE
```

Then determine whether downstream dependencies permit further work.

A blocked branch should not globally halt unrelated runnable work.

------------------------------------------------------------------------

## 11. Preserve Existing Proven Baseline

Treat the following as historical evidence unless current repository
state disproves it:

``` text
CLOSE-001  LOCAL_DONE
CLOSE-002  LOCAL_DONE
CLOSE-003  LOCAL_DONE
```

Do not rerun completed closure tasks merely to generate fresh activity.

Capture current state:

``` bash
git status --short --branch
git rev-parse HEAD
git log -10 --oneline --decorate

sop status
sop approvals
sop task CLOSE-004
```

------------------------------------------------------------------------

## 12. CLOSE-004 --- Resolve Current Lifecycle Boundary

CLOSE-004 remains the first unresolved closure boundary.

Verify the lifecycle:

``` text
IMPLEMENT_NO_PROGRESS
FIX_NO_PROGRESS
        ↓
NO_PROGRESS
        ↓
BLOCKED
```

It must not automatically become:

``` text
NEEDS_HUMAN
```

unless an independent genuine human decision exists.

Required:

``` text
NO_PROGRESS
    ↓
BLOCKED
    ↓
diagnostic/operator intervention

NOT

NO_PROGRESS
    ↓
approval record
```

------------------------------------------------------------------------

## 13. Trace NO_PROGRESS End-to-End

Inspect:

``` text
agent harness
    ↓
IMPLEMENT/FIX outcome
    ↓
failure classification
    ↓
NoProgress
    ↓
disposition
    ↓
autonomy
    ↓
lifecycle result
    ↓
task state
    ↓
approval creation
```

Verify both:

``` text
IMPLEMENT_NO_PROGRESS
FIX_NO_PROGRESS
```

Distinguish:

``` text
operator intervention required
```

from:

``` text
human approval required
```

------------------------------------------------------------------------

## 14. Required Lifecycle Tests

### IMPLEMENT no progress

Expected:

``` text
IMPLEMENT_NO_PROGRESS
task = BLOCKED
approval = none
AUTO_CONTINUE = false
```

### FIX no progress

Expected:

``` text
FIX_NO_PROGRESS
task = BLOCKED
approval = none
AUTO_CONTINUE = false
```

### Genuine human boundary

Expected:

``` text
NEEDS_HUMAN
approval record
humanized decision request
recommendation when supported
```

### Retryable failure

Existing bounded automatic behavior remains unchanged.

------------------------------------------------------------------------

## 15. Attempts Accounting

Investigate the previous observation:

``` text
CLOSE-004
Status: BLOCKED
Attempts: 0
```

Determine whether this is intentional semantics or a
lifecycle/accounting defect.

Do not modify it merely because zero appears suspicious.

If incorrect, make the smallest tested fix.

------------------------------------------------------------------------

## 16. Preserve CLOSE-004 Evidence

Preserve:

`docs/reports/pre-performance-closure/CLOSE-004-controller-deterministic-baseline.md`

if present.

Do not delete, overwrite, or fabricate evidence merely to obtain a clean
run.

Exact SOP-owned command evidence remains authoritative.

------------------------------------------------------------------------

## 17. Cross-Repository Security

Preserve:

``` text
agentic-sop
    read/write

sop-controller
    read-only
```

Controller should permit:

``` text
read/search/list
go version
gofmt -l .
go vet ./...
go test ./...
go test -race ./...
go build ./...
```

while denying repository mutation, path escape, symlink escape, and
unauthorized state access.

------------------------------------------------------------------------

## 18. Raw Evidence

For deterministic controller verification capture:

``` text
command
cwd
revision
toolchain where applicable
exit status
raw output
```

Do not substitute model summaries for required raw evidence.

------------------------------------------------------------------------

## 19. Deterministic Repository Gates

### agentic-sop

``` bash
gofmt -l .
go vet ./...
go test ./...
go test -race ./...
go build ./...
git diff --check
```

### sop-controller

``` bash
gofmt -l .
go vet ./...
go test ./...
go test -race ./...
go build ./...
git diff --check
```

All must be green before claiming deterministic baseline success.

------------------------------------------------------------------------

## 20. CLOSE-004 Governed Rerun

After lifecycle correctness and deterministic gates are proven, rerun
CLOSE-004 through the narrowest sanctioned execution path.

Do not:

``` text
outer retry loop
force retry
increase budgets to hide stalls
loosen validation
manually edit state
manufacture approval
manufacture mutation
```

If CLOSE-004 reaches genuine NO_PROGRESS:

``` text
BLOCK
```

Do not ask for approval merely because it blocked.

------------------------------------------------------------------------

## 21. Remaining Closure DAG

After CLOSE-004:

``` text
CLOSE-005 — Finish/verify current controller work
CLOSE-006 — Named-plan dogfood / human-decision verification
CLOSE-007 — Resume/idempotency
CLOSE-008 — Performance telemetry inventory
CLOSE-009 — Performance baseline
CLOSE-010 — Performance report
CLOSE-011 — Final readiness
```

Use the actual plan DAG as authority.

------------------------------------------------------------------------

## 22. Parallelize Remaining Closure Work Where Safe

After CLOSE-004 and other dependencies are satisfied, calculate runnable
tasks from the DAG.

If tasks are independent, execute them concurrently.

Conceptually:

``` text
              ┌── CLOSE-X ──┐
dependency ───┤             ├── downstream
              └── CLOSE-Y ──┘
```

Do not invent dependencies merely to serialize work.

Do not remove real dependencies merely to parallelize work.

------------------------------------------------------------------------

## 23. CLOSE-005 --- Controller Verification

Finish or verify the current controller work required by closure.

Do not expand into controller redesign.

If deterministic evidence is sufficient, continue automatically.

Do not ask for human approval merely to mark deterministic verification
complete.

------------------------------------------------------------------------

## 24. CLOSE-006 --- Human Decision Dogfood

This task verifies the genuine human-decision path.

It must not manufacture a human decision merely because the task is
named as a human-decision test.

If the existing governed workflow naturally reaches a legitimate
approval boundary, present it using the humanized decision format.

Example:

``` text
DECISION NEEDED

What happened:
The named-plan run completed all automated verification and has
reached its intentional owner-approval boundary.

Recommended:
Approve and continue.

Why:
All automated gates are green and no unresolved correctness
finding remains.

Options:
A. Approve and continue
B. Decline and stop the plan
C. Request additional verification

Suggested action:
A

Technical details:
CLOSE-006 / NEEDS_HUMAN
```

The recommendation must be based on actual evidence.

------------------------------------------------------------------------

## 25. CLOSE-007 --- Resume / Idempotency

Prove:

``` text
resume after genuine human decision
repeated resume invocation
completed work not executed again
evidence not duplicated
approvals not duplicated
repository mutations not duplicated
```

Run automatically unless a genuine human boundary appears.

------------------------------------------------------------------------

## 26. CLOSE-008 --- Performance Telemetry Inventory

Inventory existing measurements as:

``` text
AVAILABLE
PARTIAL
UNAVAILABLE
```

Do not create human approval gates for missing telemetry.

Missing telemetry becomes an observation or later backlog item unless it
prevents trustworthy measurement.

------------------------------------------------------------------------

## 27. CLOSE-009 --- Performance Baseline

Use deterministic workloads A-D.

Run:

``` text
3 independent repetitions per workload
```

Use independent disposable copies.

Where independent repetitions can safely execute concurrently, allow
bounded parallel execution.

Do not parallelize repetitions if doing so would contaminate timing
measurements or share resources in a way that invalidates the baseline.

Performance measurement integrity takes priority over speed.

Record all runs, including failures.

------------------------------------------------------------------------

## 28. CLOSE-010 --- Performance Report

Produce:

`docs/reports/PERFORMANCE-BASELINE.md`

Separate:

``` text
MEASURED
OBSERVED
INFERRED
RECOMMENDED
```

Report generation should proceed automatically.

No human approval is required merely to create the report.

------------------------------------------------------------------------

## 29. CLOSE-011 --- Final Readiness Gate

Return exactly:

``` text
PASS
NEEDS_HUMAN
FAIL
```

### PASS

All required correctness, lifecycle, controller, resume, telemetry, and
performance evidence is trustworthy.

### NEEDS_HUMAN

Use only for a genuine unresolved human decision.

When returned, include:

``` text
plain-English explanation
recommended choice
reason for recommendation
alternatives
impact of each choice
technical evidence
```

### FAIL

Use for blocking correctness/reliability defects.

Do not convert FAIL into NEEDS_HUMAN merely to obtain a human override.

------------------------------------------------------------------------

## 30. Final Deterministic Gate

Before PASS, both repositories must satisfy:

``` bash
gofmt -l .
go vet ./...
go test ./...
go test -race ./...
go build ./...
git diff --check
```

------------------------------------------------------------------------

## 31. No-New-Gates Acceptance Criterion

Pre-Performance Closure itself is successful only if it demonstrates:

``` text
[ ] NO_PROGRESS does not create approval
[ ] deterministic failures do not create approval
[ ] missing evidence does not create approval
[ ] successful task transitions do not create approval
[ ] independent runnable work can continue
[ ] safe parallelism does not require approval
[ ] genuine human boundaries still create NEEDS_HUMAN
[ ] human decision requests are understandable
[ ] recommendations are evidence-based
[ ] multiple meaningful options are presented when available
```

------------------------------------------------------------------------

## 32. Expected Human Experience

The desired operating experience is:

``` text
Start SOP
   ↓
SOP runs
   ↓
independent work parallelizes where safe
   ↓
routine technical issues handled automatically
   ↓
technical blocker?
   ├── yes → report BLOCKED
   │         unrelated work may continue
   │
   └── no
        ↓
genuine decision?
   ├── no → continue
   │
   └── yes
        ↓
"Here's what happened.
 I recommend A because X.
 B and C are also available.
 Here's the impact.
 What do you want to do?"
```

The user should not need to understand internal SOP lifecycle
terminology to make the decision.

------------------------------------------------------------------------

## 33. Immediate Execution Boundary

For the first run:

1.  Inspect current repository/lifecycle state.
2.  Verify NO_PROGRESS approval leakage.
3.  Verify genuine human approvals remain intact.
4.  Review current decision presentation and identify the smallest path
    for humanized decision briefs.
5.  Investigate `Attempts: 0`.
6.  Determine which later closure tasks can safely execute concurrently
    from the existing DAG.
7.  Run targeted lifecycle tests.
8.  Run full deterministic `agentic-sop` gates.
9.  Preserve existing CLOSE-004 evidence.
10. Report readiness.

Do not rerun CLOSE-004 yet.

Do not execute CLOSE-005 yet.

Do not globally enable Phase 9 multi-agent execution.

Do not implement Phase 10 Dynamic Worker Action Space.

End with:

``` text
CLOSE-004 READY FOR GOVERNED RERUN

Automation:
Human gates:
Humanized decisions:
Safe parallel opportunities:
Remaining blocker:
```

or:

``` text
CLOSE-004 NOT READY

Automation:
Human gates:
Humanized decisions:
Safe parallel opportunities:
Blocking defects:
```

STOP.

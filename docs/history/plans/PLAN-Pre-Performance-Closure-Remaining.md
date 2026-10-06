# Pre-Performance Closure — Remaining Work (CLOSE-005 … CLOSE-011)

> **Document class:** plan · **Lifecycle:** complete · **Authority:** historical — a record of completed work, not current planning authority.

## Project

sop-controller

## Summary

Materialize **only** the remaining documented closure work from the original Pre-Performance
Closure documentation, beginning with `CLOSE-005`. The authoritative source is
`docs/history/plans/PLAN-Finish-Pre-Performance-Closure.md` §21 "Remaining Closure DAG" and its
per-stage requirement sections (§23–§29); the final gate and acceptance criteria are §30 and
§31.

`CLOSE-001`…`CLOSE-004` are already complete. They are **not** runnable tasks here and must not
be rerun: their results are prerequisite/historical evidence only. In particular, `CLOSE-004`
must not be rerun and `CRR-001` must not be reopened.

Revision semantics (kept distinct, never conflated):

- `CLOSE-004` **execution baseline** — `sop-controller`
  `2710ce2b5209c50021507302db0e61a4b28bc9c3`.
- `CLOSE-004` **evidence commit / current `sop-controller` state** —
  `90626f302868f6f0a7d7b6f644a62d6ac9d21159`.
- Current `agentic-sop` **harness** revision —
  `bce2d6224b5847fdbd77df409d57961c037801bc` (includes the mutation-observation repair).
- The `CLOSE-004` report executed at `2710ce2`; it must not be rewritten to claim `90626f3`.
- Historical readiness and `CLOSE-004` evidence may be referenced where appropriate but must
  stay clearly identified as historical/prerequisite. **Every CLOSE task produces its own
  evidence**; earlier evidence never substitutes for a later task's deliverable.

Dependencies encode §21's ordered list as a chain. §21 and §22 name no additional parallel
edges, so `CLOSE-005` is the **only initially runnable task** and no parallel work is assumed.

## Capabilities

### Deterministic validation commands — EXISTS

- Evidence: `.agent-sdlc/config.yaml` declares build `go build ./...`, test `go test ./...`, and lint `go vet ./...` for this project.

### SOP lifecycle status, approvals, and run evidence — EXISTS

- Evidence: `sop status`, `sop approvals`, `sop task`, `sop report`, `sop validate`, `sop review`, `sop gate`; `.agent-sdlc/runs/<task-id>/` holds SOP-written `metrics.json`, `validation.json`, `review.json`, `report.json`, and per-run state.

### Authorized read-only external root and read-only sibling — EXISTS

- Evidence: `SOP_WORKSPACE_ROOTS` declares `agentic-sop` as a `mode: "read"` root; `run_command` selects it by canonical `cwd`, and a read-only root refuses writes and repository-mutating commands.

### Declared report deliverable support — EXISTS

- Evidence: a `docs/reports/**.md` path declared as a task deliverable is created with file tools, and command and git tools are withheld until it exists, so a written report is legitimate task progress.

## Assumptions

### The remaining closure work is the §21 list, nothing more

- Evidence: `docs/history/plans/PLAN-Finish-Pre-Performance-Closure.md` §21 names exactly `CLOSE-005`…`CLOSE-011`, and §33 confines the earlier first run to `FP-001`…`FP-010`.
- Consequence: no new closure stage is inferred, and `FP-001`…`FP-010`, `CRR-001`, and `CLOSE-004` are not recreated.

### Evidence-only deliverables require no production-code change

- Evidence: §24–§29 define verification, inventory, measurement, and reporting outcomes rather than code changes.
- Consequence: writing each task's report is its legitimate progress; a real defect discovered by a task is its own separately authorised change, not this plan's default.

## CLOSE-005 — Controller Verification

Finish or verify the current controller work required by closure (§23). Verify the controller
repository's deterministic correctness at the current revision and record the evidence. Do not
expand into controller redesign.

Prerequisites (record, do not conflate): the `CLOSE-004` execution baseline is `sop-controller`
`2710ce2b5209c50021507302db0e61a4b28bc9c3`; the current controller state is
`90626f302868f6f0a7d7b6f644a62d6ac9d21159` (the `CLOSE-004` evidence commit). Both are
historical/prerequisite; this task produces its own fresh evidence at the revision it observes,
and must not rerun `CLOSE-004` or reopen `CRR-001`.

### Deliverables

- docs/reports/pre-performance-closure/CLOSE-005-controller-verification.md

### Acceptance Criteria

- The deliverable exists at `docs/reports/pre-performance-closure/CLOSE-005-controller-verification.md`.
- The report records the observed `sop-controller` branch, HEAD, and working-tree status, and that HEAD equals origin/main.
- The report records the fresh outcome of the controller deterministic gates, with the command, cwd, revision, exit status, and raw output for `gofmt -l .`, `go vet ./...`, `go test -count=1 ./...`, `go test -race -count=1 ./...`, `go build ./...`, and `git diff --check`.
- The report records the observed `agentic-sop` revision and working-tree status without modifying that repository.
- The report distinguishes the `CLOSE-004` execution baseline (`2710ce2`) from the current controller state (`90626f3`) and labels the historical evidence as prerequisite.
- The report states the controller-verification verdict using only an evidence-backed `PASS`, `FAIL`, `NOT PROVEN`, or `UNAVAILABLE`, and no production-code change was required.
- The report does not rerun `CLOSE-004`, does not reopen `CRR-001`, and does not ask for human approval merely to mark deterministic verification complete.
- The report ends with exactly one verdict line: `CLOSE-005 CONTROLLER VERIFICATION PASS` or `CLOSE-005 CONTROLLER VERIFICATION FAIL`.

## CLOSE-006 — Human Decision Dogfood

Verify the genuine human-decision path (§24). Exercise the governed workflow's human-decision
presentation and confirm that a human decision boundary is raised only when the workflow
naturally reaches one. Do not manufacture a human decision merely because the task is named as
a human-decision test.

### Dependencies

- CLOSE-005

### Deliverables

- docs/reports/pre-performance-closure/CLOSE-006-human-decision-dogfood.md

### Acceptance Criteria

- The deliverable exists at `docs/reports/pre-performance-closure/CLOSE-006-human-decision-dogfood.md`.
- The report records the human-decision presentation surface actually observed (for example, the `sop approvals` / `sop approval` / `sop approve` / `sop decline` path) with the commands and observed output.
- The report states plainly whether a genuine approval boundary occurred; if none occurred, it records that no human decision was manufactured and why.
- Where a decision was presented, the report captures the humanized shape required by §24 (what happened, recommendation, why, options, suggested action, technical detail) and bases the recommendation on actual evidence.
- The report does not synthesize an approval or an operator decision, and does not create a human gate merely to satisfy this task.
- The report ends with exactly one verdict line: `CLOSE-006 HUMAN DECISION DOGFOOD PASS` or `CLOSE-006 HUMAN DECISION DOGFOOD FAIL`.

## CLOSE-007 — Resume / Idempotency

Prove resume and idempotency (§25). Run automatically unless a genuine human boundary appears.

Evidence classification (truthful classification, not a weakened gate):

- A §25 property that can be exercised and fails is recorded `FAIL`.
- A §25 property that can be exercised and passes is recorded `PASS`.
- A §25 property that cannot be exercised from within this task's harness because the required SOP lifecycle operation is unavailable is recorded `UNAVAILABLE` or `NOT PROVEN`, together with the exact reason and the missing evidence/mechanism.
- Unavailable evidence must not be converted into `PASS`.
- Unavailable evidence must not be converted into `FAIL` unless there is actual contradictory evidence.
- The report's overall conclusion distinguishes verified properties, unverified/unavailable properties, and actual failures.

### Dependencies

- CLOSE-006

### Deliverables

- docs/reports/pre-performance-closure/CLOSE-007-resume-idempotency.md

### Acceptance Criteria

- The deliverable exists at `docs/reports/pre-performance-closure/CLOSE-007-resume-idempotency.md`.
- The report records evidence for each §25 property: resume after a genuine human decision; repeated resume invocation; completed work not executed again; evidence not duplicated; approvals not duplicated; repository mutations not duplicated.
- Where a property cannot be established from available evidence, the report records it truthfully as `NOT PROVEN` or `UNAVAILABLE` with the reason rather than asserting it.
- The report records the SOP lifecycle mechanism observed (`sop resume`, `sop status`, `sop task`, and the persisted run/task state) with the commands and raw output it relied on.
- Each §25 property is classified `PASS`, `FAIL`, `UNAVAILABLE`, or `NOT PROVEN`, and every `UNAVAILABLE`/`NOT PROVEN` property states the exact reason and the missing evidence/mechanism (for example, that the required SOP lifecycle operation is unavailable from within the task harness).
- A property is recorded `FAIL` only when it was actually exercised and failed, or when contradictory evidence exists; the absence of an executable lifecycle mechanism is never itself recorded as a failure.
- The report's overall conclusion separates verified properties, unverified/unavailable properties, and actual failures, and names the unverified properties in its rationale.
- The report does not force task state, does not hand-edit `.agent-sdlc`, and does not modify `state.db`.
- The report ends with exactly one verdict line: `CLOSE-007 RESUME IDEMPOTENCY PASS` (no actual failure was observed; every property is either verified or truthfully recorded `UNAVAILABLE`/`NOT PROVEN`) or `CLOSE-007 RESUME IDEMPOTENCY FAIL` (an actual resume/idempotency defect was observed).

## CLOSE-008 — Performance Telemetry Inventory

Inventory existing measurements as `AVAILABLE`, `PARTIAL`, or `UNAVAILABLE` (§26). Do not create
human approval gates for missing telemetry; missing telemetry becomes an observation or a later
backlog item unless it prevents trustworthy measurement.

### Dependencies

- CLOSE-007

### Deliverables

- docs/reports/pre-performance-closure/CLOSE-008-performance-telemetry-inventory.md

### Acceptance Criteria

- The deliverable exists at `docs/reports/pre-performance-closure/CLOSE-008-performance-telemetry-inventory.md`.
- The report inventories the telemetry sources actually present (for example SOP run artifacts such as `metrics.json`, `validation.json`, `review.json`, and the run performance summary), marking each `AVAILABLE`, `PARTIAL`, or `UNAVAILABLE` with the observed path and content.
- The report records each gap as an observation or backlog item, and states whether any gap prevents trustworthy measurement.
- The report creates no human approval gate for missing telemetry and does not synthesize one.
- The report does not implement a telemetry subsystem and does not implement the deferred harness backlog items.
- The report ends with exactly one verdict line: `CLOSE-008 TELEMETRY INVENTORY PASS` or `CLOSE-008 TELEMETRY INVENTORY FAIL`.

## CLOSE-009 — Performance Baseline

Establish a deterministic performance baseline using deterministic workloads A–D, three
independent repetitions per workload, on independent disposable copies, recording all runs
including failures (§27). Performance measurement integrity takes priority over speed; do not
parallelize repetitions if doing so would contaminate timing or share resources in a way that
invalidates the baseline.

Operator-supplied measurement evidence (permitted).

- If this task's harness cannot create the independent disposable copies or run the measurement commands the contract requires, the operator may collect the measurement matrix outside the harness and record it as revision-bound raw evidence under `docs/reports/pre-performance-closure/`.
- When that evidence exists, this task must consume it: evaluate the recorded runs against this contract, cite the raw records it relies on, and derive the baseline and the verdict from them.
- Consuming operator evidence is not itself completion. The task must still apply the contract's criteria — three independent repetitions per workload with an observed measurement, each in an independent disposable copy, sequential — and report its verdict truthfully. Do not stamp `PASS` merely because evidence exists, and do not record `NOT PROVEN` for a run the evidence shows was produced.
- Measurements the evidence shows were produced are reported with their observed values; measurements it does not cover remain `NOT PROVEN`/`UNAVAILABLE` with the reason.

### Dependencies

- CLOSE-008

### Deliverables

- docs/reports/pre-performance-closure/CLOSE-009-performance-baseline-runs.md

### Acceptance Criteria

- The deliverable exists at `docs/reports/pre-performance-closure/CLOSE-009-performance-baseline-runs.md`.
- The report names the deterministic workloads A–D it used and makes each reproducible (the exact command or harness that produced it).
- The report records three independent repetitions per workload, including every failure, with the observed measurement and the environment it ran in.
- The report records whether repetitions were executed sequentially or in bounded parallel, and why that choice preserved measurement integrity.
- Measurements that could not be produced are recorded truthfully as `NOT PROVEN` or `UNAVAILABLE` with the reason, never substituted with an estimate presented as measured.
- The report does not implement a performance subsystem or change production behavior.
- The report identifies the operator-supplied evidence it consumed (the evidence path, the source revision, and the exported-tree archive hash) and cites the individual raw records it relies on.
- The report reports the observed exit status and duration for each of the 12 contract runs and gives per-workload summary statistics, rather than recording those runs as `NOT PROVEN`.
- The report ends with exactly one verdict line: `CLOSE-009 PERFORMANCE BASELINE PASS` or `CLOSE-009 PERFORMANCE BASELINE FAIL`.

## CLOSE-010 — Performance Report

Produce `docs/reports/PERFORMANCE-BASELINE.md` (§28), separating `MEASURED`, `OBSERVED`,
`INFERRED`, and `RECOMMENDED`. Report generation proceeds automatically; no human approval is
required merely to create the report.

### Dependencies

- CLOSE-009

### Deliverables

- docs/reports/PERFORMANCE-BASELINE.md

### Acceptance Criteria

- The deliverable exists at `docs/reports/PERFORMANCE-BASELINE.md`.
- The report separates its content into `MEASURED`, `OBSERVED`, `INFERRED`, and `RECOMMENDED` sections, and each statement is placed in the section matching how it was established.
- Every measured figure cites the `CLOSE-009` run evidence it came from; nothing measured is restated as inferred or vice versa.
- The report creates no human approval gate and no approval was required to create it.
- The report ends with exactly one verdict line: `CLOSE-010 PERFORMANCE REPORT PASS` or `CLOSE-010 PERFORMANCE REPORT FAIL`.

## CLOSE-011 — Final Readiness Gate

Return the final readiness verdict as exactly `PASS`, `NEEDS_HUMAN`, or `FAIL` (§29), applying the
final deterministic gate (§30) and the no-new-gates acceptance criteria (§31).

### Dependencies

- CLOSE-010

### Deliverables

- docs/reports/pre-performance-closure/CLOSE-011-final-readiness.md

### Acceptance Criteria

- The deliverable exists at `docs/reports/pre-performance-closure/CLOSE-011-final-readiness.md`.
- The report records the fresh deterministic gate result in both repositories (`gofmt -l .`, `go vet ./...`, `go test ./...`, `go test -race ./...`, `go build ./...`, `git diff --check`) with the observed revision and exit status, at the current revisions.
- The report addresses each §31 no-new-gates item explicitly: `NO_PROGRESS` does not create approval; deterministic failures do not create approval; missing evidence does not create approval; successful task transitions do not create approval; independent runnable work can continue; safe parallelism does not require approval; genuine human boundaries still create `NEEDS_HUMAN`; human decision requests are understandable; recommendations are evidence-based; multiple meaningful options are presented when available.
- The report cites the evidence each conclusion rests on, and does not use `NEEDS_HUMAN` for anything other than a genuine unresolved human decision, nor convert a `FAIL` into `NEEDS_HUMAN`.
- The report ends with exactly one verdict line: `CLOSE-011 FINAL READINESS PASS`, `CLOSE-011 FINAL READINESS NEEDS_HUMAN`, or `CLOSE-011 FINAL READINESS FAIL`.

## Plan-wide constraints

- Execute through normal SOP lifecycle behaviour only: no outer retry loop, no forced retry, no increased retry/fix-loop/continuation/budget limits, no manual `.agent-sdlc` edits, no `state.db` modification, no forced task state, and no synthesized or bypassed approval.
- `agentic-sop` is read-only for every task; only `sop-controller` deliverables are written, and each task's only repository write is its declared deliverable.
- Do not rerun `CLOSE-004`, do not reopen `CRR-001`, do not rerun `FP-001`…`FP-010`, and do not recreate the deferred readiness plan.
- The deferred harness backlog items "Deliverable Conformance / Evidence Progress Detection" and "Run-scoped Command Evidence" remain out of scope and must not be implemented.
- The mutation-observation repair already present at `agentic-sop` `bce2d62` is part of the current harness baseline and must not be reimplemented.
- A genuine `NEEDS_HUMAN` boundary, an unexpected repository mutation, a required change to the established closure contract, a required budget expansion, or a required backlog implementation stops the run for an operator decision.

# CLOSE-004 Rerun Readiness

> **Document class:** plan · **Lifecycle:** complete · **Authority:** historical — a record of completed work, not current planning authority.
>
> SOP-compilable restatement of the operator's readiness prompt at
> [`CLOSE-004-RERUN-READINESS-PROMPT.md`](CLOSE-004-RERUN-READINESS-PROMPT.md). The plan was
> archived `COMPLETE`; its task record, run evidence, and the accepted readiness report at
> [`../../reports/finish-pre-performance-closure/CLOSE-004-RERUN-READINESS.md`](../../reports/finish-pre-performance-closure/CLOSE-004-RERUN-READINESS.md)
> are preserved.

## Project

sop-controller

## Summary

Prepare CLOSE-004 for a governed rerun **without actually rerunning CLOSE-004**.
Resolve only five readiness questions: authorized read-only access to the sibling
`agentic-sop`; fresh deterministic gates for both repositories; provenance of the
missing historical CLOSE-004 evidence; deterministic evidence for
`FIX_NO_PROGRESS`; and classification of humanized decisions and evidence-only
tasks as blockers versus backlog.

This is an evidence-producing task, not a mutation task. Its contract is explicit:
**running the gate commands is evidence gathering, and creating or updating the
readiness report is the legitimate implementation deliverable.** The stage is not
complete merely because commands were executed: it is complete only once the report
exists and records the observed results. No production-code mutation is required
unless the investigation discovers an actual defect that needs a narrowly scoped
fix, and a defect fix is its own authorised change, not this task's deliverable.
Preserve the proven invariant
`IMPLEMENT_NO_PROGRESS → NO_PROGRESS → BLOCKED → RequiresHuman=false → no approval → AUTO_CONTINUE=false`.
Do not rerun the FP plan or CLOSE-004. Do not enable multi-agent execution
(Phase 9 orchestration stays opt-in / default-off). Do not create a human approval
gate merely to author or accept the report.

## Capabilities

### Deterministic validation commands — EXISTS

- Evidence: `.agent-sdlc/config.yaml` declares build `go build ./...`, test `go test ./...`, and lint `go vet ./...` for this project.

### SOP lifecycle status and approvals — EXISTS

- Evidence: `sop status`, `sop approvals`, `sop task`, and `sop report` answer for this project, and `.agent-sdlc/runs/` holds per-task run evidence.

### Authorized read-only external repository root — EXISTS

- Evidence: `SOP_WORKSPACE_ROOTS` declares task-authorized roots as JSON `{"path","mode"}` entries; a command selects an authorized root by its canonical `cwd`, a `read` root refuses file writes and repository-mutating commands, and a root overlapping the primary root is ignored.

### Declared report deliverable support — EXISTS

- Evidence: a `docs/reports/**.md` path declared as a task deliverable is created with file tools, and command and git tools are withheld until it exists, so a written report is legitimate task progress.

## Assumptions

### The sibling agentic-sop checkout is the SOP engine under test

- Evidence: the sibling checkout contains `cmd/sop`, `internal/planner`, `internal/ollamaagent`, and `internal/toolharness`, and the installed `sop` binary resolves on `PATH`.
- Consequence: readiness inspects the sibling read-only and never mutates it.

### A missing historical artifact need not block readiness

- Evidence: CLOSE-004 is a governed rerun that would itself generate fresh authoritative deterministic evidence.
- Consequence: the report may still return READY when provenance is truthfully determined, provided the rerun is explicitly responsible for the fresh evidence.

## Evidence review

- Existing closure evidence lives under `docs/reports/finish-pre-performance-closure/` (FP-001 … FP-009 and `READINESS.md`); the FP plan itself is archived at `docs/history/plans/PLAN-Finish-Pre-Performance-Closure.md`.
- Read FP-002, FP-004, FP-006, FP-008, FP-009, and `READINESS.md` before writing the report, and reuse their conclusions where still current rather than re-deriving them.

## CRR-001 — Prepare CLOSE-004 for Governed Rerun

This readiness task answers five questions about CLOSE-004 and produces one report.
It is evidence-only: writing the report is the deliverable, and no production-code
mutation is expected or required. Prefer `discover → verify → record` over
implementation, and record only what repository evidence supports.

1. Authorized sibling execution. Confirm the existing authorized-root mechanism
   (`SOP_WORKSPACE_ROOTS`, a JSON array of `{"path":"<abs>","mode":"read"|"read-write"}`)
   grants this task read-only access to the sibling `agentic-sop` repository, and that
   it is configured for this project. Prefer configuration over production code: if the
   mechanism works and only configuration was missing, record that and add no production
   code. Do not copy the sibling, hardcode machine-specific paths into production code,
   weaken canonical-path checks, grant unnecessary write access, disable authorization,
   or bypass the harness.

2. Fresh deterministic gates. For each repository — `sop-controller` (primary) and
   `agentic-sop` (authorized read-only sibling) — run and capture the command, cwd,
   revision, exit status, and raw output for: `gofmt -l .`, `go vet ./...`,
   `go test -count=1 ./...`, `go test -race -count=1 ./...`, `go build ./...`, and
   `git diff --check`. Select the sibling with the `cwd` of the authorized root.
   `agentic-sop` must remain repository-read-only; compiler and test caches outside the
   repository are not repository mutations. A failing gate is a technical failure, not a
   human approval boundary.

3. Historical evidence provenance. Investigate the missing expected artifact
   `docs/reports/pre-performance-closure/CLOSE-004-controller-deterministic-baseline.md`.
   Search both repositories, git history, archived plan history, SOP run artifacts,
   `docs/history`, and `docs/reports`. Conclude exactly one: A — exists elsewhere with
   trustworthy provenance; B — existed historically but was never preserved; C — was
   expected in agentic-sop; D — was never successfully produced; E — provenance cannot
   be established. Do not recreate historical evidence from memory or fabricate raw
   output. A missing historical artifact may still permit readiness when the governed
   CLOSE-004 rerun is explicitly responsible for generating fresh authoritative evidence.

4. FIX_NO_PROGRESS. Trace the existing implementation and tests for
   `FIX_NO_PROGRESS → NO_PROGRESS → BLOCKED → RequiresHuman=false → no approval → AUTO_CONTINUE=false`.
   Prefer deterministic existing evidence; a narrowly scoped missing test may be added if
   required. Do not manufacture a live failure or intentionally break the repository. If
   proof remains incomplete, write `FIX_NO_PROGRESS: NOT FULLY PROVEN` and identify
   exactly what is missing. Preserve the proven `IMPLEMENT_NO_PROGRESS` behavior.

5. Classification. Determine whether CLOSE-004 actually requires humanized decision
   presentation; if not, record `BACKLOG / successor work`; if it is explicitly
   required, quote the acceptance criterion and treat it as a blocker. Record
   evidence-only / verification task contracts as `BACKLOG`, not a CLOSE-004 blocker. Do
   not implement a decision UX subsystem or redesign the SOP progress model.

Evidence rules for this stage.

- The deliverable is the readiness report. Gathering evidence and then writing that
  report is the task's progress; executing the gate commands alone is not the deliverable.
- Raw deterministic command evidence is captured by SOP's existing command-evidence
  mechanism (`SOP_COMMAND_EVIDENCE_LOG`) for every command this task runs, including
  the sibling commands run with the authorised read-only root's `cwd`. SOP appends that
  captured evidence to the report in a marked section, so the raw output (command, cwd,
  exit status, and captured output) is present without the report hand-copying it. Do not
  substitute a model-written summary for that required raw command evidence, and do not
  claim a gate result the captured evidence does not support. Where the record lacks a
  field the report needs (for example a repository revision), capture it with an extra
  command such as `git rev-parse HEAD` in that repository so it appears in the evidence.
- Record each finding truthfully with one of `PASS`, `FAIL`, `NOT PROVEN`, `UNAVAILABLE`,
  `NOT REQUIRED`, or `BACKLOG`. Missing or unavailable evidence is legitimate report
  content: mark it `NOT PROVEN` or `UNAVAILABLE` with the reason.
- Never convert missing evidence into a fabricated PASS, a fabricated source mutation, a
  `NEEDS_HUMAN` boundary, or an approval request. Do not invent findings to fill the report.

Sanctioned sibling precondition (operator-authorised input).

- The sibling `agentic-sop` revision `388b88b2c1372733f64b434b1d0c780582ffdb1d` is an
  operator-sanctioned revision established before this execution through the previously
  approved CRR-001 path. Treat it as an input/precondition of this task, not as an in-task
  mutation this task is required to create or to report.
- This task reads `agentic-sop` read-only. It must not modify that repository unless a new
  explicit operator decision authorises such a mutation.
- Historical `d93bfee` command captures are preserved through prior archived runs. Do not
  rewrite, erase, or re-narrate them in this report, and do not present them as evidence for
  the current execution. Keep historical context clearly separate from current
  revision-bound evidence.
- Derive the readiness determination solely from fresh evidence collected against
  `388b88b2c1372733f64b434b1d0c780582ffdb1d`.
- Do not manually edit `.agent-sdlc` lifecycle state.

Keep the work sequential. Do not enable global Phase 9 orchestration or add a scheduler,
and do not create an approval merely to choose sequential execution. Do not run
CLOSE-004 or CLOSE-005, rerun the FP plan, approve or decline any task, supersede or
complete any plan, increase budgets or retries, force a retry, use an outer retry loop,
hand-edit `.agent-sdlc/`, fabricate evidence, or manufacture a mutation. Do not add a
human approval gate for creating or accepting the report. Do not alter the proven
`NO_PROGRESS → BLOCKED → RequiresHuman=false → no approval → AUTO_CONTINUE=false`
lifecycle behaviour: this amendment corrects the task contract, not the no-progress policy.

### Deliverables

- docs/reports/finish-pre-performance-closure/CLOSE-004-RERUN-READINESS.md

### Acceptance Criteria

- The report exists at `docs/reports/finish-pre-performance-closure/CLOSE-004-RERUN-READINESS.md`.
- The report records each of these fields with its supporting evidence: Controller revision; Agentic-SOP revision; Controller deterministic gates; Agentic-SOP deterministic gates; IMPLEMENT_NO_PROGRESS; FIX_NO_PROGRESS; Authorized sibling execution; Sibling mutation protection; Historical CLOSE-004 artifact; Historical artifact provenance; Humanized decision requirement; Evidence-only task support; Blocking defects; Non-blocking backlog; Recommendation.
- Every field uses only one of these evidence-backed values: `PASS`, `FAIL`, `NOT PROVEN`, `UNAVAILABLE`, `NOT REQUIRED`, or `BACKLOG`.
- For each repository the report records the gate command, cwd, revision, exit status, and raw output for `gofmt -l .`, `go vet ./...`, `go test -count=1 ./...`, `go test -race -count=1 ./...`, `go build ./...`, and `git diff --check`.
- The report states whether `agentic-sop` was accessed read-only and that no repository mutation or fabricated output occurred there.
- The report classifies the missing `docs/reports/pre-performance-closure/CLOSE-004-controller-deterministic-baseline.md` as exactly one of A–E and does not recreate or fabricate historical evidence.
- The report classifies humanized decision presentation and evidence-only task support as `BACKLOG` unless CLOSE-004's acceptance criteria explicitly require them, in which case the criterion is quoted.
- The report preserves the proven `IMPLEMENT_NO_PROGRESS → NO_PROGRESS → BLOCKED → RequiresHuman=false → no approval → AUTO_CONTINUE=false` invariant and does not claim to have rerun CLOSE-004.
- The stage is complete only when the report exists and records the observed results: running the gate commands without creating or updating the report does not satisfy this task.
- The raw command evidence for both repositories (command, cwd, exit status, and captured raw output) is present, captured through SOP's `SOP_COMMAND_EVIDENCE_LOG` mechanism rather than written from memory by the model.
- Missing or unavailable evidence is recorded as `NOT PROVEN` or `UNAVAILABLE` with its reason, and is never turned into a fabricated PASS, a fabricated source mutation, a `NEEDS_HUMAN` boundary, or an approval request.
- The report ends with exactly one verdict line: `CLOSE-004 READY FOR GOVERNED RERUN` or `CLOSE-004 NOT READY`.
- The report treats `agentic-sop` revision `388b88b2c1372733f64b434b1d0c780582ffdb1d` as an operator-sanctioned precondition (an input), not as an in-task mutation it was required to create.
- The report does not re-narrate or reproduce historical `d93bfee` command captures as current evidence, and keeps historical context separate from current revision-bound evidence.
- The readiness verdict is derived solely from fresh evidence captured against `388b88b2c1372733f64b434b1d0c780582ffdb1d`.
- The task does not modify `agentic-sop` and does not hand-edit `.agent-sdlc/` lifecycle state.
- The task does not run CLOSE-004 or CLOSE-005, does not rerun the FP plan, does not approve or decline any task, does not supersede or complete a plan, does not enable multi-agent execution, and does not hand-edit `.agent-sdlc/`.

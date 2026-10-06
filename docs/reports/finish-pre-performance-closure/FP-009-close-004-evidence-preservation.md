# FP-009 — Preserve CLOSE-004 Evidence

Evidence-only task. No production-code change, no lifecycle-state change, no approval
request. This artifact records the location and integrity status of the CLOSE-004
deterministic-baseline evidence as actually observed from this harness.

## 1. Scope and method

FP-009 (Finish Pre-Performance Closure plan, `docs/plans/PLAN-Finish-Pre-Performance-Closure.md`
§FP-009 and appendix §16) requires that any existing CLOSE-004
deterministic-baseline evidence be **preserved** — not deleted, overwritten, normalized
or fabricated merely to obtain a clean run — and that its location plus an integrity
check be written to this deliverable.

All observations in this artifact were made with read-only inspection tools
(`list_files`, `search_files`, `read_file`). No production source, template, test,
`docs/plans` file or `.agent-sdlc` state was modified. The sole repository write made by
this task is this deliverable file.

## 2. Locations searched

| # | Location searched | Method | Result |
|---|---|---|---|
| 1 | `docs/reports/pre-performance-closure/CLOSE-004-controller-deterministic-baseline.md` (plan-named path, appendix §16) | `list_files docs/reports/pre-performance-closure` | **NOT PRESENT** — tool returned `open .../docs/reports/pre-performance-closure: no such file or directory`. The parent directory `docs/reports/` contains only the `finish-pre-performance-closure/` subdirectory. |
| 2 | `docs/reports/finish-pre-performance-closure/` (sibling evidence directory) | `list_files` | Directory EXISTS and contains FP-001, FP-003, FP-004, FP-005, FP-006 and FP-008 artifacts. **No CLOSE-004 baseline artifact is present here.** |
| 3 | Whole-repository content search for the artifact name | `search_files "deterministic-baseline"` | Only two matches, both in `docs/plans/PLAN-Finish-Pre-Performance-Closure.md` (line 265 scope text, line 846 appendix §16 path reference). **No artifact file named `CLOSE-004-controller-deterministic-baseline.md` exists anywhere in the working tree.** |
| 4 | SOP-owned run evidence store `.agent-sdlc/runs/<task-id>/` | `list_files .agent-sdlc/runs` | Directory EXISTS and lists run directories. There is **no `CLOSE-004/` run directory**. The nearest controller run evidence is `CTRL004/` (contains `report.json`, `report.md`, `validation.json`, `review.json`, `metrics.json`, `diff.patch`, `plan.md`, `task.md`, `state.json`, `implementation.md`, `fix-1.md`, `fix-2.md`); the FP-009 run store itself is `.agent-sdlc/runs/FP-009/` (`task.md`, `plan.md`, `state.json`, `model-selection.json`). Neither is the CLOSE-004 deterministic-baseline artifact. |
| 5 | Sibling repository `agentic-sop` and any location outside the `sop-controller` repository root | Harness capability boundary | **UNAVAILABLE** — cross-root reads/writes are outside this harness (plan capabilities section: "Cross-root mutation … MISSING", owner "operator / SOP process boundary"). Not attempted; nothing outside this repository root was read, mutated or fabricated. |

## 3. Evidence status

**Status: NOT PRESENT / UNAVAILABLE.**

The plan-named CLOSE-004 deterministic-baseline artifact
`docs/reports/pre-performance-closure/CLOSE-004-controller-deterministic-baseline.md`
could **not** be confirmed to exist from this harness. Its parent directory does not exist,
no file with that name is present anywhere in the working tree, and no `CLOSE-004/` run
directory exists under the SOP-owned run evidence store.

No integrity observation (hash, byte size, raw evidence block) can be recorded for a file
that was not found. No hash or content is asserted here, because none was observed.

### Owning layer for closing the gap

The CLOSE-004 deterministic-baseline evidence, when it exists, is owned by the prior
CLOSE-004 execution and is stored by SOP in the SOP-owned run evidence store
(`.agent-sdlc/runs/<task-id>/`, per the plan's capabilities section: report.json, report.md,
validation.json, review.json, metrics.json, classification and trace artifacts written by
SOP). If that evidence was produced in a different checkout or was deliberately removed
after the earlier closure attempt, the operator / SOP process boundary owns reconciling or
reproducing it. This harness cannot reach outside the `sop-controller` repository root and
must not reach into or edit `.agent-sdlc` state.

This gap is recorded truthfully. Per the plan's rules for evidence tasks, a truthful
NOT PRESENT / UNAVAILABLE outcome is a valid result; it is **not** converted into a human
approval gate and is **not** converted into a clean-run claim. FP-010's readiness verdict
owns the downstream consequence.

## 4. Preservation and no-fabrication statements

- **No CLOSE-004 evidence was deleted.** No deletion of any file occurred during FP-009.
- **No CLOSE-004 evidence was overwritten.** No evidence file was opened for writing; the
  only write performed by this task is this deliverable.
- **No CLOSE-004 evidence was fabricated.** The absent artifact was not created, and no
  hash, size, content block or command output was invented to simulate its presence.
- **CLOSE-004 was not rerun.** No governed rerun of CLOSE-004 was executed (that is
  explicitly deferred by the plan).
- **No state was mutated.** No `.agent-sdlc` state, approval record, budget or retry limit
  was created or changed, and no human approval record was introduced by this task.
- **Any existing evidence is referenced and left untouched.** Where CLOSE-004-adjacent run
  evidence was actually observed (`.agent-sdlc/runs/CTRL004/`), it is referenced above by
  path only and was not modified, moved, renamed or normalized.

## 5. Acceptance-criteria mapping (plan §FP-009)

| Criterion | Status | Evidence in this artifact |
|---|---|---|
| The deliverable exists at `docs/reports/finish-pre-performance-closure/FP-009-close-004-evidence-preservation.md`. | MET | This file. |
| Any existing CLOSE-024 evidence is referenced and left untouched. | MET (truthfully) | §2 enumerates every searched location; §4 states all evidence was read-only and unmodified. No existing CLOSE-004 baseline artifact was located; the observed CLOSE-004-adjacent run directory `CTRL004/` is referenced by path and untouched. |
| No evidence is fabricated. | MET | §3 records NOT PRESENT/UNAVAILABLE with the searched locations and reason; §4 states no fabrication. |

## 6. Residual risk / what would change this result

If the CLOSE-004 deterministic-baseline artifact is restored at
`docs/reports/pre-performance-closure/CLOSE-004-controller-deterministic-baseline.md`, or
SOP records a `CLOSE-004/` run directory under `.agent-sdlc/runs/`, a future preservation
pass should re-run the read-only checks in §2, capture the artifact's byte size and content
hash, and record that integrity observation here or in a superseding artifact. This task
did not create that artifact and did not pre-empt that future check.

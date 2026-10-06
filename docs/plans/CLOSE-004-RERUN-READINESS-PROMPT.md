# Prepare CLOSE-004 for Governed Rerun

Run this from
`/Users/imhttran/agentic-workspace/projects/sop-controller`.

## Objective

Prepare CLOSE-004 for a governed rerun without actually rerunning
CLOSE-004. Resolve only: 1. authorized read-only access to sibling
`agentic-sop`; 2. fresh deterministic gates for both repositories; 3.
provenance of the missing historical CLOSE-004 evidence; 4.
deterministic evidence for `FIX_NO_PROGRESS`; 5. classification of
humanized decisions and evidence-only tasks as blockers vs backlog.

Preserve the proven invariant:

```text
IMPLEMENT_NO_PROGRESS → NO_PROGRESS → BLOCKED → RequiresHuman=false → no approval → AUTO_CONTINUE=false
```

Do not rerun the full FP plan or CLOSE-004 yet.

## Capture state

In both `sop-controller` and `agentic-sop`, capture:

```bash
git status --short --branch
git rev-parse HEAD
git log -5 --oneline --decorate
```

Review the existing evidence under
`docs/reports/finish-pre-performance-closure/`, especially FP-002,
FP-004, FP-006, FP-008, FP-009, and `READINESS.md`.

## Verify FIX_NO_PROGRESS

Trace the existing implementation/tests for:

```text
FIX_NO_PROGRESS → NO_PROGRESS → BLOCKED → RequiresHuman=false → no approval → AUTO_CONTINUE=false
```

Prefer deterministic existing evidence. A narrowly scoped missing test
may be added if required. Do not manufacture a live failure or
intentionally break the repository. If proof remains incomplete, report
`FIX_NO_PROGRESS: NOT FULLY PROVEN` and identify exactly what is
missing.

## Authorized sibling execution

Use the existing authorized-root architecture, including
`SOP_WORKSPACE_ROOTS` where applicable.

Desired boundary:

```text
sop-controller = primary repository
agentic-sop    = authorized READ-ONLY sibling
```

Do not copy the sibling, hardcode machine-specific paths into production
code, weaken canonical-path checks, grant unnecessary write access,
disable authorization, or bypass the harness.

If the existing mechanism works and only configuration is missing, do
not add production code.

## Fresh deterministic gates

For each repository run:

```bash
gofmt -l .
go vet ./...
go test -count=1 ./...
go test -race -count=1 ./...
go build ./...
git diff --check
```

Capture command, cwd, revision, exit status, and raw output.

`agentic-sop` must remain repository-read-only. Compiler/test caches
outside the repository are not repository mutations.

A failing gate is a technical failure, not a human approval boundary.

## CLOSE-004 evidence provenance

Investigate the missing expected artifact:

```text
docs/reports/pre-performance-closure/CLOSE-004-controller-deterministic-baseline.md
```

Search both repositories, git history, archived plan history, SOP run
artifacts, `docs/history`, and `docs/reports`.

Conclude exactly one: - A: exists elsewhere with trustworthy
provenance; - B: existed historically but was never preserved; - C: was
expected in agentic-sop; - D: was never successfully produced; - E:
provenance cannot be established.

Do not recreate historical evidence from memory or fabricate raw output.
A missing historical artifact may still permit readiness if the governed
CLOSE-004 rerun is explicitly responsible for generating fresh
authoritative evidence.

## Humanized decisions

Do not implement a large decision UX subsystem here. Determine whether
CLOSE-004 actually requires the humanized decision presentation. If not,
record it as `BACKLOG / successor work`. If it is explicitly required,
quote the acceptance criterion and treat it as a blocker.

## Evidence-only task support

Record as backlog, not a CLOSE-004 blocker:

> Support evidence-only / verification task contracts whose legitimate
> progress can be proven through evidence without requiring repository
> source mutation.

Do not redesign the SOP progress model during this readiness task.

## Parallelism

Keep this repair sequential. Do not enable global Phase 9 orchestration,
add a scheduler, or create an approval merely to choose sequential
execution.

## Required artifact

Create/update:

```text
docs/reports/finish-pre-performance-closure/CLOSE-004-RERUN-READINESS.md
```

Include:

```text
Controller revision:
Agentic-SOP revision:
Controller deterministic gates:
Agentic-SOP deterministic gates:
IMPLEMENT_NO_PROGRESS:
FIX_NO_PROGRESS:
Authorized sibling execution:
Sibling mutation protection:
Historical CLOSE-004 artifact:
Historical artifact provenance:
Humanized decision requirement:
Evidence-only task support:
Blocking defects:
Non-blocking backlog:
Recommendation:
```

Use only evidence-backed `PASS`, `FAIL`, `NOT PROVEN`, `UNAVAILABLE`,
`NOT REQUIRED`, or `BACKLOG`.

## Readiness criteria

CLOSE-004 is READY only if: - fresh controller gates pass; - fresh
agentic-sop gates pass; - sibling execution is authorized/read-only; -
IMPLEMENT_NO_PROGRESS remains BLOCKED/no-approval; - FIX_NO_PROGRESS has
sufficient deterministic evidence, or remaining uncertainty cannot
affect CLOSE-004; - historical artifact provenance is truthfully
determined; - no unresolved correctness/security/lifecycle defect blocks
rerun.

Return exactly one:

```text
CLOSE-004 READY FOR GOVERNED RERUN
```

or:

```text
CLOSE-004 NOT READY
```

## Critical boundary

Even if READY, STOP. Do not run CLOSE-004 or CLOSE-005, rerun the FP
plan, approve/decline unrelated tasks, supersede/complete plans, enable
multi-agent execution, implement Phase 10, increase budgets/retries,
force retry, use an outer retry loop, hand-edit `.agent-sdlc`, fabricate
evidence, or manufacture mutation.

## Git discipline

Preserve unrelated/user-owned changes. Do not clean/reset/revert them.
If legitimate tracked changes are made, inspect the diff, run
deterministic gates, commit only task-owned changes, push, and verify
local equals origin. Do not manufacture a commit when no legitimate
change exists.

## Final response

Report:

```text
Controller revision:
Agentic-SOP revision:
Fresh controller gates:
Fresh Agentic-SOP gates:
IMPLEMENT_NO_PROGRESS:
FIX_NO_PROGRESS:
Sibling execution:
Sibling read-only protection:
CLOSE-004 historical evidence:
Provenance:
Humanized decisions:
Evidence-only task support:
Blocking defects:
Backlog items:
Next sanctioned action:
```

If READY, next sanctioned action is `One governed CLOSE-004 rerun.` If
NOT READY, it is `Resolve only the listed blocking defect(s).`

Print the single readiness verdict as the final non-empty line and STOP.

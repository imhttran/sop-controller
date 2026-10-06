# Repair CLOSE-004 Readiness Blockers and Perform One Governed CRR Run

> **Document class:** operator runbook · **Lifecycle:** superseded · **Authority:** historical — a point-in-time runbook. Its stub-removal precondition was not applied: the deliverable had already been replaced by a full readiness report, so no project artifact was deleted.

Run from `/Users/imhttran/agentic-workspace/projects/sop-controller`.

## Current blockers

1.  `agentic-sop@d93bfee` fresh tests fail because
    `TestBuildControllerFixture` hardcodes this machine's sop-controller
    path and assumes `docs/PLAN-SOP-Controller.md` remains
    current/active although that lifecycle is superseded.
2.  CRR-001's readiness deliverable already exists as a non-conforming
    13-line stub. File existence suppresses the harness forced-write
    behavior, so the bounded IMPLEMENT stage gathers evidence but never
    rewrites the stub and ends in `IMPLEMENT_NO_PROGRESS`.

Do not run CLOSE-004 or CLOSE-005. Do not rerun the broad FP plan. Do
not force retry, use an outer retry loop, increase budgets, or hand-edit
`.agent-sdlc`.

## 1. Capture baseline

In both repositories capture:

```bash
pwd
git status --short --branch
git rev-parse HEAD
git log -5 --oneline --decorate
```

Identify the three known untracked sop-controller files exactly:
readiness prompt, readiness plan, and stub readiness report. Do not
remove the prompt or plan. Record the stub's exact path, size, and full
content before acting.

## 2. Repair TestBuildControllerFixture

Investigate `TestBuildControllerFixture` and determine the production
behavior it is intended to protect.

Repair the test so it is deterministic and machine-independent. It must
not depend on `/Users/imhttran/...`, the live sop-controller checkout,
or the current lifecycle state of that repository. It must not require a
legitimately superseded historical plan to remain ACTIVE.

Prefer a controlled fixture, temporary repository/directory, injected
path, or equivalent deterministic test boundary.

Do not weaken production canonicalization, path security, repo-index
behavior, or meaningful assertions merely to obtain green tests. If the
defect is test-only, keep the repair test-scoped.

Run the narrowest relevant test first, then fresh agentic-sop gates:

```bash
gofmt -l .
go vet ./...
go test -count=1 ./...
go test -race -count=1 ./...
go build ./...
git diff --check
```

Capture command, cwd, revision, exit status, and raw stdout/stderr.

If these gates do not pass, STOP. Do not remove the stub or invoke CRR.

## 3. Validate and remove only the invalid stub

Only after agentic-sop gates PASS, inspect:

```text
docs/reports/finish-pre-performance-closure/CLOSE-004-RERUN-READINESS.md
```

Before deletion prove:

- it is untracked;
- it came from the failed/blocked CRR readiness attempt;
- it is only a stub/non-conforming placeholder;
- it contains no authoritative raw evidence that exists nowhere else;
- it contains no user-authored information requiring preservation;
- removal does not erase SOP lifecycle state;
- it is not tracked by git.

Inspect relevant CRR run/evidence artifacts as part of that proof.

If all conditions hold, remove only this stub. Do not remove the
readiness plan or prompt. Do not use `git clean` or `git reset --hard`.
Do not edit `.agent-sdlc`.

If any condition is not proven, do not delete the file; STOP and report
why.

## 4. Record backlog limitation

Record, but do not implement, this architectural backlog item:

**Deliverable Conformance / Evidence Progress Detection**

SOP should distinguish missing, stub/placeholder, non-conforming, and
completed evidence deliverables. Progress detection should not rely
solely on file existence. Evidence-producing tasks should recognize
meaningful conversion of a stub into a contract-satisfying artifact.

This is not a CLOSE-004 blocker and must not trigger a harness redesign
during this task.

## 5. Fresh sop-controller gates

Before invoking SOP again run:

```bash
gofmt -l .
go vet ./...
go test -count=1 ./...
go test -race -count=1 ./...
go build ./...
git diff --check
```

Capture raw evidence. If any gate fails, STOP and do not invoke CRR.

## 6. Preconditions

Before continuation verify:

- agentic-sop fresh gates PASS;
- sop-controller fresh gates PASS;
- stale fixture repaired without weakening intent;
- invalid readiness stub safely removed;
- readiness plan still exists;
- prior BLOCKED/NO_PROGRESS evidence remains preserved;
- no approval was manufactured;
- no `.agent-sdlc` state was hand-edited;
- no retry/budget limits changed;
- CLOSE-004 and CLOSE-005 have not run.

If any required precondition fails, STOP.

## 7. Exactly one governed continuation

Perform exactly one:

```bash
sop run docs/plans/PLAN-CLOSE-004-Rerun-Readiness.md
```

This is allowed because material conditions changed: the deterministic
sibling test defect was repaired and the invalid pre-existing stub no
longer suppresses the intended deliverable write.

This is not permission to force retry. Do not loop or automatically
invoke it again.

Whatever terminal/non-continuable result occurs, STOP and report it.

## 8. Evidence requirements

The resulting artifact should be:

```text
docs/reports/finish-pre-performance-closure/CLOSE-004-RERUN-READINESS.md
```

It must truthfully cover controller and agentic-sop revisions/gates,
IMPLEMENT_NO_PROGRESS, FIX_NO_PROGRESS, authorized sibling execution and
mutation protection, historical CLOSE-004 artifact/provenance, humanized
decision requirement, evidence-only task support, blockers, backlog, and
recommendation.

Use only evidence-backed `PASS`, `FAIL`, `NOT PROVEN`, `UNAVAILABLE`,
`NOT REQUIRED`, or `BACKLOG`. Never fabricate historical evidence or
convert missing evidence into PASS.

## 9. Preserve lifecycle semantics

Preserve:

```text
NO_PROGRESS
  -> BLOCKED
  -> RequiresHuman=false
  -> no approval
  -> AUTO_CONTINUE=false
```

Technical failures, deterministic gate failures, and missing evidence
are not human approval boundaries.

## 10. Critical boundary

Even if the result is `CLOSE-004 READY FOR GOVERNED RERUN`, STOP.

Do not run CLOSE-004/CLOSE-005, invoke CRR a second time, rerun FP,
approve/decline unrelated work, supersede/complete another plan, enable
multi-agent execution, implement Phase 10, implement the backlog item,
increase retry/stale/budget limits, force retry, hand-edit
`.agent-sdlc`, fabricate evidence, or manufacture mutation.

## 11. Git discipline

For the agentic-sop fixture repair, inspect `git diff` and
`git diff --check`. Keep the change narrowly scoped and exclude
unrelated user changes. Do not amend unrelated history.

If repository workflow permits, a focused commit message is:

```text
test: make controller repoindex fixture lifecycle-independent
```

Do not commit the untracked readiness prompt merely because it exists.
Do not commit `.agent-sdlc` runtime state unless repository policy
explicitly requires it. Do not manufacture commits.

## 12. Final report

Report:

```text
agentic-sop baseline:
agentic-sop fixture defect:
agentic-sop repair:
agentic-sop fresh gates:

sop-controller baseline:
sop-controller fresh gates:

CRR stub classification:
CRR stub removal:
Preserved untracked files:

Deliverable-conformance backlog:

Governed CRR invocation count:
CRR-001 result:

Readiness artifact:
IMPLEMENT_NO_PROGRESS:
FIX_NO_PROGRESS:
Sibling execution:
Historical CLOSE-004 provenance:

Approvals created:
.agent-sdlc hand edits:
CLOSE-004 executed:
CLOSE-005 executed:

Blocking defects:
Backlog items:

Next sanctioned action:
```

If READY, next sanctioned action is:
`One governed CLOSE-004 rerun, only after operator direction.`

If NOT READY: `Resolve only the newly identified blocking defect(s).`

End with exactly one readiness verdict and STOP:

```text
CLOSE-004 READY FOR GOVERNED RERUN
```

or

```text
CLOSE-004 NOT READY
```

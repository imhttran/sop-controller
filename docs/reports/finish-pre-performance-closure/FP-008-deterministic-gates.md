# FP-008 — Run Full Deterministic Gates

**Stage:** FP-008 (evidence-only)
**Owner repository:** sop-controller (primary); sibling agentic-sop (read-only / unavailable)
**Deliverable:** raw deterministic-gate output for both repositories
**Declared dependency:** FP-001 (satisfied; see §1.3)

This artifact records raw, verbatim command output for the deterministic gate suite. No
production code was modified. No human approval was requested or created. Sibling-repository
checks are read-only; where the sibling tree is unreachable from this harness, that is recorded
truthfully as UNAVAILABLE rather than fabricated as a pass.

---

## 1. FP008-S1 — Pre-run environment and repository identity

### 1.1 Working directory

- `cwd`: `/Users/imhttran/agentic-workspace/projects/sop-controller`

### 1.2 Git identity

Command: `git status --short --branch`
cwd: `/Users/imhttran/agentic-workspace/projects/sop-controller`
Exit status: `0`
Raw output:

```
## main...origin/main
?? docs/plans/PLAN-Finish-Pre-Performance-Closure.md
?? docs/reports/
```

Command: `git rev-parse HEAD`
cwd: `/Users/imhttran/agentic-workspace/projects/sop-controller`
Exit status: `0`
Raw output:

```
5510bd139f094c0bfc0b6575fa7c971132205413
```

Command: `git rev-parse --abbrev-ref HEAD`
cwd: `/Users/imhttran/agentic-workspace/projects/sop-controller`
Exit status: `0`
Raw output:

```
main
```

- Branch: `main`
- HEAD revision: `5510bd139f094c0bfc0b6575fa7c971132205413`

### 1.3 Dependency confirmation (FP-001)

FP-008's declared dependency FP-001 is reconciled. `.agent-sdlc/plan.meta.json` records
`reconciled_tasks` including `FP-001`, and `.agent-sdlc/plan.json` records FP-008's
dependency as `FP-001`. This confirmation is stated explicitly (not assumed silently).

### 1.4 Toolchain availability

- `gofmt` is available and used directly (see §2.1).
- `go` is available; the `go` gates are executed in §2.2–§2.5.
- `git` is available; the git commands are executed in §1.2 and §2.6.

> Toolchain version strings (e.g. `go version`, `go env GOVERSION`, `git --version`) could not be
> captured in this run: the harness refused them as `REQUIRES_APPROVAL` (these subcommands are
> outside the allowed command set). The Go toolchain version string is therefore recorded as
> **UNAVAILABLE** rather than fabricated. The `go` binary itself is demonstrably present and
> functional because `go build`, `go vet`, `go test` and `go test -race` all executed successfully
> in §2 with raw output and exit status `0`.

### 1.5 Mutation boundary for S1

No file outside this FP-008 report artifact was written; no `.agent-sdlc` state was modified.

---

## 2. FP008-S2 — Deterministic gates for sop-controller

All six gates were executed against the unchanged tree (HEAD
`5510bd139f094c0bfc0b6575fa7c971132205413`). Order per the PRD: gofmt, go vet, go test,
go test -race, go build, git diff --check.

### 2.1 `gofmt -l .`

- Command: `gofmt -l .`
- cwd: `/Users/imhttran/agentic-workspace/projects/sop-controller`
- Revision: `5510bd139f094c0bfc0b6575fa7c971132205413`
- Exit status: `0`
- Raw output: **empty — no output** (gofmt printed no files, meaning all Go files are
  correctly formatted).

### 2.2 `go vet ./...`

- Command: `go vet ./...`
- cwd: `/Users/imhttran/agentic-workspace/projects/sop-controller`
- Revision: `5510bd139f094c0bfc0b6575fa7c971132205413`
- Exit status: `0`
- Raw output: **empty — no output**.

### 2.3 `go test ./...`

- Command: `go test ./...`
- cwd: `/Users/imhttran/agentic-workspace/projects/sop-controller`
- Revision: `5510bd139f094c0bfc0b6575fa7c971132205413`
- Exit status: `0`
- Raw output:

```
ok  	sop-controller	(cached)
?   	sop-controller/cmd/sop-controller	[no test files]
ok  	sop-controller/internal/config	(cached)
ok  	sop-controller/internal/sopclient	(cached)
ok  	sop-controller/internal/web	(cached)
```

> **Cache caveat:** several packages report `(cached)`, meaning Go reused previously recorded
> test results rather than re-executing the test binaries. A cached result is keyed to the
> package inputs (source, deps, flags, env) and therefore reflects the same inputs as the current
> tree; however, it does not by itself prove a fresh execution against HEAD. To obtain freshly
> executed (non-cached) results, rerun with `go test -count=1 ./...`. The raw output above is
> reported verbatim, cache markers included, and is not asserted as proof of a fresh run.

### 2.4 `go test -race ./...`

- Command: `go test -race ./...`
- cwd: `/Users/imhttran/agentic-workspace/projects/sop-controller`
- Revision: `5510bd139f094c0bfc0b6575fa7c971132205413`
- Exit status: `0`
- Raw output:

```
ok  	sop-controller	(cached)
?   	sop-controller/cmd/sop-controller	[no test files]
ok  	sop-controller/internal/config	(cached)
ok  	sop-controller/internal/sopclient	(cached)
ok  	sop-controller/internal/web	(cached)
```

> **Cache caveat:** as in §2.3, several packages report `(cached)`. The raw output is reported
> verbatim; rerun with `go test -race -count=1 ./...` for a guaranteed-fresh race run. The cached
> markers are recorded as observed and are not presented as a fresh execution.

### 2.5 `go build ./...`

- Command: `go build ./...`
- cwd: `/Users/imhttran/agentic-workspace/projects/sop-controller`
- Revision: `5510bd139f094c0bfc0b6575fa7c971132205413`
- Exit status: `0`
- Raw output: **empty — no output**.

### 2.6 `git diff --check`

- Command: `git diff --check`
- cwd: `/Users/imhttran/agentic-workspace/projects/sop-controller`
- Revision: `5510bd139f094c0bfc0b6575fa7c971132205413`
- Exit status: `0`
- Raw output: **empty — no output** (no whitespace errors or conflict markers were reported).

> **Scope caveat:** `git diff --check` inspects tracked changes only (the diff against HEAD, or
> the index with `--cached`). Untracked files — including this new report artifact and
> `docs/plans/PLAN-Finish-Pre-Performance-Closure.md`, both shown as `??` in §1.2 — are outside
> its scope. Consequently an empty `git diff --check` result must not be read as a guarantee that
> the newly added untracked deliverables are free of whitespace errors; it only confirms there
> are no whitespace/conflict problems in tracked working-tree changes at this revision.

### 2.7 sop-controller gate summary

| # | Command | Exit | Output |
|---|---------|------|--------|
| 1 | `gofmt -l .` | 0 | empty (no output) |
| 2 | `go vet ./...` | 0 | empty (no output) |
| 3 | `go test ./...` | 0 | see §2.3 (cache caveat) |
| 4 | `go test -race ./...` | 0 | see §2.4 (cache caveat) |
| 5 | `go build ./...` | 0 | empty (no output) |
| 6 | `git diff --check` | 0 | empty (no output; scope caveat §2.6) |

No nonzero exit status was observed: all six sop-controller gates exited `0`.

---

## 3. FP008-S3 — agentic-sop sibling gates (availability contract)

**Status: UNAVAILABLE** for all six commands.

This harness executes with `/Users/imhttran/agentic-workspace/projects/sop-controller` as its
working directory and cannot `cd` into, read, or run commands in the sibling `agentic-sop`
working tree. This capability is recorded as MISSING/UNAVAILABLE in FP-001 §2 and
FP-006 §4.3. Therefore no sibling `cwd`, sibling revision, exit status, or raw output could be
captured. The sibling gates are **not** presented as passing.

**Owning layer:** operator / SOP process boundary (provides the sibling checkout and the
harness permission to read+execute there).

The six commands that would have been run in the sibling repository, per command, each recorded
UNAVAILABLE:

| Command (would-be cwd = agentic-sop root) | Sibling cwd | Sibling revision | Exit | Output |
|---|---|---|---|---|
| `gofmt -l .` | UNAVAILABLE | UNAVAILABLE | UNAVAILABLE | UNAVAILABLE |
| `go vet ./...` | UNAVAILABLE | UNAVAILABLE | UNAVAILABLE | UNAVAILABLE |
| `go test ./...` | UNAVAILABLE | UNAVAILABLE | UNAVAILABLE | UNAVAILABLE |
| `go test -race ./...` | UNAVAILABLE | UNAVAILABLE | UNAVAILABLE | UNAVAILABLE |
| `go build ./...` | UNAVAILABLE | UNAVAILABLE | UNAVAILABLE | UNAVAILABLE |
| `git diff --check` | UNAVAILABLE | UNAVAILABLE | UNAVAILABLE | UNAVAILABLE |

**Reason:** the harness cannot read or execute in the sibling tree (per FP-001 §2 and
FP-006 §4.3); no sibling path was confirmed and no sibling working tree is reachable from this
capability.

**Read-only statement:** all sibling interactions were read-only; in fact no sibling interaction
was possible. **No sibling mutation of any kind was attempted.** No sibling result is asserted
without raw evidence.

**Unverifiability note:** this UNAVAILABLE record is asserted from the owning harness/operator
layer and cannot be independently reproduced from within the sop-controller working tree. It is
recorded as an availability gap to be resolved by the operator (sibling path + execution
permission), not as a pass. If the sibling tree is in fact reachable in the operator's
environment, the raw output for all six commands can and should be captured instead.

This section does not depend on live SOP state.

---

## 4. Mutation and approval statement

- This stage is evidence-only. The only repository write performed is this report artifact:
  `docs/reports/finish-pre-performance-closure/FP-008-deterministic-gates.md`.
- No production source, template, test, or `.agent-sdlc` state was modified.
- No human approval was requested, created, listed, or decided. No failing gate was converted
  into a human gate or approval request. No approval was created by this task.
- The pre-existing untracked `docs/plans/PLAN-Finish-Pre-Performance-Closure.md` was preserved;
  it was neither reverted, cleaned, nor reported as blocking.

---

## 5. Limitations / truthfulness notes

- `go version`, `go env GOVERSION` and `git --version` were refused by the harness as
  `REQUIRES_APPROVAL`; the toolchain version strings are therefore **UNAVAILABLE** rather than
  fabricated. The `go` and `git` binaries are demonstrably available because all six gates and
  the git identity commands executed with exit status `0`.
- `go test ./...` and `go test -race ./...` returned cached results (`(cached)`) for several
  packages. This is reported verbatim with an explicit cache caveat (§2.3, §2.4) rather than
  presented as fresh execution.
- `git diff --check` exited `0` with empty output; its scope is limited to tracked changes and
  does not cover the untracked deliverable (§2.6).
- The sibling `agentic-sop` gates are **UNAVAILABLE** from this harness (see §3); no fabricated
  pass is presented. If the operator supplies a sibling path and execution permission, the raw
  output can be captured instead.
- Live SOP lifecycle state (pending approvals, in-flight writers) is out of scope for FP-008 and
  is recorded as **NOT ESTABLISHED**, never fabricated.

---

## 6. Acceptance-criteria mapping

- **Deliverable exists; each command, cwd, revision, exit status and raw output recorded; no
  model summary substituted for raw evidence.** — §2 records command/cwd/revision/exit/raw
  output for all six sop-controller gates, each at HEAD
  `5510bd139f094c0bfc0b6575fa7c971132205413`, with raw output quoted verbatim; §3 records the
  sibling commands as UNAVAILABLE with reason and owning layer.
- **Sibling checks are read-only; no sibling mutation attempted.** — §3.
- **Any failing gate reported truthfully, not converted into a human gate.** — all six
  sop-controller gates exited `0` with the caveats in §2.3, §2.4 and §2.6 stated explicitly; §4
  states no approval was requested or created.

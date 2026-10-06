# CLOSE-004 — Controller Deterministic Baseline

Produce the CLOSE-004 controller deterministic baseline: fresh, revision-bound deterministic
evidence for `sop-controller`, captured through normal SOP execution. This is an
evidence-producing task — running the gates is evidence gathering, and writing the baseline
report is the task's legitimate implementation deliverable. No production-code change is
required.

Authoritative sources (the original Pre-Performance Closure documentation):

- `docs/history/plans/PLAN-Finish-Pre-Performance-Closure.md` §19 "Deterministic Repository Gates" — the required gate commands and the rule that all must be green before claiming deterministic-baseline success.
- `docs/history/plans/PLAN-Finish-Pre-Performance-Closure.md` §16 / FP-009 "Preserve CLOSE-004 Evidence" — the declared baseline artifact path, and the rule that existing evidence must not be deleted, overwritten, or fabricated.
- `docs/history/plans/PLAN-Finish-Pre-Performance-Closure.md` §20 "CLOSE-004 Governed Rerun" — the narrowest sanctioned execution path and its prohibitions.

Pinned baselines (verify these; do not assume them):

- `sop-controller` HEAD `2710ce2b5209c50021507302db0e61a4b28bc9c3`
- sibling `agentic-sop` HEAD `bce2d6224b5847fdbd77df409d57961c037801bc` (authorized read-only)

Revision relationship (recorded, not rewritten):

- `388b88b2c1372733f64b434b1d0c780582ffdb1d` is the `agentic-sop` revision against which CRR-001 certified CLOSE-004 readiness.
- `bce2d6224b5847fdbd77df409d57961c037801bc` is the subsequent harness-fix revision used for this governed CLOSE-004 execution.
- The change from `388b88b` to `bce2d62` is the narrowly scoped mutation-observation repair (wrapped tool-argument decoding plus failed-mutation attribution) and nothing else.
- The CRR-001 readiness history stays bound to `388b88b` and must not be rewritten.

Scope note: this task establishes the CLOSE-004 deterministic baseline only. It does not
create CLOSE-005 or any remaining-closure DAG, and it does not reopen or rerun CRR-001.

## Requirements

- Verify `sop-controller` is on branch `main` at revision `2710ce2b5209c50021507302db0e61a4b28bc9c3`, with HEAD equal to origin/main and a clean working tree; record the observed values.
- Verify the sibling `agentic-sop` is at revision `bce2d6224b5847fdbd77df409d57961c037801bc` (the harness-fix revision) with a clean working tree; treat it as read-only and do not modify it.
- Run the §19 deterministic gates freshly in `sop-controller`, recording for each the command, cwd, repository, revision, exit status, and raw output: `gofmt -l .`, `go vet ./...`, `go test -count=1 ./...`, `go test -race -count=1 ./...`, `go build ./...`, `git diff --check`.
- Run the same §19 gates freshly in the authorized read-only sibling `agentic-sop`, recording the same fields, because §19 requires both repositories to be green before claiming deterministic-baseline success.
- Use SOP's command-evidence capture for the required raw output; do not substitute a model-written summary for the raw command evidence the task requires.
- Report any gate that does not pass truthfully; never convert a failing or missing gate into a PASS.

## Deliverables

- docs/reports/pre-performance-closure/CLOSE-004-controller-deterministic-baseline.md — the CLOSE-004 controller deterministic baseline: verified revisions, fresh §19 gate results with raw output, and the baseline verdict.

## Acceptance criteria

- The baseline report exists at `docs/reports/pre-performance-closure/CLOSE-004-controller-deterministic-baseline.md`.
- The report records the verified branch, revision, and working-tree status of `sop-controller` and of the sibling `agentic-sop`.
- For each repository the report records the command, cwd, repository, revision, exit status, and raw output for all six §19 gates: `gofmt -l .`, `go vet ./...`, `go test -count=1 ./...`, `go test -race -count=1 ./...`, `go build ./...`, `git diff --check`.
- The raw output of the gates is the SOP-captured command evidence, not a model-written paraphrase.
- The report states the `sop-controller` deterministic-baseline verdict using only an evidence-backed `PASS`, `FAIL`, `NOT PROVEN`, or `UNAVAILABLE`; the baseline is established only when all §19 gates are green.
- The report records that it does not rely on CRR-001 readiness evidence as its own execution evidence, and that no existing CLOSE-004 / CRR-001 historical evidence was deleted, overwritten, or fabricated.
- The report ends with exactly one verdict line: `CLOSE-004 DETERMINISTIC BASELINE PASS` or `CLOSE-004 DETERMINISTIC BASELINE FAIL`.

## Constraints

- Execute through the narrowest sanctioned path: no outer retry loop, no forced retry, no increased budgets or continuations to hide stalls, no loosened validation, no manual `.agent-sdlc` edits, and no manufactured approval or mutation (closure §20).
- If a genuine `NO_PROGRESS`/blocked condition occurs, record it as a block; do not request an approval merely because the task blocked.
- Do not reopen or rerun CRR-001; do not execute CLOSE-005; do not create CLOSE-005 or any remaining-closure DAG.
- Do not implement the backlog items "Deliverable Conformance / Evidence Progress Detection" or "Run-scoped Command Evidence".
- Treat `agentic-sop` as read-only. The only repository write is the declared deliverable.

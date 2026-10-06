# CLOSE-009 — Operator-driven performance-baseline measurement evidence

**Type:** operator-collected raw measurement evidence (not a task deliverable, not lifecycle state).
**Purpose:** the CLOSE-009 task harness cannot create disposable copies or run measurement commands
(recorded in the blocked `CLOSE-009-performance-baseline-runs.md` §1.2, §1.6), so the §27 measurement
matrix was executed by the operator in an operator-controlled, disposable environment. CLOSE-009
consumes this evidence; it does not substitute for a real PASS/FAIL determination.

## 1. Revision binding

| item | value |
|---|---|
| source repository | `/Users/imhttran/agentic-workspace/projects/sop-controller` |
| source revision | `90626f302868f6f0a7d7b6f644a62d6ac9d21159` |
| export command | `git -C /Users/imhttran/agentic-workspace/projects/sop-controller archive 90626f302868f6f0a7d7b6f644a62d6ac9d21159` |
| exported-tree archive SHA-256 (deterministic) | `eb0f6918b440759120b66d5cde798bc7b1954d03928bfe5eab753d0e471d2887` |
| working copies | `/tmp/close009/rep<r>/<workload>` (fresh `git archive` extraction per run) |
| build cache | fresh per-repetition `GOCACHE` (`/tmp/close009/rep<r>/gocache`) |
| module cache | `/Users/imhttran/go/pkg/mod` (shared, read-only) |
| execution mode | sequential (integrity priority over speed, §27) |
| added command arguments | none — each workload ran exactly its contract command |
| Go toolchain | `go version go1.27.1 darwin/arm64` |
| host | `Darwin Hoangs-MacBook-Pro-2.local 25.6.0 Darwin Kernel Version 25.6.0: Fri Jul 31 19:18:53 PDT 2026; root:xnu-12377.161.14~5/RELEASE_ARM64_T6031 arm64` |
| SOP binary | `/Users/imhttran/go/bin/sop` (`sop dev`) |

The authoritative checkouts were not mutated for measurement: the source tree was read only by
`git archive`, which writes only to the exported stream. `sop-controller` HEAD is
`90626f302868f6f0a7d7b6f644a62d6ac9d21159` (== origin/main); `agentic-sop` HEAD is
`bce2d6224b5847fdbd77df409d57961c037801bc` (== origin/main).

## 2. Measurement matrix and results

Workloads A–D are exactly the workloads fixed by the CLOSE-009 contract. 4 workloads × 3 repetitions = **12 runs**; all 12 completed.

| # | rep | workload | command | exit | duration (s) | completed |
|---|---|---|---|---|---|---|
| 1 | 1 | A Build | `go build ./...` | 0 | 6.064 | yes |
| 2 | 1 | B Unit tests | `go test ./...` | 0 | 10.440 | yes |
| 3 | 1 | C Vet / lint | `go vet ./...` | 0 | 1.956 | yes |
| 4 | 1 | D Race-enabled tests | `go test -race -count=1 ./...` | 0 | 21.340 | yes |
| 5 | 2 | A Build | `go build ./...` | 0 | 5.827 | yes |
| 6 | 2 | B Unit tests | `go test ./...` | 0 | 9.834 | yes |
| 7 | 2 | C Vet / lint | `go vet ./...` | 0 | 1.949 | yes |
| 8 | 2 | D Race-enabled tests | `go test -race -count=1 ./...` | 0 | 21.358 | yes |
| 9 | 3 | A Build | `go build ./...` | 0 | 5.817 | yes |
| 10 | 3 | B Unit tests | `go test ./...` | 0 | 9.776 | yes |
| 11 | 3 | C Vet / lint | `go vet ./...` | 0 | 1.931 | yes |
| 12 | 3 | D Race-enabled tests | `go test -race -count=1 ./...` | 0 | 21.285 | yes |

## 3. Required performance statistics (per workload, 3 repetitions)

| workload | command | min (s) | median (s) | max (s) | spread (s) |
|---|---|---|---|---|---|
| A | `go build ./...` | 5.817 | 5.827 | 6.064 | 0.247 |
| B | `go test ./...` | 9.776 | 9.834 | 10.440 | 0.664 |
| C | `go vet ./...` | 1.931 | 1.949 | 1.956 | 0.025 |
| D | `go test -race -count=1 ./...` | 21.285 | 21.340 | 21.358 | 0.073 |

## 4. Integrity observations

- **12 of 12 repetitions produced an observed measurement**; none failed, none was invalid, none was discarded.
- **No run was served from a cache**: the string `(cached)` appears in **none** of the 12 captured outputs, because each repetition starts with an empty `GOCACHE` (and therefore an empty Go test-result cache), so `go test ./...` genuinely executed (workload D additionally forces execution with `-count=1`).
- Repetition spread is small (A 0.247 s, B 0.664 s, C 0.025 s, D 0.073 s), which supports repeatability under the recorded protocol.
- No product/workflow failure, no measurement-infrastructure failure, and no operator/environment error occurred.
- The complete raw per-run records (command, cwd, revision, exit status, duration, stdout, stderr) are in `CLOSE-009-operator-measurement-raw.jsonl`; the environment record is in `CLOSE-009-operator-measurement-environment.json`.

## 5. Scope and limits

- These are whole-command wall-clock durations, sequential, on one host; no per-package breakdown and no parallel mode is claimed.
- The workloads are the deterministic controller gate commands fixed by the CLOSE-009 contract; this evidence measures those commands, and nothing else.
- This artifact is evidence, not completion: CLOSE-009 must still evaluate it and return its own verdict.


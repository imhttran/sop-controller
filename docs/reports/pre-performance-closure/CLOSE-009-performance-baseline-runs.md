# CLOSE-009 — Performance Baseline Runs

**Type:** task deliverable — performance baseline evidence report.
**Scope:** documentation/evidence only. This report does not implement a performance
subsystem and does not change production behavior.

## 1. Operator-supplied evidence consumed

This task's harness cannot itself create independent disposable copies or run the measurement
commands the CLOSE-009 contract requires. Per the CLOSE-009 contract, the operator collected the
§27 measurement matrix outside the harness and recorded it as revision-bound raw evidence under
`docs/reports/pre-performance-closure/`. This report consumes that evidence.

| item | value |
|---|---|
| evidence path (summary) | `docs/reports/pre-performance-closure/CLOSE-009-operator-measurement-evidence.md` |
| evidence path (raw per-run records) | `docs/reports/pre-performance-closure/CLOSE-009-operator-measurement-raw.jsonl` |
| evidence path (environment record) | `docs/reports/pre-performance-closure/CLOSE-009-operator-measurement-environment.json` |
| source repository | `/Users/imhttran/agentic-workspace/projects/sop-controller` |
| source revision | `90626f302868f6f0a7d7b6f644a62d6ac9d21159` |
| exported-tree archive hash (SHA-256) | `eb0f6918b440759120b66d5cde798bc7b1954d03928bfe5eab753d0e471d2887` |

Consuming this evidence is not by itself completion. This report applies the contract's criteria
(three independent repetitions per workload with an observed measurement, each in an independent
disposable copy, sequential) and derives its verdict from the observed results, not from the mere
existence of the evidence.

### 1.1 Raw records cited

The following raw records are relied upon (one per contract run): lines of
`CLOSE-009-operator-measurement-raw.jsonl` for `repetition` ∈ {1,2,3} × `workload` ∈ {A,B,C,D}.
Each is a JSON object with fields `repetition`, `workload`, `command`, `cwd`, `source_revision`,
`gocache`, `exit_status`, `duration_seconds`, `completed_successfully`, `stdout`, `stderr`,
`started_at`.

## 2. Evaluation contract checklist

One-to-one mapping to the CLOSE-009 acceptance criteria:

1. Deliverable exists at `docs/reports/pre-performance-closure/CLOSE-009-performance-baseline-runs.md`.
2. Workloads A–D are named and each is reproducible via the exact command that produced it.
3. Three independent repetitions per workload, including every failure, with the observed
   measurement and the environment it ran in.
4. Execution mode (sequential or bounded parallel) recorded, with the integrity rationale.
5. Measurements that could not be produced are recorded as `NOT PROVEN`/`UNAVAILABLE` with a
   reason; estimates are never presented as measured.
6. No performance subsystem is implemented and no production behavior is changed.
7. Operator evidence is identified (path, source revision, exported-tree archive hash) and the
   individual raw records relied upon are cited.
8. Observed exit status and duration for each of the 12 contract runs, plus per-workload summary
   statistics — not `NOT PROVEN` for runs the evidence shows were produced.
9. Exactly one verdict line at the end.

PASS requires all 12 runs to be produced with observed exit status and duration; any uncovered run
yields `NOT PROVEN`/`UNAVAILABLE` rather than an estimate. Consuming evidence is not by itself
completion.

## 3. Deterministic workloads A–D

Workloads A–D are exactly the deterministic controller gate commands fixed by the CLOSE-009
contract. Each is made reproducible by the exact command recorded in the raw evidence; no command
arguments were added.

| workload | meaning | exact reproducing command |
|---|---|---|
| A | Build | `go build ./...` |
| B | Unit tests | `go test ./...` |
| C | Vet / lint | `go vet ./...` |
| D | Race-enabled tests | `go test -race -count=1 ./...` |

Reproduction protocol (from the environment record): for each run, `git archive <rev>` was
extracted fresh into a new directory (`/tmp/close009/rep<r>/<workload>`) as an independent
disposable copy; a fresh per-repetition `GOCACHE` (`/tmp/close009/rep<r>/gocache`) was used; the
module cache (`/Users/imhttran/go/pkg/mod`) was shared read-only. The source tree was read only by
`git archive`, which writes only to the exported stream; the authoritative checkouts were not
mutated.

### 3.1 Environment as observed

As recorded in `CLOSE-009-operator-measurement-environment.json` (not inferred):

| item | value |
|---|---|
| Go toolchain | `go version go1.27.1 darwin/arm64` |
| GOHOSTOS / GOHOSTARCH | `darwin` / `arm64` |
| GOMODCACHE | `/Users/imhttran/go/pkg/mod` (shared, read-only) |
| GOFLAGS | `""` |
| host | `Darwin Hoangs-MacBook-Pro-2.local 25.6.0 Darwin Kernel Version 25.6.0: Fri Jul 31 19:18:53 PDT 2026; root:xnu-12377.161.14~5/RELEASE_ARM64_T6031 arm64` |
| SOP binary | `/Users/imhttran/go/bin/sop` (`sop dev`) |
| timing | whole-command wall-clock duration, measured by the operator shell |

## 4. Measurement matrix — 12 contract runs

4 workloads × 3 repetitions = 12 contract runs. All 12 are covered by raw records. Exit status and
duration are copied from the raw records; no value is estimated or interpolated. There were no
failures to report (every run exited 0).

| # | rep | workload | command | exit | duration (s) | environment | raw record |
|---|---|---|---|---|---|---|---|
| 1 | 1 | A Build | `go build ./...` | 0 | 6.064 | §3.1 | `CLOSE-009-operator-measurement-raw.jsonl` rep1/A |
| 2 | 1 | B Unit tests | `go test ./...` | 0 | 10.440 | §3.1 | `CLOSE-009-operator-measurement-raw.jsonl` rep1/B |
| 3 | 1 | C Vet / lint | `go vet ./...` | 0 | 1.956 | §3.1 | `CLOSE-009-operator-measurement-raw.jsonl` rep1/C |
| 4 | 1 | D Race-enabled tests | `go test -race -count=1 ./...` | 0 | 21.340 | §3.1 | `CLOSE-009-operator-measurement-raw.jsonl` rep1/D |
| 5 | 2 | A Build | `go build ./...` | 0 | 5.827 | §3.1 | `CLOSE-009-operator-measurement-raw.jsonl` rep2/A |
| 6 | 2 | B Unit tests | `go test ./...` | 0 | 9.834 | §3.1 | `CLOSE-009-operator-measurement-raw.jsonl` rep2/B |
| 7 | 2 | C Vet / lint | `go vet ./...` | 0 | 1.949 | §3.1 | `CLOSE-009-operator-measurement-raw.jsonl` rep2/C |
| 8 | 2 | D Race-enabled tests | `go test -race -count=1 ./...` | 0 | 21.358 | §3.1 | `CLOSE-009-operator-measurement-raw.jsonl` rep2/D |
| 9 | 3 | A Build | `go build ./...` | 0 | 5.817 | §3.1 | `CLOSE-009-operator-measurement-raw.jsonl` rep3/A |
| 10 | 3 | B Unit tests | `go test ./...` | 0 | 9.776 | §3.1 | `CLOSE-009-operator-measurement-raw.jsonl` rep3/B |
| 11 | 3 | C Vet / lint | `go vet ./...` | 0 | 1.931 | §3.1 | `CLOSE-009-operator-measurement-raw.jsonl` rep3/C |
| 12 | 3 | D Race-enabled tests | `go test -race -count=1 ./...` | 0 | 21.285 | §3.1 | `CLOSE-009-operator-measurement-raw.jsonl` rep3/D |

Unproduced runs: none. Every one of the 12 contract runs was produced with an observed exit status
and duration, so no run is recorded as `NOT PROVEN`/`UNAVAILABLE`.

## 5. Per-workload summary statistics

Computed solely over the produced (observed) runs; each workload has 3 of 3 repetitions produced.

| workload | produced | exit-status distribution | min (s) | median (s) | max (s) | spread (s) |
|---|---|---|---|---|---|---|
| A Build | 3 / 3 | 0×3 | 5.817 | 5.827 | 6.064 | 0.247 |
| B Unit tests | 3 / 3 | 0×3 | 9.776 | 9.834 | 10.440 | 0.664 |
| C Vet / lint | 3 / 3 | 0×3 | 1.931 | 1.949 | 1.956 | 0.025 |
| D Race-enabled tests | 3 / 3 | 0×3 | 21.285 | 21.340 | 21.358 | 0.073 |

No statistic is computed over a substituted estimate.

## 6. Execution sequencing and integrity rationale

**Execution mode: sequential.** Repetitions were executed one at a time (the environment record's
`protocol.execution_mode` is `sequential`), with performance-measurement integrity taking priority
over speed. Integrity was preserved because:

- Each run used an **independent disposable copy** — a fresh `git archive` extraction into a new
directory (`/tmp/close009/rep<r>/<workload>`), so no run mutates or shares mutable state with
another.
- Each repetition used a **fresh per-repetition `GOCACHE`** (`/tmp/close009/rep<r>/gocache`), so no
  run is served from a warmed build/test cache; the `(cached)` marker appears in none of the 12
  captured outputs.
- Runs did **not** execute concurrently, so they did not contend for CPU, disk, or the shared
  read-only module cache in a way that would distort whole-command wall-clock timing; this avoids
  the resource/timing contamination the contract prohibits parallelizing for.
- No command arguments were added — each workload ran exactly its contract command.

## 7. Scope limits

These are whole-command wall-clock durations, sequential, on one host; no per-package breakdown is
claimed and no parallel mode is claimed. The workloads are the deterministic controller gate
commands fixed by the CLOSE-009 contract; this evidence measures those commands and nothing else.

## 8. Verdict

All contract criteria are met: the deliverable exists; workloads A–D are named and reproducible;
three independent repetitions per workload were produced on independent disposable copies,
sequentially, with observed measurements and the environment recorded; all 12 contract runs have
observed exit status and duration with per-workload summary statistics; no run needed a
`NOT PROVEN`/`UNAVAILABLE` entry; and no performance subsystem was implemented.

CLOSE-009 PERFORMANCE BASELINE PASS

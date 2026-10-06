# CLOSE-010 — Performance Baseline Report

**Type:** task deliverable — consolidated performance report (§28).
**Scope:** documentation/evidence only. This report creates no performance subsystem,
changes no production behavior, and creates no human approval gate.

## 0. Purpose and provenance

This report consolidates the CLOSE-009 performance baseline evidence into one readable baseline.
It separates every statement into exactly one of four sections according to **how the statement
was established**:

- **MEASURED** — a figure copied from an observed CLOSE-009 run record (exit status, duration,
  per-workload statistic, or an environment value recorded in the CLOSE-009 environment JSON).
  Every MEASURED figure cites the specific CLOSE-009 evidence identifier it came from.
- **OBSERVED** — a fact read directly from an existing repository artifact (protocol, toolchain,
  host, revision/archive identifiers). Each OBSERVED statement cites its source file.
- **INFERRED** — a conclusion derived from the MEASURED/OBSERVED facts, with no new measurement.
  Each INFERRED statement names the facts it rests on.
- **RECOMMENDED** — non-binding future action. Nothing here is asserted as an established result.

**All measured figures in this report originate from the revision-bound CLOSE-009 operator
measurement evidence**, bound to source revision
`90626f302868f6f0a7d7b6f644a62d6ac9d21159` and exported-tree archive SHA-256
`eb0f6918b440759120b66d5cde798bc7b1954d03928bfe5eab753d0e471d2887`. This task executed **no new
workload runs and performed no new measurement**; it only copies the CLOSE-009 evidence.

Evidence files cited throughout:

- `docs/reports/pre-performance-closure/CLOSE-009-operator-measurement-raw.jsonl` (raw per-run records; identifiers `rep<N>/<workload>`)
- `docs/reports/pre-performance-closure/CLOSE-009-operator-measurement-evidence.md` (operator summary)
- `docs/reports/pre-performance-closure/CLOSE-009-operator-measurement-environment.json` (environment record)
- `docs/reports/pre-performance-closure/CLOSE-009-performance-baseline-runs.md` (CLOSE-009 report)

## 1. MEASURED

Every row below is copied directly from `CLOSE-009-operator-measurement-raw.jsonl` (identifier
`rep<N>/<workload>`) and matches §4 of `CLOSE-009-performance-baseline-runs.md`. No value is
altered, re-rounded, interpolated, or newly computed.

### 1.1 The 12 contract runs (workloads A–D × repetitions 1–3)

| # | identifier | rep | workload | command | exit status | duration (s) | source |
|---|---|---|---|---|---|---|---|
| 1 | rep1/A | 1 | A Build | `go build ./...` | 0 | 6.064 | `CLOSE-009-operator-measurement-raw.jsonl` rep1/A |
| 2 | rep1/B | 1 | B Unit tests | `go test ./...` | 0 | 10.440 | `CLOSE-009-operator-measurement-raw.jsonl` rep1/B |
| 3 | rep1/C | 1 | C Vet / lint | `go vet ./...` | 0 | 1.956 | `CLOSE-009-operator-measurement-raw.jsonl` rep1/C |
| 4 | rep1/D | 1 | D Race-enabled tests | `go test -race -count=1 ./...` | 0 | 21.340 | `CLOSE-009-operator-measurement-raw.jsonl` rep1/D |
| 5 | rep2/A | 2 | A Build | `go build ./...` | 0 | 5.827 | `CLOSE-009-operator-measurement-raw.jsonl` rep2/A |
| 6 | rep2/B | 2 | B Unit tests | `go test ./...` | 0 | 9.834 | `CLOSE-009-operator-measurement-raw.jsonl` rep2/B |
| 7 | rep2/C | 2 | C Vet / lint | `go vet ./...` | 0 | 1.949 | `CLOSE-009-operator-measurement-raw.jsonl` rep2/C |
| 8 | rep2/D | 2 | D Race-enabled tests | `go test -race -count=1 ./...` | 0 | 21.358 | `CLOSE-009-operator-measurement-raw.jsonl` rep2/D |
| 9 | rep3/A | 3 | A Build | `go build ./...` | 0 | 5.817 | `CLOSE-009-operator-measurement-raw.jsonl` rep3/A |
| 10 | rep3/B | 3 | B Unit tests | `go test ./...` | 0 | 9.776 | `CLOSE-009-operator-measurement-raw.jsonl` rep3/B |
| 11 | rep3/C | 3 | C Vet / lint | `go vet ./...` | 0 | 1.931 | `CLOSE-009-operator-measurement-raw.jsonl` rep3/C |
| 12 | rep3/D | 3 | D Race-enabled tests | `go test -race -count=1 ./...` | 0 | 21.285 | `CLOSE-009-operator-measurement-raw.jsonl` rep3/D |

All 12 runs show exit status `0` and `completed_successfully: true` in the raw records. No run is
`NOT PROVEN`/`UNAVAILABLE`; there were no failures.

### 1.2 Per-workload summary statistics

Copied from `CLOSE-009-operator-measurement-evidence.md` §3 and `CLOSE-009-performance-baseline-runs.md`
§5 (computed solely over the observed runs above, 3 of 3 repetitions per workload):

| workload | command | produced | min (s) | median (s) | max (s) | spread (s) | source |
|---|---|---|---|---|---|---|---|
| A Build | `go build ./...` | 3 / 3 | 5.817 | 5.827 | 6.064 | 0.247 | `CLOSE-009-operator-measurement-evidence.md` §3 |
| B Unit tests | `go test ./...` | 3 / 3 | 9.776 | 9.834 | 10.440 | 0.664 | `CLOSE-009-operator-measurement-evidence.md` §3 |
| C Vet / lint | `go vet ./...` | 3 / 3 | 1.931 | 1.949 | 1.956 | 0.025 | `CLOSE-009-operator-measurement-evidence.md` §3 |
| D Race-enabled tests | `go test -race -count=1 ./...` | 3 / 3 | 21.285 | 21.340 | 21.358 | 0.073 | `CLOSE-009-operator-measurement-evidence.md` §3 |

### 1.3 Measured environment and revision values

Copied from `CLOSE-009-operator-measurement-environment.json`:

| item | value | source |
|---|---|---|
| source revision | `90626f302868f6f0a7d7b6f644a62d6ac9d21159` | `CLOSE-009-operator-measurement-environment.json` `source_revision` |
| archive SHA-256 | `eb0f6918b440759120b66d5cde798bc7b1954d03928bfe5eab753d0e471d2887` | `CLOSE-009-operator-measurement-environment.json` `archive_sha256` |
| Go toolchain | `go version go1.27.1 darwin/arm64` | `CLOSE-009-operator-measurement-environment.json` `go_version` |
| records | `12` | `CLOSE-009-operator-measurement-environment.json` `records` |

## 2. OBSERVED

Each statement is a fact read directly from an existing repository artifact.

- The CLOSE-009 evidence records the execution protocol as: for every run, `git archive <rev>` was
extracted fresh into a new directory (`/tmp/close009/rep<r>/<workload>`) as an independent
disposable copy; a fresh per-repetition `GOCACHE` (`/tmp/close009/rep<r>/gocache`) was used; the
module cache (`/Users/imhttran/go/pkg/mod`) was shared read-only.
  Source: `CLOSE-009-performance-baseline-runs.md` §3; `CLOSE-009-operator-measurement-environment.json` `protocol`.
- The recorded execution mode is **sequential**.
  Source: `CLOSE-009-operator-measurement-environment.json` `protocol.execution_mode` = `sequential`.
- No command arguments were added; each workload ran exactly its contract command.
  Source: `CLOSE-009-operator-measurement-environment.json` `protocol.added_command_arguments` = `none (exact contract command)`.
- The `(cached)` marker appears in **none** of the 12 captured outputs, because each repetition
starts with an empty `GOCACHE` (and therefore an empty Go test-result cache).
  Source: `CLOSE-009-operator-measurement-evidence.md` §4; `CLOSE-009-operator-measurement-raw.jsonl` (stdout fields).
- The recorded toolchain and runtime environment:
  Go toolchain `go version go1.27.1 darwin/arm64`;
  `GOHOSTOS=darwin`, `GOHOSTARCH=arm64`, `GOMODCACHE=/Users/imhttran/go/pkg/mod`, `GOFLAGS=""`.
  Source: `CLOSE-009-operator-measurement-environment.json` `go_env`.
- The recorded host is
  `Darwin Hoangs-MacBook-Pro-2.local 25.6.0 Darwin Kernel Version 25.6.0: Fri Jul 31 19:18:53 PDT 2026; root:xnu-12377.161.14~5/RELEASE_ARM64_T6031 arm64`.
  Source: `CLOSE-009-operator-measurement-environment.json` `host`.
- The recorded SOP binary is `/Users/imhttran/go/bin/sop` (`sop dev`).
  Source: `CLOSE-009-operator-measurement-environment.json` `sop_binary`, `sop_version`.
- The recorded timing method is whole-command wall-clock duration, measured by the operator shell.
  Source: `CLOSE-009-operator-measurement-environment.json` `protocol.timing`.
- The CLOSE-009 report states there were no unproduced runs and that all 12 rows show exit 0 with an
observed duration.
  Source: `CLOSE-009-performance-baseline-runs.md` §4, §5.
- CLOSE-009 binds its evidence to source revision `90626f302868f6f0a7d7b6f644a62d6ac9d21159` and
archive SHA-256 `eb0f6918b440759120b66d5cde798bc7b1954d03928bfe5eab753d0e471d2887`.
  Source: `CLOSE-009-performance-baseline-runs.md` §1; `CLOSE-009-operator-measurement-evidence.md` §1.
- The CLOSE-008 telemetry inventory classifies persisted per-run telemetry as AVAILABLE for
`metrics.json`/`validation.json`/`review.json`/`state.json`/`trace.json`/`gate.json`, PARTIAL for
`report.json`/`jev.json`/`command-evidence.jsonl`, and UNAVAILABLE for the SOP lifecycle CLI read
surface; it states no inventoried gap prevents trustworthy measurement.
  Source: `docs/reports/pre-performance-closure/CLOSE-008-performance-telemetry-inventory.md`.
- This task executed no new workload runs and performed no new measurement; no SOP lifecycle CLI
verb was invoked for this report.
  Source: this task's scope; CLOSE-008 §2/§3.2 records the same CLI unavailability.

## 3. INFERRED

Each statement is a conclusion derived from the MEASURED/OBSERVED facts above; no statement here
is a measured figure.

- **Repeatability is supported under the recorded protocol.** The per-workload spreads
  (A 0.247 s, B 0.664 s, C 0.025 s, D 0.073 s — MEASURED, §1.2) are small relative to their medians
  (MEASURED, §1.2), so the three repetitions of each workload are consistent with one another under
  the recorded sequential, fresh-`GOCACHE`, independent-disposable-copy protocol (OBSERVED, §2).
- **The figures are whole-command wall-clock durations on a single host with no per-package
  breakdown.** The recorded timing method (OBSERVED, §2) and host (OBSERVED, §2) mean the MEASURED
  durations (§1.1) aggregate the whole command and cannot, by themselves, attribute time to
  individual packages; the raw `stdout` per-package test lines are present but were not used to
  derive any reported statistic.
- **Workload D is the slowest and workload C the fastest of the four, by median.** This follows from
  the MEASURED medians (§1.2: D 21.340 s, B 9.834 s, A 5.827 s, C 1.949 s) and is an ordering
  conclusion, not a new measurement.
- **The measured durations are host- and toolchain-specific.** Because the recorded host and Go
  toolchain (OBSERVED, §2) are fixed for the baseline, the MEASURED durations (§1.1) should not be
  assumed to transfer to a different host or toolchain without re-measurement.
- **Build/test timing is dominated by the heaviest test packages.** The raw test `stdout` (OBSERVED,
  §2) shows several packages taking ~6–8 s per run, consistent with the MEASURED aggregate durations
  (§1.1), but this is an inference and not a per-package measured result.

## 4. RECOMMENDED

These items are **non-binding future actions**, not established results. None is measured or
observed here.

- Re-run the CLOSE-009 baseline on any changed host, OS, or Go toolchain **before** comparing
  performance against this report, because the measured figures are host/toolchain-specific
  (INFERRED, §3).
- Track regressions against the revision-bound baseline (source revision
  `90626f302868f6f0a7d7b6f644a62d6ac9d21159`) rather than comparing to later, unbound measurements.
- If a per-package performance breakdown is needed, add dedicated per-package or profiling
  measurement; the whole-command wall-clock figures here (INFERRED, §3) are insufficient for it.
- Consider widening repetitions and/or adding warm-cache runs in a future baseline if lower-variance,
  regression-detection-oriented figures are desired; the current baseline is three repetitions with
  a fresh `GOCACHE` per repetition.
- Keep the CLOSE-009 evidence artifacts immutable; any new measurement should be recorded as new
  revision-bound evidence rather than editing `docs/reports/pre-performance-closure/`.

## 5. Approval gate

This report was generated automatically as a documentation/evidence deliverable. **It created no
human approval gate, and no human approval was required to create it.** No approval operation was
invoked, and no gate artifact was created or synthesized for this report.

## 6. Verdict

All CLOSE-010 acceptance criteria are satisfied: the deliverable exists; its content is separated
into MEASURED, OBSERVED, INFERRED, and RECOMMENDED sections with each statement placed by how it was
established; every measured figure cites its CLOSE-009 evidence identifier; nothing measured is
restated as inferred or vice versa; no approval gate was created or required; and exactly one
verdict line follows.

CLOSE-010 PERFORMANCE REPORT PASS

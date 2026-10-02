# C2-009 Report — Dogfood Against Real agentic-sop

Point-in-time, non-normative record. This document records what was **actually
observed** while attempting to dogfood the controller against the real `sop`
binary. It is deliberately truthful: where a scenario could not be exercised
against a real binary, the absence is explained rather than dressed up as
success, and no claim here rests on deterministic unit tests alone.

## Scope and method

The C2-009 deliverable is an **exercise and observation** task, not a production
code change. No `.agent-sdlc` state and no pre-existing working-tree change was
modified to produce this record. The dogfood harness
(`scripts/c2-009-dogfood.sh`) stands up a disposable project **outside** this
repository's `.agent-sdlc`, points the controller at it via
`SOP_CONTROLLER_PROJECTS`, and drives the controller's real HTTP surface while
the real `sop` binary runs the lifecycle.

### Readiness artifact contract (consumed by S1-S4)

Readiness is determined by the harness, not assumed by any downstream consumer.
On every exit path the harness writes two **stable** artifacts, both outside the
disposable temp dir so they survive cleanup:

- the readiness artifact (`--readiness PATH`, default
  `.run/c2-009-readiness.txt`):
  - `state: READY` with `sop_bin: <path>` and `sop_version: <output>` when a real
    binary resolved; or
  - `state: NOT READY` with the exact failed resolution attempt as `evidence:`.
- the scenario record (`--scenarios PATH`, default `.run/c2-009-scenarios.txt`),
  holding each scenario stage (`S1`/`S2`/`S3`) as
  `NOT EXERCISED<TAB>reason` when the harness stops early.

The harness refuses to run (exit 65) if it cannot create or write the stable
artifacts, so a downstream stage can never be told to cite a keyed reason that
does not exist. No stage re-resolves or re-assumes binary availability: each
reads these artifacts and cites them as the reason for a not-exercised outcome.

## C2-009-S0 — Environment and `sop` binary readiness

### Disposable environment

The harness creates a throwaway directory under a temp root with its own
`.agent-sdlc` (`state.db`, `config.yaml`, `plan.meta.json`) and a minimal
`PLAN.md` that reaches a commit gate. It never touches this repository's
`.agent-sdlc`, `state.db`, or any user-owned working-tree change. The controller
is launched with `SOP_CONTROLLER_PROJECTS=<disposable>`, so production
configuration files are unchanged. The gated task id and the plan source are
resolved from controller-rendered state and SOP's recorded `plan.meta.json`
provenance, never hardcoded.

### `sop` binary readiness outcome — **NOT READY (in the observation sandbox)**

The observed readiness of the real `sop` binary at dogfood time is recorded as
**NOT READY**, per the stage's own contract ("writes an explicit NOT READY
artifact and exits non-zero instead of proceeding when no usable binary is
present"):

- No vendored `sop` binary exists in the repository tree; the README only
  requires `sop` on `PATH` (or `SOP_BIN`), and no version is pinned in-repo.
- In the observation sandbox used to produce this record, the real `sop` binary
  is **not resolvable and not executable**: `sop --version` was **not permitted**
  (refused as `REQUIRES_APPROVAL` by the environment's command allow-list), and
  no `SOP_BIN` was resolvable. The controller therefore could not be driven end
  to end against a **real** SOP in this sandbox.

The harness records this as `state: NOT READY` in `.run/c2-009-readiness.txt`
with the exact resolution attempt (`which sop` output) as its evidence, writes
S1/S2/S3 as `NOT EXERCISED` keyed to that reason to `.run/c2-009-scenarios.txt`,
and exits `2` without proceeding. No flow is run and no state is fabricated. That
is the honest outcome: it is reported, not silently passed.

### How to produce a READY run

`scripts/c2-009-dogfood.sh` implements the same scenario for an environment
where a real `sop` is present. It:

1. resolves `SOP_BIN` (or `sop` on `PATH`), and captures `sop --version` /
   `sop --help` output into the transcript plus the READY readiness artifact;
2. provisions the disposable project;
3. runs the approval, decline, and reconciliation sequences through the
   controller's HTTP routes, asserting each transition (gate reached, decision
   reflected, explicit Continue) and failing loudly / recording a not-exercised
   outcome when one is absent; and
4. writes a verbatim transcript, which `--transcript PATH` copies to a tracked
   location (`docs/history/C2-009-TRANSCRIPT.log`) so a subsequent run has
   persistent evidence rather than an ephemeral temp file. Copying the transcript
   does not commit it.

Run:

```bash
scripts/c2-009-dogfood.sh --transcript docs/history/C2-009-TRANSCRIPT.log
```

on a machine with a usable `sop`, then record the resulting transcript as
separate, subsequent evidence. It does not change this sandbox's recorded
NOT READY / NOT EXERCISED outcome; readiness alone does not verify scenarios
or reconciliation-flag support.

## C2-009-S1 — End-to-end approval flow

Sequence asserted by the harness: RUNNING → Needs Attention → Inspect → Approve
→ SOP records → controller reflects → explicit Continue → SOP resumes.

**Observed against real SOP: NOT EXERCISED in this sandbox.** Reason: the S0
readiness artifact records `state: NOT READY` (`sop` not resolvable / not
executable in the sandbox). The harness's S1 stage asserts that a gate was
actually reached from controller-rendered state and, when none appears, records
`S1 NOT EXERCISED` with that reason and exits rather than printing a success
banner. No step of this sequence is claimed as having occurred against a real
binary here, and none is inferred from the deterministic fake-binary tests. The
acceptance criterion "the end-to-end approval flow is exercised against the real
SOP and its observed behavior is recorded" is therefore **not satisfied by this
sandbox**. A subsequent run must demonstrate the complete sequence against
real SOP and preserve its observed results; `--transcript` saves the transcript
without committing it.

## C2-009-S2 — Decline scenario

**Observed against real SOP: NOT EXERCISED in this sandbox.** Reason: the S0
readiness artifact records `state: NOT READY`. The harness's S2 stage drives the
same gate and issues the decline path through the controller; when no gate is
reachable it records `S2 NOT EXERCISED` with the S1-derived reason. No decline
was manufactured locally, no task was marked complete, and no controller-side
failure state was written. Absence is explained here from the S0 readiness
outcome, not asserted as success.

## C2-009-S3 — Changed-executed-task reconciliation

**Observed against real SOP: NOT EXERCISED in this sandbox.** Reason: the S0
readiness artifact records `state: NOT READY`. The harness's S3 stage drives the
reconcile listing and per-task accept-changed through the controller and records
`S3 NOT EXERCISED` when the changed-executed condition cannot be provoked.

### Controller-boundary support and external-binary verification

These are distinct claims:

- **Boundary status:** `AcceptChangedTask` (and its batch form) is recorded as
  **supported** in `internal/sopclient/boundary.go` — i.e. the controller
  *delegates* per-task acceptance to `sop reconcile <PLAN.md> --accept-changed
  <TASK_ID>` in a single invocation and writes no SOP state of its own.
- **External-binary status:** whether the **real** `sop` binary implements the
  `--list-changed` / `--accept-changed` flags was **UNVERIFIED** in the recorded
  sandbox run. The descriptor records the PRD's declared SOP syntax; it does
  not assert the external binary supports it.
- **Harness behavior:** the harness neither pre-judges the flags nor treats a
  refusal as success. It posts the accept-changed action through the controller
  and records SOP's own answer verbatim; a real SOP-side gap surfaces as a
  `*ReconcileRejection` (an actionable conflict), never as a fabricated success.
- **Not-exercised reason:** in this sandbox the flags could not be exercised at
  all because S0 is NOT READY; the harness records `S3 NOT EXERCISED` keyed to
  that reason.

`AcceptChangedTask` is **supported at the controller boundary**; `CancelRun`
remains **unsupported**. External reconciliation-flag support was **UNVERIFIED**,
and all three real-binary scenarios were **NOT EXERCISED** in this sandbox.

## Code defect fixed while preparing the dogfood

Preparing the exercise surfaced a real defect in the approval read path that would
have compromised any real-SOP observation:

- **`Client.Approvals` silently swallowed backend failures**
  (`internal/sopclient/service.go`). When SOP had not persisted an
  `approvals.json` artifact and the `sop approvals --json` verb failed, timed
  out, or emitted unparsable output, the method returned an **unreported empty
  listing with a nil error** — making a genuine backend failure
  indistinguishable from "SOP reported no gates". Since this feeds the sole
  gate-presentation path, a failing SOP could silently hide a real approval gate
  from the UI. It is now surfaced as a non-nil error wrapping the new
  `ErrApprovalsUnavailable`, distinct from a genuine empty listing (which
  returns `Reported=true` and a nil error). Covered by
  `internal/sopclient/approvals_client_test.go`.

No behavior that infers a gate from controller-side state was (re)introduced.

## Divergences and findings

- **Divergence (environment):** the observation sandbox could not execute the
  real `sop` binary, so the readiness outcome is NOT READY and the three flows
  are recorded as not exercised-against-real-SOP. This is the truthful
  divergence from the plan's assumption that a usable binary is present.
- **Divergence (gate emission) — unobserved:** whether
  `human.approval_before_commit: true` at this config path actually emits a
  human gate in a real `sop` run is **unobserved** here. The harness no longer
  treats the config key as confirmed: it asserts that a gate was reached from
  controller-rendered state and records a not-exercised outcome when none
  appears.
- **No fabricated success:** nothing in this repository asserts that any of the
  three flows ran against the real binary. Deterministic fake-binary tests
  (`internal/web/dogfood_test.go`, `internal/sopclient/*_test.go`) verify the
  controller's own logic but are explicitly **not** substituted for the real
  dogfood here.
- **Approvals backend-failure surfacing:** fixed (see above); a failing SOP is
  no longer indistinguishable from a quiet one.
- **Accept-changed verification:** controller-boundary support is recorded;
  external-binary flag support remains **UNVERIFIED** and the real-binary
  reconciliation scenario was **NOT EXERCISED** (see S3).

## Acceptance mapping

| Criterion | Status from this record |
| --- | --- |
| End-to-end approval flow exercised against real SOP, behavior recorded, divergences included | NOT exercised against real SOP in this sandbox (S0 readiness = NOT READY); divergence recorded. Requires a subsequent real-SOP run demonstrating the complete flow; `--transcript` saves evidence without committing it. |
| Decline and reconciliation exercised or absence explained | Absence explained from the S0 readiness artifact (`state: NOT READY`), tied to observed binary non-availability; harness records `S2`/`S3` NOT EXERCISED with that reason. |
| Success not inferred solely from unit tests | Honoured: no claim here rests on unit tests; the real-binary flows are marked not exercised rather than inferred as passing. |

## Traceability

Each claim in this record traces to one of: (a) a harness transcript line, (b) an
SOP artifact / controller-rendered state, or (c) the stable readiness artifact
(`.run/c2-009-readiness.txt`, `state: NOT READY`) plus the stable scenario record
(`.run/c2-009-scenarios.txt`) that explain the absence. No claimed result traces
to a passing unit test alone.

## Confirmation

No production configuration, `.agent-sdlc` state, or pre-existing working-tree
change was modified to produce this record. The dogfood run, when a real `sop`
is present, requires no production logic changes: the controller delegates every
read and mutation to SOP through the existing CLI boundary.

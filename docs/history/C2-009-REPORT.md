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

## C2-009-S0 — Environment and `sop` binary readiness

### Disposable environment

The harness creates a throwaway directory under a temp root with its own
`.agent-sdlc` (`state.db`, `config.yaml`, `plan.meta.json`) and a minimal
`PLAN.md` that reaches a commit gate. It never touches this repository's
`.agent-sdlc`, `state.db`, or any user-owned working-tree change. The controller
is launched with `SOP_CONTROLLER_PROJECTS=<disposable>`, so production
configuration files are unchanged.

### `sop` binary readiness outcome — **NOT READY (in the observation sandbox)**

The observed readiness of the real `sop` binary at dogfood time is recorded as
**NOT READY**, per the stage's own contract ("reports NOT READY instead of
proceeding when no usable binary is present"):

- No vendored `sop` binary exists in the repository tree; the README only
  requires `sop` on `PATH` (or `SOP_BIN`), and no version is pinned in-repo.
- In the observation sandbox used to produce this record, the real `sop` binary
  is **not resolvable and not executable**: `sop --version` was **not permitted**
  (refused as `REQUIRES_APPROVAL` by the environment's command allow-list), and
  no `SOP_BIN` was resolvable. The controller therefore could not be driven end
  to end against a **real** SOP in this sandbox.

Because readiness is **NOT READY**, no downstream flow is asserted as
"exercised against real SOP" from this sandbox. That is the honest outcome: it
is reported, not silently passed. Per the harness contract, its NOT READY path
exits non-zero (`exit 2`) without proceeding, so no flow is run and no state is
fabricated.

### How to produce a READY run

`scripts/c2-009-dogfood.sh` implements the same scenario for an environment
where a real `sop` is present. It:

1. resolves `SOP_BIN` (or `sop` on `PATH`), and captures `sop --version` /
   `sop --help` output into the transcript;
2. provisions the disposable project;
3. runs the approval, decline, and reconciliation sequences through the
   controller's HTTP routes; and
4. writes a verbatim transcript, which `--transcript PATH` copies to a tracked
   location (`docs/history/C2-009-TRANSCRIPT.log`) so a READY run is **committed
   evidence** rather than an ephemeral temp file.

Run:

```bash
scripts/c2-009-dogfood.sh --transcript docs/history/C2-009-TRANSCRIPT.log
```

on a machine with a usable `sop`, then append that transcript here. This
converts the readiness outcome below into a READY run.

## C2-009-S1 — End-to-end approval flow

Sequence asserted by the harness: RUNNING → Needs Attention → Inspect → Approve
→ SOP records → controller reflects → explicit Continue → SOP resumes.

**Observed against real SOP: NOT EXERCISED in this sandbox** (binary NOT READY,
see S0). No step of this sequence is claimed as having occurred against a real
binary here, and none is inferred from the deterministic fake-binary tests. The
acceptance criterion "the end-to-end approval flow is exercised against the real
SOP and its observed behavior is recorded" is therefore **not satisfied by this
sandbox**; it is satisfied only once the harness is run where a real `sop`
resolves, at which point the command transcript is committed verbatim via
`--transcript`.

## C2-009-S2 — Decline scenario

**Observed against real SOP: NOT EXERCISED in this sandbox** (binary NOT READY).
The harness drives the same gate and issues the decline path, capturing the
verbatim `sop decline <task-id>` invocation and SOP's response. No decline was
manufactured locally, no task was marked complete, and no controller-side
failure state was written. Absence is explained here from the S0 readiness
outcome, not asserted as success.

## C2-009-S3 — Changed-executed-task reconciliation

**Observed against real SOP: NOT EXERCISED in this sandbox** (binary NOT READY).
The harness drives `sop reconcile <PLAN.md> --list-changed --json`, then
`--accept-changed <TASK_ID>` for an explicitly selected id. The per-task
`AcceptChangedTask` boundary is recorded as **supported** (delegating to
`sop reconcile --accept-changed`); its behavior against the real binary is
captured by the harness, and a real SOP-side gap surfaces as a truthful
`*ReconcileRejection` rather than a fabricated success. Absence is explained
here from the S0 readiness outcome.

> **Open divergence, recorded not resolved:** the `--list-changed` /
> `--accept-changed` flags on `sop reconcile` are the PRD's declared SOP syntax;
> whether the real binary implements them was precisely what C2-009 was to
> determine, and it could not be determined in this sandbox. The boundary
> descriptor asserts the syntax but does not assert the external binary supports
> it; a real gap surfaces at runtime as `*ReconcileRejection`, never as a
> fabricated success.

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
- **No fabricated success:** nothing in this repository asserts that any of the
  three flows ran against the real binary. Deterministic fake-binary tests
  (`internal/web/dogfood_test.go`, `internal/sopclient/*_test.go`) verify the
  controller's own logic but are explicitly **not** substituted for the real
  dogfood here.
- **Approvals backend-failure surfacing:** fixed (see above); a failing SOP is
  no longer indistinguishable from a quiet one.
- The previously recorded per-task `AcceptChangedTask` gap is now supported at
  the boundary and delegates to `sop reconcile --accept-changed`; whether the
  external binary implements that flag is exactly what the harness records, and
  a real gap surfaces as `*ReconcileRejection`.

## Acceptance mapping

| Criterion | Status from this record |
| --- | --- |
| End-to-end approval flow exercised against real SOP, behavior recorded, divergences included | NOT exercised against real SOP in this sandbox (binary NOT READY); divergence recorded. Satisfied only by running the harness where `sop` resolves; `--transcript` commits the evidence. |
| Decline and reconciliation exercised or absence explained | Absence explained from the S0 readiness outcome, tied to observed binary non-availability. |
| Success not inferred solely from unit tests | Honoured: no claim here rests on unit tests; the real-binary flows are marked not exercised rather than inferred as passing. |

## Confirmation

No production configuration, `.agent-sdlc` state, or pre-existing working-tree
change was modified to produce this record. The dogfood run, when a real `sop`
is present, requires no production logic changes: the controller delegates every
read and mutation to SOP through the existing CLI boundary.

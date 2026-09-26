# OpenJEV Specification

**Type:** Normative specification

## Purpose

Defines the OpenJEV integration contract for SOP Controller: what OpenJEV is,
what the controller reads and displays today, and what remains proposed.

The review lifecycle is defined separately in [REVIEW.md](REVIEW.md). This
document defines the OpenJEV-specific contract and does not repeat review
requirements.

## Related Specifications

- [REVIEW.md](REVIEW.md)
- [EXECUTION.md](EXECUTION.md)
- [WORKFLOW.md](WORKFLOW.md)
- [HUMAN-APPROVAL.md](HUMAN-APPROVAL.md)

## Normative Language

The terms MUST, MUST NOT, SHOULD, SHOULD NOT, and MAY are normative.

## Terminology and Ownership

**OpenJEV** (JEV) is SOP's optional, read-only engineering-analysis capability.
It produces an additional quality signal; it has no authority over any lifecycle
decision.

- SOP owns OpenJEV, its configuration, and its verdicts. The controller MUST NOT
  run OpenJEV, MUST NOT decide its outcome, and MUST NOT treat it as a workflow
  engine.
- The controller MUST report OpenJEV status and findings as SOP persisted them,
  and MUST NOT infer or invent an OpenJEV result.

## Integration Boundary

The controller observes OpenJEV through SOP's persisted run artifacts; it never
invokes OpenJEV directly.

```text
SOP  --- owns ---> OpenJEV analysis      (runs inside SOP's lifecycle)
SOP  --- persists --> run report / JEV summary
controller --- reads (read-only) --> displays JEV status, findings, quality
```

- The controller MUST read the OpenJEV summary only through
  `internal/sopclient` (the SOP boundary) and MUST NOT read OpenJEV sources
  directly.
- The controller MUST present the OpenJEV verdict distinctly from validation,
  review, and the quality status, and MUST NOT merge them into one verdict.

## Current Integration (Implemented)

What exists today:

- **Input** — SOP's run report artifact
  (`.agent-sdlc/runs/<task>/report.json`), whose `jev` object carries SOP's
  OpenJEV summary.
- **Output shown** — OpenJEV status, summary, reason, and the number of
  findings.
- **Quality status** — SOP's OpenJEV summary is reduced to a JEV/quality status
  for display. An explicit `PASS` reads as a pass; a missing artifact is
  reported as absent (`Present=false`), never as a pass.
- **Absence** — when SOP persisted no OpenJEV analysis for a run, the panel
  shows an explicit "not run" state.
- **Activity** — SOP may emit an OpenJEV activity event (action kind `jev`) in
  the structured activity stream, which the controller displays like any other
  activity event. See [ACTIVITY.md](ACTIVITY.md).

## Feature Flags

OpenJEV is configured in SOP, not in the controller. The controller MUST read
the resulting behavior, and MUST NOT expose its own OpenJEV switch.

| Location (SOP config) | Meaning |
| --- | --- |
| `quality.jev.enabled` | OpenJEV feature flag; **disabled by default**, enabling requires explicit configuration |
| `quality.jev.mode` | Execution form; only `review` is implemented today |
| `quality.jev.fail_on` | Finding severities that block; defaults to `quality.fail_on` when unset |

The controller MUST reflect whether OpenJEV ran for a task (present/absent) and
MUST NOT assume it is enabled.

## Findings and Severity

- OpenJEV findings and their severities are owned by SOP. The controller MUST
  display them as SOP reported them.
- The controller MUST NOT compute finding severities or re-rank findings.
- The blocking severities for OpenJEV are SOP's configured severities
  (`quality.jev.fail_on`, defaulting to `quality.fail_on`).

## Quality Gates

- The quality gate is deterministic and owned by SOP; it combines verification,
  review findings, OpenJEV findings, and the fix-loop budget.
- The controller MUST display SOP's quality decision and MUST NOT compute or
  override the gate.
- The controller MUST NOT let an absent OpenJEV result satisfy a gate.

## Failure Behavior

- If OpenJEV is disabled, unavailable, times out, returns malformed output, or
  produces a low-confidence result, SOP's own policy decides the fallback. The
  controller MUST report what SOP did (for example an absent OpenJEV result or
  a `NEEDS_HUMAN` disposition) and MUST NOT invent a fallback.

## Interaction with REVIEW

- OpenJEV and review are separate signals with separate owners. The controller
  MUST present them distinctly. See [REVIEW.md](REVIEW.md).

## Interaction with EXECUTION

- OpenJEV is part of SOP's execution lifecycle (a quality/JEV stage). The
  controller reports it as part of the run, like any other stage, and MUST NOT
  drive it. See [EXECUTION.md](EXECUTION.md).

## Proposed / Future (Not Implemented)

The following are **not implemented** in the controller and are recorded here as
candidate extensions. They MUST NOT be presented as current behavior.

- **Earlier pipeline checkpoints** — running OpenJEV at earlier pipeline points
  (for example before review, or on a partial change set) rather than only as
  part of SOP's quality stage.
- **Richer findings display** — per-finding navigation, severity filters, or
  links from a finding to the change it concerns.
- **OpenJEV-driven routing** — surfacing SOP decision-layer outputs that use
  OpenJEV as a decision provider. These remain SOP-owned; the controller would
  only display them.

Until implemented, the controller's OpenJEV behavior is limited to the
"Current Integration (Implemented)" section above.

## Related Documentation

- [REVIEW.md](REVIEW.md) — the review lifecycle.
- [EXECUTION.md](EXECUTION.md) — execution reporting.
- [ACTIVITY.md](ACTIVITY.md) — the activity stream that carries OpenJEV events.
- [../reference/STATUS-AND-RECOVERY.md](../reference/STATUS-AND-RECOVERY.md) —
  quality/recovery values.

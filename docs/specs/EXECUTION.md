# Execution Specification

**Type:** Normative specification

## Purpose

Defines the required behavior for how SOP Controller reports SOP task execution:
run stages, change detection, quality decisions, and structured outcomes.

## Related Specifications

- [WORKFLOW.md](WORKFLOW.md)
- [AGENT-PROVIDER.md](AGENT-PROVIDER.md)
- [REVIEW.md](REVIEW.md)
- [../reference/STATUS-AND-RECOVERY.md](../reference/STATUS-AND-RECOVERY.md)

## Normative Language

The terms MUST, MUST NOT, SHOULD, SHOULD NOT, and MAY are normative.

## Reporting the Run

- The controller MUST report the latest run's SOP lifecycle stage verbatim (for
  example `IMPLEMENTING`, `VALIDATING`, `REVIEWING`, `PASSED`, `FAILED`) and MUST
  NOT construct or invent a stage.
- The controller MUST report SOP's quality decision, fix-cycle count,
  execution mode, provider, and model verbatim as SOP persisted them.
- The controller MUST read run artifacts read-only and MUST NOT write them.
- A missing run artifact MUST be reported as absent, never as a pass.

## Change Detection

- The controller MUST rely on SOP's change detection, which includes relevant
  **untracked** files and ignores SOP's own `.agent-sdlc/` output.
- The controller MUST NOT stage or otherwise alter the working tree to make
  files visible to SOP.
- The controller MUST reflect SOP's reported change state (for example an
  adapter's `changes_expected`) rather than computing its own repository diff.

## Structured Execution Outcomes

- For implementation capabilities, the controller MUST depend on SOP acting on
  a **structured execution outcome**, not on prose.
- The controller MUST NOT parse an agent's prose to infer success or failure;
  the outcome is SOP's contract (see [AGENT-PROVIDER.md](AGENT-PROVIDER.md)).
- A `completed` outcome with `changes_expected: false` is valid for a legitimate
  verification or read-only task; the controller MUST NOT treat the absence of a
  change as failure.

## Verification-First Tasks

- A verification task MAY pass without forcing artificial changes when SOP
  records a `completed` outcome with `changes_expected: false`.
- The controller MUST present such a task as completed per SOP, and MUST NOT
  require that a change exist.

## Failures and Recovery

- The controller MUST report SOP's failure classification and recovery
  disposition verbatim, and MUST NOT decide recovery itself. See
  [WORKFLOW.md](WORKFLOW.md) and
  [../reference/STATUS-AND-RECOVERY.md](../reference/STATUS-AND-RECOVERY.md).

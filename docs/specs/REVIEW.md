# Review Specification

**Type:** Normative specification

## Purpose

Defines the review lifecycle the controller displays: how review findings,
severity, resolution status, and remediation progress are surfaced, and how the
controller may delegate review.

OpenJEV (JEV) is a related but distinct signal and is specified separately in
[OPENJEV.md](OPENJEV.md).

## Related Specifications

- [OPENJEV.md](OPENJEV.md)
- [EXECUTION.md](EXECUTION.md)
- [WORKFLOW.md](WORKFLOW.md)

## Normative Language

The terms MUST, MUST NOT, SHOULD, SHOULD NOT, and MAY are normative.

## Ownership

- Review is owned by SOP. The controller MUST NOT perform review itself and MUST
  NOT reimplement review logic.
- The only review action the controller MAY offer MUST delegate to SOP's own
  review command (`sop review`); see
  [../reference/CLI.md](../reference/CLI.md).

## What the Controller Reports

- The controller MUST display review findings and their severity as SOP
  persisted them.
- The controller MUST display each finding's resolution status and the
  remediation progress as SOP persisted them.
- The controller MUST show which severities block, based on SOP's configured
  blocking severities (`quality.fail_on` in `.agent-sdlc/config.yaml`), and MUST
  NOT invent its own blocking policy.
- The controller MUST NOT treat an absent review artifact as a pass.

## Review Engine

- The review engine is SOP configuration (`review.engine`: `self` or
  `open-code-review`, plus `review.delegation`). The controller MUST report the
  configured engine and its outcome, and MUST NOT choose the engine.

## Relationship to OpenJEV

- Review findings and OpenJEV findings are separate signals with separate
  owners. The controller MUST present them distinctly and MUST NOT merge them
  into a single verdict. See [OPENJEV.md](OPENJEV.md).

## Related Documentation

- [OPENJEV.md](OPENJEV.md) — the OpenJEV integration contract.
- [EXECUTION.md](EXECUTION.md) — how the run and its outcomes are reported.
- [../reference/STATUS-AND-RECOVERY.md](../reference/STATUS-AND-RECOVERY.md) —
  recovery dispositions after a failed review.

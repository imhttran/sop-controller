# Agent Provider Specification

**Type:** Normative specification

## Purpose

Defines the agent/provider contract SOP Controller depends on: the SOP command
provider, the structured execution outcomes it returns, and the constraints the
controller must respect.

The controller never runs an agent itself. This document records the SOP-side
contract the controller relies on so the two stay compatible.

## Related Specifications

- [EXECUTION.md](EXECUTION.md)
- [WORKFLOW.md](WORKFLOW.md)

## Normative Language

The terms MUST, MUST NOT, SHOULD, SHOULD NOT, and MAY are normative.

## Ownership

- SOP is provider-neutral and owns the agent harness and model selection. The
  controller MUST NOT select a model or provider and MUST NOT run an agent.
- The controller MUST report the provider and model SOP persisted, verbatim.
- The controller MUST NOT depend on model- or provider-specific behavior.

## The Command Provider

SOP drives this project through its `command` provider
(`.agent-sdlc/config.yaml`, `agent.harness: command`).

- The command provider MUST be an **editing agent**: it reads the
  working-tree change set as the implementation and acts on a structured
  execution outcome. A text-only model provider cannot implement.
- The adapter is a thin seam. `scripts/sop-agent.sh` contains no scheduling,
  retry, validation, or state logic; it forwards each capability to a pluggable
  implementation agent (Claude Code by default, overridable with
  `SOP_AGENT_IMPL`).

```bash
export SOP_AGENT_PROVIDER=command
export SOP_AGENT_COMMAND="sh scripts/sop-agent.sh"
```

## Adapter Protocol

Request (JSON on stdin):

```json
{"capability": "...", "task": "...", "input": "...", "output_requirements": "..."}
```

Response (stdout), per capability:

- `PLAN` / `REVIEW` → the structured JSON SOP parses.
- `IMPLEMENT` / `FIX` / `DESIGN_TESTS` → edits the tree and prints a
  **structured execution outcome** as its only stdout.
- other capabilities → prose.

## Structured Execution Outcomes

The outcome forms SOP acts on:

```json
{"status": "completed", "summary": "…", "changes_expected": true}
{"status": "needs_human", "reason": "…"}
{"status": "failed", "reason": "…"}
```

- `changes_expected` MUST reflect the actual working-tree change, so a
  `completed` outcome with `changes_expected: false` lets a verification task
  pass without forcing artificial changes.
- The controller MUST NOT parse an agent's prose to infer success or failure,
  and MUST NOT re-derive the outcome; SOP interprets the structured outcome.
  See [EXECUTION.md](EXECUTION.md).

## Change Detection

- SOP recognizes relevant untracked files without staging; its change detection
  includes untracked files and ignores its own `.agent-sdlc/` output.
- The controller and the adapter MUST NOT stage files just to make them visible
  to SOP. See [EXECUTION.md](EXECUTION.md).

## Related Documentation

- [EXECUTION.md](EXECUTION.md) — change detection and structured outcomes.
- [../reference/CONFIGURATION.md](../reference/CONFIGURATION.md) — controller
  configuration (the SOP agent is configured in SOP, not here).

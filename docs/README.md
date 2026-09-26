# SOP Controller — Documentation

Central map of SOP Controller documentation. Each entry has a one-sentence
description so a human or an AI agent can decide what to read.

**New here?** Start with [guides/GETTING-STARTED.md](guides/GETTING-STARTED.md).

## Documentation Authority

When documents disagree, authority flows in this order:

```text
Product intent
      │
      ▼
requirements/PRD.md      — what we are building and why
      │
      ▼
specs/*.md               — how the system MUST behave
      │
      ▼
architecture/*.md        — structural boundaries and design constraints
      │
      ▼
plans/*.md               — what work needs to happen
      │
      ▼
SOP execution            — SOP remains the workflow authority
```

Plans MUST NOT override specifications. Historical artifacts provide
traceability only and MUST NOT override current specifications.

## Getting Started

- [../README.md](../README.md) — project landing page: what SOP Controller is,
  its architecture, requirements, and quick start.
- [../AGENTS.md](../AGENTS.md) — routing guide for implementation/review agents.

## Requirements

- [requirements/PRD.md](requirements/PRD.md) — what problem we are solving, the
  goals, functional requirements, constraints, and success criteria.

## Architecture

- [architecture/OVERVIEW.md](architecture/OVERVIEW.md) — system structure,
  components, data flow, and repository layout.
- [architecture/SOP-BOUNDARY.md](architecture/SOP-BOUNDARY.md) — the SOP boundary:
  why SOP remains the workflow authority and the controller must not become a
  second source of truth.

## Specifications

Normative descriptions of required behavior (MUST / SHOULD / MAY).

- [specs/WORKFLOW.md](specs/WORKFLOW.md) — how the controller reflects SOP's
  workflow: status vocabulary, dependencies, scheduling ownership, restart safety.
- [specs/EXECUTION.md](specs/EXECUTION.md) — how the controller reports SOP task
  execution: run stages, change detection, and structured execution outcomes.
- [specs/REVIEW.md](specs/REVIEW.md) — the review lifecycle the controller
  displays, including findings, severity, and remediation progress.
- [specs/OPENJEV.md](specs/OPENJEV.md) — the OpenJEV (JEV) integration contract:
  what the controller reads and displays, and what remains proposed.
- [specs/HUMAN-APPROVAL.md](specs/HUMAN-APPROVAL.md) — the human-approval boundary:
  when the controller shows a human gate and how approval actions delegate to SOP.
- [specs/ACTIVITY.md](specs/ACTIVITY.md) — the live activity stream: source,
  delivery, cursor semantics, and safe-summary guarantees.
- [specs/AGENT-PROVIDER.md](specs/AGENT-PROVIDER.md) — the agent/provider contract
  the controller depends on: the SOP command provider and structured outcomes.
- [specs/SECURITY.md](specs/SECURITY.md) — local-first binding, network mode,
  CSRF, discovery boundary, and secret handling.

## Plans

What work needs to happen. Plans describe implementation work and MUST NOT
override specifications.

- [PLAN-SOP-Controller.md](PLAN-SOP-Controller.md) — **active** run-control and
  live-activity plan (CTRL001–CTRL017). SOP records this as the active plan, so it
  intentionally remains here and is executed via `sop run docs/PLAN-SOP-Controller.md`.
- [plans/PLAN-Wrap-Up.md](plans/PLAN-Wrap-Up.md) — integration verification
  workflow (WRAP-001–WRAP-012). Execute via `sop run docs/plans/PLAN-Wrap-Up.md`.
- [plans/PLAN-Hardening.md](plans/PLAN-Hardening.md) — controller hardening
  workflow (SC-001–SC-014).
- [plans/PLAN-ORGANIZATION.md](plans/PLAN-ORGANIZATION.md) — plan purposes, naming
  conventions, and documentation-organization guidelines.
- [plans/PRECHECK.md](plans/PRECHECK.md) — pre-execution validation checklist.

## Reference

Concrete interfaces, values, commands, and states.

- [reference/CONFIGURATION.md](reference/CONFIGURATION.md) — every environment
  variable, its default, and the command timeout.
- [reference/CLI.md](reference/CLI.md) — how to run the controller and the SOP CLI
  commands and HTTP routes the dashboard exposes.
- [reference/STATUS-AND-RECOVERY.md](reference/STATUS-AND-RECOVERY.md) — task
  statuses, run stages, recovery dispositions, and recovery actions.

## Guides

- [guides/GETTING-STARTED.md](guides/GETTING-STARTED.md) — install, run, and open
  the dashboard, including optional phone access.
- [guides/PROJECT-DISCOVERY.md](guides/PROJECT-DISCOVERY.md) — configure which
  SOP projects the controller observes, explicitly or by workspace discovery.
- [guides/DEVELOPMENT.md](guides/DEVELOPMENT.md) — build, test, format, and
  develop the controller.

## Historical Documentation

Point-in-time verification artifacts kept for traceability. Non-normative: they
do not define current behavior.

- [history/CTRL001/BOUNDARY-CONTRACT.md](history/CTRL001/BOUNDARY-CONTRACT.md) —
  the controller-to-SOP boundary contract (operation table and tests).
- [history/CTRL001/BOUNDARY-INVENTORY.md](history/CTRL001/BOUNDARY-INVENTORY.md) —
  inventory of controller reads and commands against the boundary.
- [history/CTRL001/BOUNDARY-VERIFICATION.md](history/CTRL001/BOUNDARY-VERIFICATION.md) —
  verification of the CTRL001 boundary.
- [history/PLAN-IDENTITY.md](history/PLAN-IDENTITY.md) — plan identity and
  named-plan resolution.
- [history/PLAN.md](history/PLAN.md) — the historical v1 implementation plan.
- [history/WRAP/](history/WRAP/) — wrap-up verification reports
  ([WRAP-006](history/WRAP/WRAP-006-VERIFICATION.md),
  [WRAP-007](history/WRAP/WRAP-007-VERIFICATION.md),
  [WRAP-008](history/WRAP/WRAP-008-VERIFICATION.md),
  [WRAP-009](history/WRAP/WRAP-009-VERIFICATION.md),
  [WRAP-011](history/WRAP/WRAP-011-VERIFICATION.md),
  [WRAP-012](history/WRAP/WRAP-012-FINAL-REPORT.md)).

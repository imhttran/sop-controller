# AGENTS.md

Routing guide for implementation and review agents working on SOP Controller.

This is a pointer file, not a README. Start at the [documentation index](docs/README.md).

## Before changing behavior

1. Read the relevant product requirement: [docs/requirements/PRD.md](docs/requirements/PRD.md).
2. Read the relevant specification(s): [docs/specs/](docs/specs/).
3. Read the architecture constraints that apply: [docs/architecture/](docs/architecture/).
4. Read the active implementation plan: [docs/PLAN-SOP-Controller.md](docs/PLAN-SOP-Controller.md).
5. Do not treat [docs/history/](docs/history/) as current specifications — it is point-in-time and non-normative.
6. SOP remains the workflow authority: read SOP state and delegate commands; never mutate SOP state or reimplement workflow logic (see [docs/architecture/SOP-BOUNDARY.md](docs/architecture/SOP-BOUNDARY.md)).

## Task-to-document mapping

```text
Execution changes:
  docs/specs/EXECUTION.md
  docs/specs/WORKFLOW.md
  docs/architecture/SOP-BOUNDARY.md

Review changes:
  docs/specs/REVIEW.md

OpenJEV changes:
  docs/specs/OPENJEV.md
  docs/specs/REVIEW.md

Approval changes:
  docs/specs/HUMAN-APPROVAL.md
  docs/architecture/SOP-BOUNDARY.md

Activity / streaming changes:
  docs/specs/ACTIVITY.md
  docs/reference/CLI.md

Agent / provider changes:
  docs/specs/AGENT-PROVIDER.md

Security changes:
  docs/specs/SECURITY.md

CLI / config changes:
  docs/reference/

Workflow status / recovery changes:
  docs/specs/WORKFLOW.md
  docs/reference/STATUS-AND-RECOVERY.md
```

## Authority order

```text
requirements/PRD.md  →  specs/*.md  →  architecture/*.md  →  plans/*.md  →  SOP execution
```

Specifications define required behavior. Plans describe work and MUST NOT
override specifications. Historical artifacts provide traceability only.

## Repository layout

```text
docs/requirements/   product intent (PRD)
docs/architecture/   structural boundaries and ownership
docs/specs/          normative required behavior
docs/plans/          implementation plans
docs/reference/      commands, states, and values
docs/guides/         how to use or develop
docs/history/        non-normative point-in-time artifacts
```

The SOP-recorded **active** plan stays at `docs/PLAN-SOP-Controller.md` (SOP
matches plans by recorded source path); do not move it while it is active.

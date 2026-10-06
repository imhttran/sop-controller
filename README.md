# SOP Controller

A local-first web dashboard for **SOP**. It gives a human a friendly
operational view of SOP projects — task state, execution progress, review
results, CI/validation state, handoff context, and failures — and exposes safe
SOP commands.

> **SOP stays the workflow authority.** The dashboard observes and commands SOP;
> it never keeps a second source of truth for task state.

## High-Level Architecture

```text
Browser
   |
   v
Go HTTP server  (html/template + HTMX + CSS)
   |
   v
SOP application/API boundary   internal/sopclient
   |
   v
SOP authoritative state        <project>/.agent-sdlc/state.db  (read-only)
                               sop CLI                        (commands)
```

Two planes, one authority:

- **`agentic-sop` / SOP** — automation and execution plane. SOP owns workflow
  state, legal state transitions, scheduling, gates, and human-approval
  boundaries.
- **`sop-controller`** — human visibility and control plane. It reads SOP's
  persisted state and delegates every action to SOP.

Read more: [docs/architecture/OVERVIEW.md](docs/architecture/OVERVIEW.md) and
[docs/architecture/SOP-BOUNDARY.md](docs/architecture/SOP-BOUNDARY.md).

## Requirements

- **Go** (1.24+). That is it at build/run time.
- The **`sop` CLI** on `PATH` (or `SOP_BIN`) for the Run/Resume/Validate/Review
  commands. Read-only views work without it.

No Node, no npm, no PostgreSQL. Server-rendered HTML plus HTMX — no SPA.

## Quick Start

```bash
go run ./cmd/sop-controller     # foreground
# → http://127.0.0.1:8080
```

By default it reads SOP state from the current directory
(`./.agent-sdlc/state.db`). Point it at other projects with
`SOP_CONTROLLER_PROJECTS=...`, or let it discover every SOP project under a
workspace root with `SOP_CONTROLLER_WORKSPACES=...`.

Full setup, including phone access: [docs/guides/GETTING-STARTED.md](docs/guides/GETTING-STARTED.md).

## Basic Run Commands

```bash
make start     # build + run in the background
make status    # is it up? (also pings /healthz)
make logs      # tail the background log
make stop      # shut it down
```

`make help` lists every target. The dashboard itself drives SOP CLI commands
(`run`, `resume`, `validate`, `review`, `report`, `retry`, `reconcile`); see
[docs/reference/CLI.md](docs/reference/CLI.md).

## Documentation

Start at the documentation index: **[docs/README.md](docs/README.md)**.

Agent routing guide: [AGENTS.md](AGENTS.md).

Major guides:

- [Getting started](docs/guides/GETTING-STARTED.md)
- [Project discovery](docs/guides/PROJECT-DISCOVERY.md)
- [Development](docs/guides/DEVELOPMENT.md)

Reference:

- [Configuration](docs/reference/CONFIGURATION.md)
- [CLI and routes](docs/reference/CLI.md)
- [Status and recovery](docs/reference/STATUS-AND-RECOVERY.md)

Specifications:

- [Workflow](docs/specs/WORKFLOW.md)
- [Execution](docs/specs/EXECUTION.md)
- [Review](docs/specs/REVIEW.md)
- [OpenJEV](docs/specs/OPENJEV.md)
- [Human approval](docs/specs/HUMAN-APPROVAL.md)
- [Activity](docs/specs/ACTIVITY.md)
- [Agent provider](docs/specs/AGENT-PROVIDER.md)
- [Security](docs/specs/SECURITY.md)

## Requirements and Plans

- [docs/requirements/PRD.md](docs/requirements/PRD.md) — product requirements

Plans (executed by SOP):

- [docs/PLAN-SOP-Controller.md](docs/PLAN-SOP-Controller.md) — **active**
  run-control and live-activity plan (drives the current work). SOP records this
  as the active plan, so it intentionally stays at `docs/PLAN-SOP-Controller.md`.
- [docs/plans/PLAN-Wrap-Up.md](docs/history/plans/PLAN-Wrap-Up.md) — integration verification workflow
  - Primary execution: `sop run docs/plans/PLAN-Wrap-Up.md`
- [docs/plans/PLAN-Hardening.md](docs/history/plans/PLAN-Hardening-2.md) — controller hardening workflow
  - Optional: `sop run docs/plans/PLAN-Hardening.md`
- [docs/plans/PLAN-ORGANIZATION.md](docs/history/plans/PLAN-ORGANIZATION.md) — plan structure,
  naming conventions, and guidelines
- [docs/plans/PRECHECK.md](docs/history/plans/PRECHECK.md) — pre-execution validation checklist

Historical reference: [docs/history/PLAN.md](docs/history/PLAN.md) — the v1
implementation plan.

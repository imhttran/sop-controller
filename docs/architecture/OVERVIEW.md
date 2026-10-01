# Architecture Overview

**Type:** Architecture documentation

## Purpose

Describes the structure of SOP Controller: its components, the flow of data
through the system, and where its information comes from.

For the ownership boundary between SOP and the controller, see
[SOP-BOUNDARY.md](SOP-BOUNDARY.md).

## System Context

SOP Controller is a local-first web dashboard for **SOP**. It is a presentation
and control surface, not a second orchestrator.

```text
        human
          |
          v
   +-----------------+          +--------------------------+
   | sop-controller   |  read    | SOP authoritative state  |
   | (this project)   | -------> | .agent-sdlc/state.db     |
   |  visibility +    |          | .agent-sdlc/runs/*       |
   |  human control   |  command | .agent-sdlc/plan.meta.json|
   |                  | -------> | sop CLI (run/resume/...) |
   +-----------------+          +--------------------------+
                                         |
                                         v
                                 agentic-sop / SOP
                                 (execution + workflow authority)
```

- **SOP** (the `agentic-sop` project) is the automation and execution plane. It
  owns workflow state, legal transitions, scheduling, gates, and human-approval
  boundaries.
- **SOP Controller** (this project) is the human visibility and control plane.
  It reads SOP's persisted state and delegates every action to SOP.

## Runtime Architecture

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

The browser never reaches SQLite. Every read and command goes through the
server, and every SOP interaction goes through `internal/sopclient`.

## Components

```text
cmd/sop-controller/    main: load config, wire handlers, serve
internal/config/       environment configuration + project discovery
internal/sopclient/    SOP application/API boundary (reads state, runs commands)
internal/web/          HTTP handlers, middleware, rendering, activity streaming
templates/             html/template views + partials
static/                app.css + htmx.min.js (embedded)
scripts/sop-agent.sh   SOP command-agent adapter (implementation-agent seam)
docs/                  documentation (this tree)
```

### `internal/sopclient` — the SOP boundary

The single place the controller reads SOP state and invokes SOP commands. It
exposes read models (projects, tasks, activity, run artifacts, plan metadata)
and command delegates (run, resume, validate, review, report, retry, reconcile).
No web handler opens `state.db` directly. See
[SOP-BOUNDARY.md](SOP-BOUNDARY.md) for the contract.

### `internal/web` — HTTP and rendering

Server-rendered `html/template` views with HTMX partial updates and polling.
Middleware enforces loopback binding, network-mode authentication, CSRF, and
request isolation. A background command runner records the latest result per
project/verb so the UI never blocks on a long SOP lifecycle.

## Data Sources

The controller reads only SOP-owned artifacts, all read-only:

| Source | Contents |
| --- | --- |
| `.agent-sdlc/state.db` | task state, status, dependencies, attempts, handoffs (read through `sopclient`) |
| `.agent-sdlc/runs/<task>/state.json` | the latest run's lifecycle **stage** |
| `.agent-sdlc/runs/<task>/classification.json` | SOP's failure **recovery** disposition |
| `.agent-sdlc/runs/<task>/report.json` | run report, quality decision, JEV summary |
| `.agent-sdlc/runs/<task>/activity.jsonl` | structured activity events (observer-only) |
| `.agent-sdlc/plan.meta.json` | SOP's recorded active plan source and final gate |
| `sop reconcile <PLAN.md> --list-changed --json` | SOP's authoritative changed-executed-task listing, resolved from the recorded plan source (C2-003; the retired `.agent-sdlc/reconcile.json` artifact is never read) |

Absent artifacts are reported as absent — never as a pass. See
[../reference/STATUS-AND-RECOVERY.md](../reference/STATUS-AND-RECOVERY.md).

## Ownership Model

```text
agentic-sop / SOP = automation and execution plane
sop-controller    = human visibility and control plane
```

The controller makes SOP understandable and safely controllable without
becoming SOP itself. This is the project's central constraint; it is specified
in [SOP-BOUNDARY.md](SOP-BOUNDARY.md) and enforced by boundary tests.

## Repository Layout

```text
cmd/sop-controller/    main
internal/config/       environment configuration
internal/sopclient/    SOP application/API boundary (reads state, runs commands)
internal/web/          HTTP handlers, middleware, rendering
sopagent_test.go       tests for the SOP command-agent adapter
scripts/sop-agent.sh   SOP command-agent adapter (implementation-agent seam)
templates/             html/template views + partials
static/                app.css + htmx.min.js (embedded)
docs/                  PRD, PLAN, pre-check report
```

## Related Documentation

- [SOP-BOUNDARY.md](SOP-BOUNDARY.md) — ownership boundary and invariants.
- [../reference/STATUS-AND-RECOVERY.md](../reference/STATUS-AND-RECOVERY.md) —
  statuses, stages, and recovery values the controller reports.
- [../reference/CONFIGURATION.md](../reference/CONFIGURATION.md) — runtime
  configuration.

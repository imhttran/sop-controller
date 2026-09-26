# Getting Started

**Type:** Guide

This guide explains how to run SOP Controller and open the dashboard. For the
full list of configuration variables, see
[../reference/CONFIGURATION.md](../reference/CONFIGURATION.md).

## Requirements

- **Go** (1.24+). That is it at build/run time.
- The **`sop` CLI** on `PATH` (or `SOP_BIN`) for the Run/Resume/Validate/Review
  commands. Read-only views work without it.

No Node, no npm, no PostgreSQL. Server-rendered HTML plus HTMX — no SPA.

## Run

```bash
go run ./cmd/sop-controller     # foreground
# → http://127.0.0.1:8080
```

Or drive it with the Makefile:

```bash
make start     # build + run in the background
make status    # is it up? (also pings /healthz)
make logs      # tail the background log
make stop      # shut it down
```

`make help` lists every target (`run`, `restart`, `build`, `test`, `fmt`,
`vet`, `tidy`, `clean`). Override the address with
`make start ADDR=127.0.0.1:9000`.

## Choose the Projects It Observes

By default the controller reads SOP state from the current directory
(`./.agent-sdlc/state.db`). Point it at other projects with
`SOP_CONTROLLER_PROJECTS=/path/to/project`, or let it discover every SOP project
under a workspace root with `SOP_CONTROLLER_WORKSPACES=/path/to/workspace`.

```bash
export SOP_CONTROLLER_WORKSPACES=~/agentic-workspace
go run ./cmd/sop-controller
```

See [PROJECT-DISCOVERY.md](PROJECT-DISCOVERY.md) for the discovery rules and
security boundary.

## Build a Binary

```bash
go build ./cmd/sop-controller && ./sop-controller
```

## Open the Dashboard

Open `http://127.0.0.1:8080`. The landing page lists the projects the controller
observes. From a project you can see its task workflow, per-task detail, activity,
recovery, review, CI, and handoff, and issue safe commands.

- [`/projects`](http://127.0.0.1:8080/projects) — project overview
- `/projects/{id}` — workflow view
- `/projects/{id}/tasks/{task}` — task detail
- `/discovery` — configured workspace roots, depth bound, and skipped candidates

## Phone / Same-Network Access

The dashboard is local-first and binds to loopback by default. To reach it from
your phone on the same Wi-Fi, network mode is an explicit opt-in that also
requires an access token:

```bash
export SOP_CONTROLLER_ALLOW_NETWORK=true
export SOP_CONTROLLER_TOKEN="choose-a-token"
export SOP_CONTROLLER_ADDR=0.0.0.0:8080
go run ./cmd/sop-controller
```

Binding a non-loopback address without network mode is refused. See
[../specs/SECURITY.md](../specs/SECURITY.md) for the full model.

## Next Steps

- [PROJECT-DISCOVERY.md](PROJECT-DISCOVERY.md) — configure what it observes.
- [DEVELOPMENT.md](DEVELOPMENT.md) — build, test, and contribute.
- [../reference/CLI.md](../reference/CLI.md) — commands and routes.

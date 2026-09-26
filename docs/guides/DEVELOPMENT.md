# Development

**Type:** Guide

How to build, test, and develop SOP Controller.

## Commands

```bash
go test ./...          # sopclient boundary + web handler tests
gofmt -l .             # formatting check
go build ./... && go vet ./...
```

The Makefile wraps the common ones:

```bash
make run       # go run ./cmd/sop-controller (foreground)
make start     # build + run in the background
make stop      # stop the background server
make restart   # stop, then start
make status    # is it running? (also pings /healthz)
make logs      # tail the background log
make build     # build ./cmd/sop-controller to .run/sop-controller
make test      # go test ./...
make fmt       # gofmt -w .
make vet       # go vet ./...
make tidy      # go mod tidy
make clean     # remove .run artifacts
```

`make help` prints the same list. Override the listen address with
`ADDR=127.0.0.1:9000`.

## Pre-commit checks

Enable the repository's pre-commit checks (gofmt + build + vet + test):

```bash
git config core.hooksPath .githooks
```

## Where things live

```text
cmd/sop-controller/    main
internal/config/       environment configuration + discovery
internal/sopclient/    SOP application/API boundary (reads state, runs commands)
internal/web/          HTTP handlers, middleware, rendering, streaming
templates/             html/template views + partials
static/                app.css + htmx.min.js (embedded)
scripts/sop-agent.sh   SOP command-agent adapter
sopagent_test.go       tests for the command-agent adapter
```

## Guardrails

- Keep the controller inside the [SOP boundary](../architecture/SOP-BOUNDARY.md):
  read SOP state and delegate commands; never mutate SOP state or reimplement
  workflow logic.
- No React, Next.js, Node runtime, or PostgreSQL in v1. Server-rendered HTML
  plus HTMX.
- Do not open `state.db` from `internal/web`; go through `internal/sopclient`.

## Related Documentation

- [../architecture/OVERVIEW.md](../architecture/OVERVIEW.md) — system structure.
- [../reference/CLI.md](../reference/CLI.md) — routes and SOP commands.
- [../requirements/PRD.md](../requirements/PRD.md) — product requirements and constraints.

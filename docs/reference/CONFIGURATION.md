# Configuration

**Type:** Reference

All environment variables are optional; see `.env.example` for a documented
template. Values may also be supplied through a `.env` file (a personal `.env`
wins and is never overwritten; `.env.dev` only fills in for development).

## Environment Variables

| Variable                         | Default          | Purpose                                                              |
| -------------------------------- | ---------------- | -------------------------------------------------------------------- |
| `SOP_CONTROLLER_ADDR`            | `127.0.0.1:8080` | Listen address (loopback enforced unless network mode is on)         |
| `SOP_CONTROLLER_PROJECTS`        | `.`              | Comma-separated explicit project roots containing `.agent-sdlc/`     |
| `SOP_CONTROLLER_WORKSPACES`      | —                | Comma-separated workspace roots scanned for SOP projects             |
| `SOP_CONTROLLER_DISCOVERY_DEPTH` | `4`              | Maximum directory depth below a workspace root during discovery      |
| `SOP_BIN`                        | `sop`            | SOP CLI used for commands                                            |
| `SOP_CONTROLLER_POLL`            | `3s`             | HTMX polling cadence for active views                                |
| `SOP_CONTROLLER_COMMAND_TIMEOUT` | `15m`            | Upper bound for a single SOP command (run, resume, validate, review) |
| `SOP_CONTROLLER_ALLOW_NETWORK`   | `false`          | Opt-in to binding a non-loopback address                             |
| `SOP_CONTROLLER_TOKEN`           | —                | Required access token when network mode is on                        |

### Notes

- `SOP_CONTROLLER_PROJECTS` and `SOP_CONTROLLER_WORKSPACES` merge. When neither
  is set, the controller observes the current directory (`"."`). When only
  workspace roots are set, the `"."` default is not forced, so the controller
  can run from outside a project.
- `SOP_CONTROLLER_ADDR` is forced back to loopback when it names a non-loopback
  host and `SOP_CONTROLLER_ALLOW_NETWORK` is not `true`.
- `SOP_CONTROLLER_ALLOW_NETWORK=true` requires `SOP_CONTROLLER_TOKEN`; the
  controller refuses to start without it.
- `SOP_CONTROLLER_DISCOVERY_DEPTH` reads a positive integer; an empty,
  unparseable, or non-positive value falls back to the default.
- The controller also reads `SOP_CONTROLLER_ENV` / `NODE_ENV` to decide whether
  to seed `.env.dev` during development. These are not part of normal runtime
  configuration.
- `SOP_ACTIVITY` is set to `on` by the controller for the SOP commands it
  launches, unless the operator already set it. Activity is observer-only in
  SOP, so this never changes execution. See
  [../specs/ACTIVITY.md](../specs/ACTIVITY.md).

## SOP-Side Configuration

Some behavior is governed by SOP's own configuration, not by a controller
environment variable. The controller only reads and reports the resulting state;
it never sets these itself:

- `human.approval_before_commit` (SOP configuration) controls whether a human
  must approve before committing. When it is on, `sop run` stops at the human
  gate and never commits; the controller reports that state rather than
  proceeding. See [../specs/HUMAN-APPROVAL.md](../specs/HUMAN-APPROVAL.md).

Approval and reconciliation are distinct human-decision domains, each with its
own SOP-owned evidence and delegated operation (see
[../specs/HUMAN-APPROVAL.md](../specs/HUMAN-APPROVAL.md)). Both are driven by the
documented SOP commands; there is no controller configuration that changes which
domain applies. There is **no** configuration to enable cancellation: SOP
exposes no cancellation operation, so cancellation remains unsupported until SOP
exposes cancellation.

### Command Timeout

`SOP_CONTROLLER_COMMAND_TIMEOUT` controls how long a single SOP command can run
before the dashboard terminates it. This applies uniformly to all commands (run,
resume, validate, review, retry, reconcile, approve, decline). The timeout is
enforced by the dashboard's command runner; exceeding it returns a timeout error
in the UI.

This is a different concern from `SOP_CONTROLLER_POLL`: the timeout bounds how
long one SOP command process (started by a user action) may run
(`internal/sopclient/commands.go`'s `Commander.Exec`), while the poll interval
only controls how often the dashboard re-fetches SOP state that SOP itself
already persisted (`internal/web/activity_stream.go`). Lowering the poll
interval does not make commands run faster or time out sooner; raising the
command timeout does not change how often the UI refreshes.

When a command exceeds `SOP_CONTROLLER_COMMAND_TIMEOUT`, the dashboard returns
a truthful error of the form `sop <verb> timed out after <duration>` (surfaced
in the command-status UI), never a silent success or a generic failure
message. This timeout is purely a dashboard-side bound on the child process:
it never writes to or otherwise mutates SOP's own persisted task/lifecycle
state under `.agent-sdlc/runs/<task>/*`. That state remains exactly whatever
SOP itself last wrote; the dashboard's `CommandRunner` only records the
timeout in its own in-memory, per-project/verb status map
(`internal/web/commands.go`), which is separate from SOP's state and is never
written to disk.

## Related Documentation

- [CLI.md](CLI.md) — commands and routes.
- [../guides/PROJECT-DISCOVERY.md](../guides/PROJECT-DISCOVERY.md) — discovery
  configuration in context.
- [../specs/SECURITY.md](../specs/SECURITY.md) — network mode and access tokens.

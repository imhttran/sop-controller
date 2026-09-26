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

### Command Timeout

`SOP_CONTROLLER_COMMAND_TIMEOUT` controls how long a single SOP command can run
before the dashboard terminates it. This applies uniformly to all commands (run,
resume, validate, review, retry). The timeout is enforced by the dashboard's
command runner; exceeding it returns a timeout error in the UI.

## Related Documentation

- [CLI.md](CLI.md) — commands and routes.
- [../guides/PROJECT-DISCOVERY.md](../guides/PROJECT-DISCOVERY.md) — discovery
  configuration in context.
- [../specs/SECURITY.md](../specs/SECURITY.md) — network mode and access tokens.

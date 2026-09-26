# Security Specification

**Type:** Normative specification

## Purpose

Defines the required security behavior of SOP Controller: local-first binding,
network mode, request safety, the discovery boundary, and secret handling.

## Related Specifications

- [WORKFLOW.md](WORKFLOW.md)
- [ACTIVITY.md](ACTIVITY.md)
- [../architecture/SOP-BOUNDARY.md](../architecture/SOP-BOUNDARY.md)
- [../reference/CONFIGURATION.md](../reference/CONFIGURATION.md)

## Normative Language

The terms MUST, MUST NOT, SHOULD, SHOULD NOT, and MAY are normative.

## Binding and Network Mode

- The controller MUST be local-first: it MUST bind to a loopback address by
  default.
- The controller MUST refuse to bind a non-loopback address unless network mode
  (`SOP_CONTROLLER_ALLOW_NETWORK=true`) is explicitly enabled.
- Network mode MUST require an access token (`SOP_CONTROLLER_TOKEN`); the
  controller MUST refuse to start without one.

## Request Safety

- State-changing requests MUST be `POST` and MUST carry a CSRF token
  (double-submit cookie).
- Requesting an action MUST NOT bypass SOP's gates: the only actions the
  controller offers MUST delegate to SOP's own operations.

## Discovery Boundary

- Project discovery MUST be an allowlist: the controller MUST read only explicit
  projects and directories beneath explicitly configured workspace roots.
- The controller MUST NOT scan the wider filesystem and MUST NOT infer roots.
- Discovery MUST NOT follow symlinks, and MUST NOT traverse metadata or build
  directories (`.git`, `node_modules`, `vendor`, `dist`, `build`, `target`,
  `.cache`).
- Discovery MUST be depth-bounded (`SOP_CONTROLLER_DISCOVERY_DEPTH`).
- A conflict where two roots claim the same project identity MUST be refused and
  reported, not silently resolved.

## Data Access

- The browser MUST NOT reach SQLite. Every read MUST be server-side through
  `internal/sopclient`.
- The controller MUST NOT write SOP persistence from the web layer.
- The controller MUST interact with SOP through the supported boundary only. See
  [../architecture/SOP-BOUNDARY.md](../architecture/SOP-BOUNDARY.md).

## Secret Handling

- The controller MUST NOT render secrets, environment variables, API keys, or
  agent credentials.
- The controller MUST NOT forward model/provider secrets to a SOP command. The
  only environment it adds is `SOP_ACTIVITY=on` (observer-only). See
  [ACTIVITY.md](ACTIVITY.md).
- Free-form text (blocking reasons, change summaries, activity detail, approval
  evidence) MUST be sanitized before display so it cannot leak prompts, secrets,
  or file contents.

## Related Documentation

- [../reference/CONFIGURATION.md](../reference/CONFIGURATION.md) — network mode
  variables.
- [../guides/PROJECT-DISCOVERY.md](../guides/PROJECT-DISCOVERY.md) — discovery in
  context.
- [../requirements/PRD.md](../requirements/PRD.md) — V1 security requirements.

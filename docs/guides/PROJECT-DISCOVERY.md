# Project Discovery

**Type:** Guide

The controller reads SOP state for a set of projects. There are two ways to
configure them, and they merge.

## Explicit allowlist

```bash
# Explicit allowlist (unchanged, backward compatible).
export SOP_CONTROLLER_PROJECTS=/path/to/project-a,/path/to/project-b
```

Each entry is a project root that contains an initialized `.agent-sdlc/` state.

## Workspace discovery

```bash
# Or: discover every SOP project beneath one or more workspace roots.
export SOP_CONTROLLER_WORKSPACES=/path/to/workspace
```

A **workspace root** is an explicit security boundary. The controller descends
only beneath the roots you configure, looking for directories that contain a
valid `.agent-sdlc/config.yaml`. It never scans `/`, `$HOME`, or any directory
outside a configured root, and it never infers extra roots. With the example
below, a single workspace root is enough to find both `workspace/agentic-sop`
and `workspace/projects/sop-controller`:

```bash
export SOP_CONTROLLER_WORKSPACES=~/agentic-workspace
```

## Discovery rules

- A directory is a project only if it has a valid `.agent-sdlc/config.yaml` and
  an initialized `.agent-sdlc/state.db`. Ordinary directories and Git repos
  without SOP state are ignored.
- Nesting such as `workspace/project` and `workspace/group/project` is supported.
  Discovery is bounded to `SOP_CONTROLLER_DISCOVERY_DEPTH` (default `4`) and
  never recurses without limit.
- Metadata and build directories (`.git`, `node_modules`, `vendor`, `dist`,
  `build`, `target`, `.cache`) are never traversed, and symlinks are never
  followed, so discovery cannot loop or escape a workspace root.
- The **project identity** is the declared `project.name` from
  `.agent-sdlc/config.yaml`, falling back to the directory name. A project has
  the same identity however it was discovered.
- The same project reached both explicitly and by discovery is registered once.
- Two different roots claiming the same identity are a conflict: the controller
  refuses to bind the identity to either and reports both roots, so it can never
  silently expose the wrong project.
- A bad directory inside a workspace (an invalid config, or a missing state
  database) produces a startup diagnostic and never hides unrelated valid
  projects.

Discovered projects behave exactly like explicit ones: the same `/projects` and
`/projects/<id>` routes, with no special handling.

## Inspect the boundary

The **Discovery** view (`/discovery`) makes the boundary visible: it lists the
configured workspace roots, the depth bound, the registered projects, and every
candidate that was skipped and why.

## Related Documentation

- [../reference/CONFIGURATION.md](../reference/CONFIGURATION.md) — the discovery
  environment variables and defaults.
- [../specs/SECURITY.md](../specs/SECURITY.md) — discovery as a security boundary.
- [GETTING-STARTED.md](GETTING-STARTED.md) — first-run setup.

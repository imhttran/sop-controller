# Activity Specification

**Type:** Normative specification

## Purpose

Defines the live activity stream the controller displays: its source, delivery,
cursor semantics, and safe-summary guarantees.

## Related Specifications

- [EXECUTION.md](EXECUTION.md)
- [WORKFLOW.md](WORKFLOW.md)
- [../reference/CLI.md](../reference/CLI.md)

## Normative Language

The terms MUST, MUST NOT, SHOULD, SHOULD NOT, and MAY are normative.

## Source

- SOP persists its structured activity stream to
  `.agent-sdlc/runs/<task>/activity.jsonl` when activity reporting is enabled.
- The controller MUST read this artifact read-only and MUST NOT write it.
- The controller MUST NOT parse terminal output to reconstruct activity.

## Enabling Activity

- The controller MUST enable activity for every SOP command it launches by
  setting `SOP_ACTIVITY=on`, unless the operator already chose a value.
- Activity is observer-only in SOP, so enabling it MUST NOT change execution.
- This MUST be the only environment the controller adds to a SOP command; it
  MUST NOT forward model, provider, or other secrets.

## Safe Summaries

- Activity events are SOP's own safe summaries (`stage` / `action` / `detail`).
- The controller MUST NOT render prompts, model output, secrets, API keys,
  environment dumps, or file contents.
- Event detail MUST be produced through the shared sanitization boundary, so a
  poll path and a stream path apply the same guarantee.

## Event Model

Each event carries:

- **TaskID** — the task whose run directory the event was read from.
- **Stage** — SOP's persisted lifecycle stage.
- **Action** — SOP's recorded action kind (see below).
- **Detail** — a safe summary (for example command name + exit status, a file
  path category, or a review/OpenJEV verdict).
- **Timestamp** — SOP's recorded transition time.

Action kinds (SOP's vocabulary): `inspect`, `mutate`, `command`, `validate`,
`review`, `jev`, `quality`, `recover`, `complete`.

## Delivery

- The controller MUST serve activity over two endpoints backed by the same read:
  an SSE stream (`/projects/{project}/activity/stream`) for active runs, and a
  bounded poll window (`/projects/{project}/activity/window`) as a fallback for
  clients without SSE.
- Events MUST carry a stable cursor. A poll refresh and a stream reconnection
  MUST be the same operation (send the last-seen cursor, receive only later
  events), so a reconnect MUST NOT duplicate already-delivered lifecycle
  entries.
- Delivery MUST be bounded: one window delivers a bounded number of events.
- The controller MUST NOT open a command operation to read activity.

## Isolation

- Reading or streaming activity MUST NOT change SOP execution or SOP state.
- Activity is one of the controller's read sources; its absence MUST be reported
  as absent, not as an error.

## Related Documentation

- [EXECUTION.md](EXECUTION.md) — the run stages activity events reflect.
- [../reference/CONFIGURATION.md](../reference/CONFIGURATION.md) — poll cadence and
  timeout configuration.

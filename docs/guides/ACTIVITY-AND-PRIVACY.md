# Activity and Privacy

**Type:** Guide

What the activity timeline shows, how SOP's structured events relate to raw
agent transcripts, and the privacy/safety rules that govern what may appear in
the dashboard. Normative behavior is in
[specs/ACTIVITY.md](../specs/ACTIVITY.md); this page is the operator view of it.

## What Activity Is

SOP writes a **structured activity stream** to
`.agent-sdlc/runs/<task>/activity.jsonl` when activity reporting is enabled. The
controller reads that artifact **read-only** and renders it as the timeline on
the project and task views:

- Project activity: `GET /projects/{project}/activity` (FR-4).
- Task activity: `GET /projects/{project}/tasks/{task}/activity`.
- Live stream: `GET /projects/{project}/activity/stream` (SSE) for active runs.
- Bounded poll: `GET /projects/{project}/activity/window` as the fallback for
  clients without SSE.

The controller enables activity for the SOP commands it launches by setting
`SOP_ACTIVITY=on`, unless you already chose a value. Activity is observer-only in
SOP, so enabling it never changes execution, and it is the only environment the
controller adds to a SOP command.

## Reading the Timeline

Each event carries SOP's own fields:

- **Stage** — SOP's persisted lifecycle stage (for example `PLANNING`,
  `IMPLEMENTING`, `REVIEWING`).
- **Action** — SOP's recorded action kind: `inspect`, `mutate`, `command`,
  `validate`, `review`, `jev`, `quality`, `recover`, `complete`.
- **Detail** — a **safe summary**, such as a command name and exit status, a file
  path category, or a review/OpenJEV verdict.
- **Timestamp** — SOP's recorded transition time.

Use the timeline to see *what SOP did and in what order*. The stages mirror the
run stages in
[../reference/STATUS-AND-RECOVERY.md](../reference/STATUS-AND-RECOVERY.md).

## Timeline vs. Raw Transcripts

**Raw agent transcripts are secondary.** SOP emits its own safe summaries; the
controller renders those summaries, not the raw model conversation. The timeline
is a faithful, ordered view of SOP's recorded events — it is **not** a full
transcript, and it does not replace SOP's authoritative persisted state
(`state.db`, `state.json`, `classification.json`, `report.json`). When the
timeline and authoritative state seem to disagree, the persisted state is the
source of truth; the activity feed is one read source among several.

## Privacy and Safety Rules

The controller applies the same guarantee to the poll path and the SSE stream:

- The controller **never renders prompts, model output, secrets, API keys,
  environment dumps, or file contents.** Event detail is produced through the
  shared sanitization boundary.
- The controller **never exposes secrets, environment variables, or agent
  credentials in dashboard output** (PRD Security). It forwards no model or
  provider secrets to a SOP command.
- The controller never parses terminal output to reconstruct activity; it reads
  SOP's structured summaries only.
- Reading or streaming activity never changes SOP execution or SOP state.
- If the activity artifact is absent, the controller reports it as **absent**,
  not as an error and not as a pass.

See [specs/ACTIVITY.md](../specs/ACTIVITY.md) for the normative guarantees and
[specs/SECURITY.md](../specs/SECURITY.md) for the wider security model.

## Related Documentation

- [specs/ACTIVITY.md](../specs/ACTIVITY.md) — normative activity specification.
- [specs/SECURITY.md](../specs/SECURITY.md) — secret handling and security.
- [FAILURE-DISPLAY.md](FAILURE-DISPLAY.md) — review, CI, and handoff display.
- [../reference/CLI.md](../reference/CLI.md) — activity routes.

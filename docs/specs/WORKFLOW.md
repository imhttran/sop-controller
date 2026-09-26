# Workflow Specification

**Type:** Normative specification

## Purpose

Defines the required behavior for how SOP Controller reflects SOP's workflow:
task status, dependencies, scheduling ownership, and restart safety.

## Related Specifications

- [EXECUTION.md](EXECUTION.md)
- [HUMAN-APPROVAL.md](HUMAN-APPROVAL.md)
- [../architecture/SOP-BOUNDARY.md](../architecture/SOP-BOUNDARY.md)
- [../reference/STATUS-AND-RECOVERY.md](../reference/STATUS-AND-RECOVERY.md)

## Normative Language

The terms MUST, MUST NOT, SHOULD, SHOULD NOT, and MAY are normative.

## Authority

- The controller MUST treat SOP as the sole workflow authority.
- The controller MUST NOT maintain a second source of truth for SOP task state.
- The controller MUST NOT modify SOP state directly.
- Every state-changing operation MUST go through the supported SOP
  application/API/CLI boundary (`internal/sopclient` → `sop` CLI verbs).
- The controller MUST report SOP lifecycle, status, and recovery information
  rather than inventing its own lifecycle semantics.

## Status Reporting

- The controller MUST mirror SOP's task status vocabulary verbatim and MUST NOT
  invent, rename, or reorder statuses.
- Any coarse state the controller derives (for example `DONE` / `RUNNING` /
  `BLOCKED` / `READY` / `PLANNED`) MUST be display-only and MUST NOT change SOP
  state or be presented as SOP's status.
- The controller MUST report a task's blocking reason as SOP reports it, and
  MUST NOT infer a reason from its own heuristics.
- The controller MUST surface SOP's recorded final plan gate verbatim when SOP
  persisted plan metadata, and MUST NOT infer it otherwise.
- A missing SOP artifact MUST be reported as absent, never as a pass.

## Dependencies and Readiness

- The controller MUST display task dependencies as SOP reports them.
- The controller MUST NOT compute task readiness or eligibility itself.

## Scheduling

- The controller MUST NOT choose which task runs next.
- The controller MAY offer Run and Resume actions that delegate task selection
  to SOP; SOP decides which runnable task is selected.

## Project Identity

- Project identity MUST be SOP's declared `project.name` (falling back to the
  directory name), so a project has the same identity however it was discovered.
- When two roots claim the same identity, the controller MUST refuse to bind the
  identity to either and MUST report the conflict.

## Restart Safety

- Restarting the controller MUST NOT lose authoritative workflow state, because
  that state lives in SOP.
- After a restart the controller MUST reflect the true SOP state on the next
  read; process-local command state MAY be refreshed rather than persisted.

## Idempotent Control

- Requesting the same command twice while it is already running MUST NOT start a
  duplicate command.

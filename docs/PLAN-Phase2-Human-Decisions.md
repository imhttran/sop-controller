# PLAN — SOP Controller Phase 2: Human Decision Integration

## Project

sop-controller

## Summary

Phase 1 of `sop-controller` (CTRL001–CTRL017) built an observation and human-control
surface over `agentic-sop`, but it treated approval and reconciliation as *gaps*: it
inferred a human boundary from SOP-persisted task status/classification and offered no
mutation because SOP exposed none. `agentic-sop` Phase 6 has since shipped authoritative
human-decision APIs:

```text
sop approvals [--json]
sop approval <task-id>
sop approve  [<task-id>] [--by NAME] [--note TEXT] [--run]
sop decline  [<task-id>] [--by NAME] [--note TEXT]
sop reconcile <PLAN.md> [--accept-changed <TASK_ID>]... [--list-changed] [--json]
```

This phase turns the controller from an inference-based observer into a safe
human-control surface that consumes those authoritative reads and delegates every
mutation to SOP.

## Objective

Make `sop-controller` consume SOP's authoritative approval and reconciliation reads,
delegate approve/decline/reconcile-accept to SOP, surface outstanding human decisions
promptly, and remove controller-side approval inference — while keeping `agentic-sop`
the sole lifecycle, routing, recovery, and approval-policy authority and the controller
purely a presentation and human-control surface.

## Architecture

```text
agentic-sop  = lifecycle + policy authority
sop-controller = presentation + human control
```

Constraints:

- The controller MUST NOT determine that an approval gate exists, that a task is
  complete/blocked/failed, or that reconciliation is required. It consumes SOP's
  authoritative reads.
- `BLOCKED` is not approval. Free-form prose, attempt counts, recovery disposition, and
  model output never manufacture a gate.
- All mutations are POST, CSRF-protected, project-scoped, and delegated to SOP through
  the existing safe command abstraction; the controller never writes SOP state.
- Decision is not execution: approving records a decision; it does not run `sop run`
  unless the operator explicitly chooses Continue.
- No arbitrary multi-minute waits; no time-based lifecycle inference.
- Do NOT change SOP routing, SMALL/MEDIUM/LARGE, JEV, provider selection, model
  resolution, recovery policy, approval policy, reconciliation policy, task scheduling,
  or retry budgets. Do NOT implement cancellation (`sop cancel` still does not exist).

## C2-001 — Adopt Authoritative Approval Reads

Replace controller-side approval inference with SOP's authoritative approval read
surface. Add a controller read that lists every applicable gate from
`sop approvals --json` and a single-task read that selects one entry from that same
structured JSON (avoid parsing `sop approval`'s human text where the JSON listing can
answer the same question). Decode SOP's schema fields verbatim: `task_id`, `kind`,
`target`, `reason`, `evidence`, `stage`, `disposition`, `status`, `requested_at`,
`task_status`. Retire the heuristics in `internal/sopclient/approval.go` so SOP's
authoritative response is the only source of truth for whether a gate exists and
whether it is currently applicable.

### Dependencies

- none

### Deliverables

- `internal/sopclient/approval.go` — authoritative approval read model and client
  operations; heuristics removed or narrowed to an explicitly isolated, non-authoritative
  legacy path that can never override SOP data.
- `internal/sopclient/` read operations `Approvals` (list) and `Approval` (single).
- Updated `internal/sopclient/types.go` / `store.go` projections that render SOP's
  values rather than derived ones.

### Acceptance Criteria

- Approval presence and applicability come only from `sop approvals --json`.
- A completed, stale, or resolved gate is not offered as actionable.
- `BLOCKED` alone, model/recovery prose, attempt counts, and inactivity do NOT produce
  an approval.
- The listing is the source for both list and single-task projection; no controller-side
  reconstruction of the gate list.
- Free-form strings are display-only and never drive controller policy.

## C2-002 — Wire Approve and Decline to SOP

Update the boundary so `OpApproveTask` and `OpDeclineTask` are supported, and delegate
the controller's approval actions to SOP. `Client.ApproveTask` runs
`sop approve <task-id> [--by NAME] [--note TEXT]` and `Client.DeclineTask` runs
`sop decline <task-id> [--by NAME] [--note TEXT]`, preserving any actor/note metadata.
Do not pass `--run`: keep decision separate from execution. Rely on SOP to validate the
gate at command time, and surface a rejected (stale/not-applicable) decision truthfully.
Do not write SOP state, mark a task complete, or manufacture a decline-as-failure.

### Dependencies

- C2-001

### Deliverables

- `internal/sopclient/boundary.go` — OpApproveTask/OpDeclineTask marked supported with
  accurate descriptors; gap reasons for approval removed.
- `internal/sopclient/` approve/decline command operations using the existing command
  runner.
- `internal/web/approval.go` — handlers delegate to the new operations.

### Acceptance Criteria

- `op approve|decline` invokes exactly `sop approve|decline <task-id>` with the task id
  passed safely (no shell interpolation of untrusted input).
- `--by`/`--note` are forwarded when supplied.
- A stale gate is rejected by SOP and reported as an actionable conflict, not a 500 and
  not a fabricated success.
- Approving never marks the task complete in controller state; declining never
  manufactures failure.
- No controller-side approval persistence exists.

## C2-003 — Adopt Reconcile Changed-Task Reads from SOP

Replace the `reconcile.json` artifact read with SOP's authoritative listing:
`sop reconcile <PLAN.md> --list-changed --json`. Resolve the active plan source from
SOP's recorded plan metadata (`plan.meta.json`) rather than guessing. Decode SOP's
document (`version`, `source`, `plan_id`, `plan_changed`, `unchanged`, `updated`,
`added`, `removed`, `changed_executed`, `removed_executed`, `auto_reconciled`).
`changed_executed` carries task ids only; enrich the display title/stage from SOP's task
store, never by computing a plan diff. Treat the listing as a pure read.

### Dependencies

- C2-001

### Deliverables

- `internal/sopclient/changed_task.go` — read model populated from the SOP listing.
- `internal/sopclient/` operation that runs `sop reconcile <PLAN.md> --list-changed --json`.
- Removal/retirement of the controller's `reconcile.json` parsing path.

### Acceptance Criteria

- The changed-executed set comes from SOP's `--list-changed --json`, not from a
  controller-computed diff or a stale artifact.
- Listing mutates nothing: task state, graph, active plan, acceptance, and provenance
  are unchanged.
- An unreported/failed listing is surfaced as such, never as an empty success.
- The active plan source is read from SOP's recorded provenance.

## C2-004 — Wire Explicit Reconciliation Acceptance

Update the boundary so `OpAcceptChangedTask` is supported and delegate acceptance to
SOP. `Client.AcceptChangedTask` runs `sop reconcile <PLAN.md> --accept-changed <TASK_ID>`
using SOP's real syntax. Support collecting multiple explicitly selected tasks into a
single SOP invocation (repeat `--accept-changed`) to preserve SOP's atomic reconciliation
semantics, rather than a controller loop of partial reconciliations. Never bulk-accept
implicitly and never treat viewing a change as accepting it.

### Dependencies

- C2-003

### Deliverables

- `internal/sopclient/boundary.go` — OpAcceptChangedTask marked supported.
- `internal/sopclient/` accept-changed operation accepting one or many explicit ids.
- `internal/web/` accept-changed handler delegation.

### Acceptance Criteria

- Acceptance invokes `sop reconcile <PLAN.md> --accept-changed <id>` with only the ids
  the human explicitly selected.
- No implicit or bulk acceptance occurs.
- A task not in SOP's reported changed set is rejected before any SOP call.
- A failed reconciliation is reported truthfully.
- Concurrent reconciliation mutations for one project are serialized.

## C2-005 — Add Project "Needs Attention" View

Add a project-level human-attention surface driven entirely by the authoritative
approval listing and the reconciliation changed set. Report counts and the specific
tasks with their reason (approval required / executed task changed). Do not derive the
list from arbitrary task statuses.

### Dependencies

- C2-001
- C2-003

### Deliverables

- `internal/sopclient/` project summary carrying `NeedsAttention` (count + items) derived
  from the authoritative reads.
- `templates/project.html` (and any partials) rendering the attention block.

### Acceptance Criteria

- The attention list equals the union of SOP's applicable approvals and SOP's
  changed-executed tasks.
- A task that is merely `BLOCKED`, retried, or inactive is not listed.
- Empty state is explicit (no attention vs. reporting failure).

## C2-006 — Add Human-Decision UI

Add the smallest coherent human-decision surface — a project-level decisions panel (or
`GET /projects/{project}/decisions`) — that lets an operator inspect a gate, approve,
decline, review a changed executed task, and accept selected changed tasks. Reuse the
existing server-rendered templates, htmx, CSRF, and command-status fragments. Stack
cleanly on narrow screens without horizontal scrolling and with adequate button spacing.

### Dependencies

- C2-002
- C2-004
- C2-005

### Deliverables

- New/extended templates (decisions panel; approval and reconciliation controls).
- `internal/web/` handlers and routes for inspection and the mutation actions.
- `static/app.css` responsive adjustments.

### Acceptance Criteria

- Approve/Decline are offered only for a gate SOP reports applicable; Accept is offered
  only for tasks SOP reports as changed-executed and pending.
- Every mutation is POST + CSRF; GET renders state only.
- Task ids, notes, actor values, and plan paths are treated as untrusted input.
- The surface is usable on a narrow viewport with no horizontal scrolling.
- No new dashboard application is introduced.

## C2-007 — Integrate Decisions with Live Progress

When SOP moves from RUNNING to a human decision state, surface it promptly through the
existing activity/status infrastructure and short configurable polling. Show "Needs your
attention" with the relevant controls rather than a generic spinner. Do not implement a
fixed multi-minute wait and do not infer a gate from inactivity.

### Dependencies

- C2-005

### Deliverables

- `internal/web/` status/partials surfacing the decision state promptly.
- Reuse of `data-poll-ms` / existing poll cadence.

### Acceptance Criteria

- A newly recorded SOP gate becomes visible shortly after SOP records it, without an
  arbitrary long wait.
- Inactivity is shown as factual elapsed time only and never becomes a lifecycle state.
- No second event system is introduced; per-project isolation is preserved.

## C2-008 — Add Contract and Integration Tests

Add deterministic tests proving the new boundary against SOP's actual contract.

### Dependencies

- C2-002
- C2-004

### Deliverables

- `internal/sopclient/` tests for approval reads, approve/decline, and
  reconcile list/accept, using a fake sop binary or injected runner.
- `internal/web/` tests for the HTTP surface.
- Updated boundary contract tests.

### Acceptance Criteria

- Tests prove approval presence comes only from `sop approvals --json`, that stale gates
  are not actionable, and that `BLOCKED`/prose/inactivity do not manufacture a gate.
- Tests prove approve/decline invoke the correct SOP verb with a safely passed task id
  and preserved metadata; a stale approval is rejected; a duplicate action does not
  bypass SOP.
- Tests prove listing is read-only, only explicitly selected tasks are accepted, and no
  implicit bulk acceptance occurs.
- Boundary tests prove CancelRun remains unsupported and every supported operation maps
  to a documented SOP operation.
- Web tests cover CSRF, method restrictions, project/task isolation, duplicate-command
  protection, status codes, and stale-gate handling.

## C2-009 — Dogfood Against Real agentic-sop

After deterministic tests pass, exercise the controller against the real `sop` binary:
a disposable scenario reaching a real human approval gate, verified end to end
(RUNNING -> Needs Attention -> Inspect -> Approve -> SOP records -> controller reflects
-> explicit Continue -> SOP resumes), plus a decline and a changed-executed-task
reconciliation scenario. Record observed behavior truthfully.

### Dependencies

- C2-006

### Deliverables

- Dogfood notes recording what was observed, in `docs/history/`.

### Acceptance Criteria

- The end-to-end approval flow is exercised against the real SOP and its observed
  behavior is recorded, including any divergence found.
- Decline and reconciliation scenarios are exercised or their absence is explained.
- Success is not inferred solely from unit tests.

## C2-010 — Update Architecture, Spec, and Reference Documentation

Update the normative documentation to the new boundary: SOP owns read and mutation
authority; the controller presents and delegates. Keep `docs/history/` historical.

### Dependencies

- C2-006
- C2-008

### Deliverables

- `docs/architecture/SOP-BOUNDARY.md`, `docs/specs/HUMAN-APPROVAL.md`,
  `docs/specs/WORKFLOW.md`, `docs/reference/CLI.md`,
  `docs/guides/APPROVAL-AND-RECONCILE.md`, and `docs/PLAN-SOP-Controller.md` updated.

### Acceptance Criteria

- Docs state that the controller determines how a decision is presented while SOP
  determines whether the decision is applicable and how it changes lifecycle state.
- Docs record which previously unsupported operations are now supported.
- Docs confirm `CancelRun` remains unsupported until SOP exposes cancellation.
- Approval and reconciliation are documented as distinct human-decision domains.
- Referenced commands, routes, and flags are accurate.

## C2-011 — Validation and Final Report

Run the repository's full validation suite and record a truthful final report.

### Dependencies

- C2-009
- C2-010

### Deliverables

- `docs/history/C2-011-REPORT.md` (or the repository's established report location).

### Acceptance Criteria

- `gofmt -l .`, `go vet ./...`, `go build ./...`, `go test ./...`, and
  `go test -race ./...` are run and their outcomes reported truthfully.
- The report covers: files changed; which previously unsupported operations are now
  supported; whether approval inference was removed or reduced; the authoritative
  approval read path; the approve/decline path; stale-gate protection; the reconciliation
  preview and acceptance paths; the Human Decisions / Needs Attention UI; activity
  integration; security and concurrency behavior; tests added; validation results;
  dogfood results; remaining unsupported boundary operations; and confirmation that the
  controller did not acquire lifecycle, routing, recovery, or approval-policy authority.

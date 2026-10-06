# FP-004 — Review Humanized Decision Presentation

Evidence-only task. This artifact reviews how a genuine SOP human decision is currently
presented, maps each required humanized decision-brief field (what happened,
recommendation, why, options, impact, technical details) onto existing behavior using
preserved evidence, states the gap for each field, assesses honest multi-option
presentation, and identifies the smallest change path — stated explicitly as a proposal
outside this task's scope. No source code, no `.agent-sdlc` state, and no approval is
mutated or invoked by this task.

Every claim cites a source. Anything that could not be observed is marked NOT OBSERVED or
UNAVAILABLE with its reason. No recommendation is given where the preserved evidence does
not support one.

## 1. Live current-run boundary — NOT OBSERVED / UNAVAILABLE

This run has no live `NEEDS_HUMAN` boundary that can be inspected from this harness, so no
rendered decision request can be quoted verbatim.

- Source: `docs/reports/finish-pre-performance-closure/FP-003-human-approval-boundaries.md`
  section 1 — the live boundary for this run is **NOT OBSERVED / UNAVAILABLE**.
- Source: `docs/reports/finish-pre-performance-closure/FP-001-repository-lifecycle-state.md`
  sections 5–6 — `sop status` and `sop approvals` were refused by this harness
  (REQUIRES_APPROVAL); the live pending-approval list is NOT ESTABLISHED.
- Source: `.agent-sdlc/plan.meta.json` (as cited by FP-003 section 2.3) — no `NEEDS_HUMAN`
  boundary or approval record for this run is recorded.

Consequence: the mapping below is grounded in preserved/authoritative evidence — the PRD's
canonical `DECISION NEEDED` templates and the archived `plan-phase2-human-decisions`
approval records (as cited by FP-003) — and **not** in observed live rendering. No live
decision request is fabricated and no `NEEDS_HUMAN` boundary is manufactured.

## 2. Authoritative presentation sources

- **PRD canonical `DECISION NEEDED` templates** —
  `docs/plans/PLAN-Finish-Pre-Performance-Closure.md` (appendix): section 4 "Humanized
  Decision Requests" gives the required conceptual format; section 5 gives the
  evidence-based recommendation rules; section 6 "Multiple Safe Options" gives the
  Options A/B/C-with-impact example; section 24 "CLOSE-006" gives a worked decision brief.
- **Preserved approval-record / decision surface** —
  `.agent-sdlc/archive/plan-phase2-human-decisions/tasks.json`,
  `.../plan.json`, `.../plan.meta.json` (plan_id `plan-phase2-human-decisions`,
  generated_at `2026-10-01T16:05:58.100228Z`). These record SOP's authoritative
  human-decision surface (`sop approvals`, `sop approval`, `sop approve`, `sop decline`,
  `sop reconcile`) and the decoded approval fields `task_id`, `kind`, `target`, `reason`,
  `evidence`, `stage`, `disposition`, `status`, `requested_at`, `task_status`
  (task C2-001 "Adopt Authoritative Approval Reads", status LOCAL_DONE).
- **Boundary reports** — `.agent-sdlc/WRAP-007-BOUNDARY-VERIFICATION.md` (all workflow
  commands delegate to the SOP CLI via `Commander.Exec`, `internal/sopclient/service.go`;
  read-only SQLite; no controller-side plan-state machine) and
  `.agent-sdlc/WRAP-007-FINAL-VERIFICATION.md` (15 handler delegations to `h.sop.*`; no
  state writes in `sopclient`), as cited by FP-003 section 2.2.
- **Controller-side renderer absent** — a repository-wide search for `humaniz` returns
  only `go.mod`/`go.sum` (`dustin/go-humanize`, an unrelated dependency) and PRD/plan
  prose; `DECISION NEEDED` occurs only in
  `docs/plans/PLAN-Finish-Pre-Performance-Closure.md`. No decision-brief renderer exists
  in `sop-controller`.

## 3. Required brief-field mapping and gaps

The required format is the PRD §4 block: `DECISION NEEDED`, then `What happened` /
`Why I need you` / `Recommended` / `Why` / `Other options` / `Impact` / `Suggested action`
/ `Technical details`. The task's field list (what happened, recommendation, why, options,
impact, technical details) maps onto that block as follows.

| Required field | Current behavior (cited) | Gap |
|---|---|---|
| **What happened** | The preserved approval record carries `kind`, `target`, `stage`, `requested_at`, `reason`, `evidence` (archived `plan-phase2-human-decisions/tasks.json`, C2-001 decoded fields; FP-003 §2.1). SOP owns the record; the controller only lists/relays it. A human-readable plain-English "what happened" sentence is not produced by any component observable in this repository. | No plain-English narrative producer is observable. The raw structured fields exist (SOP side); the humanized sentence does not. |
| **Recommendation** (recommended option) | PRD §5 requires a recommendation only when repository evidence supports one, else the literal "No recommendation — the available evidence does not distinguish the alternatives reliably." No component in `sop-controller` generates a recommendation (search for `humaniz`/`DECISION NEEDED` shows none). | No recommendation generator is observable in the controller; whether one exists in the SOP CLI/harness is not observable from this harness for a live boundary (live boundary NOT OBSERVED, §1). |
| **Why** (reasoning for the recommendation) | PRD §5 gives the reasoning format; the preserved record carries a `reason` field (FP-003 §2.1). The `reason` field is free-form record data, not a rendering of the recommendation rationale. | No producer of the recommendation-rationale prose is observable; the `reason` record field is not equivalent to a rendered "Why". |
| **Options** | The preserved decision surface is binary: `sop approve <task-id>` / `sop decline <task-id>` (archived tasks.json C2-002 "Wire Approve and Decline to SOP", status LOCAL_DONE; FP-003 §2.1). The PRD §6 mandates explicit multi-option display with per-option impact (Options A/B/C). | Binary approve/decline is the only observed mechanism; there is no observed mechanism for enumerating arbitrary alternatives. See §4. |
| **Impact** | The PRD §6 example specifies per-option impact ("fastest path…", "additional engineering work…"). The preserved record fields (`disposition`, `status`, `task_status`) describe gate state, not per-option consequences. | No per-option impact producer is observable; impact prose is specified but not rendered by any component in this repository. |
| **Technical details** (task ID / evidence / report / failure classification) | The preserved record carries `task_id`, `evidence`, `stage`, `status`, `task_status` (FP-003 §2.1) — i.e. the structured identifiers a `Technical details:` line requires. | The structured data exists; the compact rendered `Technical details:` line is not produced by any observable component in `sop-controller` (it delegates all rendering to SOP). |

Field with no observable producer overall: the entire humanized block. No decision-brief
renderer exists in this repository (search evidence, §2). The structured approval record
produced on the SOP side is the only observable data; whether the SOP CLI/harness renders
it into the §4 brief is **NOT OBSERVED** for a live boundary (reason: §1).

## 4. Honest multi-option presentation — observed surface vs. specified behavior

- **Observed surface (binary).** The preserved, superseded evidence records only a binary
  decision: `sop approve <task-id>` and `sop decline <task-id>`
  (`docs/reports/finish-pre-performance-closure/FP-003-human-approval-boundaries.md`
  §2.1, citing `.agent-sdlc/archive/plan-phase2-human-decisions/tasks.json` task C2-002 and
  `.../plan.json`). The end-to-end flow C2-009 proves the boundary as
  `RUNNING -> Needs Attention -> Inspect -> Approve -> SOP records -> controller reflects ->
  explicit Continue -> SOP resumes` — approve/continue as the actionable choice.
- **Specified behavior (multi-option).** PRD §6 requires presenting more than one valid
  action explicitly (Options A/B/C with per-option impact) and forbids reducing a
  multi-option decision to arbitrary `approve/decline`. PRD §24 gives the C-option example
  (A approve/continue, B decline/stop, C request additional verification).
- **Gap.** Honest multi-option-with-impact display is specified but its producer is not
  observable in `sop-controller`; only binary approve/decline is evidenced. The controller
  holds no decision-rendering state machine and delegates all lifecycle/approval
  operations to the SOP CLI (`WRAP-007-BOUNDARY-VERIFICATION.md`,
  `WRAP-007-FINAL-VERIFICATION.md`, cited in FP-003 §2.2). We do **not** claim multi-option
  support exists.

## 5. Smallest change path — proposal (outside FP-004 scope)

Presentation is owned by the SOP lifecycle engine / harness, not by `sop-controller`
(§2; FP-003 §2.2: all workflow commands delegate to the SOP CLI via `Commander.Exec`,
no controller-side decision state machine). The smallest coherent change is therefore in
that owning layer, and **implementing it is a separate, separately authorized change**,
outside FP-004's scope — this task mutates no source code (see §7).

Proposal (presentation layer only):

1. Where a genuine gate is recorded and rendered for a human, render the PRD §4 block from
   the already-recorded approval fields (`task_id`, `kind`, `target`, `reason`, `evidence`,
   `stage`, `disposition`, `status`, `requested_at`, `task_status`) rather than emitting
   the raw `TASK-ID NEEDS_HUMAN / approve / decline` line. This adds no new data source —
   it renders fields that already exist (§3, FP-003 §2.1).
2. Populate the `Recommended` / `Why` fields only when repository evidence supports a
   recommendation; otherwise emit the PRD §5 literal
   "No recommendation — the available evidence does not distinguish the alternatives
   reliably." (advisory only; the human remains authoritative).
3. When more than one valid action exists, enumerate the options with per-option `Impact`
   (PRD §6), instead of collapsing to approve/decline.

Rationale for "smallest": it reuses the existing recorded approval fields (no new state,
no new schema) and confines the change to the rendering/presentation step of the layer
that already owns it. No dependency, ownership, or lifecycle rule is changed. Acting on
this proposal requires the source-mutation authority that FP-004 explicitly lacks
(PLAN-Finish-Pre-Performance-Closure "Rules for evidence tasks" 2 and 6; FP-004 statement
"no production-code change is required").

## 6. Recommendations

For each finding, a recommendation is given **only** where the preserved evidence supports
it.

- **Finding: the six required brief fields are not produced by any component observable in
  this repository.**
  Recommendation: **No recommendation** — the available evidence does not distinguish the
  alternatives reliably (PRD §5). Reason: the live boundary is NOT OBSERVED (§1), so
  whether the owning SOP CLI/harness already renders the §4 brief cannot be observed, and
  the preserved evidence records only the raw structured approval fields. Without that
  observation the choice (render in the SOP layer vs. leave as-is because it is already
  rendered there) cannot be distinguished from evidence.
- **Finding: honest multi-option presentation is specified but only a binary
  approve/decline surface is evidenced (§4).**
  Recommendation: **No recommendation** — the available evidence does not distinguish the
  alternatives reliably (PRD §5). Reason: the observed surface is preserved/historical;
  the producer of option enumeration lives in a layer not observable here, so the smallest
  safe way to present options cannot be selected on evidence.
- **Finding: a smallest change path is identifiable at the presentation layer (§5) and it
  reuses existing recorded fields with no new state.**
  Recommendation: the §5 proposal is stated as a **proposal, not an implemented change**;
  whether to authorize it is a decision that requires source-mutation authority FP-004
  does not have, so no implementation recommendation is asserted here.

No other recommendations are given. Where the preserved evidence does not distinguish the
alternatives, the PRD's literal "no recommendation" wording is used explicitly.

## 7. Mutation statement

This stage is evidence-only. It creates this single report artifact under
`docs/reports/finish-pre-performance-closure/`. No production source code was changed, no
`.agent-sdlc` state was modified, and no approval command (`sop approve` / `sop decline`)
or any other approval-mutating operation was run. No live human boundary was created and no
`DECISION NEEDED` text was fabricated. The smallest change path in §5 is a proposal only;
implementing it is a separate authorized change outside FP-004's scope.

## 8. Scope boundary

This artifact stays within the bounded first-run FP-004 scope. It does not expand into plan
stages FP-005..FP-010 and does not execute the deferred CLOSE-004/CLOSE-005 work. It uses
the PRD's terminology (DECISION NEEDED, humanized decision brief, recommendation, options,
impact, technical details).

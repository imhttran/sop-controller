# FP-006 — Assess Safe Parallel Execution Opportunities

Evidence-only task. This artifact determines, from the recorded task dependency graph and
the repository/lifecycle evidence available in this harness, which runnable tasks are
genuinely independent and safe to run concurrently. It checks the conditions required by the
plan (dependencies, write ownership, shared mutable artifacts, lifecycle, approval,
evidence-order, repository-state) and recommends sequential execution wherever safety
cannot be proven. It does not invent a dependency to serialize work and does not remove a
real dependency to parallelize it.

Every claim cites a repository source. Anything that cannot be established from available
evidence is marked UNAVAILABLE / UNPROVEN with its reason and owning layer, never asserted as
a pass.

## 1. Method and evidence sources

### 1.1 Required independence conditions (verbatim from the PRD, plan appendix §7)

The PRD requires, before parallel execution, that the following conditions be verified:

```text
dependencies satisfied
no conflicting write ownership
no shared mutable artifact conflict
no lifecycle dependency
no approval dependency
no evidence-order dependency
no repository-state dependency
```

If those conditions cannot be proven: `run sequentially` (PRD §7).

### 1.2 PROVEN / UNPROVEN determination rule per condition

| Condition (PRD term) | Counts as PROVEN when | Counts as UNPROVEN when |
|---|---|---|
| dependencies satisfied | Every declared dependency of every member of the group is recorded as completed in the recorded plan/provenance, and the group is computed as runnable from the plan DAG. | A declared dependency's completion state cannot be established from recorded evidence, or the group bypasses a declared dependency. |
| no conflicting write ownership | The declared deliverable paths of the members are pairwise distinct and no member writes a path another member writes. | A shared written path exists, or live write ownership (an in-flight writer) cannot be observed. |
| no shared mutable artifact conflict | No member mutates a shared mutable artifact (`.agent-sdlc` state, a common report file, a shared build output). | A shared mutable artifact is (or may be) written by more than one member, or shared mutable state is SOP-owned and unobservable. |
| no lifecycle dependency | No member's execution changes the lifecycle state another member depends on (task transitions, reconcile, plan supersession). | A member triggers a lifecycle transition on which another member depends, or the live lifecycle state cannot be observed. |
| no approval dependency | No member requires, creates, or consumes a human approval the other member depends on. | An approval read/create is required by a member, or live pending-approval state cannot be queried. |
| no evidence-order dependency | No member consumes evidence another member produces (report ordering). | One member's deliverable is an input to another member's assessment. |
| no repository-state dependency | No member's result depends on repository state the other member changes (files, git state). | A member mutates repository files another member reads/assesses, or repository state is concurrently mutable. |

### 1.3 Evidence inputs available in this harness (all verified to exist by inspection)

- `.agent-sdlc/plan.json` — the machine plan: stages FP-001..FP-010 with objective,
  `dependencies`, `deliverables` and `acceptance_criteria` for each stage.
- `.agent-sdlc/plan.meta.json` — recorded provenance: plan_id
  `plan-finish-pre-performance-closure`, source
  `docs/plans/PLAN-Finish-Pre-Performance-Closure.md`, source_sha256
  `a0b83d5052b99c4234bcaedea4ccdf3b53bb3011e81d998d9c1d6146434f30ab`,
  `reconciled_tasks` `["FP-001"]` (quoted in FP-001 §3).
- The per-stage `Dependencies` blocks in the source plan, mirrored by the `dependencies`
  fields in `.agent-sdlc/plan.json` (authoritative DAG for this assessment).
- The declared deliverable paths per stage (recorded in `.agent-sdlc/plan.json` and the
  source plan).
- Existing FP artifacts under `docs/reports/finish-pre-performance-closure/`: FP-001,
  FP-003, FP-004, FP-005 (present; enumerated by this directory listing).
- `.agent-sdlc/config.yaml` — validation (build/test/lint), review engine, quality gate,
  and `human.approval_before_commit: true`.

### 1.4 Sources NOT available (must not be claimed as available)

- Live `sop status` / `sop task` / `sop approvals` output. FP-001 §5–§6 records both as
  refused by this harness (REQUIRES_APPROVAL), so no live task list or pending-approval
  state can be quoted. All DAG facts here are taken from recorded plan artifacts, not live
  queries.
- Live read/write ownership of in-flight concurrent writers. The capability record
  "Live read/write ownership and shared-artifact inspection surface" is PARTIAL: only
  static, read-only repository inspection is available.
- Sibling repository (`agentic-sop`) state. FP-001 §2 records it UNAVAILABLE (the harness
  cannot observe or mutate the sibling tree); the capability "Cross-root mutation of the
  sibling repository" is MISSING.
- Any implemented concurrency scheduler behind the PRD's conceptual AUTO/PARALLEL/
  SEQUENTIAL modes. The capability status is UNKNOWN (owner: SOP lifecycle engine /
  harness (execution orchestration)); no repository evidence of a controller-side worker
  pool or parallel dispatch was inspected.

## 2. Candidate parallel groups from the recorded DAG

### 2.1 Declared dependency edges (cross-checked against `.agent-sdlc/plan.json`)

From `.agent-sdlc/plan.json` `stages[].dependencies` (exact):

```text
FP-001  (no dependencies)
FP-002  -> FP-001
FP-003  -> FP-001
FP-004  -> FP-001
FP-005  -> FP-001
FP-006  -> FP-001
FP-007  -> FP-002
FP-008  -> FP-001
FP-009  -> FP-001
FP-010  -> FP-002, FP-003, FP-004, FP-005, FP-006, FP-007, FP-008, FP-009
```

These match the source plan's `Dependencies` blocks. No dependency was added or omitted:
FP-006's own declared dependency is FP-001 only, and FP-010 aggregates FP-002..FP-009.

### 2.2 Runnable set at the point of assessment

`.agent-sdlc/plan.meta.json` records `reconciled_tasks: ["FP-001"]`, i.e. only FP-001 is
reconciled as completed in the recorded provenance. The remaining first-run stages are
FP-002..FP-010. Their declared dependencies and deliverables:

| Stage | Declared dependencies | Declared deliverable path | Kind |
|---|---|---|---|
| FP-002 | FP-001 | `docs/reports/finish-pre-performance-closure/FP-002-no-progress-approval-trace.md` | evidence-only report |
| FP-003 | FP-001 | `docs/reports/finish-pre-performance-closure/FP-003-human-approval-boundaries.md` | evidence-only report |
| FP-004 | FP-001 | `docs/reports/finish-pre-performance-closure/FP-004-decision-presentation-review.md` | evidence-only report |
| FP-005 | FP-001 | `docs/reports/finish-pre-performance-closure/FP-005-attempts-accounting.md` | evidence-only report |
| FP-006 | FP-001 | `docs/reports/finish-pre-performance-closure/FP-006-parallel-safety-assessment.md` | evidence-only report |
| FP-007 | FP-002 | `docs/reports/finish-pre-performance-closure/FP-007-lifecycle-test-results.md` | evidence-only report |
| FP-008 | FP-001 | `docs/reports/finish-pre-performance-closure/FP-008-deterministic-gates.md` | evidence-only report |
| FP-009 | FP-001 | `docs/reports/finish-pre-performance-closure/FP-009-close-004-evidence-preservation.md` | evidence-only report |
| FP-010 | FP-002..FP-009 | `docs/reports/finish-pre-performance-closure/READINESS.md` | evidence-only report |

### 2.3 Maximal candidate parallel groups (derived, not invented)

A maximal candidate group is a set of stages that are simultaneously runnable (all
`dependencies` satisfied) and not pairwise dependent. Derived strictly from §2.1:

**Group A — FP-001's direct dependents once FP-001 is satisfied**
Members: FP-002, FP-003, FP-004, FP-005, FP-006, FP-008, FP-009.
Each declares exactly one dependency, FP-001, and none depends on another member of the
group. FP-007 is excluded because it declares FP-002.

**Group B — FP-007 once FP-002 is satisfied**
Members: FP-007 (alone). It is only runnable after FP-002, so it is a single-member group.

**Group C — FP-010 once FP-002..FP-009 are satisfied**
Members: FP-010 (alone). It aggregates all of FP-002..FP-009 and is not independent of any
of them.

No larger maximal group exists in the remaining first-run DAG: every other runnable stage
is a prerequisite of FP-010, and FP-007 is gated on FP-002. The grouping never bypasses a
declared dependency.

This assessment is performed at the point where FP-001 is reconciled; Group A is the only
candidate group with more than one member.

## 3. Per-group safety checks

For each condition, PROVEN requires the evidence in §1.2; otherwise UNPROVEN with reason
and owning layer.

### 3.1 Group A — FP-002, FP-003, FP-004, FP-005, FP-006, FP-008, FP-009

**Dependencies satisfied.**
- Status: PROVEN.
- Evidence: `.agent-sdlc/plan.json` records each member's `dependencies` as `["FP-001"]`
  only; `.agent-sdlc/plan.meta.json` records `reconciled_tasks: ["FP-001"]` (quoted in
  FP-001 §3), so FP-001 is satisfied and no member depends on another member.
- Owner: SOP lifecycle engine (plan DAG authority).

**No conflicting write ownership.**
- Status: PROVEN (static) / UNPROVEN (live writers).
- Evidence: declared deliverable paths are pairwise distinct (table in §2.2) and no member
  writes another member's path. No shared written file among Group A members.
- UNPROVEN part: ownership of any live, in-flight concurrent writer cannot be observed
  (capability "Live read/write ownership and shared-artifact inspection surface" is PARTIAL;
  FP-001 §5–§6 records SOP-owned live state as unreadable).
- Owning layer: SOP lifecycle engine / harness.

**No shared mutable artifact conflict.**
- Status: UNPROVEN for the `.agent-sdlc` state dimension; PROVEN only for the static report
  paths.
- Evidence: members write only their own distinct `docs/reports/...` files. However, every
  FP stage's execution is recorded by SOP into `.agent-sdlc/runs/<task-id>/` and can update
  plan/lifecycle state. `.agent-sdlc` is SOP-owned mutable state that this harness cannot
  inspect live (FP-001 §5–§6). Whether two members mutating SOP-owned state concurrently is
  safe depends on the unimplemented/UNKNOWN concurrency scheduler.
- Owning layer: SOP lifecycle engine / harness (execution orchestration).

**No lifecycle dependency.**
- Status: UNPROVEN.
- Evidence: each member, on completion, causes SOP-owned lifecycle transitions (task state,
  plan reconcile). Live lifecycle state cannot be observed from this harness (FP-001 §5–§6;
  capability "Human approval gate (list/inspect/decide)" PARTIAL). Whether concurrent
  lifecycle transitions are safe cannot be positively excluded.
- Owning layer: SOP lifecycle engine / harness.

**No approval dependency.**
- Status: UNPROVEN.
- Evidence: `.agent-sdlc/config.yaml` sets `human.approval_before_commit: true`, and the
  approval surface exists (`sop approvals`/`approve`/`decline`, FP-003 §2.1). Live
  pending-approval state cannot be queried from this harness (FP-001 §5–§6), so an approval
  dependency in a candidate group cannot be positively cleared by a live query.
- Owning layer: SOP lifecycle engine / harness.

**No evidence-order dependency.**
- Status: PROVEN (static).
- Evidence: no Group A member's deliverable is declared as an input to another Group A
  member; each is a self-contained evidence report (§2.2). FP-010, which does consume
  FP-002..FP-009 as evidence, is excluded from Group A.

**No repository-state dependency.**
- Status: UNPROVEN.
- Evidence: FP-002/FP-003/FP-004/FP-005/FP-006/FP-008/FP-009 are read-mostly, but FP-008 and
  FP-009 read git/repository state (deterministic gates; CLOSE-004 evidence preservation)
  and any concurrent mutation of the repository by another member (or by SOP) would change
  that state. Live in-flight writers and git state ownership are not observable
  (FP-001 §2, §5–§6). Whether repository state is stable across concurrent members cannot
  be proven.
- Owning layer: SOP lifecycle engine / harness.

Group A verdict: **SEQUENTIAL.** Four of the seven required conditions (shared mutable
artifact, lifecycle, approval, repository-state) are UNPROVEN, each because their safety
would depend on live SOP-owned state or on the UNKNOWN/unimplemented concurrency scheduler,
both of which cannot be observed or cleared from this harness. Per PRD §7, when the
conditions cannot be proven, run sequentially.

### 3.2 Group B — FP-007 (single member)

A single-member group has no concurrency to govern; it is trivially "sequential" and cannot
conflict with itself.

- Dependencies satisfied: PROVEN (`FP-007 -> FP-002`; runnable only after FP-002).
- No conflicting write ownership / shared artifact / lifecycle / approval / evidence-order /
  repository-state conflict **within the group**: PROVEN (single member).
- Recommendation: SEQUENTIAL (single-member group; no parallel opportunity).

### 3.3 Group C — FP-010 (single member)

- Dependencies satisfied: only when FP-002..FP-009 are complete; FP-010 is the aggregate
  readiness stage and cannot be independent of its prerequisites.
- Single member: no intra-group concurrency. Its correctness also requires FP-002..FP-009
  evidence to exist first (evidence-order), which is already enforced by its declared
  dependencies.
- Recommendation: SEQUENTIAL (single-member group; no parallel opportunity).

## 4. Recommendations

| Group | Members | Recommendation | Deciding condition(s) |
|---|---|---|---|
| A | FP-002, FP-003, FP-004, FP-005, FP-006, FP-008, FP-009 | **SEQUENTIAL** | No shared mutable artifact conflict, no lifecycle dependency, no approval dependency, and no repository-state dependency are UNPROVEN: each depends on live SOP-owned state (unobservable here) and/or on the UNKNOWN/unimplemented concurrency scheduler. |
| B | FP-007 | **SEQUENTIAL** | Single-member group; no parallel opportunity. |
| C | FP-010 | **SEQUENTIAL** | Single-member group; aggregate stage gated on FP-002..FP-009. |

Every candidate group has exactly one recommendation, and every group with an UNPROVEN
applicable condition is recommended SEQUENTIAL.

### 4.1 Concurrency-scheduler gap (recorded, not hidden)

The safety of any true concurrency in Group A would rest on the PRD's conceptual
AUTO/PARALLEL/SEQUENTIAL scheduler. That capability is recorded with status UNKNOWN (owner:
SOP lifecycle engine / harness (execution orchestration)); no concurrency scheduler, worker
pool, or parallel dispatch was observed in this repository. This is recorded as an UNPROVEN
condition and as a gap assigned to the SOP lifecycle engine / harness. No stage of this plan
declares a requirement on that UNKNOWN capability, and this assessment does not depend on it.

### 4.2 No-invention / no-removal attestation

No dependency was invented to serialize work and none was removed to parallelize it. This
rests on the FP006-S2 cross-check: the dependency edges in §2.1 are taken verbatim from
`.agent-sdlc/plan.json` `stages[].dependencies` and matched one-for-one against the source
plan's `Dependencies` blocks (FP-006 -> FP-001; FP-002/003/004/005/008/009 -> FP-001;
FP-007 -> FP-002; FP-010 -> FP-002..FP-009). Group A does not bypass any declared dependency
(FP-007 is correctly excluded because of FP-002), and no dependency was added to justify the
SEQUENTIAL recommendation — the recommendation follows solely from UNPROVEN conditions.

### 4.3 Unobservable live state recorded as UNPROVEN (not as a pass)

- Live write ownership / in-flight writers — UNPROVEN; owner SOP lifecycle engine / harness.
- Live lifecycle/approval state (`sop status`, `sop approvals`) — UNPROVEN; owner SOP
  lifecycle engine / harness (FP-001 §5–§6).
- Sibling (`agentic-sop`) state — UNAVAILABLE; owner operator / SOP process boundary.
  Every sibling-repository interaction is read-only; no sibling mutation is proposed or
  attempted.

## 5. Acceptance criteria mapping

- **The deliverable exists; every parallel candidate is justified against the required
  independence conditions.** — Section 2 enumerates every candidate group derived from the
  recorded DAG and §3 evaluates each against all seven PRD conditions with PROVEN/UNPROVEN
  status and evidence.
- **Where safety cannot be proven, sequential execution is recommended.** — §3.1 and §4
  recommend SEQUENTIAL for Group A because four conditions are UNPROVEN; Groups B and C are
  single-member groups with no parallel opportunity.
- **No dependency is invented to serialize work, and none is removed to parallelize it.** —
  §4.2 attests this, citing the §2.1 cross-check against `.agent-sdlc/plan.json` and the
  source plan.

## 6. Mutation statement

This stage is evidence-only. It creates this report artifact. No production source, template,
or `.agent-sdlc` state was modified, and no human approval was requested or created.
Sibling-repository interactions are read-only.

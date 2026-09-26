# SOP Controller — Plan Organization

**Last updated:** 2026-09-30

## Overview

Human-authored SOP plans for `sop-controller` live under `docs/plans/`. This document establishes clear purposes for each plan and provides guidelines for adding new plans.

One exception: the plan SOP records as **active** stays at the top level of `docs/`, because SOP matches a plan by its recorded source path. Today that is `docs/PLAN-SOP-Controller.md`. Moving an active plan would invalidate `.agent-sdlc/plan.meta.json` and break `sop run`, so it stays in place until SOP records a different active plan.

Other documentation — requirements, architecture, guides, reference, specifications, and history — lives in per-category subdirectories of `docs/`. [README.md](../README.md) is the documentation index; see [Documentation Categories](#documentation-categories).

## Plan Directory Structure

```
docs/
├── README.md                — Documentation index (start here)
├── PLAN-SOP-Controller.md   — ACTIVE plan (SOP-recorded path; intentionally not moved)
│
├── requirements/            — What we are building and why (PRD)
├── architecture/            — System structure and ownership boundaries
├── specs/                   — Normative required behavior (MUST / SHOULD)
├── plans/                   — What work needs to happen (PLAN-*, PRECHECK)
├── reference/               — Interfaces, values, commands, states
├── guides/                  — How to use or develop it
└── history/                 — Point-in-time artifacts (CTRL001, WRAP, PLAN.md)
```

## Documentation Categories

`docs/` separates documentation by kind so each document has one clear purpose:

| Directory | Contents |
| --- | --- |
| `requirements/` | Product intent: the PRD (what we are building and why). |
| `architecture/` | System structure and ownership boundaries (for example the SOP boundary). |
| `specs/` | Normative required behavior, using MUST / MUST NOT / SHOULD / SHOULD NOT / MAY. |
| `plans/` | Implementation work (`PLAN-*`, `PRECHECK`). |
| `reference/` | Concrete interfaces, values, commands, states, and configuration. |
| `guides/` | How a developer or operator accomplishes a task (running, discovery, development). |
| `history/` | Point-in-time artifacts, non-normative (`CTRL001/`, `WRAP/`, the historical v1 `PLAN.md`). |

Authority flows product intent → specs → architecture → plans → SOP execution;
see the [documentation index](../README.md#documentation-authority).

Guidelines:

1. **One authoritative location.** A requirement lives in exactly one document;
   related documents link to it rather than restating it.
2. **Specifications are normative.** Each specification begins with a `Type:
   Normative specification` header, a `Purpose`, a `Related Specifications`
   list, and a `Normative Language` note. Do not add MUST/SHOULD language to
   descriptive guides or reference pages.
3. **Index every document.** Add every new document to [README.md](../README.md).
4. **Plans live in `docs/plans/`.** The only plan outside `plans/` is the
   SOP-recorded **active** plan, which stays at the top level so `sop run` keeps
   working.
5. **History is non-normative.** Completed verification artifacts go to
   `docs/history/` and must not become authoritative specifications.

---

## Plan Purposes and Scopes

### docs/history/PLAN.md

**Purpose:** Establish the v1 product architecture and implementation roadmap.

**Scope:**
- 12-phase v1 dashboard implementation (Phases 0–12)
- Architecture: Go + html/template + HTMX + CSS
- Persistence: SQLite only
- Guardrails: No React, Next.js, Node, or PostgreSQL in v1

**Authorship:** Human-authored. This is the foundational product plan.

**Execution:** Not a named SOP plan (too broad for a single workflow run); serves as reference and historical record.

**When to update:**
- When v1 architecture goals change
- When phases are completed/removed
- When guardrails are refined

---

### docs/plans/PLAN-Wrap-Up.md

**Purpose:** Verify the agentic-sop + sop-controller integration and prove the controller can be developed using the SOP workflow.

**Scope:**
- 12-task wrap-up verification (WRAP-001 through WRAP-012)
- Tests: named-plan resolution, bootstrap, resume, identity, new-file detection, quality gates
- Tests: controller UI, SOP boundary, security, operations
- Produces final PASS / NEEDS_HUMAN / FAIL report

**Authorship:** Human-authored. This is an active operational plan.

**Execution:** `sop run docs/plans/PLAN-Wrap-Up.md` (primary integration workflow).

**When to use this plan:**
- Completing integration work on sop-controller
- Verifying SOP named-plan features
- Validating controller against SOP boundary

---

### docs/plans/PLAN-Hardening.md

**Purpose:** Harden the SOP Controller to make `sop run` the reliable end-to-end execution path.

**Scope:**
- 14 hardening tasks (SC-001 through SC-014)
- Tasks: SOP service boundary, lifecycle integration, concurrency protection, timeout unification, restart safety, UX improvements, failure visibility, forward compatibility
- Produces final PASS / NEEDS_HUMAN / FAIL report

**Authorship:** Human-authored. This is an active operational plan.

**Execution:** Can be run as `sop run docs/plans/PLAN-Hardening.md` if needed (optional secondary workflow).

**When to use this plan:**
- Improving reliability of `sop run` integration
- Enhancing controller's SOP boundary
- Hardening concurrency and recovery semantics

---

### docs/requirements/PRD.md

**Purpose:** Product Requirements Document defining controller behaviors, features, and constraints.

**Scope:**
- Functional requirements (FR-1 through FR-10+)
- Architecture constraints (SOP authority, no second state machine)
- Non-goals (no React, no PostgreSQL, no unrelated refactoring)

**Authorship:** Human-authored. This is reference documentation.

**Execution:** Not an SOP plan; reference for requirements validation.

---

### docs/plans/PRECHECK.md

**Purpose:** Pre-execution validation checklist before running controller workflows.

**Scope:**
- Repository identity checks
- Baseline build and test status
- Git state verification
- Environment readiness

**Authorship:** Human-authored. This is reference documentation.

**Execution:** Not an SOP plan; run manually before major workflows.

---

## Naming Conventions

### Active Named Plans (executable with `sop run`)

- **Primary plans:** `docs/plans/PLAN-{Purpose}.md` where `{Purpose}` is descriptive (e.g., `PLAN-Wrap-Up.md`)
- **Historical plans:** `docs/history/PLAN.md` — the original foundational plan (kept for reference)
- **Format:** Top-level header is the plan title; sections are task definitions with clear objectives, requirements, and acceptance criteria

### Guidelines

1. **Clear identity:** Plan filename and title must make the plan's purpose obvious
2. **Executable structure:** Use top-level sections for SOP tasks; each task has an objective, requirements, and acceptance criteria
3. **Dependencies:** Document task dependencies explicitly
4. **Commands:** Include the exact `sop run` command(s) that invoke each plan
5. **No machine-generated root artifacts:** Do not store machine-generated PLAN.md at the repository root; SOP plans stay under `docs/`

## Creating New Plans

When adding a new human-authored plan:

1. **File location:** `docs/plans/PLAN-{Purpose}.md`
2. **Naming:** Use kebab-case for multi-word purposes (e.g., `PLAN-CI-Hardening.md`)
3. **Title:** Clear, descriptive markdown heading
4. **Structure:**
   - Overview/Objective at the top
   - Task sections with: Objective, Requirements, Acceptance Criteria, Dependencies (if any)
   - Dependency graph (if complex)
5. **Execution:** Document the exact command: `sop run docs/plans/PLAN-{Purpose}.md`
6. **Review:** Ensure the plan aligns with SOP boundary principles (SOP owns workflow; controller observes)

### Template

```markdown
# PLAN-{Purpose}

## Objective

One sentence: what this plan achieves.

## Target

One sentence or brief diagram: desired end state.

## Tasks

### Task Title

**Depends on:** (none, or list prerequisite tasks)

#### Objective

What this task accomplishes.

#### Requirements

What must be true before starting.

#### Acceptance Criteria

How to know the task is complete.

---

## Dependency Graph

(Optional; include if task count > 3)

```
Task1 -> Task2 -> Task3
```

---

## Execution

```bash
sop run docs/PLAN-{Purpose}.md
```
```

## Quality Gates

All new plans must:

✓ Have a clear, single purpose  
✓ Define success/acceptance criteria for each task  
✓ Not duplicate another existing plan's scope  
✓ Live in `docs/plans/` (not repository root)  
✓ Be executable via `sop run docs/plans/PLAN-*.md`  
✓ Document the complete command  
✓ Respect SOP authority (SOP owns workflow logic; controller observes)

## No Ambiguous Artifacts

- **Repository root:** No `PLAN.md` generated by SOP or any earlier run
- **docs/ is authoritative:** All human-authored plans belong under `docs/plans/`, except the active plan at `docs/PLAN-SOP-Controller.md`
- **Category subdirectories:** Documentation lives in `docs/requirements/`, `docs/architecture/`, `docs/specs/`, `docs/plans/`, `docs/reference/`, `docs/guides/`, and `docs/history/`
- **Machine state:** SOP runtime state lives in `.agent-sdlc/` (gitignored)
- **Generated reports:** SOP reports live in `docs/reports/` (not root)

## Verification

To verify plan organization is clean:

```bash
# No PLAN files at root
find . -maxdepth 1 -name "*PLAN*" -type f
# Expected: (no output)

# Plans live in docs/plans/ (plus the active plan kept at the docs/ top level)
ls -la docs/plans/PLAN*.md docs/PLAN-SOP-Controller.md

# Verify each plan can be executed
sop run --help | grep -A5 "sop run"
sop run docs/plans/PLAN-Wrap-Up.md --dry-run  # if supported
```

## Historical Notes

- **Before 2026-09-27:** Root `PLAN.md` was generated by earlier SOP versions; no longer needed
- **2026-09-27:** WRAP-002 established docs/ as the authoritative location for human-authored plans
- **2026-09-30:** Added the documentation index (`docs/README.md`) and the `architecture/`, `guides/`, `reference/`, and `specs/` category subdirectories; at that time plans remained at the top level of `docs/`
- **2026-09-30:** Reorganized `docs/` into `requirements/`, `plans/`, and `history/` (with `history/CTRL001/` and `history/WRAP/`). The SOP-recorded active plan (`docs/PLAN-SOP-Controller.md`) was intentionally left at `docs/` so `sop run` stays valid; the archived `PLAN-Wrap-Up.md` moved to `docs/plans/`.
- **Reasoning:** Separating human-authored plans (docs/) from machine-generated state (.agent-sdlc/) makes intent clear and keeps the repository root clean

## Questions?

If a plan's purpose is unclear:
1. Check its "Objective" section first
2. Read its "Scope" section (if this doc describes it)
3. If still unclear, mark it for clarification or consolidation in the next wrap-up

---

**Document maintained as part of:** WRAP-002 — Normalize Plan Organization

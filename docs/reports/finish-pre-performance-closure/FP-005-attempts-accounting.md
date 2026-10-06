# FP-005 — Investigate Attempts Accounting

**Verdict: intentional semantics.** No defect is proven; no fix is proposed; no
production code, template, or `.agent-sdlc` state was modified by this task.

This is an evidence-only investigation (PLAN-Finish-Pre-Performance-Closure.md
FP-005). It determines, from repository evidence, whether a BLOCKED task
reporting `Attempts: 0` is intentional attempts-accounting semantics or an
accounting defect. Per the plan (§15): "Do not modify it merely because zero
appears suspicious."

---

## 1. Evidence sources

### 1.1 SOP CLI commands that expose task state and attempts

Per PLAN-Finish-Pre-Performance-Closure.md (Capabilities: "SOP plan and task
lifecycle CLI — EXISTS"):

- `sop status` — active plan and project overview.
- `sop task <id>` — one task's status, attempts, run artifacts.
- `sop approvals` / `sop approval <id>` — authoritative approval listing.
- `sop run`, `sop retry`, `sop resume` — lifecycle transitions.

These are the SOP-owned surfaces that report attempt state. *Values are
observed in §2; nothing is asserted in this section.*

### 1.2 Read-only state.db surfaces (tables)

The controller reads SOP's authoritative SQLite state read-only:

- `internal/sopclient/store.go` — `StatePath` = `<root>/.agent-sdlc/state.db`
  (store.go:29-31); `OpenStore` opens read-only with
  `mode=ro&_pragma=busy_timeout(5000)` (store.go:34-47).
- `Store.Tasks` query: `SELECT id, title, status, COALESCE(blocked_reason,''),
  attempt, max_attempts, updated_at FROM tasks ORDER BY id` (store.go).
- `Store.Task` query: same columns plus objective/acceptance_criteria, for
  `WHERE id = ?` (store.go).
- `Store.attempts` query: `SELECT number, status, COALESCE(reason,''), duration,
  timestamp FROM task_attempts WHERE task_id = ? ORDER BY number` (store.go).
- Schema (from `internal/sopclient/store_test.go`, the schema constant):
  `tasks(... status TEXT NOT NULL, blocked_reason TEXT, attempt INTEGER NOT NULL
  DEFAULT 0, max_attempts INTEGER NOT NULL DEFAULT 1, ...)`;
  `task_attempts(task_id, number, status, reason, output, duration, timestamp,
  PRIMARY KEY (task_id, number))`.

### 1.3 Run-artifact directory layout

`internal/sopclient/store.go` reads `.agent-sdlc/runs/<id>/` artifacts, including
`report.json`, `validation.json`, `review.json`, and classification artifacts;
`types.go` documents `Performance` from
`.agent-sdlc/runs/<task>/metrics.json or report.json`.

### 1.4 Controller projection / rendering path

- `store.go` scans `attempt, max_attempts` from `tasks` into
  `TaskSummary.Attempt` / `TaskSummary.MaxAttempts`.
- `internal/sopclient/types.go`: `TaskSummary.Attempt int`, `MaxAttempts int`;
  `TaskDetail.Attempts []Attempt` sourced from `task_attempts`.
- `templates/partials/task_list.html` renders `{{.Attempt}}/{{.MaxAttempts}}`
  under `<th>Attempts</th>` / `data-label="Attempts"` — a raw projection with no
  arithmetic.
- `templates/partials/attempts.html` renders per-attempt rows with
  `{{if .Attempts}}{{range .Attempts}}`.
- `internal/web/recovery_test.go` asserts `data-label="Attempts"` appears in
  rendered output.

---

## 2. Observed evidence (this checkout)

Reads performed read-only against `.agent-sdlc/` and `internal/sopclient/`.

### 2.1 Live task row for CLOSE-004

**NOT OBSERVED — task not present in this checkout's run set.**

`.agent-sdlc/runs/` contains `C2-001…C2-011`, `CTRL001…CTRL017`, `CTX-001`,
`FP-001…FP-005`, `HARD001…HARD008`, `WRAP-001…WRAP-012`, and plan-level
artifacts. **There is no `CLOSE-004/` directory** under `.agent-sdlc/runs/`, so
no run artifacts exist for CLOSE-004 here. The current `.agent-sdlc/state.db`
contents could not be read through this harness (the state database is
SOP-state-protected; `read_file` refused `.agent-sdlc/state.db`), so the live
`tasks` row for CLOSE-004 is recorded as **NOT OBSERVED**, with the reason
"SOP state is not directly readable from this harness; no CLOSE-004 run
directory is present."

Therefore the prior external observation
`CLOSE-004 / BLOCKED / Attempts: 0` (PLAN §15) is a **hypothesis to be
confirmed or refuted**; in the current checkout it **did not reproduce as a
live row** and is **NOT OBSERVED**.

### 2.2 Live `task_attempts` rows for CLOSE-004

**NOT OBSERVED** — no CLOSE-004 task or run directory exists in this checkout
(same reason as §2.1).

### 2.3 Run artifacts for CLOSE-004

**Absent** — `.agent-sdlc/runs/CLOSE-004/` does not exist.

### 2.4 Verbatim evidence: seeded BLOCKED / attempt=0 row

Because no live CLOSE-004 row exists here, the authoritative evidence for how
the controller handles `Status=BLOCKED, attempt=0, max_attempts=3` is the
repository's own tests, quoted verbatim:

`internal/sopclient/store_test.go`, `TestSummaryTasksAndBlocking`:

```sql
INSERT INTO tasks VALUES ('c','Task C','obj','ac','BLOCKED','REVIEW_UNRESOLVED',0,3,'<now>','<now>')
```

`internal/sopclient/store_test.go`, `TestFixCyclesAndRetryable`:

```sql
INSERT INTO tasks VALUES ('a','A','o','a','BLOCKED',NULL,1,3,'<now>','<now>')
INSERT INTO tasks VALUES ('b','B','o','a','BLOCKED',NULL,3,3,'<now>','<now>')
```

Assertions over these rows (verbatim intent):

```go
// a: BLOCKED with budget left -> retryable
if a := byID["a"]; a.FixCycles != 2 || !a.Retryable() { ... }
// b: BLOCKED but budget spent -> not retryable
if b := byID["b"]; b.Retryable() { ... }
```

`internal/sopclient/store_test.go`, `TestTaskDetailReadsArtifacts`:

```sql
INSERT INTO tasks VALUES ('t2','Task Two','do the thing','works; tested','FIX_REQUIRED','build failed',2,3,'<now>','<now>')
INSERT INTO task_attempts VALUES ('t2',1,'FIX_REQUIRED','build failed','boom',1500000000,'<now>')
```

Note that the seeded BLOCKED task `c` in `TestSummaryTasksAndBlocking` has
**`attempt=0` and no `task_attempts` rows**, and the test asserts only its
blocking/eligibility behavior — the repository treats `attempt=0` on a BLOCKED
task as a valid, ordinary state, not an error.

### 2.5 Verbatim semantics from types.go

```go
// Retries returns how many attempts SOP has already spent: attempt-1 when
// attempt > 0, and (0, false) when SOP recorded no attempt yet, so a genuine
// zero is distinguishable from an absence.
func (t TaskSummary) Retries() (int, bool) {
	if t.Attempt <= 0 {
		return 0, false
	}
	return t.Attempt - 1, true
}

// Retryable reports whether this is BLOCKED work SOP still has retry budget for.
func (t TaskSummary) Retryable() bool {
	return t.Status == StatusBlocked && t.Attempt < t.MaxAttempts
}
```

---

## 3. Semantics analysis and verdict

### 3.1 What each value represents

- **`tasks.attempt`** — the scalar counter of attempts SOP has already spent on
  the task. It is persisted by SOP (schema default `0`), projected verbatim by
  `store.go` into `TaskSummary.Attempt`, and rendered verbatim by
  `task_list.html`. It is **not** recomputed by the controller.
- **`task_attempts`** — the per-attempt event log (`number, status, reason,
  output, duration, timestamp`), read by `Store.attempts` and projected into
  `TaskDetail.Attempts`. It records individual attempt events.

These are two distinct sources: a scalar budget counter and an event log.

### 3.2 How retry/absence semantics interpret `attempt = 0`

- `Retries()` explicitly distinguishes a **genuine zero** ("SOP recorded no
  attempt yet") from an absence: `(0, false)` means "zero attempts spent and no
  attempts recorded," not a subtraction artifact. `Retries()` returns
  `(Attempt-1, true)` only when `Attempt > 0` — i.e., attempt counts *events
  already executed* on top of the first.
- `Retryable()` is `Status == BLOCKED && Attempt < MaxAttempts`. A BLOCKED task
  with `Attempt=0, MaxAttempts=3` is therefore **retryable by design**: SOP has
  budget left and has not spent an attempt.
- `NeedsHuman` is deliberately **not** derived from attempt counts,
  classification, stage, or BLOCKED status (types.go comment on
  `TaskSummary.NeedsHuman`: "BLOCKED status alone, run stage, model/recovery
  prose, attempt counts, and inactivity never set it"). So `Attempts: 0` is
  **not** a human-approval signal.

### 3.3 Is `Attempts: 0` on a BLOCKED row with no `task_attempts` entries consistent?

**Yes.** An `attempt = 0` BLOCKED row with no `task_attempts` entries is
consistent with SOP blocking **before spending a retryable attempt** — matching
the `NO_PROGRESS → BLOCKED` path in the plan (§12/FP-002), where no-progress
produces an operator-intervention block that does **not** create an approval.
`Retries()` returning `(0, false)` explicitly labels this a genuine absence of
spent attempts; `Retryable()` correctly reports retry budget remains, so a
subsequent `sop retry` remains a bounded, automatic action.

### 3.4 Does either source disagree?

Neither source can be shown to disagree: no CLOSE-004 row or artifact is present
in this checkout to exhibit a mismatch, and the repository's seeded BLOCKED /
`attempt=0` fixtures are asserted as valid, retryable-by-design states. A defect
would be demonstrated by an `attempt` scalar that **disagrees with the
`task_attempts` event log** (e.g., attempt events logged but the counter not
incremented, or vice versa). No such disagreement is observed in the available
evidence.

### 3.5 Verdict

**Intentional semantics.** The evidence shows `Attempts: 0` is a legitimate
state: `tasks.attempt` is a SOP-owned budget counter starting at 0, `Retries()`
labels a genuine zero explicitly, `Retryable()` treats `Attempt=0, MaxAttempts=3`
on BLOCKED as retryable budget, and `NeedsHuman` is never derived from attempt
counts. `0` is **not** classified as a defect on the strength of its appearance.
Per FP-005's acceptance criteria, `0` is not treated as a defect merely because
it appears suspicious, and no change is made.

**The prior observation `CLOSE-004 / BLOCKED / Attempts: 0` DID NOT REPRODUCE as
a live row in this checkout and is NOT OBSERVED** (no `CLOSE-004/` run directory;
SOP state not directly readable from this harness). It is therefore recorded as
a hypothesis, not a confirmed fact, and does not by itself establish a defect.

---

## 4. No fix proposed — no defect proven

**No fix proposed — no defect proven.** Because the FP-005 verdict (§3.5) is
"intentional semantics," §FP005-S4's conditional appendix is replaced by this
explicit statement. Per FP-005's acceptance criteria, there is never both a
proposal and an explicit no-proposal statement, and never neither: the report
contains exactly the latter.

Had a defect been proven, the owning layer would be identified as either the
**SOP state store/attempt increment** (sibling `agentic-sop`) or the
**controller projection** (`internal/sopclient`), and any proposal would be a
change specification only — never applied, because the harness has no capability
to mutate the sibling `agentic-sop` checkout (MISSING; PLAN capabilities). That
condition did not arise.

---

## 5. Verification checklist and no-change attestation

### 5.1 Acceptance criteria mapping

| Acceptance criterion | Where satisfied |
| --- | --- |
| Deliverable exists at declared path | This file, `docs/reports/finish-pre-performance-closure/FP-005-attempts-accounting.md` |
| Attempts semantics explained from evidence | §1 (sources), §2 (verbatim rows/queries), §3.1–3.4 |
| Zero not treated as a defect merely because it appears suspicious | §3.5 |
| No change made unless a defect is proven | §4 and §5.2 |

### 5.2 No-change attestation

Files **inspected** (read-only) during this task:

- `internal/sopclient/store.go`
- `internal/sopclient/types.go`
- `internal/sopclient/store_test.go`
- `templates/partials/task_list.html`
- `templates/partials/attempts.html`
- `docs/plans/PLAN-Finish-Pre-Performance-Closure.md`
- `docs/reports/finish-pre-performance-closure/FP-001-repository-lifecycle-state.md`
- `.agent-sdlc/` directory listing and `.agent-sdlc/runs/` listing (read-only)
- `.agent-sdlc/runs/FP-005/state.json`, `task.md`, `model-selection.json` (read-only)

**No** production source, template, or `.agent-sdlc` state was modified. No
sibling-repository mutation was attempted. The only file written by this task is
this deliverable.

## 6. Evidence appendix

Raw table/column definitions and query text are quoted inline in §1.2 and §2.4.
All quoted SQL and Go appear verbatim from the cited files. Values that could
not be established are marked **NOT OBSERVED** with their reason (§2.1–2.3).

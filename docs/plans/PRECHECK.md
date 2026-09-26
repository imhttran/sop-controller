# SOP Controller Pre-Check

Phase 0 verification gate, run before the controlled refactor from the donor
auth application to the Go + `html/template` + HTMX dashboard. Pre-checks
verify and report; they do not migrate or delete on their own.

```text
SOP Controller Pre-Check

[PASS] Go application builds
[PASS] Baseline tests pass
[PASS] SQLite is the active runtime database
[PASS] No PostgreSQL server required at runtime
[PASS] Legacy Next.js/React frontend identified and removed
[PASS] JWT / user-management / email / 2FA flows retired for V1
[PASS] SOP owns workflow state and legal transitions
[PASS] SOP service/API boundary implemented (internal/sopclient)
[PASS] Browser never accesses SQLite directly
[WARN] github.com/lib/pq was present in the donor backend
[TODO] network mode is opt-in and requires an access token

Result: READY FOR CONTROLLED REFACTOR
```

## Repository

- Expected repository and branch: `sop-controller` on `main`.
- The working tree was **not** clean at pre-check time — the SOP orchestrator
  had just rewritten `PRD.md` / `PLAN.md`. The dirty state was recorded rather
  than overwritten.
- Baseline build and tests: the donor backend built and tested under `backend/`.
  The refactor moved everything to **one Go module at the repository root**, so
  `go build ./...` / `go vet ./...` / `go test ./...` now run from the root.

## Persistence

- SQLite was already the active runtime store in the donor.
- No PostgreSQL server is required at runtime.
- `github.com/lib/pq` existed only for the one-off Postgres→SQLite importer;
  it was removed with the donor tree.
- The dashboard treats SOP's `.agent-sdlc/state.db` as **authoritative and
  read-only**; it opens it read-only and never migrates or writes it.

## Frontend

- Next.js/React and the Node/npm build pipeline were present and are removed.
- Node is no longer a runtime or build requirement.
- Target frontend confirmed: Go `html/template` + HTMX + CSS + minimal JS.

## Backend

- Inventoried and **removed** for V1: JWT auth, signup/login, user/profile
  management, email verification, password reset, 2FA/trusted devices, and the
  email queue/worker.
- **Kept/adapted**: Go HTTP service skeleton, configuration loading patterns
  (including `.env` handling), and the httptest-based test approach.

## SOP integration

- SOP owns workflow state and legal transitions; the dashboard reproduces
  neither.
- The boundary is `internal/sopclient`: reads open SOP's SQLite read-only, and
  commands shell out to the `sop` CLI (`run`, `resume`, `validate`, `review`,
  `report`).
- No duplicate authoritative task-state store exists.
- The browser never reaches SQLite; all reads are server-side.

## Notes

- Network mode (`SOP_CONTROLLER_ALLOW_NETWORK=true`) is implemented but off by
  default and requires `SOP_CONTROLLER_TOKEN`; the server otherwise binds to
  loopback only.

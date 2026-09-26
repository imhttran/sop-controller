# CTRL001 — Boundary Verification

> **Historical / non-normative.** Point-in-time artifact kept for traceability only; it does not define current behavior. See the [documentation index](../../README.md).

Verification, by code inspection and deterministic test, that the
controller-to-SOP boundary satisfies every CTRL001 acceptance criterion and adds
nothing beyond the boundary definition.

Contract: [`BOUNDARY-CONTRACT.md`](BOUNDARY-CONTRACT.md)
Code: `internal/sopclient/boundary.go`
Tests: `internal/sopclient/boundary_test.go`, `internal/web/boundary_test.go`

## Acceptance criteria

### 1. SOP remains the only lifecycle owner

- The controller never transitions task state. `internal/sopclient` reads SOP
  state and asks SOP to act only through `Commander` (`sop` CLI verbs); it has no
  code path that writes a task's status, stage, or history.
- Evidence: `internal/sopclient/boundary.go` (`Boundary` descriptors all read SOP
  artifacts or invoke a `sop` verb); `internal/sopclient/service.go` command
  methods (`Run`, `Resume`, `Retry`, `Reconcile`, ...) all delegate to `Commander.Exec`.
- Test: `TestBoundaryDoesNotMutateSOPPersistence`.

### 2. Controller operations map to SOP application operations

- All eleven PRD operations appear exactly once in `Boundary()`, each with its
  `sopclient` entry point and the SOP application operation it uses.
  `CancelRun` and `ApproveTask` are recorded as explicit unsupported gaps.
- Evidence: the mapping table in the contract document; `sopclient.Boundary` and
  `sopclient.Lookup`.
- Tests: `TestBoundaryContractMatchesPRD`, `TestReadOperationsReportSOPValues`,
  `TestCommandOperationsDelegateToSOP`.
- Gap tests: `TestUnsupportedOperationsReturnErrOperationUnsupported`.

### 3. Controller does not directly mutate SOP persistence

- No production controller code opens `state.db` for writing or writes run
  artifacts. The only direct database access is read-only
  (`OpenStore` opens `mode=ro`); command effects happen inside SOP, behind the
  `sop` CLI.
- Evidence: `internal/sopclient/store.go` (`mode=ro`), `internal/sopclient/artifacts.go`
  and `run.go` (read-only `os.ReadFile`), `internal/web/handlers.go` (uses only
  `h.sop.*`).
- Test: `TestBoundaryDoesNotMutateSOPPersistence` hashes every file under
  `.agent-sdlc/` before and after exercising every read and command operation and
  asserts they are byte-for-byte identical.

### 4. Controller does not implement scheduler decisions

- The controller never selects the next runnable task. `StartOrContinueRun`
  delegates to `sop run` / `sop resume`, which is where SOP chooses the task.
  The boundary exposes no selection operation.
- Evidence: `Boundary()` contains no selection/scheduling operation; the only
  operation that advances work is `StartOrContinueRun` → `sop run`/`sop resume`.
- Test: `TestBoundaryExposesNoSchedulerOperation`.

### 5. Existing CLI behavior remains usable

- The controller still drives SOP via the same commands. No `sop` verb, flag, or
  existing `sopclient` method changed; the boundary adds `CancelRun`/`ApproveTask`
  (which always report the gap) and documents the existing surface.
- Evidence: `internal/sopclient/commands.go` and the command methods in
  `service.go` are unchanged; `internal/web/server.go` routes are unchanged.
- Tests: `internal/web/server_test.go` (`TestCommandsReachSOPSafely`,
  `TestCommandWhitelistEnforced`, ...) and `internal/sopclient/commands_test.go`
  continue to pass.

### 6. Boundary is documented and testable

- Documented: `docs/history/CTRL001/BOUNDARY-CONTRACT.md` (invariants, operation table,
  gap records, consumer audit, test map).
- Testable: `sopclient.Boundary()`/`Lookup()` are exported data the tests assert
  against, so the contract cannot silently drift from the code.
- Tests: `internal/sopclient/boundary_test.go`,
  `internal/web/boundary_test.go`.

## Gap summary

| Operation | Status | Reason |
|-----------|--------|--------|
| `CancelRun` | unsupported | SOP exposes no `sop cancel` application operation |
| `ApproveTask` | unsupported | SOP exposes no `sop approve` application operation |

Both remain in the contract and return `ErrOperationUnsupported`. Neither is
simulated in the controller.

## Scope statement

No functionality beyond the CTRL001 boundary definition was added: no new
orchestration service, no scheduler logic, no cancellation/approval
implementation, no change to SOP persistence, and no change to existing SOP
commands. `agentic-sop` was not modified.

## Validation

    gofmt -l .            # no files
    go build ./...        # ok
    go vet ./...          # ok
    go test ./...         # ok

Baseline (before the change) and final runs are both green across
`sop-controller`, `internal/sopclient`, and `internal/web`.

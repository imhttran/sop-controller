# HARD008 Report --- Validation and Final Report

Point-in-time, non-normative record produced by the SOP plan for HARD008,
following the same audit format as [HARD001-REPORT.md](HARD001-REPORT.md) and
[HARD004-REPORT.md](HARD004-REPORT.md).

## Scope

HARD008 is the validation and reporting stage of the SOP Controller hardening
plan (`docs/PLAN-Hardening.md`, HARD001--HARD008). It runs the full Go
validation suite and answers twelve required points about the fixed-wait
monitoring work and the resulting observation model.

## 1. Origin of the original 240-second wait

The `sleep 240` / `tail -25 .run/sop-run.log` sequence is an **external, manual
operator shell workflow**. It is described only in the hardening plan's own
narrative (`docs/PLAN-Hardening.md:12-13`, restated at `:17` and `:164`):

```bash
sleep 240
tail -25 .run/sop-run.log
sop status ...
```

It was never a controller construct. An operator watching an active run reached
for a four-minute pause because no documented "observe without waiting"
workflow pointed them at the controller's own activity surfaces.

## 2. Category it existed in

**Only an operator/debugging workflow** --- not production code, not tests, not
scripts, and not any documentation other than the plan's own example of that
workflow. Verified by repository-wide search (see
[HARD001-REPORT.md](HARD001-REPORT.md) for the full transcript):

- No `.go` file contains `240`, `sleep 240`, or a `sop-run.log` reference.
- No `*.sh` file contains any of those terms.
- The only `time.Sleep` calls are 5--10 ms synchronization sleeps in
  `internal/web/{dogfood,reconcile,server}_test.go`, unrelated to the pattern.
- `docs/PLAN-Hardening.md` is the sole documentation match, and it quotes the
  external workflow to explain what NOT to do.

No code change was invented to "remove" a wait that never existed in the
controller. Per the plan, the observability gap behind the manual wait was
closed by documenting and testing the existing native surfaces instead.

## 3. Files changed

The HARD001--HARD007 hardening work landed across prior SOP tasks; HARD008 itself
adds only this report (`docs/history/HARD008-REPORT.md`, new). The
HARD008-relevant, already-present files that implement the observation model are:

- `internal/web/activity_stream.go` --- SSE stream + bounded-poll window.
- `internal/web/commands.go` --- asynchronous `CommandRunner` (running/done/error).
- `internal/web/handlers.go`, `internal/web/server.go` --- command start/status
  routes and runner wiring.
- `internal/web/approval.go`, `internal/web/reconcile_test.go` --- human-gate
  surfacing/delegation.
- `static/activity-live.js`, `templates/project.html` --- browser SSE consumer
  with bounded-poll fallback.
- `internal/config/config.go` --- `PollInterval` (default 3s), `CommandTimeout`
  (default 15m).
- `internal/web/observability_test.go` and
  `internal/web/observability_hard007_test.go` --- deterministic HARD007 tests.
- `docs/guides/EXECUTION-AND-OBSERVATION.md`,
  `docs/guides/DEVELOPMENT.md` --- execution vs observation vs polling vs
  timeout documentation.

## 4. How active commands are now observed

Starting a command and observing it are separate operations. The HTTP request
that starts a command (`POST /projects/{project}/commands/{verb}`) returns
immediately with a status fragment; it never blocks for the command's duration
(`internal/web/commands.go` records `CommandState{State: "running"}` and the
handler renders it without waiting). The in-flight command is then observable at
`GET /projects/{project}/commands/{verb}`; `CommandState` also surfaces the real
`TaskID`/`Stage`/`Attempt` when SOP exposes them, and never fabricates them
(`internal/web/handlers.go` `renderCommand`). Proven by
`TestProjectCommandObservedWhileRunning` and
`TestTaskCommandReturnsImmediatelyWithStage`.

## 5. How frequently fallback polling occurs

Polling is a *fallback delivery mode*, on the controller's existing configured
cadence `SOP_CONTROLLER_POLL` (default `3s`) --- not a new hard-coded value and
never a multi-minute wait. The server's `pollInterval()`
(`internal/web/activity_stream.go`) derives the ticker interval from that
configuration, and the browser client (`static/activity-live.js`) reads the same
cadence from `data-poll-ms`. `TestActivityPanelExposesConfiguredPollMs` asserts a
configured `250ms` reaches the rendered page, proving the cadence is
configurable rather than fixed.

## 6. How activity streaming is used

Activity streaming is the primary observation path. `GET
/projects/{project}/activity/stream` is an SSE endpoint
(`internal/web/activity_stream.go`) backed solely by the read-only
`sopclient.ActivityWindow` over SOP's persisted activity; it emits each newly
persisted event while a run is active and re-checks on the configured poll
cadence. It holds no lifecycle opinion: a transport failure simply ends the
stream so the client can reconnect (with `Last-Event-ID`) or fall back to
polling. The browser consumer `static/activity-live.js` subscribes via
`EventSource`, deduplicates by the stable cursor, and rebases on a `reset`
window. No second logging/event system was introduced. Proven by
`internal/web/activity_stream_test.go` and the activity isolation tests.

## 7. How terminal states are detected

Terminal states (`DONE`, `FAILED`, `BLOCKED`, `PASS`) are read from SOP's own
persisted `status`/`stage` fields --- never inferred from elapsed time. The
observation path renders SOP's status verbatim through pure string-to-string
mappings: `TaskState` (`internal/sopclient/types.go`), `statusClass`,
`stageClass`, and `levelClass` (`internal/web/render.go`). No function compares a
duration against a threshold to select a state; `elapsed()` and `since()`
produce a duration string or an explicit absence literal only. Confirmed by the
[HARD004-REPORT.md](HARD004-REPORT.md) audit and
`TestNoActivityDoesNotInferFailureOrBlockage`.

## 8. How human-decision states are surfaced

SOP's human-decision boundary is read from SOP's own `Approval` projection on the
task detail (`internal/web/approval.go`). When SOP reports a boundary
(`Approval.Present`) the UI surfaces "Needs your attention" with the appropriate
controls instead of reading as generic running work; when SOP reports no
boundary, nothing is surfaced. Approve/decline delegate only to SOP's
application operations and refuse with an explicit 409 when SOP exposes no such
operation, so the controller never infers or manufactures an approval. Changed
executed tasks are surfaced as actionable human decisions (reconcile is refused
while tasks are pending approval) rather than being polled for self-resolution.
Proven by `TestHumanGateDecisionSurfacedNotGenericRunning` and
`TestControllerAnswersOperationalQuestions`.

## 9. Whether CommandTimeout changed and why

`CommandTimeout` is a distinct concept from observation/polling and is unchanged
by HARD008: `SOP_CONTROLLER_COMMAND_TIMEOUT`, default `15m`
(`internal/config/config.go`), wired through `Options.CommandTimeout` into
`NewCommandRunner` (`internal/web/server.go`). It bounds one command's
execution, returns a truthful timeout error, never silently marks the SOP task
failed, and never mutates SOP lifecycle state --- proven by `TestCommandTimeout`
and `TestCommandTimeoutDoesNotMutateSOPLifecycleState`. It was deliberately left
as-is because the hardening goal is observability, not changing the execution
timeout.

## 10. Tests added

HARD007's deterministic observability tests are the ones added for this work;
all use short bounded intervals or channel synchronization and none use a real
multi-minute sleep:

- `internal/web/observability_test.go`:
  - `TestRunningCommandObservableWithoutWaiting`
  - `TestCommandFailureSurfacesPromptly`
  - `TestProjectCommandObservedWhileRunning`
- `internal/web/observability_hard007_test.go`:
  - `TestHumanGateDecisionSurfacedNotGenericRunning`
  - `TestNoActivityDoesNotInferFailureOrBlockage`
  - `TestQuietActivityWindowIsNotAFailure`
  - `TestCommandTimeoutSeparateFromPolling`
- Support tests referenced above: `TestTaskCommandReturnsImmediatelyWithStage`,
  `TestTaskCommandNoFabricatedStage` (HARD002), and
  `TestActivityPanelExposesConfiguredPollMs` (HARD003).

## 11. Validation results

Run from the module root (`go.mod` at the repository root):

| Command | Outcome |
| --- | --- |
| `gofmt -l .` | PASS (no output --- all files formatted) |
| `go vet ./...` | PASS (exit 0) |
| `go build ./...` | PASS (exit 0) |
| `go test ./...` | PASS (`sop-controller`, `internal/config`, `internal/sopclient`, `internal/web` all ok; `cmd/sop-controller` has no test files) |
| `go test -race ./...` | PASS on re-run with `-count=1` (all packages ok) |

**Truthful note on the race run.** A first `go test -race ./...` invocation
without `-count=1` exited non-zero with one `internal/web` failure:
`TestDogfoodNormalFlow` reported "state.db unchanged after the scripted run;
fixture mutation did not take effect". This is a pre-existing, order-dependent
flakiness in the dogfood fixture (which mutates a shared project root), not a
HARD008 defect: the test passes in isolation (`-run TestDogfoodNormalFlow
-count=1`) and the full suite passes with `-count=1`. It is recorded here as an
observed limitation rather than a passing result.

## 12. Remaining observability limitations

- **Dogfood test flakiness (recorded above):** `TestDogfoodNormalFlow` can fail
  under a first cached/parallel race run because the fake `sop` mutates a shared
  fixture root; it is stable with `-count=1`. Not fixed here (out of HARD008
  scope) and not masked as a pass.
- **SOP-side run log is not a controller surface:** the manual
  `tail -25 .run/sop-run.log` workflow the plan describes is an SOP-side
  artifact; the controller documents `.run/sop-controller.log` as the process log
  instead (`docs/guides/DEVELOPMENT.md`).
- **Approval applicability depends on SOP:** approve/decline/reconcile are
  offered only where SOP reports a boundary and exposes the operation; where SOP
  exposes none the action is refused with an explicit conflict rather than
disabled silently --- a truthful read-only gap, not a controller limitation.
- **No inactivity-based inference anywhere:** by design the controller never
  converts silence into a state; an operator sees no fabricated `BLOCKED`/
  `STUCK`, which means a genuinely quiet run looks quiet (truthful, but requires
  the operator to consult SOP's own state).

## Conclusion

The 240-second wait originated only in the plan's narrative as an external,
manual operator workflow; no production code, test, script, or doc other than the
plan example ever contained it, and no code change was invented to remove it. The
observation model relies on SOP's authoritative state, an SSE activity stream as
the primary path, a short configurable poll fallback, read-only terminal-state
detection, and explicit human-boundary surfacing. `CommandTimeout` is unchanged.
All five validation commands pass; the one observed non-deterministic race-run
failure is recorded truthfully as a pre-existing dogfood-fixture limitation.

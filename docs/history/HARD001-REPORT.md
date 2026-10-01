# HARD001 Report --- Fixed-Wait Monitoring Source

Point-in-time, non-normative record produced by the SOP plan `plan-hardening`.

Re-verified 2026-10-01: the search below was re-run against the current
working tree (clean, `0858633`) and reproduces the same matches and
conclusion as the original run. No new `.go`/`.sh` occurrence of `sleep 240`,
`240`, or `tail -25`/`sop-run.log` was introduced since the original report.

## HARD001-REPORT

### Search transcript

Terms searched: `sleep 240`, `time.Sleep`, `tail -25`, `sop-run.log`, literal
`240`, `poll`/`ticker`/`PollInterval`, `CommandTimeout`, across `*.go`, `*.sh`,
and `*.md`.

```
$ grep -rn "sleep 240" --include="*.go" --include="*.sh" --include="*.md" .
docs/PLAN-Hardening.md:12:sleep 240

$ grep -rn "time.Sleep" --include="*.go" .
internal/web/dogfood_test.go:289:   time.Sleep(10 * time.Millisecond)
internal/web/reconcile_test.go:98:  time.Sleep(10 * time.Millisecond)
internal/web/server_test.go:199:    time.Sleep(5 * time.Millisecond)
internal/web/server_test.go:270:    time.Sleep(10 * time.Millisecond)

$ grep -rn "tail -25\|sop-run.log" --include="*.go" --include="*.sh" --include="*.md" .
docs/PLAN-Hardening.md:13:tail -25 .run/sop-run.log
docs/PLAN-Hardening.md:80-81: (restates the search instruction itself)
docs/PLAN-Hardening.md:164: (quotes tail -f .run/sop-run.log as an alternative)

$ grep -rn "\b240\b" --include="*.go" --include="*.sh" .
(no matches)

$ grep -rln "ticker\|PollInterval\|pollInterval" --include="*.go" .
cmd/sop-controller/main.go
internal/config/config.go
internal/web/activity_stream.go
internal/web/activity_stream_test.go
internal/web/activity_isolation_test.go

$ grep -rln "CommandTimeout" --include="*.go" .
cmd/sop-controller/main.go
internal/config/config.go
internal/web/server.go
internal/web/dogfood_test.go
internal/web/server_test.go
internal/web/recovery_test.go
```

### Classification

The `sleep 240` / `tail -25 .run/sop-run.log` fixed-wait sequence exists
**only in this plan document's own narrative** (`docs/PLAN-Hardening.md:12-13`,
and its description quoted again at `:164`) — it describes an external,
manual operator shell workflow. It is **not present** in:

- production code — no `.go` file in the repository contains it, `240`, or a
  `sop-run.log` reference;
- scripts — `scripts/sop-agent.sh` has none of these terms;
- tests — the only `time.Sleep` calls in the repo are 5-10ms sleeps in
  `internal/web/{dogfood,reconcile,server}_test.go`, used for goroutine/command
  synchronization in tests, unrelated to the 240s pattern;
- any other documentation file.

The production observation surface already uses two distinct, unrelated
mechanisms instead of a fixed sleep:

- `internal/web/activity_stream.go` runs a `time.NewTicker` driven by
  `PollInterval` (`internal/config/config.go`, default 3s) for its SSE/poll
  fallback — a short, configurable cadence, not a hard-coded 240s wait.
- `CommandTimeout` (`internal/config/config.go`, default 15m, wired in
  `internal/web/server.go`) bounds one command's execution; it is a distinct,
  already-configurable value, not the UI observation interval.

**Conclusion: only an operator/debugging workflow** — not production code,
not tests, not scripts, and not any doc other than this plan's own example of
that workflow. No code change is invented to "remove" it, per the acceptance
criteria.

### Observability gap

The gap the manual `sleep 240; tail -25 .run/sop-run.log; sop status` sequence
implies is the absence of a documented, deterministic "observe without
waiting" workflow: an operator watching a long run had no guidance pointing
them at the controller's own SSE stream (`activity_stream.go`) or the short
poll fallback, so they reached for an arbitrary sleep instead. The mechanisms
to close that gap (`activity_stream.go`'s ticker and bounded poll window)
already exist; whether they are fully verified and documented as the
recommended workflow is the subject of HARD002-HARD007, out of scope here.

No production code, configuration value, or default timeout was changed by
this task.


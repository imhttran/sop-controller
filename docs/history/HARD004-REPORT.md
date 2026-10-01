# HARD004 Report --- Inactivity and Timeout Are Not Lifecycle States

Point-in-time, non-normative record produced by the SOP plan for HARD004,
following the same audit format as [HARD001-REPORT.md](HARD001-REPORT.md).

## HARD004-REPORT

### Search transcript

Terms searched: `elapsed >` / `elapsed>`, `no output`, `STUCK`, `"FAILED"`,
`"BLOCKED"`, `time.Since`, across `*.go`, `*.js`, and `*.html`.

```
$ grep -rn "elapsed >\|elapsed>" --include="*.go" --include="*.js" --include="*.html" .
internal/web/server_test.go:254:	if elapsed > 3*time.Second {

$ grep -rni "no output" --include="*.go" --include="*.js" --include="*.html" .
(no matches)

$ grep -rn "STUCK\|\"FAILED\"\|\"BLOCKED\"" --include="*.go" --include="*.js" --include="*.html" . | grep -v _test.go
internal/sopclient/run.go:30:	StageFailed          = "FAILED"
internal/sopclient/types.go:28:	StatusBlocked        = "BLOCKED"
internal/sopclient/types.go:58:		return "BLOCKED"
internal/sopclient/approval.go:39:	ApprovalKindBlocked = "BLOCKED"
internal/web/render.go:60:	case "BLOCKED":

$ grep -rn "time.Since" --include="*.go" .
internal/web/render.go:251:	d := time.Since(start)
internal/web/render.go:286:	d := time.Since(t)
internal/web/middleware.go:99:	log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
internal/web/server_test.go:237:	elapsed := time.Since(start)
```

### Classification

- The only `elapsed >` occurrence is a test assertion
  (`internal/web/server_test.go:254`) that a POST returns within 3s — a
  responsiveness check on an HTTP handler, not a lifecycle-state inference.
- `"no output"` has zero occurrences anywhere in the tree: there is no
  `no output for N => blocked` construct, or any variant of it, in Go, JS, or
  template sources.
- `STUCK` has zero occurrences anywhere in the tree.
- Every `"FAILED"` / `"BLOCKED"` occurrence is a literal SOP status constant
  (`internal/sopclient/run.go:30`, `internal/sopclient/types.go:28`,
  `internal/sopclient/approval.go:39`) or a `switch` arm that styles a status
  string SOP already returned (`internal/sopclient/types.go:58`'s
  `TaskState`, consumed by `internal/web/render.go:60`'s `statusClass`). Both
  `TaskState` and `statusClass` take `status string` as their only input —
  neither reads a timestamp, a duration, or an elapsed value. Comments at
  `internal/sopclient/types.go:50-51` ("display-only grouping... It never
  changes SOP state") and `internal/web/render.go:54-55` ("only styles the
  status SOP already chose; it never classifies the workflow itself")
  document this as deliberate.
- `time.Since` has four call sites total: two in `render.go` (`elapsed()` at
  :251, `since()` at :286), formatting a duration string or an explicit
  absence literal (`notRunLabel` / `"—"`) for display — see "Elapsed/
  since are display-only" below; one in `middleware.go:99`, a request-latency
  log line unrelated to task lifecycle; one in `server_test.go:237`, the same
  responsiveness-check helper as above.

### Elapsed/since are display-only

`internal/web/render.go`'s `elapsed(t sopclient.TaskDetail)` (:246-262) and
`since(t time.Time)` (:282-299) are the two functions in the repository that
convert a timestamp into a duration a human reads. Both are exhaustively
guarded:

- `elapsed()` returns `notRunLabel` when `startedAt(t)` is zero (SOP recorded
  no attempt timestamp), otherwise formats `time.Since(start)` into one of
  `< 1m` / `Nm` / `Nh` / `Nd`. There is no branch that returns `"FAILED"`,
  `"BLOCKED"`, `"STUCK"`, or any lifecycle word.
- `since()` returns `"—"` when the timestamp is zero, otherwise formats
  `time.Since(t)` into `just now` / `Nm ago` / `Nh ago` / `Nd ago`. Same
  guarantee: no lifecycle-word return path.

Both functions' only inputs are timestamps SOP itself persisted
(`TaskDetail.Attempts[].Timestamp`, or a caller-supplied `time.Time` sourced
from SOP's run/activity records) — neither reads a threshold constant, neither
compares the duration against a cutoff to select a different return branch.
A caller cannot get `FAILED`/`BLOCKED`/`STUCK` out of either function; that
comes only from SOP's own `status` string passed into `TaskState`/
`statusClass`, as shown above.

### Activity delivery and polling

`internal/web/activity_stream.go`'s SSE/poll loop (`pollInterval()`,
`time.NewTicker` at :157) only re-reads SOP's already-persisted activity
stream (`activity.jsonl`) on a fixed cadence (`SOP_CONTROLLER_POLL`, default
3s); it never compares elapsed time against a threshold to synthesize a
lifecycle state, and `static/activity-live.js` only renders whatever
activity/status payload the server already read from SOP — it contains no
inactivity-based inference either (confirmed by the same grep set above
covering `*.js`).

### Conclusion

No `elapsed > N => finished` or `no output for N => blocked` style construct
exists anywhere in the observation path (`internal/web/render.go`,
`internal/web/activity_stream.go`, `static/activity-live.js`,
`templates/*.html`). Elapsed/since rendering is factual-only: it can produce
a duration string or an explicit absence literal, never a lifecycle state.
Lifecycle words (`FAILED`, `BLOCKED`, `DONE`, etc.) only ever flow from SOP's
own persisted `status` field through pure string-to-string mappings
(`TaskState`, `statusClass`) that take no time-based input.

No production code, configuration value, or default was changed by this
task.

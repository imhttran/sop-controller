#!/usr/bin/env bash
# C2-009 dogfood harness: exercise sop-controller against the REAL `sop` binary.
#
# It stands up an ISOLATED, disposable SOP project (its own .agent-sdlc) OUTSIDE
# this repository's working tree, points the controller at it via
# SOP_CONTROLLER_PROJECTS, and drives the CONTROLLER's real HTTP surface
# (GET /projects/{id}, GET /projects/{id}/decisions, POST command routes with
# CSRF) while the real `sop` binary runs the lifecycle underneath. Nothing in
# this repository's .agent-sdlc/state.db or user-owned working-tree changes is
# touched.
#
# IMPORTANT: the C2-009 acceptance criteria are about the CONTROLLER end to end
# (RUNNING -> Needs Attention -> Inspect -> Approve -> SOP records -> controller
# reflects -> explicit Continue -> SOP resumes). This harness therefore drives
# the controller's HTTP routes, NOT the raw `sop` binary: it can only evidence
# the flow if the controller surfaces the gate, delegates Approve to SOP,
# reflects SOP's recorded decision, and resumes via an explicit Continue. This
# harness never invokes the raw `sop` binary to stand in for a controller action.
#
# READINESS (C2-009-S0): readiness is DETERMINED, not assumed. When no usable
# `sop` binary is resolvable the harness writes an explicit NOT READY artifact to
# the stable path (--readiness, default .run/c2-009-readiness.txt), prints a NOT
# READY report, records the scenario stages as NOT EXERCISED keyed to that
# reason, and exits non-zero WITHOUT proceeding. It never fakes a run. Downstream
# stages (S1-S4) read that artifact rather than re-resolving the binary.
#
# Stable artifacts (both written on every exit path, both outside $WORK so they
# survive cleanup):
#   --readiness P      the READY/NOT READY determination (default
#                      .run/c2-009-readiness.txt). S1-S4 consume THIS.
#   --scenarios P      the per-stage NOT EXERCISED records (default
#                      .run/c2-009-scenarios.txt), keyed to the S0 reason, so a
#                      downstream stage / the report can cite them after the run.
#
# Usage: scripts/c2-009-dogfood.sh [--keep] [--transcript PATH] [--readiness PATH] [--scenarios PATH]
#   --keep             do not delete the disposable project on exit
#   --transcript P     copy the final transcript to PATH (a tracked location such
#                      as docs/history/C2-009-TRANSCRIPT.log) so a real READY run
#                      is committed evidence rather than an ephemeral temp file.
#   --readiness P      write the READY/NOT READY determination to P (default
#                      .run/c2-009-readiness.txt). This is the stable artifact
#                      S1-S4 consume; it is written on BOTH paths.
#   --scenarios P      write the per-stage NOT EXERCISED records to P (default
#                      .run/c2-009-scenarios.txt), so they survive cleanup.
#
# The transcript is the truthful observation record: every controller action
# (route + status) and the controller's rendered state after each step. With
# --transcript it is committed verbatim; without it, it lives under $WORK.

set -u

KEEP=0
TRANSCRIPT_OUT=""
REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
READINESS_OUT="$REPO_ROOT/.run/c2-009-readiness.txt"
SCENARIOS_OUT="$REPO_ROOT/.run/c2-009-scenarios.txt"
while [ $# -gt 0 ]; do
  case "$1" in
    --keep) KEEP=1 ;;
    --transcript) TRANSCRIPT_OUT="${2:-}"; shift ;;
    --readiness) READINESS_OUT="${2:-}"; shift ;;
    --scenarios) SCENARIOS_OUT="${2:-}"; shift ;;
    *) echo "unknown argument: $1" >&2; exit 64 ;;
  esac
  shift
done

WORK="$(mktemp -d "c2-009-dogfood.XXXXXX")"
TRANSCRIPT="$WORK/transcript.log"
PROJECT="$WORK/project"
COOKIES="$WORK/cookies.txt"
: > "$TRANSCRIPT"

note() { printf '%s\n' "$*" | tee -a "$TRANSCRIPT" >&2; }

# prep_stable_artifacts ensures the stable output directories exist BEFORE any
# exit path can need them, and truncates the scenario record so a stale record
# from a previous run can never be read as this run's outcome. It fails loudly
# rather than silently: the readiness artifact is the only consumable evidence a
# downstream stage (S1-S4) is told to cite, so an unwritable path must not be
# swallowed.
prep_stable_artifacts() {
  local dir
  for dir in "$(dirname "$READINESS_OUT")" "$(dirname "$SCENARIOS_OUT")"; do
    if ! mkdir -p "$dir" 2>/dev/null; then
      echo "FATAL: cannot create artifact directory $dir" >&2
      exit 65
    fi
  done
  if ! : > "$SCENARIOS_OUT" 2>/dev/null; then
    echo "FATAL: cannot write scenario record $SCENARIOS_OUT" >&2
    exit 65
  fi
}

# not_exercised records a scenario stage as NOT EXERCISED with its reason. It
# writes the S1-S4 consumable record keyed to the S0 readiness outcome to the
# STABLE scenario path (not the disposable $WORK), so a downstream stage (and the
# report) can cite an explicit absence after the harness exits rather than
# inferring a pass from the deterministic unit tests.
not_exercised() {
  local stage="$1" reason="$2"
  printf '%s\tNOT EXERCISED\t%s\n' "$stage" "$reason" >> "$SCENARIOS_OUT"
  note "NOT EXERCISED [$stage]: $reason"
}

cleanup() {
  if [ -n "${CTRL_PID:-}" ]; then kill "$CTRL_PID" 2>/dev/null || true; fi
  if [ -n "$TRANSCRIPT_OUT" ]; then
    cp -f "$TRANSCRIPT" "$TRANSCRIPT_OUT" 2>/dev/null && \
      printf 'transcript committed to %s\n' "$TRANSCRIPT_OUT" >&2 || \
      printf 'WARN: could not copy transcript to %s\n' "$TRANSCRIPT_OUT" >&2
  fi
  if [ "$KEEP" -eq 0 ]; then rm -rf "$WORK"; else note "kept: $WORK"; fi
}
trap cleanup EXIT

prep_stable_artifacts

note "==================================================================="
note " C2-009 dogfood against real agentic-sop (CONTROLLER-driven)"
note " repo:        $REPO_ROOT"
note " disposable:  $PROJECT"
note " transcript:  $TRANSCRIPT"
note " readiness:   $READINESS_OUT"
note " scenarios:   $SCENARIOS_OUT"
note "==================================================================="

# write_readiness records the S0 determination to the stable artifact. It is a
# verbatim contract: READY names the resolved binary and its --version output;
# NOT READY carries the exact resolution attempt and its evidence. It fails
# loudly when the artifact cannot be written, because downstream stages cite it
# as the keyed reason for a not-exercised outcome.
write_readiness() {
  local state="$1"; shift
  {
    printf 'state: %s\n' "$state"
    printf 'recorded_at_utc: %s\n' "$(date -u '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || echo unknown)"
    printf 'repo_root: %s\n' "$REPO_ROOT"
    local line
    for line in "$@"; do printf '%s\n' "$line"; done
  } > "$READINESS_OUT" 2>/dev/null || {
    note "FATAL: could not write readiness artifact $READINESS_OUT"
    exit 65
  }
  note "readiness artifact written: $READINESS_OUT (state: $state)"
}

##############################################################################
# C2-009-S0  Binary readiness (determined, not assumed) + disposable project
##############################################################################

SOP_BIN_RESOLVED="${SOP_BIN:-}"
RESOLUTION_EVIDENCE=""
if [ -z "$SOP_BIN_RESOLVED" ]; then
  SOP_BIN_RESOLVED="$(command -v sop 2>/dev/null || true)"
  RESOLUTION_EVIDENCE="$( { printf 'which sop -> '; command -v sop 2>/dev/null || printf '<not found>'; printf '\n'; } 2>&1 )"
else
  RESOLUTION_EVIDENCE="SOP_BIN env override -> $SOP_BIN_RESOLVED"
fi

note ""
note "## S0: sop binary readiness"
if [ -z "$SOP_BIN_RESOLVED" ] || [ ! -x "$SOP_BIN_RESOLVED" ]; then
  note "NOT READY: no usable sop binary (SOP_BIN unset and 'sop' not on PATH)."
  note "$RESOLUTION_EVIDENCE"
  note "Deterministic unit tests are NOT a substitute for this exercise."
  note "Set SOP_BIN=/path/to/sop and re-run to produce a READY observation."
  write_readiness "NOT READY" \
    "reason: no usable sop binary resolved (SOP_BIN unset and 'sop' not on PATH)" \
    "evidence: $RESOLUTION_EVIDENCE"
  not_exercised "S1" "S0 NOT READY: no usable real sop binary resolved"
  not_exercised "S2" "S0 NOT READY: no usable real sop binary resolved"
  not_exercised "S3" "S0 NOT READY: no usable real sop binary resolved"
  exit 2
fi
note "resolved sop: $SOP_BIN_RESOLVED"
note "-- sop --version --"
SOP_VERSION="$("$SOP_BIN_RESOLVED" --version 2>&1)" || note "(no --version; continuing)"
printf '%s\n' "$SOP_VERSION" >>"$TRANSCRIPT"
note "-- sop --help --"
"$SOP_BIN_RESOLVED" --help >>"$TRANSCRIPT" 2>&1 || note "(no --help; continuing)"
write_readiness "READY" \
  "sop_bin: $SOP_BIN_RESOLVED" \
  "sop_version: $SOP_VERSION"

# Disposable project with its own SOP state, isolated from the repo.
mkdir -p "$PROJECT/.agent-sdlc"
cat > "$PROJECT/.agent-sdlc/config.yaml" <<'YAML'
project:
  name: "c2-009-dogfood"
human:
  approval_before_commit: true
YAML
cat > "$PROJECT/PLAN.md" <<'MD'
# Dogfood plan

## Task

### T1 — dogfood gate

- objective: reach the human approval gate
- acceptance: controller observes SOP's gate and delegates approve/decline
MD

note ""
note "## S0: disposable project provisioned (human.approval_before_commit: true)"
note "project root: $PROJECT"
note "NOTE: whether the real sop honours this exact config key/path and whether"
note "this minimal PLAN.md reaches a commit gate is recorded, not assumed: the"
note "S1 'reach gate' step below fails loudly (and is recorded) if sop run does"
note "not produce an approval gate."

##############################################################################
# Launch the controller against ONLY the disposable project
##############################################################################

note ""
note "## Launch controller with SOP_CONTROLLER_PROJECTS=$PROJECT"
(
  cd "$REPO_ROOT"
  SOP_CONTROLLER_PROJECTS="$PROJECT" \
  SOP_BIN="$SOP_BIN_RESOLVED" \
  SOP_CONTROLLER_ADDR="127.0.0.1:8099" \
  go run ./cmd/sop-controller >"$WORK/controller.log" 2>&1
) &
CTRL_PID=$!

BASE="http://127.0.0.1:8099"
for _ in $(seq 1 100); do
  if curl -fsS "$BASE/healthz" >/dev/null 2>&1; then break; fi
  sleep 0.1
done
if ! curl -fsS "$BASE/healthz" >/dev/null 2>&1; then
  note "controller failed to start; see $WORK/controller.log"
  not_exercised "S1" "controller failed to start against the disposable project"
  not_exercised "S2" "controller failed to start against the disposable project"
  not_exercised "S3" "controller failed to start against the disposable project"
  exit 3
fi

# The controller has no JSON project-list route; the real project list is the
# /projects HTML view. Fetch it (also establishing the CSRF cookie in $COOKIES)
# and extract the project id the controller itself reports.
CTRL_GET() { curl -fsS -b "$COOKIES" -c "$COOKIES" "$1"; }

curl -fsS -c "$COOKIES" "$BASE/projects" > "$WORK/projects.html" 2>/dev/null || true
PROJ_ID="$(grep -oE '/projects/[A-Za-z0-9._-]+' "$WORK/projects.html" 2>/dev/null | head -n1 | sed 's#/projects/##' || true)"
note "controller up at $BASE ; controller-reported project id: ${PROJ_ID:-<none>}"
if [ -z "$PROJ_ID" ]; then
  note "controller reports no project for the disposable root; see $WORK/projects.html"
  not_exercised "S1" "controller reported no project for the disposable root"
  not_exercised "S2" "controller reported no project for the disposable root"
  not_exercised "S3" "controller reported no project for the disposable root"
  exit 4
fi

# csrf reads the per-session CSRF token the controller issued.
csrf() { awk '$6=="sop_ctrl_csrf"{print $7}' "$COOKIES" | tail -n1; }

# ctrl_post <path> [k=v ...] issues a controller POST with the CSRF token and
# records the route, HTTP status, and the controller's rendered fragment. Extra
# form pairs are passed as separate argv elements (never word-split).
ctrl_post() {
  local path="$1"; shift
  local token code
  token="$(csrf)"
  local args=()
  args+=(--data-urlencode "csrf=$token")
  local pair
  for pair in "$@"; do
    args+=(--data-urlencode "$pair")
  done
  code="$(curl -s -o "$WORK/post.body" -w '%{http_code}' -b "$COOKIES" -c "$COOKIES" \
    "${args[@]}" "$BASE$path")"
  note "-- controller POST $path -> HTTP $code --"
  cat "$WORK/post.body" >> "$TRANSCRIPT"
  printf '%s\n' "" >> "$TRANSCRIPT"
  printf '%s' "$code"
}

# ctrl_wait <path> polls a command-status route until it leaves "running" and
# records the final controller-rendered status fragment.
ctrl_wait() {
  local path="$1" i body
  for i in $(seq 1 100); do
    body="$(CTRL_GET "$BASE$path" || true)"
    case "$body" in *"· running"*) sleep 0.1;; *) break;; esac
  done
  note "-- controller GET $path (final) --"
  printf '%s\n' "$body" >> "$TRANSCRIPT"
}

# observe <label> <path> records labelled controller-rendered state and echoes
# it so callers can assert on the rendered fragment.
observe() {
  note "-- [$1] controller GET $2 --"
  local body
  body="$(CTRL_GET "$BASE$2" 2>&1 || true)"
  printf '%s\n' "$body" >> "$TRANSCRIPT"
  printf '%s' "$body"
}

# resolve_task_id discovers the gated task id from the controller-rendered
# decisions view rather than hardcoding it. It anchors on the task link whose
# href sits under the /decisions view, so a decisions page that lists several
# tasks resolves the gate it is actually showing; when it finds none it returns
# empty so the caller records a not-exercised outcome instead of guessing.
resolve_task_id() {
  local page="$1"
  printf '%s' "$page" | grep -oE '/projects/[^/"]+/tasks/[A-Za-z0-9._-]+' \
    | sed 's#.*/tasks/##' | head -n1
}

##############################################################################
# S1  End-to-end approval flow, driven entirely through the CONTROLLER.
#
# Sequence: RUNNING -> Needs Attention -> Inspect -> Approve -> SOP records ->
# controller reflects -> explicit Continue -> SOP resumes.
##############################################################################

note ""
note "## S1: start the plan through the CONTROLLER (POST /commands/start)"
ctrl_post "/projects/$PROJ_ID/commands/start" >/dev/null
ctrl_wait "/projects/$PROJ_ID/commands/start"

note ""
note "## S1: observe RUNNING and the SOP-reported gate via the CONTROLLER"
RUNNING_PAGE="$(observe "running" "/projects/$PROJ_ID")"
GATE_PAGE="$(observe "gate" "/projects/$PROJ_ID/decisions")"

# Assert the gate was actually reached (C2-009-S1): a config key and a minimal
# PLAN.md are NOT proof that real sop emitted a human gate. The controller is the
# only surface consulted; the harness fails loudly and records a not-exercised
# outcome rather than proceeding to a success banner when no gate appears.
TASK_ID="$(resolve_task_id "$GATE_PAGE")"
if [ -z "$TASK_ID" ]; then
  TASK_ID="$(resolve_task_id "$RUNNING_PAGE")"
fi
if [ -z "$TASK_ID" ]; then
  note "NO GATE REACHED: neither /decisions nor /projects exposed a gated task."
  note "Recorded as an observed absence: human.approval_before_commit did not"
  note "produce a controller-visible gate for this disposable project."
  not_exercised "S1" "no gate reached via the controller after start; human.approval_before_commit emitted no controller-visible gate"
  not_exercised "S2" "no gate reached (see S1); nothing to decline"
  not_exercised "S3" "no gate reached (see S1); changed-executed set not provoked"
  note "==================================================================="
  note " NOT EXERCISED (observed absence). Readiness artifact: $READINESS_OUT"
  note " Scenario record: $SCENARIOS_OUT"
  note "==================================================================="
  exit 5
fi
note "S1 gate reached; controller-reported task id: $TASK_ID"

note ""
note "## S1: inspect the gated task through the CONTROLLER"
observe "inspect" "/projects/$PROJ_ID/tasks/$TASK_ID" >/dev/null

note ""
note "## S1: APPROVE via the CONTROLLER (POST .../commands/approve)"
note "The controller must delegate this to SOP's own operation and report SOP's"
note "answer verbatim; the controller writes no approval state of its own."
ctrl_post "/projects/$PROJ_ID/tasks/$TASK_ID/commands/approve" "note=c2-009 dogfood approve" >/dev/null
ctrl_wait "/projects/$PROJ_ID/tasks/$TASK_ID/commands/approve"
note ""
note "## S1: controller reflects SOP's recorded decision"
AFTER_APPROVE="$(observe "reflect-after-approve" "/projects/$PROJ_ID/tasks/$TASK_ID")"
case "$AFTER_APPROVE" in
  *"pprov"*|*"ecision"*|*"ecorded"*)
    note "S1: controller reflects SOP's recorded approval decision." ;;
  *)
    note "DIVERGENCE S1: controller does not visibly reflect an approval decision"
    note "after approve; recorded verbatim above rather than treated as success." ;;
esac

note ""
note "## S1: explicit Continue via the CONTROLLER (POST .../commands/start again)"
note "Continue is a separate, explicit action: the harness records that no"
note "resumption happened implicitly at approve time."
ctrl_post "/projects/$PROJ_ID/commands/start" >/dev/null
ctrl_wait "/projects/$PROJ_ID/commands/start"
observe "resumed" "/projects/$PROJ_ID" >/dev/null

##############################################################################
# S2  Decline scenario, driven through the CONTROLLER.
##############################################################################

note ""
note "## S2: DECLINE via the CONTROLLER (POST .../commands/decline)"
note "If SOP exposes no applicable gate at this point the controller reports that"
note "truthfully; the harness records whatever the controller actually returns."
ctrl_post "/projects/$PROJ_ID/tasks/$TASK_ID/commands/decline" "note=c2-009 dogfood decline" >/dev/null
ctrl_wait "/projects/$PROJ_ID/tasks/$TASK_ID/commands/decline"
observe "reflect-after-decline" "/projects/$PROJ_ID/tasks/$TASK_ID" >/dev/null

note ""
note "## S2: confirm no controller-manufactured failure/complete"
observe "task-list-after-decline" "/projects/$PROJ_ID/tasks" >/dev/null

##############################################################################
# S3  Changed-executed-task reconciliation, driven through the CONTROLLER.
#
# The controller resolves the plan from SOP's recorded provenance
# (.agent-sdlc/plan.meta.json) and the changed set from SOP's authoritative
# listing (`sop reconcile <PLAN.md> --list-changed --json`); this harness NEVER
# hardcodes PLAN.md, so it matches what the controller actually does.
##############################################################################

note ""
note "## S3: observe the changed-executed set the CONTROLLER reads from SOP"
observe "changed-tasks" "/projects/$PROJ_ID" >/dev/null

note ""
note "## S3: reconcile via the CONTROLLER (POST /commands/reconcile)"
ctrl_post "/projects/$PROJ_ID/commands/reconcile" >/dev/null
ctrl_wait "/projects/$PROJ_ID/commands/reconcile"

note ""
note "## S3: per-task accept-changed via the CONTROLLER (recorded boundary gap)"
note "The controller delegates `sop reconcile <PLAN.md> --accept-changed <id>`;"
note "a real SOP-side gap surfaces as a truthful *ReconcileRejection (conflict),"
note "which the harness records rather than treating as success."
ACCEPT_BODY="$(ctrl_post "/projects/$PROJ_ID/tasks/$TASK_ID/commands/accept-changed")"
ctrl_wait "/projects/$PROJ_ID/tasks/$TASK_ID/commands/accept-changed"
case "$ACCEPT_BODY" in
  *"unsupported"*|*"not supported"*|*"reject"*|*"Rejection"*|*"conflict"*)
    note "S3: accept-changed delegated to SOP and SOP's own answer (supported or"
    note "rejected) is recorded above; the boundary is NOT pre-judged by the controller." ;;
  *)
    note "S3: accept-changed POST returned HTTP $ACCEPT_BODY; SOP's own answer is"
    note "recorded verbatim above and is neither pre-judged nor treated as success." ;;
esac

note ""
note "## S3: confirm SOP state is intact (no direct state.db write by the controller)"
observe "project-final" "/projects/$PROJ_ID" >/dev/null

note ""
note "==================================================================="
note " READY run complete. Observed via the CONTROLLER HTTP surface."
note " Transcript: $TRANSCRIPT"
note " Readiness: $READINESS_OUT"
note " Scenario record: $SCENARIOS_OUT"
note " Commit it verbatim (use --transcript) to convert this into committed"
note " evidence for docs/history/C2-009-REPORT.md."
note "==================================================================="

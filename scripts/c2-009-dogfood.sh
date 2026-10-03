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
# stages use the resolved binary in this process; the artifact records S0.
#
# Stable artifacts (both written on every exit path, both outside $WORK so they
# survive cleanup):
#   --readiness P      the READY/NOT READY determination (default
#                      .run/c2-009-readiness.txt).
#   --scenarios P      the per-stage PASS / FAIL / NOT EXERCISED records (default
#                      .run/c2-009-scenarios.txt), keyed to the S0 reason, so a
#                      downstream stage / the report can cite them after the run.
#
# Usage: scripts/c2-009-dogfood.sh [--keep] [--transcript PATH] [--readiness PATH] [--scenarios PATH]
#   --keep             do not delete the disposable project on exit
#   --transcript P     append the transcript to PATH, preserving previous runs.
#                      Use docs/history/ for evidence intended for review/commit.
#   --readiness P      write the READY/NOT READY determination to P (default
#                      .run/c2-009-readiness.txt), written on BOTH paths.
#   --scenarios P      write the per-stage results to P (default
#                      .run/c2-009-scenarios.txt), so they survive cleanup.
#
# The transcript is the truthful observation record: every controller action
# (route + status) and the controller's rendered state after each step. With
# --transcript it is saved verbatim, without committing; otherwise it is scratch.

set -uo pipefail

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

WORK="$(mktemp -d "${TMPDIR:-/tmp}/c2-009-dogfood.XXXXXX")" || exit 65
TRANSCRIPT="$WORK/transcript.log"
PROJECT="$WORK/project"
COOKIES="$WORK/cookies.txt"
: > "$TRANSCRIPT" || exit 65
RESULT=0

note() { printf '%s\n' "$*" | tee -a "$TRANSCRIPT" >&2; }

# prep_stable_artifacts ensures the stable output directories exist BEFORE any
# exit path can need them, and truncates the scenario record so a stale record
# from a previous run can never be read as this run's outcome. It fails loudly
# rather than silently: the readiness artifact is the only consumable evidence a
# downstream report is told to cite, so an unwritable path must not be
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
# writes the result keyed to the S0 readiness outcome to the
# STABLE scenario path (not the disposable $WORK), so a downstream stage (and the
# report) can cite an explicit absence after the harness exits rather than
# inferring a pass from the deterministic unit tests.
not_exercised() {
  record_result "$1" "NOT EXERCISED" "$2"
}

record_result() {
  local stage="$1" result="$2" reason="$3"
  if ! printf '%s\t%s\t%s\n' "$stage" "$result" "$reason" >> "$SCENARIOS_OUT"; then
    note "FATAL: cannot write scenario record $SCENARIOS_OUT"
    exit 65
  fi
  note "$result [$stage]: $reason"
  [ "$result" = PASS ] || RESULT=1
  return 0
}

setup_failed() {
  note "SETUP FAILED: $1"
  not_exercised S1 "$1"
  not_exercised S2 "$1"
  not_exercised S3 "$1"
  exit 3
}

cleanup() {
  local code=$?
  trap - EXIT
  if [ -n "${CTRL_PID:-}" ]; then
    kill "$CTRL_PID" 2>/dev/null || true
    wait "$CTRL_PID" 2>/dev/null || true
  fi
  if [ -f "$WORK/controller.log" ]; then
    note "-- controller process log --"
    cat "$WORK/controller.log" >> "$TRANSCRIPT" || code=65
  fi
  if [ -f "$SCENARIOS_OUT" ]; then
    note "-- scenario results --"
    cat "$SCENARIOS_OUT" >> "$TRANSCRIPT" || code=65
  fi
  if [ "$KEEP" -eq 1 ]; then note "kept: $WORK"; fi
  if [ -n "$TRANSCRIPT_OUT" ]; then
    if mkdir -p "$(dirname "$TRANSCRIPT_OUT")" && cat "$TRANSCRIPT" >> "$TRANSCRIPT_OUT"; then
      printf 'transcript saved to %s\n' "$TRANSCRIPT_OUT" >&2
    else
      printf 'FATAL: could not save transcript to %s; retained %s\n' "$TRANSCRIPT_OUT" "$WORK" >&2
      KEEP=1
      code=65
    fi
  fi
  if [ "$KEEP" -eq 0 ]; then rm -rf "$WORK"; fi
  exit "$code"
}
trap cleanup EXIT

prep_stable_artifacts

note "==================================================================="
note " C2-009 dogfood against real agentic-sop (CONTROLLER-driven)"
note " repo:        $REPO_ROOT"
note " recorded UTC: $(date -u '+%Y-%m-%dT%H:%M:%SZ')"
note " controller revision: $(git -C "$REPO_ROOT" rev-parse HEAD) (working-tree harness)"
note " disposable:  $PROJECT"
note " transcript:  $TRANSCRIPT"
note " readiness:   $READINESS_OUT"
note " scenarios:   $SCENARIOS_OUT"
note "==================================================================="

# write_readiness records the S0 determination to the stable artifact. It is a
# verbatim contract: READY names the resolved binary and its version output;
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
note "-- $SOP_BIN_RESOLVED version --"
SOP_VERSION="$("$SOP_BIN_RESOLVED" version 2>&1)" || setup_failed "sop version failed: $SOP_VERSION"
printf '%s\n' "$SOP_VERSION" >>"$TRANSCRIPT"
note "-- binary build metadata --"
go version -m "$SOP_BIN_RESOLVED" >> "$TRANSCRIPT" 2>&1 || note "build metadata unavailable"
note "-- sop --help --"
"$SOP_BIN_RESOLVED" --help >>"$TRANSCRIPT" 2>&1 || setup_failed "sop --help failed"
write_readiness "READY" \
  "sop_bin: $SOP_BIN_RESOLVED" \
  "sop_version: $SOP_VERSION"

# Only setup invokes SOP directly. No database or run state is supplied by the
# harness: sop init creates the schema using the real project's state store.
for tool in git go curl jq; do
  command -v "$tool" >/dev/null || setup_failed "required tool unavailable: $tool"
done
mkdir -p "$PROJECT/.agent-sdlc" || setup_failed "cannot create project directory"
git -C "$PROJECT" init --quiet > "$WORK/git-init.log" 2>&1 || setup_failed "git init failed: $(cat "$WORK/git-init.log")"
printf 'Disposable C2-009 project. No controller-owned workflow state.\n' > "$PROJECT/README.md" || setup_failed "cannot write README.md"
cat > "$PROJECT/check.sh" <<'SH' || setup_failed "cannot write check.sh"
#!/bin/sh
test -f DOGFOOD.txt && test "$(cat DOGFOOD.txt)" = 'C2-009 real SOP dogfood'
SH
cat > "$PROJECT/.agent-sdlc/config.yaml" <<'YAML'
project:
  name: "c2-009-dogfood"
human:
  approval_before_commit: true
validation:
  test:
    - sh check.sh
YAML
if [ $? -ne 0 ]; then setup_failed "cannot write SOP configuration"; fi
cat > "$PROJECT/PLAN.md" <<'MD'
# Dogfood plan

## Project

c2-009-dogfood

## Summary

Exercise a minimal real SOP lifecycle in a disposable repository.

## T1 — Write the dogfood record

Create DOGFOOD.txt containing exactly one line: C2-009 real SOP dogfood.
Keep all existing project files and the validation command intact.

### Deliverables

- DOGFOOD.txt

### Acceptance Criteria

- sh check.sh passes.
MD
if [ $? -ne 0 ]; then setup_failed "cannot write PLAN.md"; fi

note ""
note "## S0: real SOP initialization"
[ ! -e "$PROJECT/.agent-sdlc/state.db" ] || setup_failed "state.db unexpectedly exists before sop init"
note "state.db before init: absent"
note "command: (cd \"$PROJECT\" && \"$SOP_BIN_RESOLVED\" init)"
(cd "$PROJECT" && "$SOP_BIN_RESOLVED" init) > "$WORK/sop-init.log" 2>&1
INIT_CODE=$?
note "initialization exit code: $INIT_CODE"
cat "$WORK/sop-init.log" >> "$TRANSCRIPT" || setup_failed "cannot record sop init output"
[ "$INIT_CODE" -eq 0 ] || setup_failed "real sop init exited $INIT_CODE; output recorded above"
[ -f "$PROJECT/.agent-sdlc/state.db" ] || setup_failed "sop init succeeded but state.db is absent"
note "state.db after init: exists ($(wc -c < "$PROJECT/.agent-sdlc/state.db" | tr -d ' ') bytes), created by real sop init"
note "Commit policy is enabled; only an applicable SOP-reported gate proves S1/S2 prerequisites."

##############################################################################
# Launch the controller against ONLY the disposable project
##############################################################################

note ""
note "## Launch controller with SOP_CONTROLLER_PROJECTS=$PROJECT"
# Run the built executable directly so cleanup owns the server PID, not a go-run
# wrapper. A pre-existing listener must not masquerade as this controller.
go -C "$REPO_ROOT" build -o "$WORK/sop-controller" ./cmd/sop-controller > "$WORK/controller-build.log" 2>&1 || setup_failed "controller build failed: $(cat "$WORK/controller-build.log")"
if curl -fsS "http://127.0.0.1:8099/healthz" >/dev/null 2>&1; then
  setup_failed "port 8099 already serves /healthz; refusing to exercise another controller"
fi
(
  cd "$PROJECT" || exit 3
  SOP_CONTROLLER_PROJECTS="$PROJECT" \
  SOP_CONTROLLER_WORKSPACES="" \
  SOP_CONTROLLER_ALLOW_NETWORK="false" \
  SOP_BIN="$SOP_BIN_RESOLVED" \
  SOP_CONTROLLER_ADDR="127.0.0.1:8099" \
  SOP_CONTROLLER_COMMAND_TIMEOUT="15m" \
  exec "$WORK/sop-controller" >"$WORK/controller.log" 2>&1
) &
CTRL_PID=$!

BASE="http://127.0.0.1:8099"
for _ in $(seq 1 100); do
  kill -0 "$CTRL_PID" 2>/dev/null || break
  if curl -fsS "$BASE/healthz" >/dev/null 2>&1; then break; fi
  sleep 0.1
done
if ! kill -0 "$CTRL_PID" 2>/dev/null || ! curl -fsS "$BASE/healthz" >/dev/null 2>&1; then
  note "controller failed to start; see $WORK/controller.log"
  not_exercised "S1" "controller failed to start against the disposable project"
  not_exercised "S2" "controller failed to start against the disposable project"
  not_exercised "S3" "controller failed to start against the disposable project"
  exit 3
fi

# The controller has no JSON project-list route; the real project list is the
# /projects HTML view. Fetch it (also establishing the CSRF cookie in $COOKIES)
# and extract the project id the controller itself reports.
CTRL_GET() { curl -fsS --max-time 30 -b "$COOKIES" -c "$COOKIES" "$1"; }

curl -fsS -c "$COOKIES" "$BASE/projects" > "$WORK/projects.html" || setup_failed "GET /projects failed"
cat "$WORK/projects.html" >> "$TRANSCRIPT" || setup_failed "cannot record project discovery"
PROJ_ID="$(sed -n 's/.*class="card" href="\/projects\/\([A-Za-z0-9._-]*\)".*/\1/p' "$WORK/projects.html")"
note "controller up at $BASE ; controller-reported project id: ${PROJ_ID:-<none>}"
if [ -z "$PROJ_ID" ] || [ "$(printf '%s\n' "$PROJ_ID" | wc -l | tr -d ' ')" -ne 1 ]; then
  note "controller reports no project for the disposable root; see $WORK/projects.html"
  not_exercised "S1" "controller reported no project for the disposable root"
  not_exercised "S2" "controller reported no project for the disposable root"
  not_exercised "S3" "controller reported no project for the disposable root"
  exit 4
fi

# csrf reads the per-session CSRF token the controller issued.
csrf() { awk '$6=="sop_ctrl_csrf"{print $7}' "$COOKIES" | tail -n1; }

# Check transport and HTTP acceptance before polling the asynchronous operation.
ctrl_post() {
  local path="$1"; shift
  local token code transport pair
  token="$(csrf)"
  [ -n "$token" ] || { note "POST refused: controller issued no CSRF token"; return 1; }
  local args=(--data-urlencode "csrf=$token")
  for pair in "$@"; do args+=(--data-urlencode "$pair"); done
  code="$(curl -sS --max-time 30 -o "$WORK/post.body" -w '%{http_code}' -b "$COOKIES" -c "$COOKIES" \
    "${args[@]}" "$BASE$path")"
  transport=$?
  note "-- controller POST $path -> HTTP $code --"
  if [ -f "$WORK/post.body" ]; then cat "$WORK/post.body" >> "$TRANSCRIPT" || return 1; fi
  printf '\n' >> "$TRANSCRIPT" || return 1
  [ "$transport" -eq 0 ] && [ "$code" = 200 ]
}

# cmd-done is published only after the synchronous SOP process returns
# successfully. Errors, idle, malformed responses and transport failure are not
# completion. The deadline matches the controller's 15-minute command limit.
# Polling observes completion; it is not a delay to hide initialization failure.
ctrl_wait() {
  local path="$1" body deadline=$((SECONDS + 900))
  while [ "$SECONDS" -lt "$deadline" ]; do
    body="$(CTRL_GET "$BASE$path")" || { note "GET command status failed: $path"; return 1; }
    if printf '%s\n' "$body" | grep -x '<div class="cmd cmd-running">' >/dev/null; then
      sleep 0.1
      continue
    fi
    note "-- controller GET $path (terminal) --"
    printf '%s\n' "$body" >> "$TRANSCRIPT" || return 1
    printf '%s\n' "$body" | grep -x '<div class="cmd cmd-done">' >/dev/null
    return $?
  done
  note "command did not complete within the controller's 15-minute limit: $path"
  return 1
}

observe() {
  note "-- [$1] controller GET $2 --"
  local body
  body="$(CTRL_GET "$BASE$2")" || return 1
  printf '%s\n' "$body" >> "$TRANSCRIPT" || return 1
  printf '%s' "$body"
}

# These forms exist only for actions SOP reports applicable. Navigation/task
# links do not establish a gate. Never choose arbitrarily among several gates.
action_tasks() {
  printf '%s\n' "$1" | grep -oE 'hx-post="/projects/[A-Za-z0-9._-]+/tasks/[A-Za-z0-9._-]+/commands/[a-z-]+"' \
    | awk -F/ -v project="$PROJ_ID" -v verb="$2" '$3 == project && $7 == verb "\"" {print $5}' | sort -u
}

resolve_task_id() {
  local tasks
  tasks="$(action_tasks "$1" "$2")"
  if [ -z "$tasks" ] || [ "$(printf '%s\n' "$tasks" | wc -l | tr -d ' ')" -ne 1 ]; then
    note "No unambiguous SOP-reported $2 gate (task ids: ${tasks:-none})."
    return 1
  fi
  printf '%s' "$tasks"
}

# Read SOP-produced artifacts only; never write them or invoke a raw SOP
# decision/lifecycle command once the controller is running.
decision_recorded() {
  local task="$1" status="$2" approved="$3" decision_note="$4"
  local file="$PROJECT/.agent-sdlc/runs/$task/approval.json"
  [ -f "$file" ] || return 1
  note "-- authoritative SOP decision: $file --"
  cat "$file" >> "$TRANSCRIPT" || return 1
  jq -e --arg task "$task" --arg status "$status" --arg note "$decision_note" --argjson approved "$approved" \
    '.task_id == $task and .status == $status and .decision.task_id == $task
     and .decision.request_id == .id and .decision.approved == $approved
     and .decision.decided_by == "c2-009-dogfood" and .decision.note == $note
     and (.decision.decided_at | length > 0)' "$file" >/dev/null
}

# A successful resume command can merely report the next legal action. S1 may
# pass only if SOP actually finished the gated run and the task hero reflects it.
continuation_reflected() {
  local task="$1" page
  page="$(observe after-explicit-continue "/projects/$PROJ_ID/tasks/$task")" || return 1
  jq -e --arg task "$task" '.id == $task and .stage == "PASSED"' \
    "$PROJECT/.agent-sdlc/runs/$task/state.json" >/dev/null || return 1
  printf '%s\n' "$page" | awk '/<section class="hero task-hero">/{hero=1} hero{print} /<\/section>/{hero=0}' \
    | grep -F '>stage PASSED</span' >/dev/null
}

##############################################################################
# S1: Run -> applicable human gate -> inspect -> approve -> explicit Continue.
##############################################################################
note ""
note "## S1: start the plan through the CONTROLLER (POST /commands/start)"
ctrl_post "/projects/$PROJ_ID/commands/start" || setup_failed "initial controller start POST failed"
# A genuine human boundary may make sop run exit nonzero. Record that outcome;
# only an applicable form plus SOP's PENDING request can prove a parked gate.
START_OK=1
ctrl_wait "/projects/$PROJ_ID/commands/start" || START_OK=0
observe after-start "/projects/$PROJ_ID" >/dev/null || setup_failed "project observation failed"
GATE_PAGE="$(observe gate "/projects/$PROJ_ID/decisions")" || setup_failed "gate observation failed"

if TASK_ID="$(resolve_task_id "$GATE_PAGE" approve)"; then
  if ! jq -e --arg task "$TASK_ID" '.task_id == $task and .status == "PENDING"' \
      "$PROJECT/.agent-sdlc/runs/$TASK_ID/approval.json" >/dev/null; then
    record_result S1 FAIL "controller gate has no matching authoritative PENDING SOP request"
  elif ! observe inspect "/projects/$PROJ_ID/tasks/$TASK_ID" >/dev/null; then
    record_result S1 FAIL "gated-task inspection failed"
  elif ! ctrl_post "/projects/$PROJ_ID/tasks/$TASK_ID/commands/approve" "by=c2-009-dogfood" "note=c2-009 dogfood approve" \
      || ! ctrl_wait "/projects/$PROJ_ID/tasks/$TASK_ID/commands/approve" \
      || ! decision_recorded "$TASK_ID" APPROVED true "c2-009 dogfood approve"; then
    record_result S1 FAIL "approval transport, completion or task-specific SOP decision verification failed"
  elif ! AFTER_APPROVE="$(observe reflect-after-approve "/projects/$PROJ_ID/decisions")" \
      || action_tasks "$AFTER_APPROVE" approve | grep -Fxq "$TASK_ID"; then
    record_result S1 FAIL "controller did not reflect removal of the approved gate"
  else
    note "S1: task-specific APPROVED decision recorded by SOP; explicit Continue follows."
    if ctrl_post "/projects/$PROJ_ID/commands/start" \
        && ctrl_wait "/projects/$PROJ_ID/commands/start" \
        && continuation_reflected "$TASK_ID"; then
      record_result S1 PASS "applicable gate inspected, SOP approval recorded/reflected, explicit Continue reached SOP PASSED and controller reflected it"
    else
      record_result S1 FAIL "explicit continuation failed or did not reach a SOP PASSED run reflected by the controller"
    fi
  fi
else
  RUN_STAGE="$(jq -r '.stage // "unreported"' "$PROJECT/.agent-sdlc/runs/T1/state.json" 2>/dev/null)" || RUN_STAGE=unreported
  not_exercised S1 "no unambiguous applicable SOP approval gate after start; SOP T1 stage: $RUN_STAGE, successful command: $START_OK; commit policy alone is not an approval request"
fi

##############################################################################
# S2: Decline an applicable gate, not the already-approved gate from S1.
##############################################################################
note ""
note "## S2: applicable decline through the CONTROLLER"
if ! DECLINE_PAGE="$(observe before-decline "/projects/$PROJ_ID/decisions")"; then
  record_result S2 FAIL "cannot observe applicable decline gates"
elif DECLINE_TASK="$(resolve_task_id "$DECLINE_PAGE" decline)"; then
  if ctrl_post "/projects/$PROJ_ID/tasks/$DECLINE_TASK/commands/decline" "by=c2-009-dogfood" "note=c2-009 dogfood decline" \
      && ctrl_wait "/projects/$PROJ_ID/tasks/$DECLINE_TASK/commands/decline" \
      && decision_recorded "$DECLINE_TASK" DECLINED false "c2-009 dogfood decline" \
      && AFTER_DECLINE="$(observe reflect-after-decline "/projects/$PROJ_ID/decisions")" \
      && ! action_tasks "$AFTER_DECLINE" decline | grep -Fxq "$DECLINE_TASK" \
      && observe task-after-decline "/projects/$PROJ_ID/tasks/$DECLINE_TASK" >/dev/null; then
    record_result S2 PASS "SOP recorded the task-specific decline; controller no longer offers that gate"
  else
    record_result S2 FAIL "decline transport, completion, SOP decision or controller reflection verification failed"
  fi
else
  not_exercised S2 "SOP exposes no unambiguous applicable decline gate; an already-approved gate is not a decline scenario"
fi

##############################################################################
# S3: Edit the disposable source plan, then let SOP determine the changed set.
##############################################################################
note ""
note "## S3: intentional disposable-plan edit, then SOP-authoritative changed listing"
if [ ! -f "$PROJECT/.agent-sdlc/plan.meta.json" ]; then
  not_exercised S3 "SOP did not record an active plan; changed-executed reconciliation prerequisite absent"
elif ! printf '\n- DOGFOOD.txt contains no additional lines.\n' >> "$PROJECT/PLAN.md"; then
  record_result S3 FAIL "could not edit the disposable source plan"
elif ! CHANGED_PAGE="$(observe changed-tasks "/projects/$PROJ_ID/decisions")"; then
  record_result S3 FAIL "changed-task observation failed"
elif action_tasks "$CHANGED_PAGE" accept-changed | grep -Fxq T1; then
  note "SOP lists T1 as changed-executed; controller per-task acceptance delegates to real SOP."
  if ctrl_post "/projects/$PROJ_ID/tasks/T1/commands/accept-changed" \
      && ctrl_wait "/projects/$PROJ_ID/tasks/T1/commands/accept-changed" \
      && AFTER_ACCEPT="$(observe after-accept "/projects/$PROJ_ID/decisions")" \
      && ! action_tasks "$AFTER_ACCEPT" accept-changed | grep -Fxq T1 \
      && printf '%s\n' "$AFTER_ACCEPT" | grep -F 'SOP reports no changed executed tasks awaiting accept.' >/dev/null \
      && jq -e '.stages[] | select(.id == "T1") | .acceptance_criteria | index("DOGFOOD.txt contains no additional lines.") != null' \
        "$PROJECT/.agent-sdlc/plan.json" >/dev/null \
      && ctrl_post "/projects/$PROJ_ID/commands/reconcile" \
      && ctrl_wait "/projects/$PROJ_ID/commands/reconcile" \
      && observe project-final "/projects/$PROJ_ID" >/dev/null; then
    record_result S3 PASS "SOP listed changed-executed T1, per-task acceptance updated SOP's plan, controller reflected removal and reconciliation completed"
  else
    record_result S3 FAIL "accept-changed transport, completion, SOP plan update, reflection or reconciliation failed"
  fi
else
  not_exercised S3 "SOP did not expose T1 as changed-executed after the source edit; authoritative listing/absence or error is recorded above"
fi

# Persist useful native observations in the transcript, not just ephemeral .run
# links. These are read-only artifacts produced by the real SOP lifecycle.
for file in "$PROJECT/.agent-sdlc/plan.meta.json" "$PROJECT/.agent-sdlc/plan.json" \
    "$PROJECT/.agent-sdlc/runs/"*/state.json "$PROJECT/.agent-sdlc/runs/"*/classification.json \
    "$PROJECT/.agent-sdlc/runs/"*/approval.json "$PROJECT/.agent-sdlc/runs/"*/report.json; do
  if [ -f "$file" ]; then
    note "-- SOP-produced evidence: $file --"
    cat "$file" >> "$TRANSCRIPT" || { note "FATAL: cannot record $file"; exit 65; }
    printf '\n' >> "$TRANSCRIPT" || exit 65
  fi
done

note ""
note "==================================================================="
note " Run finished with result exit code $RESULT (0 requires S1/S2/S3 PASS)."
note " Transcript: $TRANSCRIPT"
note " Readiness: $READINESS_OUT"
note " Scenario record: $SCENARIOS_OUT"
note "==================================================================="
exit "$RESULT"

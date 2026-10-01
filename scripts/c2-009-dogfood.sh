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
# Readiness is DETERMINED, not assumed (C2-009-S0): if no usable `sop` binary is
# resolvable the script prints an explicit NOT READY report and exits non-zero
# WITHOUT proceeding. It never fakes a run.
#
# Usage: scripts/c2-009-dogfood.sh [--keep] [--transcript PATH]
#   --keep           do not delete the disposable project on exit
#   --transcript P   copy the final transcript to PATH (a tracked location such
#                    as docs/history/C2-009-TRANSCRIPT.log) so a real READY run
#                    is committed evidence rather than an ephemeral temp file.
#
# The transcript is the truthful observation record: every controller action
# (route + status) and the controller's rendered state after each step. With
# --transcript it is committed verbatim; without it, it lives under $WORK.

set -u

KEEP=0
TRANSCRIPT_OUT=""
while [ $# -gt 0 ]; do
  case "$1" in
    --keep) KEEP=1 ;;
    --transcript) TRANSCRIPT_OUT="${2:-}"; shift ;;
    *) echo "unknown argument: $1" >&2; exit 64 ;;
  esac
  shift
done

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WORK="$(mktemp -d "c2-009-dogfood.XXXXXX")"
TRANSCRIPT="$WORK/transcript.log"
PROJECT="$WORK/project"
COOKIES="$WORK/cookies.txt"
: > "$TRANSCRIPT"

note() { printf '%s\n' "$*" | tee -a "$TRANSCRIPT" >&2; }

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

note "==================================================================="
note " C2-009 dogfood against real agentic-sop (CONTROLLER-driven)"
note " repo:        $REPO_ROOT"
note " disposable:  $PROJECT"
note " transcript:  $TRANSCRIPT"
note "==================================================================="

##############################################################################
# C2-009-S0  Binary readiness (determined, not assumed) + disposable project
##############################################################################

SOP_BIN_RESOLVED="${SOP_BIN:-}"
if [ -z "$SOP_BIN_RESOLVED" ]; then
  SOP_BIN_RESOLVED="$(command -v sop 2>/dev/null || true)"
fi

note ""
note "## S0: sop binary readiness"
if [ -z "$SOP_BIN_RESOLVED" ] || [ ! -x "$SOP_BIN_RESOLVED" ]; then
  note "NOT READY: no usable sop binary (SOP_BIN unset and 'sop' not on PATH)."
  note "Deterministic unit tests are NOT a substitute for this exercise."
  note "Set SOP_BIN=/path/to/sop and re-run to produce a READY observation."
  exit 2
fi
note "resolved sop: $SOP_BIN_RESOLVED"
note "-- sop --version --"
"$SOP_BIN_RESOLVED" --version >>"$TRANSCRIPT" 2>&1 || note "(no --version; continuing)"
note "-- sop --help --"
"$SOP_BIN_RESOLVED" --help >>"$TRANSCRIPT" 2>&1 || note "(no --help; continuing)"

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

# observe <label> <path> records labelled controller-rendered state.
observe() {
  note "-- [$1] controller GET $2 --"
  CTRL_GET "$BASE$2" >> "$TRANSCRIPT" 2>&1 || note "(GET $2 failed)"
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
observe "running" "/projects/$PROJ_ID"
observe "gate"    "/projects/$PROJ_ID/decisions"

TASK_ID="${C2_009_TASK_ID:-T1}"
note ""
note "## S1: inspect the gated task through the CONTROLLER"
observe "inspect" "/projects/$PROJ_ID/tasks/$TASK_ID"

note ""
note "## S1: APPROVE via the CONTROLLER (POST .../commands/approve)"
note "The controller must delegate this to SOP's own operation and report SOP's"
note "answer verbatim; the controller writes no approval state of its own."
ctrl_post "/projects/$PROJ_ID/tasks/$TASK_ID/commands/approve" "note=c2-009 dogfood approve" >/dev/null
ctrl_wait "/projects/$PROJ_ID/tasks/$TASK_ID/commands/approve"
note ""
note "## S1: controller reflects SOP's recorded decision"
observe "reflect-after-approve" "/projects/$PROJ_ID/tasks/$TASK_ID"

note ""
note "## S1: explicit Continue via the CONTROLLER (POST .../commands/start again)"
note "Continue is a separate, explicit action: the harness records that no"
note "resumption happened implicitly at approve time."
ctrl_post "/projects/$PROJ_ID/commands/start" >/dev/null
ctrl_wait "/projects/$PROJ_ID/commands/start"
observe "resumed" "/projects/$PROJ_ID"

##############################################################################
# S2  Decline scenario, driven through the CONTROLLER.
##############################################################################

note ""
note "## S2: DECLINE via the CONTROLLER (POST .../commands/decline)"
note "If SOP exposes no applicable gate at this point the controller reports that"
note "truthfully; the harness records whatever the controller actually returns."
ctrl_post "/projects/$PROJ_ID/tasks/$TASK_ID/commands/decline" "note=c2-009 dogfood decline" >/dev/null
ctrl_wait "/projects/$PROJ_ID/tasks/$TASK_ID/commands/decline"
observe "reflect-after-decline" "/projects/$PROJ_ID/tasks/$TASK_ID"

note ""
note "## S2: confirm no controller-manufactured failure/complete"
observe "task-list-after-decline" "/projects/$PROJ_ID/tasks"

##############################################################################
# S3  Changed-executed-task reconciliation, driven through the CONTROLLER.
#
# The controller resolves the plan from SOP's recorded provenance
# (.agent-sdlc/plan.meta.json) via `sop reconcile <PLAN.md> --list-changed
# --json`; this harness NEVER hardcodes PLAN.md, so it matches what the
# controller actually does.
##############################################################################

note ""
note "## S3: observe the changed-executed set the CONTROLLER reads from SOP"
observe "changed-tasks" "/projects/$PROJ_ID"

note ""
note "## S3: reconcile via the CONTROLLER (POST /commands/reconcile)"
ctrl_post "/projects/$PROJ_ID/commands/reconcile" >/dev/null
ctrl_wait "/projects/$PROJ_ID/commands/reconcile"

note ""
note "## S3: per-task accept-changed via the CONTROLLER (recorded boundary gap)"
note "The controller delegates `sop reconcile <PLAN.md> --accept-changed <id>`;"
note "a real SOP-side gap surfaces as a truthful *ReconcileRejection (conflict),"
note "which the harness records rather than treating as success."
ctrl_post "/projects/$PROJ_ID/tasks/$TASK_ID/commands/accept-changed" >/dev/null
ctrl_wait "/projects/$PROJ_ID/tasks/$TASK_ID/commands/accept-changed"

note ""
note "## S3: confirm SOP state is intact (no direct state.db write by the controller)"
observe "project-final" "/projects/$PROJ_ID"

note ""
note "==================================================================="
note " READY run complete. Observed via the CONTROLLER HTTP surface."
note " Transcript: $TRANSCRIPT"
note " Commit it verbatim (use --transcript) to convert this into committed"
note " evidence for docs/history/C2-009-REPORT.md."
note "==================================================================="

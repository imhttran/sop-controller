package sopclient

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SOP writes structured run artifacts under <root>/.agent-sdlc/runs/<task-id>/.
// This file reads them, read-only. They are SOP's own persisted record of what a
// run did and why it stopped: the controller reports them and never writes here,
// never infers workflow state, and never feeds them back into a decision.

// Run stages mirror agentic-sop's internal/run.Stage vocabulary: the
// deterministic lifecycle stage of one run. The controller reports these
// verbatim; it never constructs or invents a stage.
const (
	StageCreated         = "CREATED"
	StagePlanning        = "PLANNING"
	StageImplementing    = "IMPLEMENTING"
	StageValidating      = "VALIDATING"
	StageReviewing       = "REVIEWING"
	StageFixing          = "FIXING"
	StageWaitingForHuman = "WAITING_FOR_HUMAN"
	StagePassed          = "PASSED"
	StageFailed          = "FAILED"
)

// Failure dispositions mirror agentic-sop's internal/failure.Disposition. They
// are SOP's verdict on what it will do about a failure; the controller displays
// them and never decides them.
const (
	DispositionAutoFix    = "AUTO_FIX"
	DispositionContinue   = "CONTINUE"
	DispositionRetry      = "RETRY"
	DispositionReplan     = "REPLAN"
	DispositionNeedsHuman = "NEEDS_HUMAN"
)

// Activity action kinds group the meaningful SOP activity this model exposes.
// They mirror the PRD's action categories (file inspection, repository mutation,
// command execution, validation, review, JEV, quality decisions, recovery,
// completion) and use SOP's own vocabulary rather than controller-invented
// stages. The reader reports the Action SOP persisted; these constants document
// the kinds a consumer can recognize without re-deriving them.
const (
	// ActionInspect: inspecting files/state (read-only observation).
	ActionInspect = "inspect"
	// ActionMutate: mutating the repository (edit/write/patch).
	ActionMutate = "mutate"
	// ActionCommand: executing a command.
	ActionCommand = "command"
	// ActionValidate: running validation (build/test/lint).
	ActionValidate = "validate"
	// ActionReview: running review.
	ActionReview = "review"
	// ActionJEV: a JEV (independent verification) analysis.
	ActionJEV = "jev"
	// ActionQuality: a quality decision (accept/reject a result).
	ActionQuality = "quality"
	// ActionRecover: recovering from a failure (retry/auto-fix).
	ActionRecover = "recover"
	// ActionComplete: completion / lifecycle transition.
	ActionComplete = "complete"
)

// ActivityEvent is one structured SOP activity event: a short, machine-readable
// summary of a meaningful lifecycle transition or action (agentic-sop
// internal/activity.Event).
//
// Field meanings:
//
//	TaskID    the task whose run directory the event was read from.
//	Stage     SOP's persisted lifecycle stage (one of the Stage* constants),
//	          e.g. IMPLEMENTING, VALIDATING, REVIEWING, PASSED.
//	Action    the action kind SOP recorded (e.g. one of the Action* constants).
//	Detail    a safe summary (command name + exit status, file path category,
//	          review/JEV verdict). Never a prompt, secret, API key, environment
//	          dump, or unrestricted command/file content - see sanitizeDetail.
//	Timestamp SOP's recorded transition time (from activity.jsonl "timestamp").
//
// Safe-summary contract: Detail is produced only through sanitizeDetail, which
// strips/redacts prompts, secrets, API keys, environment dumps, and unrestricted
// command/file contents. Both the CLI and the controller consume this same model
// through internal/sopclient, so the guarantee is shared, not duplicated.
type ActivityEvent struct {
	TaskID    string
	Stage     string
	Action    string
	Detail    string
	Timestamp time.Time
}

// Classification mirrors SOP's failure classification
// (agentic-sop internal/failure.Classification): SOP's own verdict on why a run
// stopped short of a pass and what disposition it applied. The controller only
// reads and displays it.
type Classification struct {
	Kind        string
	Disposition string
	Confidence  string
	Reason      string
}

// HumanRequired reports whether SOP classified the failure as needing a human.
// Only this disposition (or an explicit approval boundary) is a human boundary;
// AUTO_FIX, CONTINUE, and RETRY are recovery SOP performs itself.
func (c *Classification) HumanRequired() bool {
	return c != nil && c.Disposition == DispositionNeedsHuman
}

// JEV is SOP's optional JEV analysis summary from a run report.
type JEV struct {
	Status   string
	Reason   string
	Summary  string
	Findings int
}

// RunInfo is the latest run's structured record for one task, assembled from
// SOP's run artifacts. Present is true when any artifact exists.
type RunInfo struct {
	Present        bool
	Stage          string
	Decision       string
	Reasons        []string
	FixCycles      int
	Provider       string
	Model          string
	ExecutionMode  string
	VerifiedFirst  bool
	GeneratedAt    time.Time
	UpdatedAt      time.Time
	Classification *Classification
	JEV            *JEV
}

type stateDoc struct {
	ID        string    `json:"id"`
	Stage     string    `json:"stage"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type classificationDoc struct {
	Kind        string `json:"kind"`
	Disposition string `json:"disposition"`
	Confidence  string `json:"confidence"`
	Reason      string `json:"reason"`
}

type activityDoc struct {
	Stage     string    `json:"stage"`
	Action    string    `json:"action"`
	Detail    string    `json:"detail"`
	Timestamp time.Time `json:"timestamp"`
}

type reportDoc struct {
	ID             string             `json:"id"`
	Stage          string             `json:"stage"`
	Provider       string             `json:"provider"`
	Model          string             `json:"model"`
	Decision       string             `json:"decision"`
	Reasons        []string           `json:"reasons"`
	FixCycles      int                `json:"fix_cycles"`
	ExecutionMode  string             `json:"execution_mode"`
	VerifiedFirst  bool               `json:"verified_first"`
	Classification *classificationDoc `json:"classification"`
	JEV            *struct {
		Status   string            `json:"status"`
		Reason   string            `json:"reason"`
		Summary  string            `json:"summary"`
		Findings []json.RawMessage `json:"findings"`
	} `json:"jev"`
	GeneratedAt time.Time `json:"generated_at"`
}

type planMeta struct {
	Source    string `json:"source"`
	PlanID    string `json:"plan_id"`
	FinalGate string `json:"final_gate"`
}

// runDir is the directory holding one task's run artifacts.
func (s *Store) runDir(taskID string) string {
	return filepath.Join(s.root, ".agent-sdlc", "runs", taskID)
}

// readJSON decodes a JSON file into v, reporting whether it existed and parsed.
func readJSON(path string, v any) bool {
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return json.Unmarshal(raw, v) == nil
}

// runInfo assembles the latest run's structured record for a task from SOP's
// artifacts. Missing artifacts yield a zero RunInfo (Present=false) rather than
// an error: a task that has never run simply has no run info.
func (s *Store) runInfo(taskID string) RunInfo {
	var ri RunInfo
	dir := s.runDir(taskID)

	var st stateDoc
	if readJSON(filepath.Join(dir, "state.json"), &st) {
		ri.Present = true
		ri.Stage = st.Stage
		ri.UpdatedAt = st.UpdatedAt
	}

	var rep reportDoc
	if readJSON(filepath.Join(dir, "report.json"), &rep) {
		ri.Present = true
		if ri.Stage == "" {
			ri.Stage = rep.Stage
		}
		ri.Decision = rep.Decision
		ri.Reasons = rep.Reasons
		ri.FixCycles = rep.FixCycles
		ri.Provider = rep.Provider
		ri.Model = rep.Model
		ri.ExecutionMode = rep.ExecutionMode
		ri.VerifiedFirst = rep.VerifiedFirst
		ri.GeneratedAt = rep.GeneratedAt
		if rep.Classification != nil {
			ri.Classification = toClassification(*rep.Classification)
		}
		if rep.JEV != nil {
			ri.JEV = &JEV{
				Status:   rep.JEV.Status,
				Reason:   rep.JEV.Reason,
				Summary:  rep.JEV.Summary,
				Findings: len(rep.JEV.Findings),
			}
		}
	}

	// classification.json is the reliable source: SOP writes it whenever a run
	// did not pass, including a run that stopped on an infrastructure error and
	// therefore wrote no report.json.
	if ri.Classification == nil {
		var cls classificationDoc
		if readJSON(filepath.Join(dir, "classification.json"), &cls) && cls.Disposition != "" {
			ri.Present = true
			ri.Classification = toClassification(cls)
		}
	}
	return ri
}

func toClassification(c classificationDoc) *Classification {
	return &Classification{
		Kind:        c.Kind,
		Disposition: c.Disposition,
		Confidence:  c.Confidence,
		Reason:      c.Reason,
	}
}

// runMeta returns the cheap display fields for the task list: the latest run
// stage, SOP's recovery disposition, and the run's reported fix-cycle count.
// It reads only the small state, classification, and report artifacts, so it
// stays inexpensive when listing many tasks.
func (s *Store) runMeta(taskID string) (stage, recovery string, fixCycles int) {
	dir := s.runDir(taskID)
	var st stateDoc
	if readJSON(filepath.Join(dir, "state.json"), &st) {
		stage = st.Stage
	}
	var cls classificationDoc
	if readJSON(filepath.Join(dir, "classification.json"), &cls) {
		recovery = cls.Disposition
	}
	var rep reportDoc
	if readJSON(filepath.Join(dir, "report.json"), &rep) {
		fixCycles = rep.FixCycles
	}
	return stage, recovery, fixCycles
}

// maxActivityEvents bounds the activity returned for one task. SOP appends to
// activity.jsonl across retries and once per tool interaction, so an unbounded
// read could grow large; the controller keeps the most recent events.
const maxActivityEvents = 200

// activity returns the structured activity SOP persisted for a task, oldest
// first, capped to the most recent maxActivityEvents. It returns nil when no
// activity artifact exists (for example a run that predates activity reporting,
// or a run SOP executed without it enabled).
//
// Ordering guarantee: events are returned in activity.jsonl append order (the
// order SOP wrote them, oldest first). Because a file's append order is stable,
// repeated reads of the same artifact yield the same order. Each event carries
// SOP's recorded Timestamp; when SOP records non-decreasing timestamps, the
// returned events are monotonic non-decreasing. Malformed lines are skipped
// rather than failing the whole view.
//
// The controller is a pass-through: it reports the Stage and Action SOP
// persisted and never invents a stage, decides a transition, or re-derives
// workflow state. Detail is passed through sanitizeDetail so every consumer
// shares one safe-summary guarantee.
func (s *Store) activity(taskID string) []ActivityEvent {
	raw, ok := s.runFile(taskID, "activity.jsonl")
	if !ok {
		return nil
	}
	var out []ActivityEvent
	sc := bufio.NewScanner(bytes.NewReader(raw))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var e activityDoc
		if json.Unmarshal(line, &e) != nil {
			continue // skip a malformed line rather than fail the whole view
		}
		out = append(out, ActivityEvent{
			TaskID:    taskID,
			Stage:     e.Stage,
			Action:    e.Action,
			Detail:    sanitizeDetail(e.Detail),
			Timestamp: e.Timestamp,
		})
	}
	if len(out) > maxActivityEvents {
		out = out[len(out)-maxActivityEvents:]
	}
	return out
}

// PlanSource returns the active plan's source path recorded by SOP in
// .agent-sdlc/plan.meta.json, or ("", false) when no plan is recorded. The
// controller uses it to name the plan for `sop reconcile` without guessing.
func (s *Store) PlanSource() (string, bool) {
	var meta planMeta
	if !readJSON(filepath.Join(s.root, ".agent-sdlc", "plan.meta.json"), &meta) {
		return "", false
	}
	src := strings.TrimSpace(meta.Source)
	return src, src != ""
}

// Plan returns the active plan and its final plan gate as SOP persisted them.
// When no plan metadata exists, the returned PlanGate has Recorded=false and
// FinalGate=StatusUnknown: absence is explicit and never inferred to be gated.
func (s *Store) Plan() PlanGate {
	var meta planMeta
	if !readJSON(filepath.Join(s.root, ".agent-sdlc", "plan.meta.json"), &meta) {
		return PlanGate{FinalGate: StatusUnknown}
	}
	src := strings.TrimSpace(meta.Source)
	gate := strings.TrimSpace(meta.FinalGate)
	if gate == "" {
		gate = StatusUnknown
	}
	return PlanGate{
		Recorded:   true,
		Source:     src,
		PlanID:     meta.PlanID,
		FinalGate:  gate,
		InProgress: gate == StatusUnknown,
	}
}

// reportArtifact is the filename SOP uses for a run's report.
const reportArtifact = "report.json"

// ReportRef returns the location/reference of the task's persisted report
// artifact. A missing report yields Present=false and an empty Path, never a
// synthesized path that could imply a result.
func (s *Store) ReportRef(taskID string) ReportRef {
	ref := ReportRef{TaskID: taskID}
	path := filepath.Join(s.runDir(taskID), reportArtifact)
	if _, err := os.Stat(path); err != nil {
		return ref
	}
	ref.Present = true
	ref.Path = path
	ref.Name = reportArtifact
	return ref
}

package sopclient

import (
	"path/filepath"
	"strings"
	"time"
)

// This file defines the controller-facing performance read model: where SOP
// recorded a task or a run spending its time, and the operation counts SOP
// recorded (agent calls, validation/review runs, fix cycles).
//
// Source-of-truth rule (performance): PERFORMANCE IS SOP'S MEASUREMENT, NOT THE
// CONTROLLER'S. agentic-sop owns timing through its internal/perf package and
// persists it as diagnostic artifacts:
//
//	.agent-sdlc/runs/<task-id>/metrics.json   per task (agentic-sop perf.Task)
//	.agent-sdlc/runs/<plan-id>/metrics.json   per run  (agentic-sop perf.Run)
//	.agent-sdlc/runs/<task-id>/report.json    the task's "performance" field
//
// The controller only READS and PROJECTS these values. It starts no timer,
// derives no lifecycle duration from its own clock, an HTTP request, a polling
// interval, a task-status transition, or process-local state, and stores no
// authoritative performance state. Performance is diagnostic metadata only: it
// never influences task status, selection, retry, recovery, approval,
// reconciliation, validation, review, quality gating, or routing. The controller
// therefore never decides whether a task advances from anything in this file.
//
// Missing-data rule: when SOP persisted no performance record (a task that never
// ran, a run that predates instrumentation, or an unreadable/malformed artifact)
// the record is absent (Present=false) and every field stays at its zero value.
// The controller never fabricates a zero-duration measurement and never
// zero-fills an absent one, so a renderer can show an explicit absence instead of
// a misleading row of 0s.

// SOP's stage and validation-category keys, mirroring agentic-sop
// internal/perf (StagesMS/ValidationMS map keys). They are SOP's own vocabulary;
// the controller uses them only to look values up.
const (
	perfStagePlan       = "plan"
	perfStageImplement  = "implement"
	perfStageValidation = "validation"
	perfStageReview     = "review"
	perfStageFix        = "fix"

	perfCategoryBuild = "build"
	perfCategoryTest  = "test"
	perfCategoryLint  = "lint"
)

// metricsFileName is the artifact SOP writes for both a task's and a run's
// performance record.
const metricsFileName = "metrics.json"

// Performance is the controller-facing projection of SOP's persisted per-task
// performance record (agentic-sop internal/perf.Task). Every duration is a SOP
// measurement reported verbatim; the controller computes none of them.
type Performance struct {
	// Present is true when SOP persisted a performance record carrying at least
	// one measurement or count. When false the remaining fields are zero and a
	// renderer must show an explicit absence, never a 0s measurement.
	Present bool

	// Total is the task's wall-clock duration SOP measured.
	Total time.Duration

	// Stage durations (SOP's StagesMS), each zero when SOP recorded no time in
	// that stage (SOP omits an unmeasured stage; the zero is an absence, not a
	// measured 0).
	Plan       time.Duration
	Implement  time.Duration
	Validation time.Duration
	Review     time.Duration
	Fix        time.Duration

	// Validation breakdown (SOP's ValidationMS): the build/test/lint split of the
	// Validation stage. Zero when SOP recorded no per-category time.
	Build time.Duration
	Test  time.Duration
	Lint  time.Duration

	// Operation counts SOP recorded (agentic-sop perf.Counts), reported verbatim.
	AgentCalls        int
	AgentCallsAvoided int
	ValidationRuns    int
	ValidationReused  int
	ReviewRuns        int
	ReviewReused      int
	FixCycles         int
	PlanRepairs       int
}

// Agent is the task's agent time as SOP's own report defines it: the plan,
// implement, and fix stages. It is arithmetic over SOP-recorded durations (the
// same category rule agentic-sop's perf.Run.CategoryMS applies), not a new
// measurement.
func (p Performance) Agent() time.Duration { return p.Plan + p.Implement + p.Fix }

// CategoryShare is one measured category's duration and its whole-percent share
// of the measured stage time. It is presentation only: arithmetic over
// SOP-recorded durations that carries no controller judgement (it never labels a
// category slow, bad, or needing optimization).
type CategoryShare struct {
	// Name is the category label (Agent / Validation / Review).
	Name string
	// Duration is the category's SOP-measured total.
	Duration time.Duration
	// Percent is Duration's whole-percent share of the measured stage time,
	// rounded the same way SOP's own report rounds it.
	Percent int
}

// Shares returns the agent/validation/review split SOP's own report renders, with
// each category's whole-percent share of the measured stage time. It is
// arithmetic over SOP-recorded durations only. When no stage time was measured it
// returns nil, so a renderer shows no percentage rather than 0%.
func (p Performance) Shares() []CategoryShare {
	return categoryShares(p.Agent(), p.Validation, p.Review)
}

// PlanPerformance is the controller-facing projection of SOP's persisted
// plan-level performance aggregate (agentic-sop internal/perf.Run), written by a
// full `sop run` at .agent-sdlc/runs/<plan-id>/metrics.json.
type PlanPerformance struct {
	// Present is true when SOP persisted a run aggregate for the active plan.
	Present bool
	// PlanID is SOP's recorded plan identity the aggregate belongs to
	// (plan.meta.json "plan_id").
	PlanID string
	// Tasks is the number of task records SOP folded into the aggregate.
	Tasks int
	// Total is the run's wall-clock duration SOP measured.
	Total time.Duration
	// Agent/Validation/Review are the measured category totals across the run's
	// task records, using SOP's own category rule (agent = plan + implement +
	// fix). They are arithmetic over SOP-recorded durations, not a new
	// measurement; the per-run operation counts below come straight from SOP's
	// already-aggregated record.
	Agent      time.Duration
	Validation time.Duration
	Review     time.Duration

	// Operation counts SOP already aggregated for the run (perf.Run.Counts).
	AgentCalls        int
	AgentCallsAvoided int
	ValidationRuns    int
	ValidationReused  int
	ReviewRuns        int
	ReviewReused      int
	FixCycles         int
	PlanRepairs       int
}

// Shares returns the run's agent/validation/review split with whole-percent
// shares, or nil when no stage time was measured.
func (p PlanPerformance) Shares() []CategoryShare {
	return categoryShares(p.Agent, p.Validation, p.Review)
}

// categoryShares is the single place the agent/validation/review share is
// derived, so a task and a run view cannot disagree about the arithmetic.
func categoryShares(agent, validation, review time.Duration) []CategoryShare {
	measured := agent + validation + review
	if measured <= 0 {
		return nil
	}
	return []CategoryShare{
		{Name: "Agent", Duration: agent, Percent: percentOf(agent, measured)},
		{Name: "Validation", Duration: validation, Percent: percentOf(validation, measured)},
		{Name: "Review", Duration: review, Percent: percentOf(review, measured)},
	}
}

// percentOf rounds part/whole to a whole percent, matching agentic-sop's own
// percentage rule so the controller's share cannot differ from SOP's report.
func percentOf(part, whole time.Duration) int {
	w := whole.Milliseconds()
	if w <= 0 {
		return 0
	}
	return int((part.Milliseconds()*100 + w/2) / w)
}

// metricsCountsDoc mirrors agentic-sop perf.Counts' JSON encoding.
type metricsCountsDoc struct {
	AgentCalls        int `json:"agent_calls"`
	AgentCallsAvoided int `json:"agent_calls_avoided"`
	ValidationRuns    int `json:"validation_runs"`
	ValidationReused  int `json:"validation_reused"`
	ReviewRuns        int `json:"review_runs"`
	ReviewReused      int `json:"review_reused"`
	FixCycles         int `json:"fix_cycles"`
	PlanRepairs       int `json:"plan_repairs"`
}

// taskMetricsDoc mirrors agentic-sop perf.Task's JSON encoding.
type taskMetricsDoc struct {
	ID           string           `json:"id"`
	TotalMS      int64            `json:"total_ms"`
	StagesMS     map[string]int64 `json:"stages_ms"`
	ValidationMS map[string]int64 `json:"validation_ms"`
	Counts       metricsCountsDoc `json:"counts"`
}

// runMetricsDoc mirrors agentic-sop perf.Run's JSON encoding (the plan-level
// aggregate).
type runMetricsDoc struct {
	StartedAt time.Time        `json:"started_at"`
	TotalMS   int64            `json:"total_ms"`
	Tasks     []taskMetricsDoc `json:"tasks"`
	Counts    metricsCountsDoc `json:"counts"`
}

func msToDuration(ms int64) time.Duration { return time.Duration(ms) * time.Millisecond }

// performanceFromTaskDoc projects a parsed perf.Task record. ok is false when the
// record carries no measurement, so a present-but-empty document is never
// rendered as a real measurement.
func performanceFromTaskDoc(doc taskMetricsDoc) (Performance, bool) {
	p := Performance{
		Total:             msToDuration(doc.TotalMS),
		Plan:              msToDuration(doc.StagesMS[perfStagePlan]),
		Implement:         msToDuration(doc.StagesMS[perfStageImplement]),
		Validation:        msToDuration(doc.StagesMS[perfStageValidation]),
		Review:            msToDuration(doc.StagesMS[perfStageReview]),
		Fix:               msToDuration(doc.StagesMS[perfStageFix]),
		Build:             msToDuration(doc.ValidationMS[perfCategoryBuild]),
		Test:              msToDuration(doc.ValidationMS[perfCategoryTest]),
		Lint:              msToDuration(doc.ValidationMS[perfCategoryLint]),
		AgentCalls:        doc.Counts.AgentCalls,
		AgentCallsAvoided: doc.Counts.AgentCallsAvoided,
		ValidationRuns:    doc.Counts.ValidationRuns,
		ValidationReused:  doc.Counts.ValidationReused,
		ReviewRuns:        doc.Counts.ReviewRuns,
		ReviewReused:      doc.Counts.ReviewReused,
		FixCycles:         doc.Counts.FixCycles,
		PlanRepairs:       doc.Counts.PlanRepairs,
	}
	if !p.hasAny() {
		return Performance{}, false
	}
	p.Present = true
	return p, true
}

// hasAny reports whether a projected record carries at least one SOP-measured
// value, so a present-but-empty document is treated as absent.
func (p Performance) hasAny() bool {
	return p.Total > 0 || p.Plan > 0 || p.Implement > 0 || p.Validation > 0 ||
		p.Review > 0 || p.Fix > 0 ||
		p.AgentCalls != 0 || p.AgentCallsAvoided != 0 || p.ValidationRuns != 0 ||
		p.ValidationReused != 0 || p.ReviewRuns != 0 || p.ReviewReused != 0 ||
		p.FixCycles != 0 || p.PlanRepairs != 0
}

// Performance returns the latest performance record SOP persisted for a task,
// read-only. It first reads SOP's per-task metrics artifact
// (.agent-sdlc/runs/<task-id>/metrics.json); when that artifact is absent or
// unreadable it falls back to the redundant performance field of the task's
// report.json, matching SOP's own read order. A missing or malformed record
// yields Present=false and never an error: a task that never ran, and a run that
// predates performance instrumentation, simply have no performance record.
//
// It reads files only: it opens no timer, writes nothing under .agent-sdlc, and
// mutates no SOP state.
func (s *Store) Performance(taskID string) Performance {
	var doc taskMetricsDoc
	if readJSON(filepath.Join(s.runDir(taskID), metricsFileName), &doc) {
		if p, ok := performanceFromTaskDoc(doc); ok {
			return p
		}
	}
	// Fallback: SOP embeds the same task record in report.json's "performance".
	var rep struct {
		Performance taskMetricsDoc `json:"performance"`
	}
	if readJSON(filepath.Join(s.runDir(taskID), reportArtifact), &rep) {
		if p, ok := performanceFromTaskDoc(rep.Performance); ok {
			return p
		}
	}
	return Performance{}
}

// PlanPerformance reads SOP's plan-level performance aggregate for the active
// plan, read-only. The plan identity comes from SOP's recorded plan.meta.json
// ("plan_id"), never from a heuristic or a filename guess; when SOP recorded no
// plan, or persisted no aggregate for it, the record is absent (Present=false)
// rather than guessed.
//
// Like Performance it reads files only and mutates no SOP state.
func (s *Store) PlanPerformance() PlanPerformance {
	var meta planMeta
	if !readJSON(filepath.Join(s.root, ".agent-sdlc", "plan.meta.json"), &meta) {
		return PlanPerformance{}
	}
	planID := strings.TrimSpace(meta.PlanID)
	if planID == "" {
		return PlanPerformance{}
	}
	var doc runMetricsDoc
	if !readJSON(filepath.Join(s.root, ".agent-sdlc", "runs", planID, metricsFileName), &doc) {
		return PlanPerformance{}
	}
	return planPerformanceFromRunDoc(planID, doc)
}

// planPerformanceFromRunDoc projects a parsed perf.Run aggregate. The per-run
// counts come straight from SOP's already-aggregated record; only the
// agent/validation/review category split is summed from the task records, using
// SOP's own category rule.
func planPerformanceFromRunDoc(planID string, doc runMetricsDoc) PlanPerformance {
	p := PlanPerformance{
		PlanID:            planID,
		Tasks:             len(doc.Tasks),
		Total:             msToDuration(doc.TotalMS),
		AgentCalls:        doc.Counts.AgentCalls,
		AgentCallsAvoided: doc.Counts.AgentCallsAvoided,
		ValidationRuns:    doc.Counts.ValidationRuns,
		ValidationReused:  doc.Counts.ValidationReused,
		ReviewRuns:        doc.Counts.ReviewRuns,
		ReviewReused:      doc.Counts.ReviewReused,
		FixCycles:         doc.Counts.FixCycles,
		PlanRepairs:       doc.Counts.PlanRepairs,
	}
	for _, t := range doc.Tasks {
		p.Agent += msToDuration(t.StagesMS[perfStagePlan] + t.StagesMS[perfStageImplement] + t.StagesMS[perfStageFix])
		p.Validation += msToDuration(t.StagesMS[perfStageValidation])
		p.Review += msToDuration(t.StagesMS[perfStageReview])
	}
	if !p.hasAny() {
		return PlanPerformance{}
	}
	p.Present = true
	return p
}

// hasAny reports whether the aggregate carries at least one SOP-measured value.
func (p PlanPerformance) hasAny() bool {
	return len(p.PlanID) > 0 && (p.Tasks > 0 || p.Total > 0 || p.Agent > 0 ||
		p.Validation > 0 || p.Review > 0 ||
		p.AgentCalls != 0 || p.AgentCallsAvoided != 0 || p.ValidationRuns != 0 ||
		p.ValidationReused != 0 || p.ReviewRuns != 0 || p.ReviewReused != 0 ||
		p.FixCycles != 0 || p.PlanRepairs != 0)
}

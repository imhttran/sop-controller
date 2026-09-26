package sopclient

// This file defines the controller-facing structured status model (CTRL003):
// enough SOP-persisted state for the controller to render the current plan and
// execution status without ever opening SOP state storage directly.
//
// Missing-data rule (CTRL003): every status field reports absence explicitly.
// A missing artifact yields an absent/unknown value ("" for a field that does
// not apply, or the explicit StatusUnknown constant); it never resolves to a
// PASS. The controller renders completed/passing states only from SOP-persisted
// evidence. Constants below come from SOP's own vocabulary where SOP defines
// one; the aggregate status values used here are SOP's Status values or the
// explicit no-evidence value.

// StatusUnknown is the explicit representation of "SOP persisted no evidence".
// It is distinct from every real SOP status (PASS, FAIL, LOCAL_DONE, ...): a
// renderer can therefore show "unknown" without implying a passing result.
const StatusUnknown = "UNKNOWN"

// Aggregate validation statuses. ValidationCheck carries SOP's own per-check
// Status; the aggregate reports SOP's own PASS/FAIL/SKIP vocabulary, or
// StatusUnknown when SOP persisted no validation artifact.
const (
	ValidationPass    = "PASS"
	ValidationFail    = "FAIL"
	ValidationSkipped = "SKIP"
)

// Aggregate review outcomes over SOP's Review data. StatusUnknown is reported
// when SOP persisted no review artifact.
const (
	ReviewNotRun = "NOT_RUN"
	ReviewPassed = "PASS"
	ReviewFailed = "FAIL"
)

// PlanGate is the controller-friendly projection of SOP's active plan and its
// final plan gate (plan.meta.json). Recorded distinguishes "SOP persisted a
// plan" from "no plan metadata exists"; Source/PlanID/FinalGate are reported
// verbatim from SOP. FinalGate is StatusUnknown when SOP recorded none, so a
// plan with no gate evidence never renders as gated.
type PlanGate struct {
	// Recorded is true when SOP persisted plan metadata.
	Recorded bool
	// Source is SOP's recorded active plan path (plan.meta.json "source").
	Source string
	// PlanID is SOP's recorded plan id (plan.meta.json "plan_id"), if any.
	PlanID string
	// FinalGate is SOP's recorded final plan gate value, or StatusUnknown.
	FinalGate string
	// InProgress is true when SOP recorded an active plan whose final gate is
	// not a completed gate value.
	InProgress bool
}

// StatusLevel is a render-ready status string plus whether SOP persisted any
// evidence for it. Level is StatusUnknown when the artifact is missing; a
// missing artifact never yields PASS.
type StatusLevel struct {
	// Level is the reported status value (SOP vocabulary, or StatusUnknown).
	Level string
	// Present is true when SOP persisted evidence backing Level.
	Present bool
}

// RunStatus exposes the current run's stage and decision as distinct fields the
// controller can render, without opening SOP state storage.
type RunStatus struct {
	// Present is true when any run artifact exists for the task.
	Present bool
	// Stage is SOP's current lifecycle stage, or "" when the task has not run.
	Stage string
	// Decision is SOP's run decision (e.g. PASS/FAIL), or "".
	Decision string
	// Active reports whether the run is in a non-terminal, in-flight stage.
	Active bool
	// Blocked is true when SOP parked the run at a human boundary
	// (WAITING_FOR_HUMAN) rather than an active or completed stage.
	Blocked bool
}

// Status projects the run record into the CTRL003 render-ready run status. A
// missing run leaves Present=false; Decision stays "" and never becomes PASS.
func (r RunInfo) Status() RunStatus {
	return RunStatus{
		Present:  r.Present,
		Stage:    r.Stage,
		Decision: r.Decision,
		Active:   activeStage(r.Stage),
		Blocked:  r.Stage == StageWaitingForHuman,
	}
}

// activeStage reports whether SOP's run stage is a non-terminal, in-flight
// stage. It derives nothing SOP did not persist: it only groups SOP's Stage*
// constants for display.
func activeStage(stage string) bool {
	switch stage {
	case StageCreated, StagePlanning, StageImplementing, StageValidating, StageReviewing, StageFixing:
		return true
	default:
		return false
	}
}

// aggregateValidation reduces SOP's per-check validation results into one
// status. No checks yields an absent StatusLevel (Present=false, Level
// StatusUnknown): a missing validation artifact is never PASS. Any failing check
// makes the aggregate FAIL; otherwise the aggregate is PASS.
func aggregateValidation(checks []ValidationCheck) StatusLevel {
	if len(checks) == 0 {
		return StatusLevel{Level: StatusUnknown}
	}
	level := ValidationPass
	for _, c := range checks {
		switch c.Status {
		case ValidationFail:
			return StatusLevel{Level: ValidationFail, Present: true}
		case ValidationSkipped:
			if level == ValidationPass {
				level = ValidationSkipped
			}
		}
	}
	return StatusLevel{Level: level, Present: true}
}

// aggregateReview reduces SOP's review result into one status. A review with no
// artifact is absent (Present=false, StatusUnknown) and never PASS; a review
// with one or more findings is FAIL; a recorded review with no findings is PASS.
func aggregateReview(r Review) StatusLevel {
	if r.Summary == "" && len(r.Findings) == 0 {
		return StatusLevel{Level: StatusUnknown}
	}
	if len(r.Findings) > 0 {
		return StatusLevel{Level: ReviewFailed, Present: true}
	}
	return StatusLevel{Level: ReviewPassed, Present: true}
}

// aggregateQuality reduces SOP's JEV summary into the JEV/quality status. A
// missing JEV artifact is absent (Present=false, StatusUnknown), never PASS.
func aggregateQuality(j *JEV) StatusLevel {
	if j == nil {
		return StatusLevel{Level: StatusUnknown}
	}
	status := j.Status
	if status == "" {
		status = StatusUnknown
	}
	return StatusLevel{Level: status, Present: true}
}

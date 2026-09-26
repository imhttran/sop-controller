package sopclient

// This file defines the controller-facing checkpoint / bounded-progress model
// (CTRL010): bounded progress reported by SOP for work that spans multiple agent
// invocations, such as coverage analysis ("11/15 cases analyzed, covered: 11,
// missing: 4, next: ..."), remaining acceptance criteria, or other checkpoints
// SOP explicitly reports.
//
// Source-of-truth rule (CTRL010): CheckpointProgress carries ONLY values SOP
// persisted in its run artifacts or structured activity. The controller never
// computes a percentage, interpolates a count, or extrapolates a total. When SOP
// reported no checkpoint, the record is absent (Present=false) and every numeric
// field stays at its zero value with its Has* flag false, following the CTRL003
// missing-data rule.
//
// Partial progress is never task completion: a checkpoint with an analyzed count
// below its total is bounded, in-flight progress. It does not, and must not,
// resolve to a passed/complete state. Task completion still comes only from SOP's
// own task status and run decision.

// CheckpointProgress is the controller-friendly projection of the most recent
// checkpoint SOP reported for a task. It reports SOP's values verbatim: no
// rounding, interpolation, or extrapolation.
//
// Present distinguishes "SOP reported a checkpoint" from "no checkpoint
// evidence". When Present is false, a renderer shows an explicit absence and must
// not display a number.
type CheckpointProgress struct {
	// Present is true when SOP reported a checkpoint. When false, the remaining
	// fields are zero/empty and a renderer must show an explicit absence.
	Present bool
	// Label is SOP's checkpoint label/source identifier (e.g. "JEV012" or a
	// coverage-analysis line). Reported verbatim; never synthesized.
	Label string
	// Source names where SOP reported the checkpoint (artifact or structured
	// activity), so a reader can trace the value to its origin. It falls back to
	// the artifact filename only as a fact about where the data was read from;
	// it is never a synthesis of a SOP-reported field (see checkpointDoc.progress).
	Source string
	// Analyzed and Total are SOP's reported progress counters. HasAnalyzed and
	// HasTotal distinguish a genuine zero from an absent counter; the controller
	// never invents either value, and never derives one from the other.
	Analyzed    int
	HasAnalyzed bool
	Total       int
	HasTotal    bool
	// Covered and Missing are SOP's reported covered/missing counters, each with
	// a Has* flag so zero is distinguishable from absence.
	Covered    int
	HasCovered bool
	Missing    int
	HasMissing bool
	// Next is SOP's reported next-step hint (e.g. "add deterministic
	// coverage"), reported verbatim. Empty when SOP reported none.
	Next string
}

// Complete reports whether SOP's reported checkpoint is bounded progress that
// reached its total. It returns false whenever SOP reported no analyzed/total
// pair, so an absent checkpoint is never a completion. Even when true it means
// only that this checkpoint reached its total; it never overrides SOP's own task
// status as the source of task completion.
func (c CheckpointProgress) Complete() bool {
	if !c.Present || !c.HasAnalyzed || !c.HasTotal {
		return false
	}
	return c.Analyzed >= c.Total
}

// InProgress reports whether SOP reported a partial (bounded, in-flight)
// checkpoint: evidence is present and the analyzed count has not reached the
// reported total. Partial progress is never task completion.
func (c CheckpointProgress) InProgress() bool {
	if !c.Present || !c.HasAnalyzed || !c.HasTotal {
		return false
	}
	return c.Analyzed < c.Total
}

// newCheckpoint builds a CheckpointProgress from SOP-reported evidence, applying
// the shared safe-summary sanitization to free-form detail (Label/Source/Next)
// and marking the counters present only when SOP actually reported them. It
// never derives a counter from another: an absent Total is not inferred from
// Analyzed, and vice versa. The caller passes has* flags that reflect SOP's
// persisted record, not a controller guess. Free-form detail is bounded by the
// shared sanitizeDetail rule (maxDetailLen), so both read paths share one bound.
func newCheckpoint(label, source string, analyzed, total, covered, missing int, hasAnalyzed, hasTotal, hasCovered, hasMissing bool, next string) CheckpointProgress {
	return CheckpointProgress{
		Present:     true,
		Label:       sanitizeDetail(label),
		Source:      sanitizeDetail(source),
		Analyzed:    analyzed,
		HasAnalyzed: hasAnalyzed,
		Total:       total,
		HasTotal:    hasTotal,
		Covered:     covered,
		HasCovered:  hasCovered,
		Missing:     missing,
		HasMissing:  hasMissing,
		Next:        sanitizeDetail(next),
	}
}

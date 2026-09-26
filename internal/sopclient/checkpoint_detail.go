package sopclient

import (
	"regexp"
	"strconv"
	"strings"
)

// This file extracts a bounded-progress checkpoint from one structured activity
// entry (CTRL010). SOP's activity Detail is a safe summary (sanitizeDetail has
// already been applied at the Store.activity boundary), and for work that spans
// multiple agent invocations SOP may report a bounded-progress line such as:
//
//	JEV012, 11/15 cases analyzed, covered: 11, missing: 4, next: add deterministic coverage
//
// checkpointFromDetail reports such a line VERBATIM. It extracts a counter only
// when SOP explicitly stated it in the detail; it never computes a percentage,
// never derives one counter from another (a missing "covered" is not inferred
// from analyzed/total), and never zero-fills an absent counter. When the detail
// carries no recognizable bounded-progress line it returns ok=false so the
// caller keeps looking and ultimately reports an explicit absence.

// analyzedTotalRe matches the "<analyzed>/<total>" progress form, e.g. "11/15".
var analyzedTotalRe = regexp.MustCompile(`\b(\d+)\s*/\s*(\d+)\b`)

// checkpointCounterRes match SOP's labelled counters. Each is anchored on the
// label SOP uses so an arbitrary number elsewhere in a detail line is never
// mistaken for a counter. They are case-insensitive to tolerate SOP's casing.
var (
	coveredRe = regexp.MustCompile(`(?i)\bcovered\s*[:=]?\s*(\d+)\b`)
	missingRe = regexp.MustCompile(`(?i)\bmissing\s*[:=]?\s*(\d+)\b`)
	nextRe    = regexp.MustCompile(`(?i)\bnext\s*[:=]\s*(.+)$`)
	labelRe   = regexp.MustCompile(`^([A-Za-z][A-Za-z0-9_-]*[0-9][A-Za-z0-9_-]*)\s*[,:]`)
)

// analyzedTotalKeywords gates the analyzed/total form: a bare "11/15" is only
// accepted as a checkpoint when SOP's detail also names the unit of work it
// counted (e.g. "cases analyzed"), so an unrelated ratio in a detail line is
// never misread as checkpoint progress.
var analyzedTotalKeywords = []string{"analyzed", "cases", "checks", "criteria", "items", "files", "test"}

// checkpointFromDetail derives a checkpoint from one SOP activity entry. It
// reports ok=false when the entry's Detail carries no bounded-progress evidence,
// so the caller can continue searching older entries and, if none match, report
// an explicit absent checkpoint rather than a fabricated one. Present counters
// are reported verbatim with their Has* flags; absent counters keep a zero value
// and a false flag, so a renderer never shows an invented number.
func checkpointFromDetail(taskID string, e ActivityEvent) (CheckpointProgress, bool) {
	detail := strings.TrimSpace(e.Detail)
	if detail == "" {
		return CheckpointProgress{}, false
	}

	label := ""
	if m := labelRe.FindStringSubmatch(detail); m != nil {
		label = m[1]
	}

	var (
		analyzed, total, covered, missing             int
		hasAnalyzed, hasTotal, hasCovered, hasMissing bool
	)

	// Accept the analyzed/total form only when the detail names the unit counted,
	// so an unrelated ratio is never misread as checkpoint progress.
	lower := strings.ToLower(detail)
	namesUnit := false
	for _, kw := range analyzedTotalKeywords {
		if strings.Contains(lower, kw) {
			namesUnit = true
			break
		}
	}
	if namesUnit {
		if m := analyzedTotalRe.FindStringSubmatch(detail); m != nil {
			if v, err := strconv.Atoi(m[1]); err == nil {
				analyzed, hasAnalyzed = v, true
			}
			if v, err := strconv.Atoi(m[2]); err == nil {
				total, hasTotal = v, true
			}
		}
	}
	if m := coveredRe.FindStringSubmatch(detail); m != nil {
		if v, err := strconv.Atoi(m[1]); err == nil {
			covered, hasCovered = v, true
		}
	}
	if m := missingRe.FindStringSubmatch(detail); m != nil {
		if v, err := strconv.Atoi(m[1]); err == nil {
			missing, hasMissing = v, true
		}
	}

	// Recognizable checkpoint evidence requires at least one counter SOP
	// explicitly reported. A label alone (e.g. "JEV012, starting") is context,
	// not bounded progress, and is not treated as a checkpoint.
	if !hasAnalyzed && !hasTotal && !hasCovered && !hasMissing {
		return CheckpointProgress{}, false
	}

	next := ""
	if m := nextRe.FindStringSubmatch(detail); m != nil {
		next = strings.TrimSpace(m[1])
	}

	// The label falls back to SOP's stage/action vocabulary only when the detail
	// name no explicit scope id, so the surface still names what was counted
	// without the controller inventing an identifier SOP did not report.
	if label == "" {
		label = strings.TrimSpace(e.Stage)
	}

	// Source names the SOP read path (structured activity) so a reader can trace
	// the value; the counters stay verbatim SOP-reported values.
	return newCheckpoint(
		label,
		"activity",
		analyzed, total, covered, missing,
		hasAnalyzed, hasTotal, hasCovered, hasMissing,
		next,
	), true
}

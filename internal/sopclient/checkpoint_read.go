package sopclient

import (
	"path/filepath"
	"strings"
)

// This file reads SOP-reported checkpoint / bounded-progress evidence (CTRL010)
// from SOP's run artifacts and structured activity. It is read-only and never
// synthesizes a value: an absent source yields an absent CheckpointProgress.
//
// Source-of-truth note: whether SOP persists a dedicated checkpoint artifact is
// not determinable from the controller repository (.agent-sdlc/ is SOP-owned and
// the controller must not inspect it directly). The read path therefore treats a
// dedicated checkpoint record as OPTIONAL and present-or-absent: when SOP emits
// it, its values are reported verbatim; when SOP emits nothing, the checkpoint is
// reported absent rather than fabricated. Structured activity is also consulted,
// because the PRD's own example ("JEV012, 11/15 cases analyzed", a JEV artifact)
// suggests bounded progress may arrive through activity or a JEV artifact rather
// than a dedicated file.

// checkpointArtifact is the optional filename a run may carry a bounded-progress
// checkpoint line in. It is optional: SOP writes it only for work that spans
// multiple agent invocations and reports bounded progress, so its absence is
// normal and never an error.
const checkpointArtifact = "checkpoint.json"

// checkpointDoc is a persisted checkpoint record. Every counter is a pointer so
// the controller can tell a genuine zero SOP persisted from a field SOP did not
// report at all: it never infers one counter from another.
type checkpointDoc struct {
	Label    string `json:"label"`
	Source   string `json:"source"`
	Analyzed *int   `json:"analyzed"`
	Total    *int   `json:"total"`
	Covered  *int   `json:"covered"`
	Missing  *int   `json:"missing"`
	Next     string `json:"next"`
}

// Checkpoint returns the most recent SOP-reported checkpoint for a task.
//
// Selection rule: the most recent SOP source wins. A dedicated checkpoint
// artifact (checkpoint.json) is preferred when present, because it is the most
// explicit SOP-reported checkpoint; otherwise the latest structured activity
// entry that carries a bounded-progress line is used. When SOP reported none, the
// returned record has Present=false and no counters, so a renderer shows an
// explicit absence rather than a fabricated number.
//
// It reads SOP's run directory through the same read-only boundary as runInfo
// and activity; the controller never opens SOP state storage directly and never
// computes a percentage, interpolation, or extrapolation of these values.
func (s *Store) Checkpoint(taskID string) CheckpointProgress {
	path := filepath.Join(s.runDir(taskID), checkpointArtifact)
	var doc checkpointDoc
	if readJSON(path, &doc) {
		return doc.progress("")
	}
	// Fall back to the latest structured activity entry that reported bounded
	// progress (SOP's activity stream is a documented, supported read path).
	if cp, ok := s.checkpointFromActivity(taskID); ok {
		return cp
	}
	return CheckpointProgress{}
}

// progress projects a persisted checkpoint document into the read model. A
// counter is marked present only when SOP recorded it (non-nil pointer); an
// absent counter stays at zero with its Has* flag false. The reported values are
// verbatim: no rounding, interpolation, or extrapolation.
//
// fallbackSource names the SOP source when the record itself carries no "source"
// field. It is used only as a provenance label; when it is empty the Source stays
// empty so the renderer shows nothing rather than the controller naming its own
// filename as if SOP had.
func (d checkpointDoc) progress(fallbackSource string) CheckpointProgress {
	source := strings.TrimSpace(d.Source)
	if source == "" {
		source = strings.TrimSpace(fallbackSource)
	}
	return newCheckpoint(
		d.Label,
		source,
		deref(d.Analyzed), deref(d.Total), deref(d.Covered), deref(d.Missing),
		d.Analyzed != nil, d.Total != nil, d.Covered != nil, d.Missing != nil,
		d.Next,
	)
}

// checkpointFromActivity derives a checkpoint from the latest structured
// activity entry that reported a bounded-progress line. SOP's activity Detail is
// a safe summary and may carry a checkpoint identifier plus analyzed/total
// counters (the PRD's "JEV012, 11/15 cases analyzed, covered: 11, missing: 4"
// example). It reports the entry's Detail verbatim as the label and source, and
// extracts counters only when SOP's detail explicitly states them; it never
// invents a counter or computes a percentage. The newest such entry wins.
//
// The activity slice is materialized exactly once per call, so indexing stays
// consistent and no redundant re-read/re-sanitization occurs per iteration.
func (s *Store) checkpointFromActivity(taskID string) (CheckpointProgress, bool) {
	events := s.activity(taskID)
	for i := len(events) - 1; i >= 0; i-- {
		if cp, ok := checkpointFromDetail(taskID, events[i]); ok {
			return cp, true
		}
	}
	return CheckpointProgress{}, false
}

// deref returns the pointed-to int or 0 when SOP reported no value.
func deref(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

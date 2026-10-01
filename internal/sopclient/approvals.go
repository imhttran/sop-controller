package sopclient

// This file defines the controller-facing read of SOP's authoritative approval
// listing (C2-001): the machine-readable surface by which SOP reports every
// human approval gate that currently applies to a project's tasks.
//
// Ownership rule: SOP owns approval and every lifecycle gate. The controller
// never decides that approval is required, never fabricates a gate, and never
// approves work itself. The ONLY source of truth for whether a gate exists and
// whether it is currently applicable is SOP's structured approval listing; the
// controller-side inference this file replaces (run classification, run stage,
// BLOCKED status) is retired. When SOP reports no applicable gate for a task,
// this model reports none and no approval may be taken.
//
// The listing is SOP's `sop approvals --json` surface. SOP persists it as the
// project artifact <root>/.agent-sdlc/approvals.json (the same present-or-absent
// artifact contract as reconcile.json / plan.meta.json), and the controller
// reads it back verbatim. Every decoded field name matches SOP's schema exactly
// (task_id, kind, target, reason, evidence, stage, disposition, status,
// requested_at, task_status); nothing is renamed, inferred, or reconstructed.
//
// The listing is the single source for BOTH projections: Store.Tasks (list /
// project-hero NeedsHuman/ApprovalKind) and Store.Task (TaskDetail.Approval)
// select from the same decoded listing, so the two views can never disagree.
//
// Free-form SOP strings (reason, evidence) are DISPLAY-ONLY: they are sanitized
// and shown, but they never drive presence, applicability, or any controller
// policy. A gate is never derived from BLOCKED status alone, model/recovery
// prose, attempt counts, or inactivity.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// approvalsArtifact is the SOP-owned approval listing artifact, read from
// <root>/.agent-sdlc/approvals.json, the file SOP's `approvals --json` verb
// writes, exactly like reconcile.json and plan.meta.json. It is optional and
// present-or-absent: SOP writes it when it has computed the applicable gate
// listing, and its absence is normal and never an error. The controller reads it
// verbatim and NEVER reconstructs a gate list of its own to fill the gap.
const approvalsArtifact = "approvals.json"

// Approval status values SOP uses to mark a gate entry that is no longer
// actionable. A gate carrying any of these is never offered as applicable. They
// are SOP-reported structured values, matched verbatim; the controller does not
// invent a mapping and does not scan free-form prose.
const (
	// ApprovalStatusPending marks a gate SOP still reports as open.
	ApprovalStatusPending = "PENDING"
	// ApprovalStatusApplied / ApprovalStatusCompleted / ApprovalStatusResolved
	// mark a gate SOP reports as already answered or closed out.
	ApprovalStatusApplied   = "APPLIED"
	ApprovalStatusCompleted = "COMPLETED"
	ApprovalStatusResolved  = "RESOLVED"
	// ApprovalStatusStale marks a gate SOP reports as superseded/no longer current.
	ApprovalStatusStale = "STALE"
)

// ApprovalEntry is one SOP-reported approval gate, decoded verbatim from SOP's
// approval listing. Every field is SOP's own value; the controller neither
// creates entries nor re-ranks or filters them beyond the applicability check
// over SOP-reported structured fields.
type ApprovalEntry struct {
	// TaskID is SOP's identifier for the task the gate concerns (task_id).
	TaskID string
	// Kind is SOP's gate kind (kind), verbatim.
	Kind string
	// Target is SOP's identifier for what the gate concerns (target), verbatim.
	Target string
	// Reason is SOP's free-form explanation (reason). DISPLAY-ONLY: it never
	// drives presence or applicability.
	Reason string
	// Evidence is SOP's free-form evidence/detail (evidence). DISPLAY-ONLY: it
	// never drives presence or applicability.
	Evidence string
	// Stage is SOP's lifecycle stage for the gate (stage), verbatim.
	Stage string
	// Disposition is SOP's gate disposition (disposition), verbatim.
	Disposition string
	// Status is SOP's gate status (status), verbatim. It is the structured field
	// the applicability check reads.
	Status string
	// RequestedAt is SOP's recorded request time (requested_at), verbatim as the
	// string SOP reported.
	RequestedAt string
	// TaskStatus is SOP's task status at the time of the gate (task_status),
	// verbatim.
	TaskStatus string
	// Applicable is true when SOP reports this gate as currently actionable. It
	// is derived ONLY from SOP's structured status/disposition fields; it is never
	// derived from reason/evidence prose, BLOCKED status alone, attempt counts, or
	// inactivity.
	Applicable bool
}

// ApprovalsListing is the result of the authoritative approval read. Reported
// distinguishes "SOP reported a structured listing" from "SOP reported none": a
// caller must never render an unreported listing as an empty success. When the
// listing is unsupported/absent, Reported is false and Entries is empty; the
// controller must NOT reconstruct gates from any other source.
type ApprovalsListing struct {
	// Reported is true only when SOP reported a structured approval listing. When
	// false, Entries is empty and the caller shows an explicit "not reported by
	// SOP" state rather than an empty success.
	Reported bool
	// Source names where SOP reported the listing (the artifact filename), so a
	// reader can trace the value. Empty when Reported is false.
	Source string
	// Entries are the approval gates SOP reported, in SOP's own order.
	Entries []ApprovalEntry
}

// approvalsDoc is the persisted shape of SOP's approval listing. Only SOP-owned
// fields are decoded; the controller adds nothing. The JSON tag of every field
// matches SOP's schema name exactly.
type approvalsDoc struct {
	Approvals []approvalEntryDoc `json:"approvals"`
}

type approvalEntryDoc struct {
	TaskID      string `json:"task_id"`
	Kind        string `json:"kind"`
	Target      string `json:"target"`
	Reason      string `json:"reason"`
	Evidence    string `json:"evidence"`
	Stage       string `json:"stage"`
	Disposition string `json:"disposition"`
	Status      string `json:"status"`
	RequestedAt string `json:"requested_at"`
	TaskStatus  string `json:"task_status"`
}

// decodeApprovals parses SOP's approval-listing JSON. It returns the decoded
// listing and whether the bytes were a well-formed SOP listing. Decoding never
// derives or renames a field; a malformed document yields Reported=false so the
// caller shows an explicit absence rather than a partial gate list. Blank
// output (SOP reported nothing) is a well-formed empty listing, not an error.
func decodeApprovals(data []byte) (ApprovalsListing, bool) {
	if len(strings.TrimSpace(string(data))) == 0 {
		return ApprovalsListing{Reported: true, Source: approvalsArtifact}, true
	}
	var doc approvalsDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return ApprovalsListing{}, false
	}
	out := ApprovalsListing{Reported: true, Source: approvalsArtifact}
	for _, e := range doc.Approvals {
		entry := ApprovalEntry{
			TaskID:      strings.TrimSpace(e.TaskID),
			Kind:        strings.TrimSpace(e.Kind),
			Target:      strings.TrimSpace(e.Target),
			Reason:      sanitizeDetail(e.Reason),
			Evidence:    sanitizeDetail(e.Evidence),
			Stage:       strings.TrimSpace(e.Stage),
			Disposition: strings.TrimSpace(e.Disposition),
			Status:      strings.TrimSpace(e.Status),
			RequestedAt: strings.TrimSpace(e.RequestedAt),
			TaskStatus:  strings.TrimSpace(e.TaskStatus),
		}
		entry.Applicable = applicableStatus(entry.Status, entry.Disposition)
		out.Entries = append(out.Entries, entry)
	}
	return out, true
}

// applicableStatus is the applicability rule, stated ONLY in terms of
// SOP-reported structured fields. SOP must affirmatively report the gate as
// open; a completed, applied, resolved, or stale gate is not actionable, and a
// blank status never manufactures a gate. This reads no free-form prose and is
// independent of task status (so BLOCKED alone is never a gate), attempt counts,
// and inactivity.
func applicableStatus(status, disposition string) bool {
	if closedStatus(status) || closedStatus(disposition) {
		return false
	}
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case ApprovalStatusPending, "OPEN", "REQUESTED", "REQUESTED_APPROVAL":
		return true
	default:
		return false
	}
}

// closedStatus reports whether a SOP-reported structured value marks a gate as
// no longer actionable (completed / applied / resolved / stale). Matching is
// verbatim over SOP's structured fields only.
func closedStatus(v string) bool {
	switch strings.ToUpper(strings.TrimSpace(v)) {
	case ApprovalStatusApplied, ApprovalStatusCompleted, ApprovalStatusResolved, ApprovalStatusStale,
		"CLOSED", "DONE", "CANCELLED", "CANCELED":
		return true
	default:
		return false
	}
}

// Approvals reads SOP's authoritative approval listing for this project. It
// reads SOP's own approvals artifact read-only, through the same present-or-
// absent artifact contract as reconcile.json and plan.meta.json; the controller
// never opens SOP state storage for write and never reconstructs a gate list.
// When SOP reports no listing the result has Reported=false and no entries, so a
// caller shows an explicit absence instead of an empty success.
func (s *Store) Approvals() ApprovalsListing {
	raw, err := os.ReadFile(filepath.Join(s.root, ".agent-sdlc", approvalsArtifact))
	if err != nil {
		return ApprovalsListing{}
	}
	listing, ok := decodeApprovals(raw)
	if !ok {
		return ApprovalsListing{}
	}
	return listing
}

// Lookup returns the SOP-reported approval entry for one task, selected from the
// authoritative listing (no second data source, no per-task human-text parse).
// It returns the first applicable entry for the task and whether one exists. A
// task whose only entries are completed/stale/resolved yields ok=false, so a
// resolved gate is never offered as actionable.
func (l ApprovalsListing) Lookup(taskID string) (ApprovalEntry, bool) {
	if !l.Reported {
		return ApprovalEntry{}, false
	}
	for _, e := range l.Entries {
		if e.TaskID != taskID {
			continue
		}
		if !e.Applicable {
			continue
		}
		return e, true
	}
	return ApprovalEntry{}, false
}

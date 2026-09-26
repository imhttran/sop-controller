package sopclient

import (
	"strings"
	"unicode"
)

// This file holds the single safe-summary point for the activity model. SOP
// persists each activity event with a short, human-oriented Detail string; the
// controller re-sanitizes it here so every consumer (the CLI and the web/controller
// layer) shares one guarantee: prompts, secrets, API keys, environment dumps, and
// unrestricted command/file contents never reach a caller through ActivityEvent.
//
// The sanitization is applied at the read boundary (Store.activity), so no
// consumer can accidentally bypass it by reading activity.jsonl itself.

// maxDetailLen bounds a safe Detail summary so an over-long free-form string
// (for example a dump SOP mis-recorded) cannot be surfaced verbatim.
const maxDetailLen = 200

// redacted replaces any span suppressed by sanitizeDetail.
const redacted = "[redacted]"

// secretMarkers are substrings that indicate a value is a credential rather than
// a safe summary. Matching is case-insensitive.
var secretMarkers = []string{
	"api_key", "apikey", "api-key",
	"secret", "token", "password", "passwd",
	"authorization", "bearer ", "private_key", "privatekey",
	"-----begin", "access_key", "accesskey",
}

// envDumpMarkers indicate the detail is an environment dump rather than a
// summary. Matching is case-insensitive.
var envDumpMarkers = []string{
	"environ", "env=", "env var", "envvar",
	"os.environ", "$env:", "printenv", "getenv",
}

// promptMarkers indicate the detail carries prompt/model text rather than a
// safe action summary. Matching is case-insensitive.
var promptMarkers = []string{
	"prompt:", "system prompt", "user prompt", "messages=",
	"role: system", "role: user", "role: assistant",
}

// sanitizeDetail returns a safe, bounded summary of a raw activity Detail.
//
// Policy:
//   - A summary that looks like a secret/credential, an environment dump, or raw
//     prompt text is replaced wholesale with a redaction marker; such content is
//     never partially surfaced.
//   - A long, free-form value (unrestricted command output or file content) is
//     truncated, because a safe summary is short by construction.
//   - Otherwise the value is returned unchanged: the reader reports what SOP
//     recorded, it does not invent a summary.
//
// It is a pure function so the guarantee is identical on every read path.
func sanitizeDetail(detail string) string {
	s := strings.TrimSpace(detail)
	if s == "" {
		return ""
	}
	lower := strings.ToLower(s)
	for _, marker := range secretMarkers {
		if strings.Contains(lower, marker) {
			return redacted
		}
	}
	for _, marker := range envDumpMarkers {
		if strings.Contains(lower, marker) {
			return redacted
		}
	}
	for _, marker := range promptMarkers {
		if strings.Contains(lower, marker) {
			return redacted
		}
	}
	// A detail that looks like a multi-line blob is an unrestricted dump (full
	// file contents or command output), not a one-line summary.
	if strings.ContainsAny(s, "\n\r") {
		return redacted
	}
	if len(s) > maxDetailLen {
		s = truncate(s, maxDetailLen) + "..."
	}
	return s
}

// truncate returns the first n runes of s, never splitting a UTF-8 rune.
func truncate(s string, n int) string {
	count := 0
	for i := range s {
		if count == n {
			return s[:i]
		}
		count++
	}
	return s
}

// isSafeDetail reports whether a raw detail survives sanitization unchanged. It
// is used only by tests.
func isSafeDetail(detail string) bool {
	if strings.TrimSpace(detail) == "" {
		return true
	}
	return sanitizeDetail(detail) == strings.TrimSpace(detail) && !strings.ContainsRune(strings.Map(dropNonPrint, detail), unicode.ReplacementChar)
}

func dropNonPrint(r rune) rune {
	if unicode.IsPrint(r) || r == '\n' || r == '\r' || r == '\t' {
		return r
	}
	return unicode.ReplacementChar
}

package web

import (
	"html/template"
	"io"
	"io/fs"
	"strconv"
	"strings"
	"time"

	"sop-controller/internal/sopclient"
)

// Views renders the server-side HTML templates.
type Views struct {
	t *template.Template
}

// NewViews parses templates/*.html and templates/partials/*.html from fsys.
func NewViews(fsys fs.FS) (*Views, error) {
	funcs := template.FuncMap{
		"statusClass":            statusClass,
		"severityClass":          severityClass,
		"stageClass":             stageClass,
		"dispositionClass":       dispositionClass,
		"categoryClass":          categoryClass,
		"categoryLabel":          categoryLabel,
		"classificationCategory": classificationCategory,
		"levelClass":             levelClass,
		"activeTask":             activeTask,
		"since":                  since,
		"evidenceState":          evidenceState,
		"unavailable":            unavailable,
		"absentLiteral":          absentLiteral,
		"hasRetries":             hasRetries,
		"retryCount":             retryCount,
		"blocked":                blocked,
		"hasProviderModel":       hasProviderModel,
		"startedAt":              startedAt,
		"elapsed":                elapsed,
	}
	t, err := template.New("root").Funcs(funcs).ParseFS(fsys, "templates/*.html", "templates/partials/*.html")
	if err != nil {
		return nil, err
	}
	return &Views{t: t}, nil
}

// Render executes the named template.
func (v *Views) Render(w io.Writer, name string, data any) error {
	return v.t.ExecuteTemplate(w, name, data)
}

// statusClass maps SOP's task status to a badge style. It only styles the status
// SOP already chose; it never classifies the workflow itself.
func statusClass(status string) string {
	switch sopclient.TaskState(status) {
	case "DONE":
		return "s-completed"
	case "BLOCKED":
		return "s-blocked"
	case "FAILED":
		return "s-failed"
	case "READY":
		return "s-ready"
	case "PLANNED":
		return "s-planned"
	default:
		return "s-running"
	}
}

// stageClass maps SOP's run stage to a badge style.
func stageClass(stage string) string {
	switch strings.ToUpper(stage) {
	case sopclient.StagePassed, "PASS":
		return "s-completed"
	case sopclient.StageFailed, "FAIL":
		return "s-failed"
	case sopclient.StageWaitingForHuman:
		return "s-blocked"
	case "":
		return "s-planned"
	default:
		return "s-running"
	}
}

// dispositionClass maps SOP's failure disposition to a badge style. Each
// disposition gets a distinct treatment so a human can tell at a glance whether
// SOP will recover on its own (AUTO_FIX/CONTINUE/RETRY) or needs them.
func dispositionClass(disposition string) string {
	switch disposition {
	case sopclient.DispositionAutoFix:
		return "d-autofix"
	case sopclient.DispositionContinue:
		return "d-continue"
	case sopclient.DispositionRetry:
		return "d-retry"
	case sopclient.DispositionReplan:
		return "d-replan"
	case sopclient.DispositionNeedsHuman:
		return "d-human"
	default:
		return "s-planned"
	}
}

// classificationCategory derives the display-only category for a classification
// from the Kind SOP persisted (CTRL009). It delegates entirely to sopclient so
// the controller never reclassifies the failure; a nil or unrecognized Kind
// yields CategoryUnknown.
func classificationCategory(c *sopclient.Classification) sopclient.Category {
	return c.Category()
}

// categoryLabel is the display label for a classification category. An unknown
// category renders explicitly as "Unknown" so it can never read as a pass, a
// provider failure, or a human boundary. It delegates to sopclient so the label
// vocabulary cannot drift from the category contract.
func categoryLabel(c sopclient.Category) string {
	return c.Label()
}

// categoryClass maps a classification category to a badge style. Provider and
// deterministic/validation categories are visually distinct, budget is its own
// treatment (never the human style), and an unknown category is neutral so it
// can never read as PASS, provider, or human.
func categoryClass(c sopclient.Category) string {
	switch c {
	case sopclient.CategoryProvider:
		return "c-provider"
	case sopclient.CategoryDeterministic:
		return "c-deterministic"
	case sopclient.CategoryBudget:
		return "c-budget"
	case sopclient.CategoryHuman:
		return "c-human"
	default:
		return "c-unknown"
	}
}

// levelClass maps an aggregate status level (PASS/FAIL/SKIP/NOT_RUN/UNKNOWN,
// and the JEV statuses) to a badge style. Only an explicit PASS reads as
// complete; everything else, including the explicit-absence UNKNOWN, is neutral
// or failing, so a missing artifact never looks like a pass.
func levelClass(level string) string {
	switch strings.ToUpper(level) {
	case "PASS":
		return "s-completed"
	case "FAIL", "ERROR":
		return "s-failed"
	default:
		return "s-planned"
	}
}

// notRunLabel is the wording used for data SOP did not persist in the CTRL008
// detail sections. It is deliberately not "PASS", "OK", or any success word.
const notRunLabel = "not run"

// unavailable is the CTRL008 template affordance for a field SOP persisted no
// value for. It renders the explicit not-run literal so a missing field reads as
// absent rather than as success, and it is never styled as PASS.
func unavailable() string { return notRunLabel }

// absentLiteral is the explicit absence value used by the aggregate status
// panel: it is SOP's StatusUnknown ("UNKNOWN"), distinct from every real status
// so a missing artifact never reads as a pass. It exists as a template func so
// the template does not embed the literal and cannot drift from the sopclient
// constant or render a success word.
func absentLiteral() string { return sopclient.StatusUnknown }

// evidenceState renders the CTRL008 missing-data affordance: a status-like value
// that SOP did not persist is shown as the explicit "not run"/"unavailable"
// literal with a neutral (never PASS-styled) badge, so absent data can never read
// as success. A value SOP did persist is returned verbatim. It never invents a
// value and never styles absence as complete.
func evidenceState(present bool, value string) string {
	if !present || strings.TrimSpace(value) == "" {
		return notRunLabel
	}
	return value
}

// activeTask returns the id of the project's active task, or "" - a
// template-friendly form of ProjectDetail.ActiveTask (Go templates cannot take a
// two-value return in an action).
func activeTask(p sopclient.ProjectDetail) string {
	id, _ := p.ActiveTask()
	return id
}

// hasRetries reports whether SOP recorded any attempt evidence for the task, so
// a genuine zero retries is distinguishable from an absence. It is the
// template-friendly form of TaskSummary.Retries (Go templates cannot take a
// two-value return in an action).
func hasRetries(t sopclient.TaskDetail) bool {
	_, ok := t.Retries()
	return ok
}

// retryCount returns how many attempts SOP has already spent (attempt-1), or 0
// when SOP recorded no attempt yet. Pair it with hasRetries to render absence
// explicitly rather than as a zero.
func retryCount(t sopclient.TaskDetail) int {
	n, _ := t.Retries()
	return n
}

// blocked reports whether SOP reports a blocked task: an explicit blocked
// status, or dependencies not yet completed. It reads SOP's own fields and never
// infers a block the workflow did not record.
func blocked(t sopclient.TaskDetail) bool {
	return t.Status == sopclient.StatusBlocked || len(t.BlockedBy) > 0
}

// hasProviderModel reports whether SOP persisted any provider/model detail for
// the run, so the template can render the pair only when at least one half is
// present instead of a dangling separator.
func hasProviderModel(r sopclient.RunInfo) bool {
	return strings.TrimSpace(r.Provider) != "" || strings.TrimSpace(r.Model) != ""
}

// startedAt returns SOP's earliest recorded attempt timestamp for the task, or
// the zero time when SOP persisted no attempt evidence. CTRL008 presents
// Started/elapsed only from data SOP actually persisted (attempt timestamps); it
// never invents or infers a start time.
func startedAt(t sopclient.TaskDetail) time.Time {
	var earliest time.Time
	for _, a := range t.Attempts {
		if a.Timestamp.IsZero() {
			continue
		}
		if earliest.IsZero() || a.Timestamp.Before(earliest) {
			earliest = a.Timestamp
		}
	}
	return earliest
}

// elapsed renders the time since the task's earliest recorded attempt as a
// human-readable duration, or the explicit not-run literal when SOP persisted no
// timestamp. It never fabricates a duration for absent data.
func elapsed(t sopclient.TaskDetail) string {
	start := startedAt(t)
	if start.IsZero() {
		return notRunLabel
	}
	d := time.Since(start)
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Minute:
		return "< 1m"
	case d < time.Hour:
		return strconv.Itoa(int(d.Minutes())) + "m"
	case d < 24*time.Hour:
		return strconv.Itoa(int(d.Hours())) + "h"
	default:
		return strconv.Itoa(int(d.Hours()/24)) + "d"
	}
}

func severityClass(sev string) string {
	switch strings.ToLower(sev) {
	case "critical":
		return "s-failed"
	case "high":
		return "s-blocked"
	case "medium":
		return "s-waiting"
	case "low":
		return "s-ready"
	default:
		return "s-planned"
	}
}

func since(t time.Time) string {
	if t.IsZero() {
		return "\u2014"
	}
	d := time.Since(t)
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return strconv.Itoa(int(d.Minutes())) + "m ago"
	case d < 24*time.Hour:
		return strconv.Itoa(int(d.Hours())) + "h ago"
	default:
		return strconv.Itoa(int(d.Hours()/24)) + "d ago"
	}
}

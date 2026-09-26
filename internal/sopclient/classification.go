package sopclient

import "strings"

// SOP owns failure classification (agentic-sop internal/failure.Classification);
// the controller only reads and displays it. This file is a DISPLAY-ONLY map
// from the Kind SOP persisted to a coarse presentation category, so the UI can
// distinguish provider failures from code/validation failures and from
// tool/iteration-budget exhaustion without the controller reclassifying the
// failure or transforming SOP's disposition.
//
// Boundary: the controller does not classify failures. It keys the category off
// the Kind string SOP already wrote (report.json classification /
// classification.json) and never infers a category from the disposition or from
// the stage. The authoritative Kind vocabulary is owned by agentic-sop and is
// not fully enumerated in this repository; only the kinds observed in SOP's
// artifacts/tests are named here. Any Kind this map does not recognize yields
// CategoryUnknown - never Provider, Human, or Deterministic - so an unrecognized
// kind stays unknown rather than being guessed.

// Category is a display-only grouping of SOP's classification Kind. It is a
// presentation hint for styling only; it is never SOP's verdict and never
// changes SOP state.
type Category string

const (
	// CategoryProvider: an infrastructure/model-provider failure (the model,
	// transport, or provider call failed rather than the code under test).
	CategoryProvider Category = "PROVIDER"
	// CategoryDeterministic: a deterministic code/validation failure that will
	// reproduce until the code changes (tests, validation, build, review).
	CategoryDeterministic Category = "DETERMINISTIC"
	// CategoryBudget: a tool/iteration/retry budget or resource exhaustion. It
	// is explicitly NOT a human boundary on its own.
	CategoryBudget Category = "BUDGET"
	// CategoryHuman: SOP itself classified the failure as needing a human
	// (an ambiguous contract or an explicit human boundary).
	CategoryHuman Category = "HUMAN"
	// CategoryUnknown: the Kind is absent or not one this controller recognizes.
	// It is never inferred to be provider, deterministic, budget, or human.
	CategoryUnknown Category = "UNKNOWN"
)

// Kind values observed in SOP's persisted classification artifacts and the
// controller's fixtures. These document the kinds the controller recognizes for
// display; the authoritative set is owned by agentic-sop and may grow. An
// unrecognized kind maps to CategoryUnknown.
const (
	// Provider / transient infrastructure kinds.
	KindTransientProvider = "TRANSIENT_PROVIDER"
	KindProviderError     = "PROVIDER_ERROR"
	KindModelError        = "MODEL_ERROR"

	// Deterministic code/validation kinds.
	KindTestFailure              = "TEST_FAILURE"
	KindValidationFailure        = "VALIDATION_FAILURE"
	KindBuildFailure             = "BUILD_FAILURE"
	KindIncompleteImplementation = "INCOMPLETE_IMPLEMENTATION"

	// Tool/iteration-budget exhaustion kinds.
	KindToolBudgetExhausted      = "TOOL_BUDGET_EXHAUSTED"
	KindIterationBudgetExhausted = "ITERATION_BUDGET_EXHAUSTED"
	KindRetryBudgetExhausted     = "RETRY_BUDGET_EXHAUSTED"

	// Human-boundary kinds.
	KindAmbiguousContract = "AMBIGUOUS_CONTRACT"
	KindHumanRequired     = "HUMAN_REQUIRED"

	// KindReplanRequired is a non-human failure kind seen in fixtures.
	KindReplanRequired = "REPLAN_REQUIRED"
)

// CategoryOf maps an SOP classification Kind to a display-only Category. It is a
// pure mapping of the Kind string SOP persisted; it does not read or change the
// disposition and does not classify the failure. An empty or unrecognized Kind
// yields CategoryUnknown.
func CategoryOf(kind string) Category {
	switch strings.ToUpper(strings.TrimSpace(kind)) {
	case KindTransientProvider, KindProviderError, KindModelError:
		return CategoryProvider
	case KindTestFailure, KindValidationFailure, KindBuildFailure, KindIncompleteImplementation:
		return CategoryDeterministic
	case KindToolBudgetExhausted, KindIterationBudgetExhausted, KindRetryBudgetExhausted:
		return CategoryBudget
	case KindAmbiguousContract, KindHumanRequired:
		return CategoryHuman
	default:
		return CategoryUnknown
	}
}

// Category returns the display-only category for this classification from the
// Kind SOP persisted. A nil classification is unknown. It never inspects the
// disposition: a deterministic kind is never labeled provider, and a
// budget-exhaustion kind is never labeled human, regardless of disposition.
func (c *Classification) Category() Category {
	if c == nil {
		return CategoryUnknown
	}
	return CategoryOf(c.Kind)
}

// Label is the human-readable label for a Category, used by the templates. An
// unknown category is rendered explicitly as "Unknown" so it can never read as a
// pass, a provider failure, or a human boundary.
func (c Category) Label() string {
	switch c {
	case CategoryProvider:
		return "Provider"
	case CategoryDeterministic:
		return "Code or validation"
	case CategoryBudget:
		return "Budget exhausted"
	case CategoryHuman:
		return "Human"
	default:
		return "Unknown"
	}
}

# Plan Identity Verification Implementation

> **Historical / non-normative.** Point-in-time artifact kept for traceability only; it does not define current behavior. See the [documentation index](../README.md).

## Overview

This document describes the implementation of plan identity verification to prevent accidental reuse of wrong machine plans or task graphs in the sop-controller. The solution ensures that named plans cannot accidentally be resumed with stale source content, and that different plans are isolated from one another.

## Architecture

### Core Components

#### 1. **PlanMetadata** (`metadata.go`)
Persists plan identity, fingerprint, and execution history.

- **PlanIdentity**: Stable, unique identifier derived from plan filename (e.g., `PLAN-Wrap-Up`)
- **SourcePath**: Relative or absolute path to the plan file
- **SourceFingerprint**: SHA256 hash of the plan source at execution time
- **LastExecutedAt**: Timestamp of last execution
- **TaskHistory**: List of completed task IDs

Storage: `.agent-sdlc/plan-metadata/{planID}.json`

#### 2. **PlanContext** (`plan_context.go`)
Tracks current execution context for plan isolation.

- **PlanIdentity**: Currently executing plan's ID
- **PlanPath**: Path to the plan being executed
- **ExecutionID**: Unique ID for this execution session
- **StartedAt**: Execution start timestamp

Storage: `.agent-sdlc/current-plan-context.json`

#### 3. **Store Extensions** (`store.go`)
Extended Store type with plan metadata and context stores.

Methods:
- `ValidateResumeForPlan(planPath)`: Checks if resumption is safe (isolation, staleness)
- `RecordPlanExecution(planPath, executionID)`: Records execution metadata
- `ClearPlanContext()`: Removes execution context (cleanup on completion)

#### 4. **Service Methods** (`service.go`)
Public Client API for plan operations.

Methods:
- `ValidatePlanResumption()`: Validate before resuming
- `RecordPlanExecution()`: Record execution start
- `GetPlanTaskHistory()`: Retrieve completed tasks
- `ClearPlanContext()`: Cleanup on completion

#### 5. **ResumptionValidation** (`types.go`)
Structured result of resumption validation.

Fields:
- `CanResume`: Whether execution can proceed
- `PlanIdentity`: Stable plan ID
- `IsFirstRun`: True for first execution
- `IsStale`: True if source has changed
- `CompletedTasks`: List of previously completed tasks
- `Error`: Human-readable error message

## Acceptance Criteria Implementation

### AC-1: Re-running unchanged PLAN-Wrap-Up.md resumes the same execution

**Implementation**: `ValidateResumeForPlan()` checks:
1. Plan isolation (no other plan currently executing)
2. Metadata existence (first run = can proceed)
3. Fingerprint match (unchanged = not stale)
4. Returns completed task list for resumption

**Test**: `TestAcceptanceCriteria_UnchangedPlanResumes`

### AC-2: Editing the plan makes the machine representation stale

**Implementation**: `IsMetadataStale()` compares current SHA256 fingerprint with stored fingerprint.

**Test**: `TestAcceptanceCriteria_EditedPlanIsStale`

### AC-3: Running another named plan cannot silently reuse this plan's task graph

**Implementation**: `PlanContextStore.ValidatePlanIdentity()` maintains single active plan identity in `.agent-sdlc/current-plan-context.json`. Attempting to execute a different plan raises an error.

**Test**: `TestAcceptanceCriteria_IsolationPreventsReuse`

### AC-4: Completed task history is not silently destroyed when reconciliation requires human intervention

**Implementation**: Task history is stored independently from plan state. Even if a plan is detected as stale, history remains accessible via `GetTaskHistory()` for manual intervention.

**Test**: `TestAcceptanceCriteria_HistoryNotDestroyed`

### AC-5: Persisted metadata identifies at minimum source path, source content fingerprint, and plan identity

**Implementation**: `PlanMetadata` struct enforces these three required fields. JSON schema includes all three. Validation ensures all fields are populated before persistence.

**Test**: `TestAcceptanceCriteria_PersistedMetadata`

## Data Flow

### First Execution
```
1. Load plan file
2. ValidateResumeForPlan() → IsFirstRun=true, CanResume=true
3. RecordPlanExecution()
   - Compute SHA256 fingerprint
   - Save PlanContext (isolation)
   - Save PlanMetadata (identity + fingerprint)
4. Execute plan
5. RecordTaskCompletion() for each completed task
6. ClearPlanContext() on completion
```

### Resume Unchanged Plan
```
1. Load plan file
2. ValidateResumeForPlan()
   - Check isolation (no other plan active)
   - Load PlanMetadata
   - Compute SHA256, compare with stored fingerprint → not stale
   - Return CompletedTasks for resumption
3. Execute remaining tasks
4. RecordTaskCompletion() for new completions
5. ClearPlanContext() on completion
```

### Stale Plan Detection
```
1. Load plan file
2. ValidateResumeForPlan()
   - Check isolation
   - Load PlanMetadata
   - Compute SHA256, compare → stale!
   - Return IsStale=true, Error message
3. Require human intervention to proceed
4. History still accessible for recovery
```

### Plan Isolation
```
1. Plan A starts: SaveCurrentContext(PlanContext{PlanIdentity: "PLAN-A"})
2. Attempt Plan B: ValidatePlanIdentity("PLAN-B") → Error
3. Plan A completes: ClearCurrentContext()
4. Plan B can now start
```

## Storage Layout

```
.agent-sdlc/
├── state.db                           # SOP's authoritative task state
├── current-plan-context.json          # Active plan identity (singleton)
└── plan-metadata/
    ├── PLAN-Wrap-Up.json              # Metadata for PLAN-Wrap-Up
    ├── PLAN-Hardening.json            # Metadata for PLAN-Hardening
    └── ...
```

### PlanMetadata JSON Example
```json
{
  "plan_identity": "PLAN-Wrap-Up",
  "source_path": "docs/plans/PLAN-Wrap-Up.md",
  "source_fingerprint": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
  "last_executed_at": "2026-09-27T19:58:08Z",
  "task_history": ["WRAP-001", "WRAP-002", "WRAP-003"]
}
```

### PlanContext JSON Example
```json
{
  "plan_identity": "PLAN-Wrap-Up",
  "plan_path": "docs/plans/PLAN-Wrap-Up.md",
  "execution_id": "exec-1234567890",
  "started_at": "2026-09-27T19:58:08Z"
}
```

## Test Coverage

### Unit Tests (`plan_identity_test.go`)
- `TestPlanMetadataRoundTrip`: Metadata serialization
- `TestFingerprintDeterministic`: SHA256 consistency
- `TestFingerprintChanges`: Detection of modifications
- `TestStaleDetection`: Stale state identification
- `TestPlanContextIsolation`: Isolation enforcement
- `TestTaskHistoryPreservation`: History persistence
- `TestNormalizePlanID`: Plan ID normalization
- `TestResumptionValidationFirstRun`: First run validation
- `TestResumptionValidationStaleDetection`: Stale detection in validation
- `TestTaskHistoryNotDestroyedOnStale`: History survives stale state

### Integration Tests (`plan_identity_integration_test.go`)
- `TestAcceptanceCriteria_UnchangedPlanResumes`: AC-1
- `TestAcceptanceCriteria_EditedPlanIsStale`: AC-2
- `TestAcceptanceCriteria_IsolationPreventsReuse`: AC-3
- `TestAcceptanceCriteria_HistoryNotDestroyed`: AC-4
- `TestAcceptanceCriteria_PersistedMetadata`: AC-5
- `TestCompleteWorkflow`: End-to-end lifecycle

**Coverage**: 18 tests, all passing ✓

## Integration with SOP Controller

### Service Layer (`service.go`)
The Client type exposes methods for the web layer to call:

```go
// Validate before running a plan
validation, err := client.ValidatePlanResumption(projectID, planPath)
if !validation.CanResume {
    return fmt.Errorf("cannot resume: %s", validation.Error)
}

// Record when plan execution starts
err = client.RecordPlanExecution(projectID, planPath, executionID)

// Get completed tasks for resumption
history, err := client.GetPlanTaskHistory(projectID, planPath)

// Cleanup when plan completes
err = client.ClearPlanContext(projectID)
```

### Web Layer Integration Points
The web handlers (in `internal/web/`) can call these methods:

1. **Before Run/Resume**: Validate plan identity and staleness
2. **After Run Start**: Record execution metadata
3. **During Execution**: Track task completions
4. **After Plan Completes**: Clear context for next plan

## Design Decisions

### Plan ID Derivation
Plan identity is derived from filename (normalized) rather than random UUID:
- ✓ Deterministic across runs
- ✓ Human-readable in filesystem
- ✓ Matches plan naming conventions (PLAN-*.md)
- ✓ No UUID mapping table needed

### Metadata Storage Format
JSON files in `.agent-sdlc/plan-metadata/` rather than SQLite:
- ✓ Easy to inspect/debug
- ✓ Separate from SOP's state.db (authority boundary)
- ✓ Simple structure, no schema evolution needed
- ✓ Atomic per-plan updates

### Fingerprinting Algorithm
SHA256 hash of complete file contents:
- ✓ Deterministic (identical content = identical hash)
- ✓ Sensitive to any change (single byte change detected)
- ✓ Standard, proven algorithm
- ✓ No custom logic to maintain

### Single Active Plan Context
Only one plan can be active at a time (context stored in `current-plan-context.json`):
- ✓ Prevents concurrent execution confusion
- ✓ Clear error message if violated
- ✓ Simple enforcement mechanism
- ✓ Matches typical SOP workflow (one plan at a time)

## Future Enhancements

1. **Plan Versioning**: Track multiple versions of the same plan
2. **Diff on Stale**: Show what changed in a stale plan
3. **Auto-Recovery**: Automatic recomputation of fingerprint after human review
4. **Multi-Plan Support**: Allow concurrent execution of different plans (requires execution ID tracking in tasks)
5. **Metadata Cleanup**: Automatic pruning of old metadata for completed plans
6. **Plan Dependency Tracking**: Prevent plan dependencies from being silently reordered

## Troubleshooting

### "Plan source has changed; cannot resume from stale state"
The plan file has been modified since last execution. Options:
1. Review changes with `git diff docs/plans/PLAN-Wrap-Up.md`
2. Revert to last committed version if changes were unintended
3. Delete `.agent-sdlc/plan-metadata/PLAN-Wrap-Up.json` to reset (loses history)

### "plan isolation violation: current plan is X, but Y was requested"
Another plan is still executing. Options:
1. Wait for first plan to complete
2. Check task status in dashboard
3. Manually delete `.agent-sdlc/current-plan-context.json` if first plan crashed

### Task history lost
Metadata files may be corrupted. Check:
1. `.agent-sdlc/plan-metadata/{planID}.json` exists and is valid JSON
2. `task_history` array contains expected task IDs
3. Restore from backup if available

---

**Last Updated**: 2026-09-27
**Status**: Implemented and tested
**Coverage**: 5/5 acceptance criteria verified

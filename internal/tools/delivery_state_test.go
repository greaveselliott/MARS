/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/convergence-state-machine.md
- docs/design-docs/tools-glossary.md
- docs/features/F-005-agent-execution-runtime.md
*/
package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEngineerDeliveryState_claimedByDefault(t *testing.T) {
	t.Parallel()
	state := Session{Role: "engineer"}.EngineerDeliveryState()
	assert.Equal(t, DeliveryPhaseClaimed, state.Phase)
	assert.Equal(t, RepairLaneNone, state.RepairLane)
}

func TestEngineerDeliveryState_implementingAfterProductWrite(t *testing.T) {
	t.Parallel()
	state := Session{
		Role:       "engineer",
		ToolCounts: map[string]int{"file_write": 2},
	}.EngineerDeliveryState()
	assert.Equal(t, DeliveryPhaseImplementing, state.Phase)
}

func TestEngineerDeliveryState_validatingAfterValidationCommand(t *testing.T) {
	t.Parallel()
	state := Session{
		Role: "engineer",
		ToolState: map[string]string{
			testBuildValidationCommandKey: `shell_exec {"argv":["go","test","./..."]}`,
		},
	}.EngineerDeliveryState()
	assert.Equal(t, DeliveryPhaseValidating, state.Phase)
}

func TestEngineerDeliveryState_validationFailedTestBuildLane(t *testing.T) {
	t.Parallel()
	state := Session{
		Role: "engineer",
		ToolState: map[string]string{
			testBuildValidationCommandKey: `shell_exec {"argv":["go","test","./..."]}`,
		},
		ToolCounts: map[string]int{testBuildValidationOutstandingKey: 1},
	}.EngineerDeliveryState()
	assert.Equal(t, DeliveryPhaseValidationFailed, state.Phase)
	assert.Equal(t, RepairLaneTestBuild, state.RepairLane)
}

func TestEngineerDeliveryState_validationFailedRuntimeLane(t *testing.T) {
	t.Parallel()
	state := Session{
		Role:       "engineer",
		ToolCounts: map[string]int{unexpectedRuntimeValidationOutstandingKey: 1},
	}.EngineerDeliveryState()
	assert.Equal(t, DeliveryPhaseValidationFailed, state.Phase)
	assert.Equal(t, RepairLaneRuntime, state.RepairLane)
}

func TestEngineerDeliveryState_validatedAfterSuccessfulValidation(t *testing.T) {
	t.Parallel()
	state := Session{
		Role: "engineer",
		ToolState: map[string]string{
			testBuildValidationCommandKey: `shell_exec {"argv":["go","test","./..."]}`,
		},
		ToolCounts: map[string]int{validationCommandSuccessKey: 1},
	}.EngineerDeliveryState()
	assert.Equal(t, DeliveryPhaseValidated, state.Phase)
}

func TestEngineerInValidatedPhase(t *testing.T) {
	t.Parallel()
	assert.False(t, engineerInValidatedPhase(Session{Role: "engineer"}))
	assert.True(t, engineerInValidatedPhase(Session{
		Role:       "engineer",
		ToolCounts: map[string]int{validationCommandSuccessKey: 1},
	}))
	assert.True(t, engineerInValidatedPhase(Session{
		Role: "engineer",
		ToolCounts: map[string]int{
			validationCommandSuccessKey: 1,
			"tool:git_commit:success":   1,
		},
	}))
}

func TestEngineerDeliveryState_postValidationPhases(t *testing.T) {
	t.Parallel()
	state := Session{
		Role: "engineer",
		ToolCounts: map[string]int{
			validationCommandSuccessKey: 1,
			"tool:git_commit:success":   1,
		},
	}.EngineerDeliveryState()
	assert.Equal(t, DeliveryPhaseCommitting, state.Phase)

	state = Session{
		Role: "engineer",
		ToolCounts: map[string]int{
			validationCommandSuccessKey: 1,
			"tool:git_commit:success":   1,
			"tool:file_write:success":   1,
		},
	}.EngineerDeliveryState()
	assert.Equal(t, DeliveryPhaseEvidenceRecording, state.Phase)

	state = Session{
		Role: "engineer",
		ToolCounts: map[string]int{
			validationCommandSuccessKey: 1,
			ticketDoneMoveSuccessKey:    1,
		},
	}.EngineerDeliveryState()
	assert.Equal(t, DeliveryPhaseClosing, state.Phase)
}

func TestReviewDeliveryState_phases(t *testing.T) {
	t.Parallel()
	state := Session{Role: "qa"}.ReviewDeliveryState()
	assert.Equal(t, DeliveryPhaseClaimed, state.Phase)

	state = Session{
		Role:       "qa",
		ToolCounts: map[string]int{buildCommandSuccessKey: 1},
	}.ReviewDeliveryState()
	assert.Equal(t, DeliveryPhaseValidating, state.Phase)

	state = Session{
		Role:       "qa",
		ToolCounts: map[string]int{validationCommandSuccessKey: 1},
	}.ReviewDeliveryState()
	assert.Equal(t, DeliveryPhaseValidated, state.Phase)

	state = Session{
		Role:       "security",
		ToolCounts: map[string]int{reviewTerminalDispositionRequiredKey: 1},
	}.ReviewDeliveryState()
	assert.Equal(t, DeliveryPhaseTerminalDisposition, state.Phase)
}

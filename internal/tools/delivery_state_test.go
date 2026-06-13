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

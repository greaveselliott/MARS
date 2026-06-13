/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/convergence-state-machine.md
- docs/design-docs/tools-glossary.md
- docs/features/F-005-agent-execution-runtime.md
*/
package tools

import "strings"

// DeliveryPhase names an engineer delivery lifecycle state (AD-286).
type DeliveryPhase string

const (
	DeliveryPhaseClaimed             DeliveryPhase = "claimed"
	DeliveryPhaseImplementing        DeliveryPhase = "implementing"
	DeliveryPhaseValidating          DeliveryPhase = "validating"
	DeliveryPhaseValidationFailed    DeliveryPhase = "validation-failed"
	DeliveryPhaseValidated           DeliveryPhase = "validated"
	DeliveryPhaseCommitting          DeliveryPhase = "committing"
	DeliveryPhaseEvidenceRecording   DeliveryPhase = "evidence-recording"
	DeliveryPhaseClosing             DeliveryPhase = "closing"
	DeliveryPhaseTerminalDisposition DeliveryPhase = "terminal-disposition"
)

// RepairLane names the active validation-failed repair lane when applicable.
type RepairLane string

const (
	RepairLaneNone      RepairLane = "none"
	RepairLaneTestBuild RepairLane = "test-build"
	RepairLaneRuntime   RepairLane = "runtime"
)

// DeliveryState is the explicit convergence state machine position for the
// current job, derived from accumulated session evidence (AD-286 / T-029).
type DeliveryState struct {
	Phase           DeliveryPhase
	RepairLane      RepairLane
	ClaimedTicketID string
}

// EngineerDeliveryState derives the engineer machine position from the same
// session evidence the point rules already consult. It adds no second source
// of truth; policy rules migrate to this helper one cluster at a time.
func (s Session) EngineerDeliveryState() DeliveryState {
	state := DeliveryState{
		Phase:      DeliveryPhaseClaimed,
		RepairLane: RepairLaneNone,
	}
	if id := engineerCompletedTicketID(s); id != "" {
		state.ClaimedTicketID = id
	}

	if engineerOutstandingTestBuildValidationFailures(s) > 0 {
		state.Phase = DeliveryPhaseValidationFailed
		state.RepairLane = RepairLaneTestBuild
		return state
	}
	if engineerOutstandingRuntimeValidationFailures(s) > 0 {
		state.Phase = DeliveryPhaseValidationFailed
		state.RepairLane = RepairLaneRuntime
		return state
	}

	if s.ToolCounts != nil && s.ToolCounts[validationCommandSuccessKey] > 0 {
		state.Phase = DeliveryPhaseValidated
		return state
	}

	if s.ToolState != nil {
		if strings.TrimSpace(s.ToolState[testBuildValidationCommandKey]) != "" ||
			strings.TrimSpace(s.ToolState[unexpectedRuntimeValidationCommandKey]) != "" {
			state.Phase = DeliveryPhaseValidating
			return state
		}
	}

	if s.ToolCounts != nil {
		if n := s.ToolCounts["file_write"]; n > 0 {
			state.Phase = DeliveryPhaseImplementing
		}
	}
	return state
}

func engineerInValidatedPhase(session Session) bool {
	state := session.EngineerDeliveryState()
	return state.Phase == DeliveryPhaseValidated
}

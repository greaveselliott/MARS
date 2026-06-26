/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/guardrails.md
- docs/design-docs/tools-glossary.md
- docs/features/F-005-agent-execution-runtime.md
- docs/features/F-007-guardrails-and-safety.md
*/
package tools

import (
	"testing"
)

func TestCapabilityCoverageTreatsPiecesAsTetrominoesForLocking(t *testing.T) {
	t.Parallel()
	surface := `
## Scenario Schedule

1. F-001-S006 - Tetrominoes Lock Into Stack On Contact

### F-001-S006: Tetrominoes Lock Into Stack On Contact

Given a tetromino is falling
When it makes contact with the bottom or another piece
Then the tetromino locks into the stack and a new tetromino begins
`
	if !capabilityPhraseCovered(surface, "lock pieces into the stack") {
		t.Fatal("expected tetromino locking language to cover piece-locking capability")
	}
}

func TestCapabilityMatchingIgnoresIncludingAndDetectionGlue(t *testing.T) {
	t.Parallel()

	if !capabilityPhraseCovered("visible board grid is displayed", "core product including visible grid") {
		t.Fatal("expected including/core product glue not to block visible grid coverage")
	}
	if !capabilityPhraseCovered("game over is detected when stack fills the playfield", "game over detection") {
		t.Fatal("expected detection glue not to block game-over coverage")
	}
	if !capabilityPhraseCovered("game over is detected when stack fills playfield", "show game over when the stack fills") {
		t.Fatal("expected show/display glue not to block game-over coverage")
	}
}

func TestCapabilityExtractionIgnoresReviewerValidationInstructions(t *testing.T) {
	t.Parallel()

	for _, phrase := range []string{
		"enough validation instructions for a reviewer to confirm the game works",
		"reviewer validation steps for keyboard movement",
		"manual validation that verifies the restart flow",
	} {
		if !isValidationEvidenceCapabilityPhrase(phrase) {
			t.Fatalf("expected reviewer validation phrase %q to be treated as validation procedure", phrase)
		}
	}
}

func TestCapabilityMatchingAcceptsSplitDirectionalMovementScenarios(t *testing.T) {
	t.Parallel()

	surface := `
## Scenario Schedule

1. F-001-S004 - User Can Control A Falling Block With Left Movement
2. F-001-S005 - User Can Control A Falling Block With Right Movement
3. F-001-S006 - User Can Control A Falling Block With Down Movement

### F-001-S004: User Can Control A Falling Block With Left Movement
### F-001-S005: User Can Control A Falling Block With Right Movement
### F-001-S006: User Can Control A Falling Block With Down Movement
`
	if !capabilityPhraseCovered(surface, "basic left right down movement") {
		t.Fatal("expected split left/right/down movement scenarios to cover compound movement capability")
	}
}

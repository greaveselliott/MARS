/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/foundation-operating-model.md
- docs/design-docs/validation-matrix-gating.md
- docs/validation/README.md
*/
package validation

import (
	"strings"
	"testing"
)

func TestCheckClosureTextAllowsUnconfirmedBlockedReplay(t *testing.T) {
	report := CheckClosureText(`
# Validation Report

## Pass/fail against AD-284/AD-285

| Run | Archetype | Verdict |
| --- | --- | --- |
| 1 | static-browser | **PASS** |
| 2 | api-service | **BLOCKED** (COO max_turns) |

## Closure verdict

**Foundation improvement plan closure: UNCONFIRMED.** Run 2 is blocked.
`)

	if !report.OK() {
		t.Fatalf("expected report to pass, got problems: %v", report.Problems)
	}
	if len(report.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(report.Rows))
	}
}

func TestCheckClosureTextRejectsConfirmedBlockedReplay(t *testing.T) {
	report := CheckClosureText(`
# Validation Report

## Pass/fail against AD-284/AD-285

| Run | Archetype | Verdict |
| --- | --- | --- |
| 1 | static-browser | **PASS** |
| 2 | api-service | **BLOCKED** (COO max_turns) |

## Closure verdict

**Foundation improvement plan closure: confirmed.** Run 2 is blocked.
`)

	if report.OK() {
		t.Fatal("expected confirmed blocked report to fail")
	}
	if got := strings.Join(report.Problems, "\n"); !strings.Contains(got, "claims confirmed/complete") {
		t.Fatalf("expected confirmed/complete problem, got %q", got)
	}
}

func TestCheckClosureTextRejectsMissingClosureVerdict(t *testing.T) {
	report := CheckClosureText(`
# Validation Report

## Pass/fail against AD-284/AD-285

| Run | Archetype | Verdict |
| --- | --- | --- |
| 1 | static-browser | **PASS** |
`)

	if report.OK() {
		t.Fatal("expected missing closure verdict to fail")
	}
}

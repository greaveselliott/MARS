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
	"fmt"
	"os"
	"regexp"
	"strings"
)

// ClosureReport captures the AD-284 closure status table and verdict text.
type ClosureReport struct {
	Path        string
	Rows        []ClosureRow
	VerdictText string
	Problems    []string
}

// ClosureRow is one row from a report's pass/fail table.
type ClosureRow struct {
	Run       string
	Archetype string
	Verdict   string
}

// OK reports whether the closure report is internally consistent.
func (r ClosureReport) OK() bool {
	return len(r.Problems) == 0
}

// Summary returns a compact human-readable result.
func (r ClosureReport) Summary() string {
	if r.OK() {
		return fmt.Sprintf("closure report ok: %d run rows checked", len(r.Rows))
	}
	return fmt.Sprintf("closure report failed: %d problem(s)", len(r.Problems))
}

// CheckClosureReport reads and validates a closure report.
func CheckClosureReport(path string) (ClosureReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ClosureReport{}, fmt.Errorf("validation check-closure: read %s: %w", path, err)
	}
	report := CheckClosureText(string(data))
	report.Path = path
	return report, nil
}

// CheckClosureText validates closure-report text.
func CheckClosureText(text string) ClosureReport {
	report := ClosureReport{
		Rows:        parseClosureRows(text),
		VerdictText: parseClosureVerdict(text),
	}
	if len(report.Rows) == 0 {
		report.Problems = append(report.Problems, "missing pass/fail table with Run, Archetype, and Verdict columns")
	}
	if strings.TrimSpace(report.VerdictText) == "" {
		report.Problems = append(report.Problems, "missing Closure verdict section")
	}
	if claimsClosureConfirmed(report.VerdictText) {
		for _, row := range report.Rows {
			if closureRowBlocked(row.Verdict) {
				report.Problems = append(report.Problems, fmt.Sprintf("closure verdict claims confirmed/complete while %s %s is %q", row.Run, row.Archetype, row.Verdict))
			}
		}
	}
	return report
}

func parseClosureRows(text string) []ClosureRow {
	lines := strings.Split(text, "\n")
	rows := make([]ClosureRow, 0)
	inTargetTable := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "|") || !strings.HasSuffix(trimmed, "|") {
			inTargetTable = false
			continue
		}
		cells := markdownTableCells(trimmed)
		if len(cells) < 3 {
			continue
		}
		if isMarkdownSeparatorRow(cells) {
			continue
		}
		if tableHeaderMatches(cells, "run", "archetype", "verdict") {
			inTargetTable = true
			continue
		}
		if !inTargetTable {
			continue
		}
		rows = append(rows, ClosureRow{
			Run:       cells[0],
			Archetype: cells[1],
			Verdict:   cells[2],
		})
	}
	return rows
}

func markdownTableCells(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	parts := strings.Split(line, "|")
	cells := make([]string, 0, len(parts))
	for _, part := range parts {
		cells = append(cells, strings.TrimSpace(part))
	}
	return cells
}

func isMarkdownSeparatorRow(cells []string) bool {
	for _, cell := range cells {
		if strings.Trim(cell, ":- ") != "" {
			return false
		}
	}
	return true
}

func tableHeaderMatches(cells []string, want ...string) bool {
	if len(cells) < len(want) {
		return false
	}
	for i, expected := range want {
		if strings.ToLower(strings.TrimSpace(cells[i])) != expected {
			return false
		}
	}
	return true
}

func parseClosureVerdict(text string) string {
	lines := strings.Split(text, "\n")
	inSection := false
	var section []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			if inSection {
				break
			}
			inSection = strings.EqualFold(strings.TrimSpace(strings.TrimPrefix(trimmed, "## ")), "Closure verdict")
			continue
		}
		if inSection {
			section = append(section, line)
		}
	}
	return strings.TrimSpace(strings.Join(section, "\n"))
}

var confirmedVerdictPattern = regexp.MustCompile(`\b(confirm(?:ed)?|complete(?:d)?)\b`)

func claimsClosureConfirmed(text string) bool {
	normalized := strings.ToLower(strings.Join(strings.Fields(text), " "))
	if normalized == "" {
		return false
	}
	for _, negative := range []string{"unconfirmed", "not confirmed", "not complete", "not completed"} {
		if strings.Contains(normalized, negative) {
			return false
		}
	}
	return confirmedVerdictPattern.MatchString(normalized)
}

func closureRowBlocked(verdict string) bool {
	normalized := strings.ToUpper(verdict)
	for _, marker := range []string{"BLOCKED", "FAIL", "FAILED", "PENDING"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

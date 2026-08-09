/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/self-reflective-telemetry.md
- docs/features/F-012-self-improvement-loop.md
*/
package foundationtelemetry

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"time"
)

const SchemaVersion = 1

var safeValueRe = regexp.MustCompile(`^[a-zA-Z0-9._+ -]*$`)

// AnonymousReport is the sanitized envelope a deployed harness may send to a
// foundation collector. It deliberately contains aggregates only.
type AnonymousReport struct {
	SchemaVersion           int                `json:"schema_version"`
	ReportKey               string             `json:"report_key"`
	HarnessVersion          string             `json:"harness_version"`
	GeneratedHarnessVersion string             `json:"generated_harness_version,omitempty"`
	OS                      string             `json:"os"`
	Arch                    string             `json:"arch"`
	HardwareTier            string             `json:"hardware_tier"`
	OrchestrationMode       string             `json:"orchestration_mode"`
	WindowStart             time.Time          `json:"window_start"`
	WindowEnd               time.Time          `json:"window_end"`
	Patterns                []AnonymousPattern `json:"patterns"`
}

// AnonymousPattern is one non-identifying recurring failure aggregate.
type AnonymousPattern struct {
	Signature    string `json:"signature"`
	RoleDomain   string `json:"role_domain"`
	RoleMode     string `json:"role_mode"`
	Category     string `json:"category"`
	Target       string `json:"target"`
	Severity     string `json:"severity"`
	Count        int    `json:"count"`
	DistinctJobs int    `json:"distinct_jobs"`
}

// ReportBatch is the collector request shape.
type ReportBatch struct {
	Reports []AnonymousReport `json:"reports"`
}

// AggregatedPattern is the foundation collector's cross-report view.
type AggregatedPattern struct {
	Signature          string
	ReportHash         string
	ReportKey          string
	FirstSeen          time.Time
	LastSeen           time.Time
	ReportCount        int
	InstallWindowCount int
	HarnessVersions    []string
	Category           string
	Target             string
	Severity           string
}

// FoundationTelemetryStore is the pluggable collector storage interface. V1
// ships SQLite; a future Postgres/Neon adapter can implement the same contract.
type FoundationTelemetryStore interface {
	SaveReport(ctx context.Context, report AnonymousReport) error
	UpsertPattern(ctx context.Context, pattern AggregatedPattern) error
	PatternsSince(ctx context.Context, since time.Time) ([]AggregatedPattern, error)
}

// PayloadHash returns a stable hash of a sanitized report.
func PayloadHash(report AnonymousReport) (string, error) {
	data, err := json.Marshal(report)
	if err != nil {
		return "", fmt.Errorf("foundation telemetry: marshal report for hash: %w", err)
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum[:]), nil
}

// PatternSignature hashes the non-identifying dimensions of a pattern.
func PatternSignature(parts ...string) string {
	for i, part := range parts {
		parts[i] = strings.ToLower(strings.TrimSpace(part))
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return "sig-" + fmt.Sprintf("%x", sum[:12])
}

// ValidateReport enforces the anonymous telemetry privacy contract.
func ValidateReport(report AnonymousReport) error {
	if report.SchemaVersion != SchemaVersion {
		return fmt.Errorf("foundation telemetry: unsupported schema_version %d", report.SchemaVersion)
	}
	required := map[string]string{
		"report_key":         report.ReportKey,
		"harness_version":    report.HarnessVersion,
		"os":                 report.OS,
		"arch":               report.Arch,
		"hardware_tier":      report.HardwareTier,
		"orchestration_mode": report.OrchestrationMode,
	}
	for field, value := range required {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("foundation telemetry: %s is required", field)
		}
		if !safeValueRe.MatchString(value) {
			return fmt.Errorf("foundation telemetry: %s contains unsafe characters", field)
		}
	}
	if report.GeneratedHarnessVersion != "" && !safeValueRe.MatchString(report.GeneratedHarnessVersion) {
		return fmt.Errorf("foundation telemetry: generated_harness_version contains unsafe characters")
	}
	if report.WindowStart.IsZero() || report.WindowEnd.IsZero() || !report.WindowStart.Before(report.WindowEnd) {
		return fmt.Errorf("foundation telemetry: valid window_start and window_end are required")
	}
	for i, pattern := range report.Patterns {
		if err := ValidatePattern(pattern); err != nil {
			return fmt.Errorf("foundation telemetry: pattern %d: %w", i, err)
		}
	}
	return nil
}

// ValidatePattern validates one anonymous aggregate.
func ValidatePattern(pattern AnonymousPattern) error {
	values := map[string]string{
		"signature":   pattern.Signature,
		"role_domain": pattern.RoleDomain,
		"role_mode":   pattern.RoleMode,
		"category":    pattern.Category,
		"target":      pattern.Target,
		"severity":    pattern.Severity,
	}
	for field, value := range values {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", field)
		}
		if !safeValueRe.MatchString(value) {
			return fmt.Errorf("%s contains unsafe characters", field)
		}
	}
	if pattern.Count <= 0 {
		return fmt.Errorf("count must be positive")
	}
	if pattern.DistinctJobs <= 0 {
		return fmt.Errorf("distinct_jobs must be positive")
	}
	return nil
}

// DecodeBatch decodes collector input with unknown fields rejected so callers
// cannot smuggle raw telemetry alongside the allowlisted envelope.
func DecodeBatch(data []byte) (ReportBatch, error) {
	var batch ReportBatch
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&batch); err != nil {
		return ReportBatch{}, fmt.Errorf("foundation telemetry: decode batch: %w", err)
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		if err == nil {
			return ReportBatch{}, fmt.Errorf("foundation telemetry: decode batch: multiple JSON values")
		}
		return ReportBatch{}, fmt.Errorf("foundation telemetry: decode batch: trailing data: %w", err)
	}
	if len(batch.Reports) == 0 {
		return ReportBatch{}, fmt.Errorf("foundation telemetry: reports is required")
	}
	for _, report := range batch.Reports {
		if err := ValidateReport(report); err != nil {
			return ReportBatch{}, err
		}
	}
	return batch, nil
}

// AggregatesFromReport converts one report into pattern upserts.
func AggregatesFromReport(report AnonymousReport) []AggregatedPattern {
	out := make([]AggregatedPattern, 0, len(report.Patterns))
	reportHash, _ := PayloadHash(report)
	for _, pattern := range report.Patterns {
		out = append(out, AggregatedPattern{
			Signature:          pattern.Signature,
			ReportHash:         reportHash,
			ReportKey:          report.ReportKey,
			FirstSeen:          report.WindowStart,
			LastSeen:           report.WindowEnd,
			ReportCount:        1,
			InstallWindowCount: 1,
			HarnessVersions:    []string{report.HarnessVersion},
			Category:           pattern.Category,
			Target:             pattern.Target,
			Severity:           pattern.Severity,
		})
	}
	return out
}

func mergeVersions(existing []string, next []string) []string {
	seen := map[string]bool{}
	for _, value := range append(existing, next...) {
		value = strings.TrimSpace(value)
		if value != "" {
			seen[value] = true
		}
	}
	out := make([]string, 0, len(seen))
	for value := range seen {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

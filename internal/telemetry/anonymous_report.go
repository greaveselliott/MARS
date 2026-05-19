/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/self-reflective-telemetry.md
- docs/features/F-012-self-improvement-loop.md
*/
package telemetry

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/greaveselliott/mars-harness/internal/foundationtelemetry"
)

// RoleMetadata is the non-identifying role shape allowed in anonymous reports.
type RoleMetadata struct {
	Domain string
	Mode   string
}

// AnonymousReportOptions controls report generation from local raw telemetry.
type AnonymousReportOptions struct {
	RepoID                  string
	ReportKeySeed           string
	HarnessVersion          string
	GeneratedHarnessVersion string
	OS                      string
	Arch                    string
	HardwareTier            string
	OrchestrationMode       string
	WindowStart             time.Time
	WindowEnd               time.Time
	Roles                   map[string]RoleMetadata
}

// LoadOrCreateReportKeySeed returns the local secret seed used only to derive
// rotating anonymous report keys. The raw seed is never included in reports.
func LoadOrCreateReportKeySeed(baseDir string) (string, error) {
	baseDir = strings.TrimSpace(baseDir)
	if baseDir == "" {
		return "", fmt.Errorf("telemetry: base directory is required for anonymous report key seed")
	}
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return "", fmt.Errorf("telemetry: create base directory for anonymous report key seed: %w", err)
	}
	path := filepath.Join(baseDir, "anonymous-telemetry-seed")
	if data, err := os.ReadFile(path); err == nil {
		if seed := strings.TrimSpace(string(data)); seed != "" {
			return seed, nil
		}
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("telemetry: read anonymous report key seed: %w", err)
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("telemetry: generate anonymous report key seed: %w", err)
	}
	seed := fmt.Sprintf("%x", buf)
	if err := os.WriteFile(path, []byte(seed+"\n"), 0o600); err != nil {
		return "", fmt.Errorf("telemetry: write anonymous report key seed: %w", err)
	}
	return seed, nil
}

// BuildAnonymousReport builds an allowlisted aggregate from local raw telemetry.
func (s *Store) BuildAnonymousReport(opts AnonymousReportOptions) (foundationtelemetry.AnonymousReport, error) {
	if s == nil || s.db == nil {
		return foundationtelemetry.AnonymousReport{}, fmt.Errorf("telemetry: store unavailable")
	}
	if opts.WindowEnd.IsZero() {
		opts.WindowEnd = time.Now().UTC().Truncate(time.Hour)
	}
	if opts.WindowStart.IsZero() {
		opts.WindowStart = opts.WindowEnd.Add(-PatternWindow)
	}
	if strings.TrimSpace(opts.OS) == "" {
		opts.OS = runtime.GOOS
	}
	if strings.TrimSpace(opts.Arch) == "" {
		opts.Arch = runtime.GOARCH
	}
	counts, err := s.RoleCategoryCountsBetween(opts.WindowStart, opts.WindowEnd)
	if err != nil {
		return foundationtelemetry.AnonymousReport{}, err
	}
	report := foundationtelemetry.AnonymousReport{
		SchemaVersion:           foundationtelemetry.SchemaVersion,
		ReportKey:               rotatingReportKey(opts.ReportKeySeed, opts.WindowStart),
		HarnessVersion:          defaultAnonymousValue(opts.HarnessVersion, "unknown"),
		GeneratedHarnessVersion: strings.TrimSpace(opts.GeneratedHarnessVersion),
		OS:                      defaultAnonymousValue(opts.OS, "unknown"),
		Arch:                    defaultAnonymousValue(opts.Arch, "unknown"),
		HardwareTier:            defaultAnonymousValue(opts.HardwareTier, "unknown"),
		OrchestrationMode:       defaultAnonymousValue(opts.OrchestrationMode, "unknown"),
		WindowStart:             opts.WindowStart.UTC(),
		WindowEnd:               opts.WindowEnd.UTC(),
	}
	for _, count := range counts {
		if strings.TrimSpace(opts.RepoID) != "" && count.RepoID != opts.RepoID {
			continue
		}
		if count.DistinctJobs <= 0 {
			count.DistinctJobs = count.Count
		}
		if count.DistinctJobs <= 0 || !ReportableFoundationCategory(count.Category) {
			continue
		}
		proposal := TriagePattern(Pattern{
			RepoID:   count.RepoID,
			Role:     count.Role,
			Category: count.Category,
			Count:    count.DistinctJobs,
			Window:   "24h",
		})
		role := opts.Roles[count.Role]
		domain := defaultAnonymousValue(role.Domain, "unknown")
		mode := defaultAnonymousValue(role.Mode, "unknown")
		report.Patterns = append(report.Patterns, foundationtelemetry.AnonymousPattern{
			Signature: foundationtelemetry.PatternSignature(
				domain,
				mode,
				string(count.Category),
				string(proposal.Target),
				proposal.Severity,
				report.GeneratedHarnessVersion,
			),
			RoleDomain:   domain,
			RoleMode:     mode,
			Category:     string(count.Category),
			Target:       string(proposal.Target),
			Severity:     proposal.Severity,
			Count:        count.Count,
			DistinctJobs: count.DistinctJobs,
		})
	}
	if err := foundationtelemetry.ValidateReport(report); err != nil {
		return foundationtelemetry.AnonymousReport{}, err
	}
	return report, nil
}

// ReportableFoundationCategory reports whether a local failure category belongs
// to foundation telemetry rather than target backlog intervention debt.
func ReportableFoundationCategory(cat FailureCategory) bool {
	switch cat {
	case CategoryContextOverflow,
		CategoryLLMUnreachable,
		CategoryInferenceCrash,
		CategoryModelUnavailable,
		CategoryToolTimeout,
		CategoryCircleDetected,
		CategoryMaxTurns,
		CategoryBudgetExceeded,
		CategoryManifestError,
		CategoryTicketGate,
		CategoryDispatchProtocol,
		CategoryGuardrailBlock,
		CategoryWorkspaceHygiene,
		CategoryUnknown:
		return true
	default:
		return false
	}
}

func rotatingReportKey(seed string, windowStart time.Time) string {
	seed = strings.TrimSpace(seed)
	if seed == "" {
		seed = "ephemeral"
	}
	year, week := windowStart.ISOWeek()
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%04d-W%02d", seed, year, week)))
	return "rk-" + fmt.Sprintf("%x", sum[:12])
}

func defaultAnonymousValue(value, fallback string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return fallback
	}
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_' || r == '.' || r == '+':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return fallback
	}
	return out
}

package guardrails

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEngine_hardRuleBlocksViolation(t *testing.T) {
	t.Parallel()
	rules := []Rule{{
		ID:       "no-secrets",
		Name:     "No Hardcoded Secrets",
		Severity: SeverityHard,
		Scope:    "global",
		Pattern:  `(?i)password\s*=\s*"[^"]+"|AKIA[0-9A-Z]{16}`,
		Message:  "Do not hardcode secrets in source files",
	}}

	eng, err := New(rules)
	require.NoError(t, err)

	err = eng.CheckFile("ci-fixer", "config.go", `var password = "hunter2"`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no-secrets")

	err = eng.CheckFile("ci-fixer", "config.go", `var password = os.Getenv("PASSWORD")`)
	require.NoError(t, err)
}

func TestEngine_advisoryInPrompt(t *testing.T) {
	t.Parallel()
	rules := []Rule{
		{
			ID:       "prefer-tables",
			Name:     "Use Table-Driven Tests",
			Severity: SeverityAdvisory,
			Scope:    "global",
			Message:  "Prefer table-driven tests in Go test files",
		},
		{
			ID:       "no-fmt",
			Name:     "No fmt in Production",
			Severity: SeverityHard,
			Scope:    "global",
			Pattern:  `fmt\.Print`,
			Message:  "Use slog instead of fmt.Print",
		},
	}

	eng, err := New(rules)
	require.NoError(t, err)

	prompt := eng.AdvisoryPrompt("ci-fixer")
	require.Contains(t, prompt, "Use Table-Driven Tests")
	require.Contains(t, prompt, "Prefer table-driven tests")
	require.NotContains(t, prompt, "No fmt in Production")
}

func TestEngine_scopeFiltering(t *testing.T) {
	t.Parallel()
	rules := []Rule{{
		ID:       "ci-only",
		Name:     "CI Only Rule",
		Severity: SeverityHard,
		Scope:    "ci-fixer",
		Pattern:  `FORBIDDEN`,
		Message:  "This pattern is forbidden for ci-fixer",
	}}

	eng, err := New(rules)
	require.NoError(t, err)

	err = eng.CheckFile("ci-fixer", "main.go", "FORBIDDEN content")
	require.Error(t, err)

	err = eng.CheckFile("pr-writer", "main.go", "FORBIDDEN content")
	require.NoError(t, err)
}

func TestEngine_overrideAllowsBlocked(t *testing.T) {
	t.Parallel()
	rules := []Rule{{
		ID:       "no-eval",
		Name:     "No Eval",
		Severity: SeverityHard,
		Scope:    "global",
		Pattern:  `eval\(`,
		Message:  "Do not use eval()",
	}}

	eng, err := New(rules)
	require.NoError(t, err)

	content := `result := eval("1+1")`

	err = eng.CheckFile("ci-fixer", "exec.go", content)
	require.Error(t, err)

	future := time.Now().UTC().Add(24 * time.Hour)
	overrides := []Override{{
		RuleID:    "no-eval",
		Principal: "admin",
		Reason:    "Legacy code migration",
		ExpiresAt: &future,
	}}

	err = eng.CheckWithOverride("ci-fixer", "exec.go", content, overrides)
	require.NoError(t, err)
}

func TestEngine_staleRules90Days(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)

	rules := []Rule{
		{
			ID:        "old-rule",
			Name:      "Old Rule",
			Severity:  SeverityAdvisory,
			Scope:     "global",
			Message:   "Outdated",
			CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			StaleDays: 90,
		},
		{
			ID:        "new-rule",
			Name:      "New Rule",
			Severity:  SeverityAdvisory,
			Scope:     "global",
			Message:   "Fresh",
			CreatedAt: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
			StaleDays: 90,
		},
		{
			ID:        "immortal-rule",
			Name:      "Never Stale",
			Severity:  SeverityHard,
			Scope:     "global",
			Pattern:   `TODO`,
			Message:   "Always enforced",
			CreatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			StaleDays: -1,
		},
	}

	eng, err := New(rules)
	require.NoError(t, err)

	stale := eng.StaleRules(now)
	require.Len(t, stale, 1)
	require.Equal(t, "old-rule", stale[0].ID)
}

func TestEngine_invalidRegexRejected(t *testing.T) {
	t.Parallel()
	rules := []Rule{{
		ID:       "bad-regex",
		Name:     "Bad Regex",
		Severity: SeverityHard,
		Scope:    "global",
		Pattern:  `[unclosed`,
		Message:  "This should fail",
	}}

	_, err := New(rules)
	require.Error(t, err)
	require.Contains(t, err.Error(), "bad-regex")
	require.Contains(t, err.Error(), "invalid regex")
}

func TestEngine_filePatternMatching(t *testing.T) {
	t.Parallel()
	rules := []Rule{{
		ID:          "no-secrets-in-yaml",
		Name:        "No Secrets in YAML",
		Severity:    SeverityHard,
		Scope:       "global",
		Pattern:     `secret_key`,
		FilePattern: "*.yaml",
		Message:     "Do not put secrets in YAML files",
	}}

	eng, err := New(rules)
	require.NoError(t, err)

	err = eng.CheckFile("ci-fixer", "config.yaml", "secret_key: abc123")
	require.Error(t, err)

	err = eng.CheckFile("ci-fixer", "config.go", "secret_key = getValue()")
	require.NoError(t, err)
}

func TestEngine_deduplicateAdvisory(t *testing.T) {
	t.Parallel()
	rules := []Rule{
		{
			ID:       "dup-rule",
			Name:     "Duplicate Advisory",
			Severity: SeverityAdvisory,
			Scope:    "global",
			Message:  "This appears twice in the rule set",
		},
		{
			ID:       "dup-rule",
			Name:     "Duplicate Advisory",
			Severity: SeverityAdvisory,
			Scope:    "global",
			Message:  "This appears twice in the rule set",
		},
		{
			ID:       "unique-rule",
			Name:     "Unique Advisory",
			Severity: SeverityAdvisory,
			Scope:    "global",
			Message:  "This is unique",
		},
	}

	eng, err := New(rules)
	require.NoError(t, err)

	prompt := eng.AdvisoryPrompt("ci-fixer")
	count := strings.Count(prompt, "Duplicate Advisory")
	require.Equal(t, 1, count, "expected deduplicated advisory to appear exactly once")
	require.Contains(t, prompt, "Unique Advisory")
}

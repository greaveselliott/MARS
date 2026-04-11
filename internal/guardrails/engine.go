package guardrails

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Severity controls enforcement behaviour.
type Severity string

const (
	SeverityAdvisory Severity = "advisory"
	SeverityHard     Severity = "hard"
)

const defaultStaleDays = 90

// Rule is a single guardrail rule.
type Rule struct {
	ID          string
	Name        string
	Severity    Severity
	Scope       string // role name or "global"
	Pattern     string // regex pattern for content matching
	FilePattern string // glob for file path matching
	Message     string
	CreatedAt   time.Time
	StaleDays   int // 0 = never stale; default 90
}

// Override records a temporary exemption from a hard rule.
type Override struct {
	RuleID    string
	Principal string
	Reason    string
	ExpiresAt *time.Time
}

type compiledRule struct {
	rule        Rule
	contentRe   *regexp.Regexp // nil if Pattern is empty
	filePattern string         // normalised glob, empty if none
}

// Engine evaluates guardrail rules against operations.
type Engine struct {
	rules []compiledRule
}

// New creates an engine with the given rules. Returns an error if any
// rule contains an invalid regex pattern.
func New(rules []Rule) (*Engine, error) {
	compiled := make([]compiledRule, 0, len(rules))
	for _, r := range rules {
		cr := compiledRule{rule: r, filePattern: r.FilePattern}
		if r.Pattern != "" {
			re, err := regexp.Compile(r.Pattern)
			if err != nil {
				return nil, fmt.Errorf("guardrails: rule %q has invalid regex %q: %w", r.ID, r.Pattern, err)
			}
			cr.contentRe = re
		}
		compiled = append(compiled, cr)
	}
	return &Engine{rules: compiled}, nil
}

// CheckFile evaluates hard rules against a file write operation.
// Returns nil if allowed, error with rule name if blocked.
func (e *Engine) CheckFile(role, filePath, content string) error {
	for _, cr := range e.rules {
		if cr.rule.Severity != SeverityHard {
			continue
		}
		if !scopeMatches(cr.rule.Scope, role) {
			continue
		}
		if !filePatternMatches(cr.filePattern, filePath) {
			continue
		}
		if cr.contentRe != nil && cr.contentRe.MatchString(content) {
			slog.Warn("guardrails: hard rule violation",
				"rule", cr.rule.ID,
				"name", cr.rule.Name,
				"file", filePath,
				"role", role,
			)
			return fmt.Errorf("guardrails: blocked by rule %q (%s): %s", cr.rule.ID, cr.rule.Name, cr.rule.Message)
		}
	}
	return nil
}

// CheckWithOverride checks file with active overrides applied.
// Overridden rules are skipped during evaluation.
func (e *Engine) CheckWithOverride(role, filePath, content string, overrides []Override) error {
	active := activeOverrideSet(overrides)

	for _, cr := range e.rules {
		if cr.rule.Severity != SeverityHard {
			continue
		}
		if _, overridden := active[cr.rule.ID]; overridden {
			slog.Debug("guardrails: rule overridden", "rule", cr.rule.ID)
			continue
		}
		if !scopeMatches(cr.rule.Scope, role) {
			continue
		}
		if !filePatternMatches(cr.filePattern, filePath) {
			continue
		}
		if cr.contentRe != nil && cr.contentRe.MatchString(content) {
			return fmt.Errorf("guardrails: blocked by rule %q (%s): %s", cr.rule.ID, cr.rule.Name, cr.rule.Message)
		}
	}
	return nil
}

// AdvisoryPrompt returns advisory rules formatted for inclusion in the
// system prompt. Deduplicates so each advisory appears at most once.
func (e *Engine) AdvisoryPrompt(role string) string {
	seen := make(map[string]struct{})
	var lines []string

	for _, cr := range e.rules {
		if cr.rule.Severity != SeverityAdvisory {
			continue
		}
		if !scopeMatches(cr.rule.Scope, role) {
			continue
		}
		if _, dup := seen[cr.rule.ID]; dup {
			continue
		}
		seen[cr.rule.ID] = struct{}{}
		lines = append(lines, fmt.Sprintf("- [%s] %s", cr.rule.Name, cr.rule.Message))
	}

	if len(lines) == 0 {
		return ""
	}
	return "## Guardrails\n" + strings.Join(lines, "\n")
}

// StaleRules returns rules older than their StaleDays threshold.
// Rules with StaleDays == 0 are never stale.
func (e *Engine) StaleRules(now time.Time) []Rule {
	var stale []Rule
	for _, cr := range e.rules {
		staleDays := cr.rule.StaleDays
		if staleDays == 0 {
			staleDays = defaultStaleDays
		}
		if staleDays < 0 {
			continue
		}
		cutoff := cr.rule.CreatedAt.Add(time.Duration(staleDays) * 24 * time.Hour)
		if now.After(cutoff) {
			stale = append(stale, cr.rule)
		}
	}
	return stale
}

func scopeMatches(scope, role string) bool {
	return scope == "global" || scope == role
}

func filePatternMatches(pattern, filePath string) bool {
	if pattern == "" {
		return true
	}
	matched, err := filepath.Match(pattern, filepath.Base(filePath))
	if err != nil {
		slog.Warn("guardrails: bad file pattern", "pattern", pattern, "error", err)
		return false
	}
	return matched
}

func activeOverrideSet(overrides []Override) map[string]struct{} {
	now := time.Now().UTC()
	active := make(map[string]struct{})
	for _, o := range overrides {
		if o.ExpiresAt != nil && o.ExpiresAt.Before(now) {
			continue
		}
		active[o.RuleID] = struct{}{}
	}
	return active
}

package safety

import (
	"regexp"
	"strings"
)

// ScanResult is a potential secret detection.
type ScanResult struct {
	File    string
	Line    int
	Pattern string
	Match   string
}

type secretPattern struct {
	name string
	re   *regexp.Regexp
}

var patterns = []secretPattern{
	{
		name: "AWS Access Key",
		re:   regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
	},
	{
		name: "GitHub Token (ghp/gho/ghs)",
		re:   regexp.MustCompile(`(?:ghp|gho|ghs)_[A-Za-z0-9_]{36,}`),
	},
	{
		name: "Private Key Block",
		re:   regexp.MustCompile(`-----BEGIN\s+(?:RSA|DSA|EC|OPENSSH|PGP)?\s*PRIVATE KEY-----`),
	},
	{
		name: "Password in URL",
		re:   regexp.MustCompile(`://[^/\s:]+:[^/\s@]+@`),
	},
	{
		name: "Generic API Key Assignment",
		re:   regexp.MustCompile(`(?i)(?:api[_-]?key|apikey|secret[_-]?key|access[_-]?token)\s*[:=]\s*["']?[A-Za-z0-9/+=_\-]{16,}["']?`),
	},
}

// ScanForSecrets scans file content for common secret patterns.
func ScanForSecrets(filename, content string) []ScanResult {
	var results []ScanResult
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		for _, p := range patterns {
			if matches := p.re.FindAllString(line, -1); len(matches) > 0 {
				for _, m := range matches {
					results = append(results, ScanResult{
						File:    filename,
						Line:    i + 1,
						Pattern: p.name,
						Match:   m,
					})
				}
			}
		}
	}

	return results
}

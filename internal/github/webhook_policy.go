/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/github-app-integration.md
- docs/features/F-011-optional-github-integration.md
- docs/features/F-017-open-source-publication.md
- docs/product-specs/product-surface.md
*/
package github

import (
	"fmt"
	"strings"
	"unicode"
)

const (
	MaxWebhookActorIDs = 256
	MaxRepositoryName  = 255
	MaxBranchName      = 255
)

// NormalizeWebhookActorIDs validates, bounds, and deduplicates trusted numeric
// GitHub actor IDs while preserving their configured order.
func NormalizeWebhookActorIDs(ids []int64) ([]int64, error) {
	seen := make(map[int64]struct{}, len(ids))
	result := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return nil, fmt.Errorf("webhook actor policy contains invalid actor ID %d; IDs must be positive numeric GitHub actor IDs", id)
		}
		if _, ok := seen[id]; ok {
			continue
		}
		if len(result) >= MaxWebhookActorIDs {
			return nil, fmt.Errorf("webhook actor policy contains more than %d unique IDs", MaxWebhookActorIDs)
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result, nil
}

// NormalizeRepository validates exact owner/repo syntax and returns its
// case-normalized identity. Whitespace, controls, URL syntax, and path escapes
// are rejected rather than trimmed or interpreted.
func NormalizeRepository(value string) (string, error) {
	if value == "" || len(value) > MaxRepositoryName || value != strings.TrimSpace(value) {
		return "", fmt.Errorf("invalid GitHub repository %q; use a bounded exact owner/repo identifier", value)
	}
	parts := strings.Split(value, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" || parts[0] == "." || parts[0] == ".." || parts[1] == "." || parts[1] == ".." {
		return "", fmt.Errorf("invalid GitHub repository %q; use exact owner/repo syntax", value)
	}
	for _, r := range value {
		if r == '/' {
			continue
		}
		if !isASCIIAlphaNumeric(r) && r != '-' && r != '_' && r != '.' {
			return "", fmt.Errorf("invalid GitHub repository %q; owner/repo may contain only ASCII letters, digits, period, underscore, and hyphen", value)
		}
	}
	return strings.ToLower(value), nil
}

// ValidateBranch rejects missing, oversized, ambiguous, URL-shaped, control,
// whitespace, or invalid Git ref names while preserving case.
func ValidateBranch(value string) error {
	if value == "" || len(value) > MaxBranchName || value != strings.TrimSpace(value) {
		return fmt.Errorf("invalid GitHub branch %q; use a bounded exact branch name without whitespace", value)
	}
	if strings.HasPrefix(value, ".") || strings.HasPrefix(value, "-") || strings.HasPrefix(value, "/") || strings.HasSuffix(value, "/") || strings.HasSuffix(value, ".") ||
		strings.HasSuffix(value, ".lock") || strings.Contains(value, "..") || strings.Contains(value, "@{") || strings.Contains(value, "//") {
		return fmt.Errorf("invalid GitHub branch %q; use a valid exact Git branch name", value)
	}
	for _, component := range strings.Split(value, "/") {
		if strings.HasPrefix(component, ".") {
			return fmt.Errorf("invalid GitHub branch %q; dot-prefixed path components are not allowed", value)
		}
		if strings.HasSuffix(component, ".lock") {
			return fmt.Errorf("invalid GitHub branch %q; components ending in .lock are not allowed", value)
		}
	}
	for _, r := range value {
		if unicode.IsSpace(r) || unicode.IsControl(r) || strings.ContainsRune("\\?#~^:*[", r) {
			return fmt.Errorf("invalid GitHub branch %q; whitespace, controls, backslash, query/fragment, and Git ref metacharacters are not allowed", value)
		}
	}
	return nil
}

func validBoundedToken(value string, limit int) bool {
	if value == "" || len(value) > limit || value != strings.TrimSpace(value) {
		return false
	}
	for _, r := range value {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func isASCIIAlphaNumeric(r rune) bool {
	return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9'
}

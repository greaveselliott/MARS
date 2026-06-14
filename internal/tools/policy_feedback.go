/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/delivery-operating-model.md
- docs/design-docs/tools-glossary.md
- docs/features/F-005-agent-execution-runtime.md
- docs/features/F-006-queue-and-orchestration.md
*/
package tools

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const policyBlockCountKeyPrefix = "policy:block:count:"

func recordRepeatedPolicyFailure(session *Session, stage, name string, err error) int {
	if session == nil || err == nil {
		return 0
	}
	if session.ToolState == nil {
		session.ToolState = make(map[string]string)
	}
	key := policyBlockCountKey(session.Role, stage, name, err.Error())
	count, _ := strconv.Atoi(session.ToolState[key])
	count++
	session.ToolState[key] = strconv.Itoa(count)
	return count
}

func policyBlockCountKey(role, stage, name, message string) string {
	fingerprint := strings.Join([]string{
		normalizePolicyFeedbackField(role),
		normalizePolicyFeedbackField(stage),
		normalizePolicyFeedbackField(name),
		normalizePolicyFeedbackField(message),
	}, "|")
	sum := sha256.Sum256([]byte(fingerprint))
	return policyBlockCountKeyPrefix + fmt.Sprintf("%x", sum[:8])
}

func normalizePolicyFeedbackField(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(value))), " ")
}

func withPolicyFailureRepairFeedback(root Root, session *Session, stage, name string, err error, count int) error {
	if err == nil || count < 2 {
		return err
	}
	feedback := policyFailureRepairFeedback(root, session, stage, name, err, count)
	if strings.TrimSpace(feedback) == "" {
		return err
	}
	return fmt.Errorf("%v\n\n%s", err, feedback)
}

func policyFailureRepairFeedback(root Root, session *Session, stage, name string, err error, count int) string {
	if session != nil && strings.EqualFold(strings.TrimSpace(session.Role), "coo") && name == "job_disposition_record" {
		if guidance := cooFeatureSpecificityRepairFeedback(root, err); guidance != "" {
			return guidance
		}
	}
	if name == "shell_exec" {
		if guidance := shellExecValidationLaneRepairFeedback(stage, name, err, count); guidance != "" {
			return guidance
		}
	}
	return fmt.Sprintf("Guardrail repair required:\n- Repeated policy block #%d for %s during %s-policy checks.\n- Do not call %s again with the same payload until the blocker is repaired.\n- Read the policy error above, perform the allowed repair action it names, then retry.", count, name, stage, name)
}

func shellExecValidationLaneRepairFeedback(stage, name string, err error, count int) string {
	if err == nil {
		return ""
	}
	message := strings.ToLower(err.Error())
	if !strings.Contains(message, "failing test or build command is unresolved") {
		return ""
	}
	lines := []string{
		"Guardrail repair required:",
		fmt.Sprintf("- Repeated policy block #%d for %s during %s-policy checks.", count, name, stage),
		"- Stop trying alternate shell_exec or dependency commands; they remain blocked until the failed test/build lane passes.",
		"- Use file_read/file_write to repair the source, test, fixture, or package/build config for the exact unresolved command named above.",
		"- Rerun the exact focused test/build command successfully before ticket moves, evidence, commits, runtime probes, or more shell exploration.",
	}
	if strings.Contains(message, "github.com/stretchr/testify") ||
		strings.Contains(message, "no required module provides package") ||
		strings.Contains(message, "missing go.sum entry") {
		lines = append(lines, "- If a newly written Go test introduced a missing assertion dependency, rewrite it with standard-library testing assertions or remove the new dependency before rerunning go test.")
	}
	return strings.Join(lines, "\n")
}

func cooFeatureSpecificityRepairFeedback(root Root, err error) string {
	if err == nil {
		return ""
	}
	message := err.Error()
	if !strings.Contains(strings.ToLower(message), "product brief capabilities") &&
		!strings.Contains(strings.ToLower(message), "starter-placeholder") {
		return ""
	}
	capabilities := policyMissingCapabilityPhrases(message)
	featurePaths := cooRepairFeatureContractPaths(root)
	sourcePaths := cooRepairSourcePaths(root, featurePaths)
	if len(featurePaths) == 0 {
		featurePaths = []string{"docs/features/F-001-product-walking-skeleton.md"}
	}
	capabilityText := "the missing product capabilities"
	if len(capabilities) > 0 {
		capabilityText = strings.Join(capabilities, ", ")
	}
	return fmt.Sprintf("Guardrail repair required:\n- Do not call job_disposition_record again for this same payload.\n- Missing product capability coverage: %s.\n- Call file_read on %s.\n- Call file_write on %s to update Scenario Schedule/scenario headings or Descoped Scenarios.\n- Retry job_disposition_record only after coverage includes: %s.", capabilityText, strings.Join(sourcePaths, " and "), strings.Join(featurePaths, " or "), capabilityText)
}

func policyMissingCapabilityPhrases(message string) []string {
	lower := strings.ToLower(message)
	for _, marker := range []string{
		"scenario schedule does not cover product brief capabilities:",
		"scenario outline does not break out product brief capabilities:",
		"active feature contracts list explicit product brief capabilities under out of scope without descoped scenarios rationale:",
	} {
		idx := strings.Index(lower, marker)
		if idx < 0 {
			continue
		}
		tail := strings.TrimSpace(message[idx+len(marker):])
		if end := strings.Index(tail, "."); end >= 0 {
			tail = tail[:end]
		}
		var out []string
		for _, part := range strings.Split(tail, ",") {
			phrase := cleanCapabilityPhrase(part)
			if phrase != "" {
				out = append(out, phrase)
			}
		}
		return out
	}
	return nil
}

func cooRepairFeatureContractPaths(root Root) []string {
	if refs := existingProjectBriefFeatureReferencePaths(root); len(refs) > 0 {
		return refs
	}
	var out []string
	for _, id := range activePlanFeatureIDs(root) {
		if rel := relPathFromAbs(root, featureContractPath(root, id)); rel != "" {
			out = append(out, rel)
		}
	}
	if len(out) > 0 {
		return dedupeStrings(out)
	}
	if rel := relPathFromAbs(root, featureContractPath(root, "F-001")); rel != "" {
		return []string{rel}
	}
	for _, id := range featureContractIDs(root) {
		if rel := relPathFromAbs(root, featureContractPath(root, id)); rel != "" {
			return []string{rel}
		}
	}
	return nil
}

func cooRepairSourcePaths(root Root, featurePaths []string) []string {
	var paths []string
	for _, rel := range []string{
		"README.md",
		filepath.Join("docs", "product-specs", "vision.md"),
		filepath.Join("docs", "goals", "active.md"),
	} {
		if rootRelExists(root, rel) {
			paths = append(paths, filepath.ToSlash(rel))
		}
	}
	paths = append(paths, featurePaths...)
	if len(paths) == 0 {
		return []string{"README.md"}
	}
	return dedupeStrings(paths)
}

func existingProjectBriefFeatureReferencePaths(root Root) []string {
	var out []string
	for _, rel := range projectBriefFeatureReferencePaths(root) {
		if rootRelExists(root, rel) {
			out = append(out, rel)
		}
	}
	return dedupeStrings(out)
}

func projectBriefFeatureReferencePaths(root Root) []string {
	text := projectBriefSourceText(root)
	if strings.TrimSpace(text) == "" {
		return nil
	}
	pattern := regexp.MustCompile(`docs/features/[A-Za-z0-9._/-]+\.md`)
	var out []string
	for _, match := range pattern.FindAllString(text, -1) {
		out = append(out, filepath.ToSlash(filepath.Clean(match)))
	}
	return dedupeStrings(out)
}

func rootRelExists(root Root, rel string) bool {
	abs, err := root.ResolvePath(rel)
	if err != nil {
		return false
	}
	_, err = os.Stat(abs)
	return err == nil
}

func relPathFromAbs(root Root, abs string) string {
	abs = strings.TrimSpace(abs)
	if abs == "" {
		return ""
	}
	rel, err := filepath.Rel(root.Abs(), abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return ""
	}
	return filepath.ToSlash(rel)
}

func dedupeStrings(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

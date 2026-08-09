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
	"path/filepath"
	"regexp"
	"strings"
)

func projectBriefMentionsFramework(root Root, framework string) bool {
	framework = strings.ToLower(strings.TrimSpace(framework))
	if framework == "" {
		return false
	}
	for _, rel := range []string{
		"README.md",
		filepath.Join("docs", "product-specs", "vision.md"),
		filepath.Join("docs", "goals", "active.md"),
		filepath.Join("docs", "features", "F-001-product-walking-skeleton.md"),
	} {
		data, err := root.RepoFS().ReadFile(rel)
		if err != nil {
			continue
		}
		if strings.Contains(strings.ToLower(string(data)), framework) {
			return true
		}
	}
	return false
}

func projectBriefHasConcreteProductIntent(root Root) bool {
	for _, rel := range []string{
		"README.md",
		filepath.Join("docs", "product-specs", "vision.md"),
		filepath.Join("docs", "goals", "active.md"),
	} {
		data, err := root.RepoFS().ReadFile(rel)
		if err != nil {
			continue
		}
		lower := strings.ToLower(string(data))
		for _, marker := range []string{"create ", "build ", "implement ", "game", "app", "application", "service", "tool", "website", "dashboard"} {
			if strings.Contains(lower, marker) {
				return true
			}
		}
	}
	return false
}

func projectBriefCapabilityPhrases(root Root) []string {
	text := projectBriefSourceText(root)
	if strings.TrimSpace(text) == "" {
		return nil
	}
	labelTokens := projectBriefLabelTokens(root)
	markers := []string{
		" should include ",
		" must include ",
		" include ",
		" includes ",
		" including ",
		" should implement ",
		" must implement ",
		" implement ",
		" implements ",
		" should support ",
		" must support ",
		" support ",
		" supports ",
		" should detect ",
		" must detect ",
		" should allow ",
		" must allow ",
		" should let ",
		" must let ",
		" features:",
		" features include ",
	}
	seen := map[string]bool{}
	var phrases []string
	for _, sentence := range splitBriefSentences(text) {
		lower := " " + strings.ToLower(sentence) + " "
		for _, marker := range markers {
			idx := strings.Index(lower, marker)
			if idx < 0 {
				continue
			}
			segment := strings.TrimSpace(lower[idx+len(marker):])
			for _, phrase := range splitCapabilitySegment(segment) {
				if isValidationEvidenceCapabilityPhrase(phrase) {
					continue
				}
				phrase = stripCapabilityLabelTokens(phrase, labelTokens)
				if len(capabilityKeywords(phrase)) == 0 {
					continue
				}
				key := strings.ToLower(phrase)
				if seen[key] {
					continue
				}
				seen[key] = true
				phrases = append(phrases, phrase)
			}
		}
	}
	return phrases
}

func projectBriefLabelTokens(root Root) map[string]bool {
	out := map[string]bool{}
	for _, rel := range []string{
		"README.md",
		filepath.Join("docs", "product-specs", "vision.md"),
		filepath.Join("docs", "goals", "active.md"),
	} {
		data, err := root.RepoFS().ReadFile(rel)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(line)
			if !strings.HasPrefix(trimmed, "#") {
				continue
			}
			trimmed = strings.TrimLeft(trimmed, "#")
			trimmed = strings.TrimSpace(trimmed)
			if trimmed == "" {
				continue
			}
			for _, field := range strings.Fields(normalizeCapabilitySurface(trimmed)) {
				key := capabilityKeyword(field)
				if key == "" || capabilityLabelKeepWords[key] {
					continue
				}
				out[key] = true
			}
		}
	}
	return out
}

func stripCapabilityLabelTokens(phrase string, labels map[string]bool) string {
	if len(labels) == 0 {
		return phrase
	}
	fields := strings.Fields(normalizeCapabilitySurface(phrase))
	if len(fields) == 0 {
		return phrase
	}
	var kept []string
	removed := false
	for _, field := range fields {
		key := capabilityKeyword(field)
		if key != "" && labels[key] {
			removed = true
			continue
		}
		kept = append(kept, field)
	}
	candidate := cleanCapabilityPhrase(strings.Join(kept, " "))
	if candidate == "" || len(capabilityKeywords(candidate)) == 0 {
		if removed {
			return ""
		}
		return phrase
	}
	return candidate
}

func projectBriefSourceText(root Root) string {
	var b strings.Builder
	for _, rel := range []string{
		"README.md",
		filepath.Join("docs", "product-specs", "vision.md"),
		filepath.Join("docs", "goals", "active.md"),
	} {
		data, err := root.RepoFS().ReadFile(rel)
		if err != nil {
			continue
		}
		b.WriteByte('\n')
		b.Write(data)
	}
	return b.String()
}

func splitBriefSentences(text string) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	numberedListPattern := regexp.MustCompile(`^\d+\.\s+`)
	var normalized strings.Builder
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			normalized.WriteString(". ")
			continue
		}
		if strings.HasPrefix(trimmed, "- ") ||
			strings.HasPrefix(trimmed, "* ") ||
			strings.HasPrefix(trimmed, "#") ||
			numberedListPattern.MatchString(trimmed) {
			normalized.WriteString(". ")
		}
		normalized.WriteString(trimmed)
		normalized.WriteByte(' ')
	}
	fields := regexp.MustCompile(`[.!?]+`).Split(normalized.String(), -1)
	var out []string
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field != "" {
			out = append(out, field)
		}
	}
	return out
}

func splitCapabilitySegment(segment string) []string {
	segment = strings.TrimSpace(segment)
	segment = strings.Trim(segment, " .:-")
	segment = stripCapabilityCategoryPrefix(segment)
	if segment == "" {
		return nil
	}
	segment = regexp.MustCompile(`\b(and|plus)\b`).ReplaceAllString(segment, ",")
	rawParts := strings.FieldsFunc(segment, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n'
	})
	var phrases []string
	skipValidationTail := false
	for _, part := range rawParts {
		phrase := cleanCapabilityPhrase(part)
		if phrase == "" {
			continue
		}
		if isValidationEvidenceCapabilityPhrase(phrase) {
			skipValidationTail = true
			continue
		}
		if skipValidationTail && isValidationEvidenceTailPhrase(phrase) {
			continue
		}
		skipValidationTail = false
		if len(capabilityKeywords(phrase)) == 0 {
			continue
		}
		phrases = append(phrases, phrase)
	}
	return phrases
}

func stripCapabilityCategoryPrefix(segment string) string {
	idx := strings.Index(segment, ":")
	if idx < 0 {
		return segment
	}
	prefix := strings.ToLower(strings.TrimSpace(segment[:idx]))
	if strings.Contains(prefix, "mechanic") ||
		strings.Contains(prefix, "capabilit") ||
		strings.Contains(prefix, "feature") ||
		strings.Contains(prefix, "behavior") ||
		strings.Contains(prefix, "behaviour") {
		return strings.TrimSpace(segment[idx+1:])
	}
	return segment
}

func cleanCapabilityPhrase(phrase string) string {
	phrase = strings.TrimSpace(strings.ToLower(phrase))
	phrase = stripCapabilityReferenceTail(phrase)
	phrase = strings.Trim(phrase, "`*_ .:-")
	for _, prefix := range []string{"a ", "an ", "the "} {
		phrase = strings.TrimPrefix(phrase, prefix)
	}
	for _, suffix := range []string{" behavior", " behaviour", " feature", " features", " functionality", " flow", " capability", " capabilities"} {
		phrase = strings.TrimSuffix(phrase, suffix)
	}
	phrase = strings.Join(strings.Fields(phrase), " ")
	if len(phrase) < 3 {
		return ""
	}
	return phrase
}

func stripCapabilityReferenceTail(phrase string) string {
	lower := strings.ToLower(phrase)
	cut := len(phrase)
	for _, marker := range []string{
		" described in ",
		" described by ",
		" documented in ",
		" documented by ",
		" defined in ",
		" defined by ",
		" specified in ",
		" specified by ",
		" covered in ",
		" covered by ",
	} {
		if idx := strings.Index(lower, marker); idx >= 0 && idx < cut {
			cut = idx
		}
	}
	for _, marker := range []string{
		" docs/features/",
		"(docs/features/",
		"`docs/features/",
		"[docs/features/",
	} {
		if idx := strings.Index(lower, marker); idx >= 0 && idx < cut {
			cut = idx
		}
	}
	if cut < len(phrase) {
		return strings.TrimSpace(phrase[:cut])
	}
	return phrase
}

func isValidationEvidenceCapabilityPhrase(phrase string) bool {
	normalized := normalizeCapabilitySurface(phrase)
	if normalized == "" {
		return false
	}
	for _, marker := range []string{
		"evidence",
		"smoke evidence",
		"smoke test",
		"validation evidence",
		"validation instruction",
		"validation instructions",
		"manual validation",
		"reviewer validation",
		"test evidence",
		"prove",
		"proves",
		"proving",
		"verified by",
		"build artifact",
	} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	if strings.Contains(normalized, "reviewer") &&
		(strings.Contains(normalized, "confirm") ||
			strings.Contains(normalized, "verify") ||
			strings.Contains(normalized, "check")) {
		return true
	}
	return false
}

func isValidationEvidenceTailPhrase(phrase string) bool {
	switch normalizeCapabilitySurface(phrase) {
	case "mount", "mounts", "play", "plays", "load", "loads", "run", "runs":
		return true
	default:
		return false
	}
}

func featureScenarioSurface(content string) string {
	lower := strings.ToLower(content)
	start := strings.Index(lower, "## scenario schedule")
	if start < 0 {
		start = strings.Index(lower, "## scenarios")
	}
	if start < 0 {
		return ""
	}
	end := strings.Index(lower[start:], "\n## out of scope")
	if end >= 0 {
		return lower[start : start+end]
	}
	return lower[start:]
}

func featureScenarioOutlineSurface(content string) string {
	lower := strings.ToLower(content)
	var parts []string
	start := strings.Index(lower, "## scenario schedule")
	if start >= 0 {
		end := strings.Index(lower[start+1:], "\n## ")
		if end >= 0 {
			parts = append(parts, lower[start:start+1+end])
		} else {
			parts = append(parts, lower[start:])
		}
	}
	for _, line := range strings.Split(lower, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "### f-") {
			parts = append(parts, line)
		}
	}
	return strings.Join(parts, "\n")
}

func featureOutOfScopeSurface(content string) string {
	lower := strings.ToLower(content)
	start := strings.Index(lower, "## out of scope")
	if start < 0 {
		return ""
	}
	end := strings.Index(lower[start+1:], "\n## ")
	if end >= 0 {
		return lower[start : start+1+end]
	}
	return lower[start:]
}

func outOfScopeSurfaceRequiresDescoping(surface, phrase string) bool {
	for _, line := range strings.Split(surface, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "-")
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "##") {
			continue
		}
		if outOfScopeLineIsExplanation(line) {
			continue
		}
		if outOfScopeLineLeavesBasicCapabilityInScope(line) {
			continue
		}
		if capabilityPhraseCovered(line, phrase) {
			return true
		}
	}
	return false
}

func outOfScopeLineIsExplanation(line string) bool {
	normalized := normalizeCapabilitySurface(line)
	if strings.HasPrefix(normalized, "the following") ||
		strings.HasPrefix(normalized, "following ") ||
		strings.HasPrefix(normalized, "none ") {
		return true
	}
	for _, marker := range []string{
		"clear reason",
		"clear reasons",
		"explicit rationale",
		"explicit rationales",
		"listed under out of scope",
	} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func outOfScopeLineLeavesBasicCapabilityInScope(line string) bool {
	normalized := normalizeCapabilitySurface(line)
	switch normalized {
	case "animation", "animations", "animation polish", "animation only polish", "animation-only polish",
		"visual polish", "visual effects",
		"preview", "previews", "next piece preview", "next piece previews", "piece preview", "piece previews",
		"sound", "sounds", "sound effects", "audio", "audio feedback",
		"multiplayer", "multiplayer support", "multiplayer functionality",
		"mobile touch controls", "touch controls", "touch input",
		"hold piece", "hold queue", "hard drop":
		return true
	}
	for _, prefix := range []string{
		"animation for ",
		"animations for ",
		"animated ",
		"animation polish for ",
		"animation-only polish for ",
		"visual polish for ",
		"visual effects for ",
		"preview for ",
		"previews for ",
		"next piece preview for ",
		"sound for ",
		"sounds for ",
		"audio for ",
		"multiplayer for ",
		"mobile touch controls for ",
		"touch controls for ",
	} {
		if strings.HasPrefix(normalized, prefix) {
			return true
		}
	}
	if strings.Contains(normalized, "advanced") && strings.Contains(normalized, "beyond basic") {
		return true
	}
	if strings.Contains(normalized, "advanced") &&
		(strings.Contains(normalized, "scoring") ||
			strings.Contains(normalized, "score") ||
			strings.Contains(normalized, "combo") ||
			strings.Contains(normalized, "back to back") ||
			strings.Contains(normalized, "changes basic")) {
		return true
	}
	if strings.Contains(normalized, "combo") || strings.Contains(normalized, "back to back") {
		return true
	}
	if strings.Contains(normalized, "beyond") {
		return true
	}
	if strings.Contains(normalized, "high score") || strings.Contains(normalized, "persistence") || strings.Contains(normalized, "persisted") {
		return true
	}
	return false
}

func featureDescopedSurface(content string) string {
	lower := strings.ToLower(content)
	start := strings.Index(lower, "## descoped scenarios")
	if start < 0 {
		return ""
	}
	end := strings.Index(lower[start+1:], "\n## ")
	if end >= 0 {
		return lower[start : start+1+end]
	}
	return lower[start:]
}

func capabilityPhraseCovered(surface, phrase string) bool {
	surface = normalizeCapabilitySurface(surface)
	phrase = normalizeCapabilitySurface(phrase)
	if phrase == "" {
		return true
	}
	if strings.Contains(surface, phrase) {
		return true
	}
	surfaceKeys := capabilityKeywordSet(surface)
	keys := capabilityKeywords(phrase)
	if len(keys) == 0 {
		return true
	}
	if directionalMovementCapabilityCovered(surfaceKeys, keys) {
		return true
	}
	for _, key := range keys {
		if key == "move" && (surfaceKeys["left"] || surfaceKeys["right"] || surfaceKeys["down"] || (surfaceKeys["control"] && surfaceKeys["keyboard"])) {
			continue
		}
		if !surfaceKeys[key] {
			return false
		}
	}
	return true
}

func directionalMovementCapabilityCovered(surfaceKeys map[string]bool, keys []string) bool {
	requiredDirections := map[string]bool{}
	hasMovement := false
	for _, key := range keys {
		switch key {
		case "move":
			hasMovement = true
		case "left", "right", "down":
			requiredDirections[key] = true
		}
	}
	if !hasMovement || len(requiredDirections) < 2 {
		return false
	}
	for direction := range requiredDirections {
		if !surfaceKeys[direction] {
			return false
		}
	}
	return surfaceKeys["move"] || surfaceKeys["control"] || surfaceKeys["keyboard"]
}

func normalizeCapabilitySurface(text string) string {
	text = strings.ToLower(text)
	text = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(text, " ")
	return strings.Join(strings.Fields(text), " ")
}

func capabilityKeywordSet(text string) map[string]bool {
	out := map[string]bool{}
	for _, field := range strings.Fields(text) {
		if key := capabilityKeyword(field); key != "" {
			out[key] = true
		}
	}
	return out
}

func capabilityKeywords(phrase string) []string {
	seen := map[string]bool{}
	var out []string
	for _, field := range strings.Fields(normalizeCapabilitySurface(phrase)) {
		key := capabilityKeyword(field)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, key)
	}
	return out
}

func capabilityKeyword(token string) string {
	token = strings.TrimSpace(strings.ToLower(token))
	if token == "" || capabilityStopWords[token] {
		return ""
	}
	switch {
	case strings.HasPrefix(token, "rotat"):
		return "rotat"
	case strings.HasPrefix(token, "scor"):
		return "score"
	case strings.HasPrefix(token, "track"):
		return "score"
	case strings.HasPrefix(token, "clear"):
		return "clear"
	case strings.HasPrefix(token, "mov"):
		return "move"
	case token == "over" || strings.HasPrefix(token, "end"):
		return "gameover"
	case strings.HasPrefix(token, "restart"):
		return "restart"
	case strings.HasPrefix(token, "playfield"):
		return "playfield"
	case strings.HasPrefix(token, "keyboard"):
		return "keyboard"
	case strings.HasPrefix(token, "browser"):
		return "browser"
	case strings.HasPrefix(token, "canvas"):
		return "canvas"
	case strings.HasPrefix(token, "collision"):
		return "collision"
	case strings.HasPrefix(token, "lock"):
		return "lock"
	}
	for _, suffix := range []string{"ing", "ed", "es", "s"} {
		if len(token) > len(suffix)+3 && strings.HasSuffix(token, suffix) {
			token = strings.TrimSuffix(token, suffix)
			break
		}
	}
	if len(token) < 4 || capabilityStopWords[token] {
		return ""
	}
	return token
}

func projectBriefNamesGoBackend(root Root) bool {
	for _, rel := range []string{
		"README.md",
		filepath.Join("docs", "product-specs", "vision.md"),
	} {
		data, err := root.RepoFS().ReadFile(rel)
		if err != nil {
			continue
		}
		lower := strings.ToLower(string(data))
		for _, marker := range []string{"go backend", "golang backend", "go server", "golang server", "go cli", "golang cli"} {
			if strings.Contains(lower, marker) {
				return true
			}
		}
	}
	return false
}

var capabilityStopWords = map[string]bool{
	"a": true, "an": true, "and": true, "another": true, "are": true, "as": true, "be": true, "by": true,
	"basic": true, "can": true, "complete": true, "condition": true, "conditions": true, "described": true,
	"doc": true, "docs": true, "document": true, "documents": true, "documentation": true,
	"core": true, "detect": true, "detected": true, "detection": true, "display": true, "displayed": true, "displays": true,
	"fall": true, "falling": true, "feature": true, "features": true, "fill": true, "fills": true, "filled": true, "for": true, "from": true, "full": true,
	"gameplay": true, "handle": true, "handled": true, "game": true, "games": true, "in": true,
	"include": true, "includes": true, "including": true, "inspect": true, "inspected": true, "into": true, "local": true, "locally": true, "of": true,
	"markdown": true, "md": true,
	"mechanic": true, "mechanics": true, "on": true, "open": true, "opened": true, "or": true, "product": true, "project": true,
	"piece": true, "pieces": true, "playable": true, "player": true, "players": true, "reach": true, "reaches": true, "round": true, "rounds": true, "run": true, "see": true, "stack": true,
	"show": true, "showing": true, "shows": true, "that": true, "the": true, "to": true, "using": true, "user": true, "users": true,
	"usable": true, "useful": true, "version": true, "when": true, "with": true,
}

var capabilityLabelKeepWords = map[string]bool{
	"application": true,
	"board":       true,
	"calendar":    true,
	"chat":        true,
	"dashboard":   true,
	"editor":      true,
	"form":        true,
	"service":     true,
	"site":        true,
	"task":        true,
	"tracker":     true,
	"workflow":    true,
}

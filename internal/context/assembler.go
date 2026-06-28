/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/context-efficiency.md
- docs/features/F-005-agent-execution-runtime.md
*/
package context

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/greaveselliott/mars/internal/llm"
)

const (
	headerRole        = "## ROLE"
	headerRunMetadata = "## RUN METADATA"
	headerGuards      = "## GUARDRAILS"
	headerLearnings   = "## REPO LEARNINGS"
	headerKnowledge   = "## KNOWLEDGE ROUTES"
	headerSkills      = "## SKILLS"
	headerTrigger     = "## TRIGGER CONTEXT"
	headerCodeGraph   = "## CODE GRAPH CONTEXT"
	headerRepo        = "## REPO SUMMARY"
	headerTickets     = "## TICKET INDEX"
)

// block is an internal mutable slice used for budget trimming (lower truncPri drops first).
type block struct {
	name     string
	header   string
	body     string
	truncPri int // smaller = truncated earlier under budget pressure
}

// Assemble builds the additive system prompt with section headers (AD-006).
// Truncation order when over TokenBudget starts with low-priority repo summary,
// then trigger/knowledge-style routing sections. Role text is never truncated.
func Assemble(in Input) (system string, stats []SectionStat, err error) {
	role, err := loadRolePrompt(in)
	if err != nil {
		return "", nil, err
	}
	scope := strings.TrimSpace(in.RoleScope)

	var parts []block

	parts = append(parts, block{name: "role", header: headerRole, body: strings.TrimSpace(role), truncPri: 100})
	if md := runMetadataBody(in.CurrentTime); md != "" {
		parts = append(parts, block{name: "run_metadata", header: headerRunMetadata, body: md, truncPri: 100})
	}

	guardBodies := filterGuardrails(in.Guardrails, scope)
	if len(guardBodies) > 0 {
		var b strings.Builder
		for _, g := range guardBodies {
			title := strings.TrimSpace(g.Title)
			if title != "" {
				fmt.Fprintf(&b, "### %s\n%s\n\n", title, strings.TrimSpace(g.Body))
			} else {
				fmt.Fprintf(&b, "%s\n\n", strings.TrimSpace(g.Body))
			}
		}
		parts = append(parts, block{name: "guardrails", header: headerGuards, body: strings.TrimSpace(b.String()), truncPri: 80})
	}

	if strings.TrimSpace(in.Learnings) != "" {
		parts = append(parts, block{name: "learnings", header: headerLearnings, body: strings.TrimSpace(in.Learnings), truncPri: 60})
	}

	if len(in.KnowledgeRoutes) > 0 {
		var b strings.Builder
		for _, kr := range in.KnowledgeRoutes {
			when := strings.TrimSpace(kr.When)
			paths := strings.TrimSpace(kr.Paths)
			if when == "" && paths == "" {
				continue
			}
			if when == "" {
				when = "this repository"
			}
			if paths == "" {
				continue
			}
			fmt.Fprintf(&b, "- When working on %s, read %s\n", when, paths)
		}
		if b.Len() > 0 {
			parts = append(parts, block{name: "knowledge", header: headerKnowledge, body: strings.TrimSpace(b.String()), truncPri: 40})
		}
	}

	if len(in.Skills) > 0 {
		var b strings.Builder
		for _, sk := range in.Skills {
			title := strings.TrimSpace(sk.Name)
			body := strings.TrimSpace(sk.Body)
			if body == "" {
				continue
			}
			if title != "" {
				fmt.Fprintf(&b, "### %s\n%s\n\n", title, body)
			} else {
				fmt.Fprintf(&b, "%s\n\n", body)
			}
		}
		if b.Len() > 0 {
			parts = append(parts, block{name: "skills", header: headerSkills, body: strings.TrimSpace(b.String()), truncPri: 50})
		}
	}

	triggerBody := triggerContextBody(in.PayloadMode, in.Trigger)
	if strings.TrimSpace(triggerBody) != "" {
		parts = append(parts, block{name: "trigger", header: headerTrigger, body: strings.TrimSpace(triggerBody), truncPri: 20})
	}
	if strings.TrimSpace(in.TicketIndex) != "" {
		parts = append(parts, block{name: "tickets", header: headerTickets, body: strings.TrimSpace(in.TicketIndex), truncPri: 75})
	}
	if strings.TrimSpace(in.CodeGraphContext) != "" {
		parts = append(parts, block{name: "code_graph", header: headerCodeGraph, body: strings.TrimSpace(in.CodeGraphContext), truncPri: 70})
	}
	if strings.TrimSpace(in.RepoSummary) != "" {
		parts = append(parts, block{name: "repo", header: headerRepo, body: strings.TrimSpace(in.RepoSummary), truncPri: 10})
	}

	if in.TokenBudget > 0 {
		shrinkToBudget(&parts, in.TokenBudget)
	}

	var out strings.Builder
	for _, p := range parts {
		if strings.TrimSpace(p.body) == "" {
			continue
		}
		fmt.Fprintf(&out, "%s\n\n%s\n\n", p.header, p.body)
		n := llm.EstimateTokens([]llm.Message{{Role: "system", Content: p.header + "\n\n" + p.body}}, nil)
		stats = append(stats, SectionStat{Name: p.name, Tokens: n})
	}
	return strings.TrimSpace(out.String()), stats, nil
}

func triggerContextBody(payloadMode, trigger string) string {
	payloadMode = strings.TrimSpace(payloadMode)
	trigger = strings.TrimSpace(trigger)
	if payloadMode == "" {
		return trigger
	}
	if trigger == "" {
		return fmt.Sprintf("payload_mode: %s", payloadMode)
	}
	return fmt.Sprintf("payload_mode: %s\n\n%s", payloadMode, trigger)
}

func runMetadataBody(now time.Time) string {
	if now.IsZero() {
		return ""
	}
	zoneName, zoneOffset := now.Zone()
	if strings.TrimSpace(zoneName) == "" {
		zoneName = "local"
	}
	return fmt.Sprintf("- current_date: %s\n- current_time: %s\n- timezone: %s (%s)\n\nUse `current_date` for dated evidence paths, report dates, release entries, and ticket timestamps. Do not infer a date from examples or model memory; if this metadata is absent, omit the date or record it as unknown instead of inventing one.",
		now.Format("2006-01-02"),
		now.Format(time.RFC3339),
		zoneName,
		formatUTCOffset(zoneOffset),
	)
}

func formatUTCOffset(offsetSeconds int) string {
	sign := "+"
	if offsetSeconds < 0 {
		sign = "-"
		offsetSeconds = -offsetSeconds
	}
	hours := offsetSeconds / 3600
	minutes := (offsetSeconds % 3600) / 60
	return fmt.Sprintf("%s%02d:%02d", sign, hours, minutes)
}

func loadRolePrompt(in Input) (string, error) {
	path := strings.TrimSpace(in.RolePromptPath)
	if path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("context: read role prompt file %q: %w", path, err)
		}
		s := strings.TrimSpace(string(b))
		if s == "" {
			return "", fmt.Errorf("context: role prompt file %q is empty", path)
		}
		return s, nil
	}
	s := strings.TrimSpace(in.RolePrompt)
	if s == "" {
		return "", fmt.Errorf("context: role prompt is empty; set RolePrompt or RolePromptPath")
	}
	return in.RolePrompt, nil
}

func filterGuardrails(guards []Guardrail, roleScope string) []Guardrail {
	roleScope = strings.TrimSpace(strings.ToLower(roleScope))
	var out []Guardrail
	for _, g := range guards {
		sc := strings.TrimSpace(strings.ToLower(g.Scope))
		if sc == "" || sc == "all" || sc == roleScope {
			out = append(out, g)
		}
	}
	return out
}

func shrinkToBudget(parts *[]block, budget int) {
	for iter := 0; iter < 2000; iter++ {
		s := renderBlocks(*parts)
		if s == "" {
			return
		}
		got := llm.EstimateTokens([]llm.Message{{Role: "system", Content: s}}, nil)
		if got <= budget {
			return
		}
		// Find lowest truncPri with non-empty body (excluding role).
		idx := -1
		bestPri := int(^uint(0) >> 1)
		for i := range *parts {
			p := &(*parts)[i]
			if p.name == "role" || p.name == "run_metadata" {
				continue
			}
			if strings.TrimSpace(p.body) == "" {
				continue
			}
			if p.truncPri < bestPri {
				bestPri = p.truncPri
				idx = i
			}
		}
		if idx < 0 {
			// Cannot shrink further without touching the role prompt (forbidden by MH-004).
			return
		}
		b := &(*parts)[idx]
		newLen := len(b.body) * 3 / 4
		if newLen < 1 {
			b.body = ""
			continue
		}
		b.body = trimEndPreserveNewline(b.body, newLen)
		b.body += "\n\n(context: section truncated for token budget)"
	}
}

func trimEndPreserveNewline(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	return s[:maxBytes]
}

func renderBlocks(parts []block) string {
	var out strings.Builder
	for _, p := range parts {
		if strings.TrimSpace(p.body) == "" {
			continue
		}
		fmt.Fprintf(&out, "%s\n\n%s\n\n", p.header, p.body)
	}
	return out.String()
}

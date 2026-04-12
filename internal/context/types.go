package context

// Guardrail is one scoped policy block (YAML parsing is MH-014; callers pass structs).
type Guardrail struct {
	Scope string // empty means global (applies to all roles)
	Title string
	Body  string
}

// KnowledgeRoute is a single routing hint (YAML parsing deferred; callers pass structs).
type KnowledgeRoute struct {
	When  string // e.g. "TypeScript CI failures"
	Paths string // e.g. "docs/ci.md, scripts/check.ts"
}

// Skill is a structured instruction loaded from .harness/skills/.
type Skill struct {
	Name  string // skill name from YAML frontmatter
	Scope string // role scope filter (empty = all roles)
	Body  string // markdown body with instructions
}

// Input is everything needed to build the additive system prompt (MH-004 / AD-006).
type Input struct {
	RoleScope string // current role id for guardrail filtering, e.g. "engineer"

	// RolePrompt is inline role text. If RolePromptPath is set, the file overrides RolePrompt.
	RolePrompt     string
	RolePromptPath string

	Guardrails      []Guardrail
	KnowledgeRoutes []KnowledgeRoute
	Skills          []Skill // from .harness/skills/, filtered by role scope
	Trigger         string  // ticket body, CI excerpt, etc.
	RepoSummary     string  // directory tree or short manifest

	// Learnings is a pre-formatted text block of per-repo conventions, lessons, and excludes.
	Learnings string

	// TokenBudget is estimated tokens for the full system string; 0 = unlimited (MH-004).
	TokenBudget int
}

// SectionStat records estimated tokens for one emitted block (for logs / traces).
type SectionStat struct {
	Name   string
	Tokens int
}

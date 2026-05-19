/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/self-reflective-telemetry.md
- docs/features/F-012-self-improvement-loop.md
*/
package remediation

import (
	"sort"
	"strings"

	"github.com/greaveselliott/mars-harness/internal/telemetry"
)

// SafetyClass describes whether a recipe can be used automatically or should
// remain an operator-visible blocker until a later policy explicitly permits it.
type SafetyClass string

const (
	SafetyAutoSafe         SafetyClass = "auto_safe"
	SafetyOperatorRequired SafetyClass = "operator_required"
	SafetyApprovalRequired SafetyClass = "approval_required"
)

// AttemptStatus is the deterministic planning status for an applicable recipe.
type AttemptStatus string

const (
	AttemptReady                   AttemptStatus = "ready"
	AttemptSkippedOperatorRequired AttemptStatus = "skipped_operator_required"
	AttemptSkippedApprovalRequired AttemptStatus = "skipped_approval_required"
)

// Signal is the normalized failure context used to select deterministic
// remediation recipes before involving an LLM repair role.
type Signal struct {
	Category telemetry.FailureCategory
	Message  string
	Role     string
	RepoPath string
	Phase    string
}

// Recipe describes one deterministic remediation candidate. Recipes do not
// execute themselves; callers decide which ready recipes are allowed in a given
// trust and guardrail context.
type Recipe struct {
	ID              string
	Title           string
	Summary         string
	Target          string
	Categories      []telemetry.FailureCategory
	MessageContains []string
	Commands        []string
	CandidateFiles  []string
	Safety          SafetyClass
	Destructive     bool
	NextAction      string
}

// Attempt records what the planner would do with an applicable recipe.
type Attempt struct {
	RecipeID   string
	Status     AttemptStatus
	Safety     SafetyClass
	Reason     string
	Commands   []string
	NextAction string
}

// Plan is the deterministic remediation plan for one signal.
type Plan struct {
	Signal   Signal
	Attempts []Attempt
}

// Registry stores known deterministic recipes.
type Registry struct {
	recipes []Recipe
}

// NewRegistry returns a registry with recipes sorted by ID for stable output.
func NewRegistry(recipes []Recipe) Registry {
	out := append([]Recipe(nil), recipes...)
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return Registry{recipes: out}
}

// DefaultRegistry returns the initial deterministic remediation catalog.
func DefaultRegistry() Registry {
	return NewRegistry([]Recipe{
		{
			ID:              "dirty-worktree:blocker",
			Title:           "Dirty Working Tree Before Run",
			Summary:         "Detect user or generated changes before model work and return an actionable blocker instead of reverting.",
			Target:          "workspace",
			Categories:      []telemetry.FailureCategory{telemetry.CategoryWorkspaceHygiene, telemetry.CategoryGuardrailBlock},
			MessageContains: []string{"dirty working tree", "uncommitted changes", "worktree has local changes"},
			Commands:        []string{"git status --short"},
			CandidateFiles:  []string{"internal/tools/workspace_hygiene.go", "internal/serve/server.go"},
			Safety:          SafetyOperatorRequired,
			NextAction:      "Inspect git status, separate user work from generated churn, and retry only after the operator or owning role makes the worktree truthful.",
		},
		{
			ID:              "doctor:known-remediation",
			Title:           "Known Doctor Remediation",
			Summary:         "Surface doctor checks with known repair commands as deterministic guidance before model repair.",
			Target:          "doctor",
			Categories:      []telemetry.FailureCategory{telemetry.CategoryUnknown},
			MessageContains: []string{"doctor", "known remediation", "health check failed"},
			Commands:        []string{"mars-harness doctor --repo <repo>"},
			CandidateFiles:  []string{"internal/doctor", "docs/product-specs/product-surface.md"},
			Safety:          SafetyOperatorRequired,
			NextAction:      "Run doctor, apply the named non-destructive remediation command, and record any missing optional dependency honestly.",
		},
		{
			ID:              "generated-docs:update-missing-defaults",
			Title:           "Missing Generated Harness Defaults",
			Summary:         "Fill missing generated target defaults through the non-destructive update harness path.",
			Target:          "generated-target",
			Categories:      []telemetry.FailureCategory{telemetry.CategoryManifestError, telemetry.CategoryUnknown},
			MessageContains: []string{"missing generated docs", "operating-model drift", "missing harness defaults"},
			Commands:        []string{"mars-harness update harness --repo <repo>"},
			CandidateFiles:  []string{"internal/scanner/init.go", "internal/updatecheck"},
			Safety:          SafetyAutoSafe,
			NextAction:      "Run update harness to write missing defaults only; stale user-owned files remain reported rather than overwritten.",
		},
		{
			ID:              "manifest:validate-or-init",
			Title:           "Missing Or Invalid Manifest",
			Summary:         "Detect missing or invalid .harness/manifest.yaml and route to init, upgrade, or manifest repair.",
			Target:          "manifest",
			Categories:      []telemetry.FailureCategory{telemetry.CategoryManifestError},
			MessageContains: []string{"manifest", ".harness/manifest.yaml", "bundle"},
			Commands:        []string{"mars-harness init --repo <repo>", "mars-harness upgrade --repo <repo>"},
			CandidateFiles:  []string{"internal/bundle", "internal/scanner/init.go"},
			Safety:          SafetyOperatorRequired,
			NextAction:      "Initialize missing harness scaffolding or repair the invalid manifest without overwriting user-owned role configuration.",
		},
		{
			ID:              "model-artifact:checksum-mismatch",
			Title:           "Model Artifact Checksum Mismatch",
			Summary:         "Treat corrupt model or binary artifacts as explicit cache repair work, not a blind retry.",
			Target:          "models",
			Categories:      []telemetry.FailureCategory{telemetry.CategoryModelUnavailable, telemetry.CategoryInferenceCrash},
			MessageContains: []string{"checksum mismatch", "sha256 mismatch", "corrupt model", "corrupt artifact"},
			Commands:        []string{"mars-harness setup", "mars-harness update tool"},
			CandidateFiles:  []string{"internal/models", "internal/selfupdate"},
			Safety:          SafetyApprovalRequired,
			Destructive:     true,
			NextAction:      "Keep the corrupt artifact for inspection unless the operator approves cache cleanup or a fresh verified download path.",
		},
		{
			ID:              "optional-tool:install-guidance",
			Title:           "Missing Optional Tool Guidance",
			Summary:         "Surface absent optional tools as install, skip, or blocker guidance instead of false success.",
			Target:          "tooling",
			Categories:      []telemetry.FailureCategory{telemetry.CategoryToolTimeout, telemetry.CategoryUnknown},
			MessageContains: []string{"missing optional tool", "optional tool", "optional dependency", "tool not found", "not found in path", "llama-server not found", "podman not found", "gh not found", "github cli not found"},
			Commands:        []string{"mars-harness doctor --repo <repo>"},
			CandidateFiles:  []string{"internal/doctor", "internal/setup", "docs/design-docs/self-reflective-telemetry.md"},
			Safety:          SafetyOperatorRequired,
			NextAction:      "Install the named optional tool only when the workflow needs it, or record a skip/blocker; do not mark remediation successful merely because the tool is optional.",
		},
		{
			ID:              "scanner:dedupe-duplicate-tickets",
			Title:           "Repeated Scanner Duplicate Tickets",
			Summary:         "Route duplicate scanner findings through canonical ticket_create dedupe instead of creating more backlog churn.",
			Target:          "scanner",
			Categories:      []telemetry.FailureCategory{telemetry.CategoryUnknown},
			MessageContains: []string{"duplicate ticket", "scanner duplicate", "repeated scanner"},
			Commands:        []string{"mars-harness scan --repo <repo>"},
			CandidateFiles:  []string{"internal/scanner", "internal/tools/ticket_create.go"},
			Safety:          SafetyOperatorRequired,
			NextAction:      "Inspect existing ticket dedupe keys and update the matching ticket instead of adding another scanner ticket.",
		},
		{
			ID:              "stale-ticket:drain-in-progress",
			Title:           "Stale In-Progress Ticket Drain",
			Summary:         "Route stale in-progress tickets to explicit drain, blocker, or Janitor handling before new backlog work.",
			Target:          "tickets",
			Categories:      []telemetry.FailureCategory{telemetry.CategoryStaleTicket},
			MessageContains: []string{"stale in-progress ticket", "stale ticket", "stale_in_progress_ticket"},
			Commands:        []string{"mars-harness scan --repo <repo>"},
			CandidateFiles:  []string{"internal/scanner", "docs/tickets/in-progress/"},
			Safety:          SafetyOperatorRequired,
			NextAction:      "Complete, block, or return the stale ticket with blocker metadata before claiming ordinary backlog work.",
		},
		{
			ID:              "dependency:sync-before-repair",
			Title:           "Missing Dependency Setup",
			Summary:         "Use dependency_sync with workspace hygiene preflight instead of asking an LLM to guess install commands.",
			Target:          "dependencies",
			Categories:      []telemetry.FailureCategory{telemetry.CategoryWorkspaceHygiene, telemetry.CategoryToolTimeout},
			MessageContains: []string{"missing dependency setup", "dependency setup", "node_modules", "package manager"},
			Commands:        []string{"mars-harness tools run dependency_sync --repo <repo> --trust contributor"},
			CandidateFiles:  []string{"internal/tools/dependency_sync.go", "internal/tools/workspace_hygiene.go"},
			Safety:          SafetyOperatorRequired,
			NextAction:      "Run dependency_sync only when package-manager mutation is allowed and generated artifacts are ignored or tracked deliberately.",
		},
	})
}

// List returns the known recipes in stable order.
func (r Registry) List() []Recipe {
	return append([]Recipe(nil), r.recipes...)
}

// Find returns a recipe by ID.
func (r Registry) Find(id string) (Recipe, bool) {
	for _, recipe := range r.recipes {
		if recipe.ID == id {
			return recipe, true
		}
	}
	return Recipe{}, false
}

// Applicable returns recipes that match a signal by category or message.
func (r Registry) Applicable(signal Signal) []Recipe {
	var out []Recipe
	for _, recipe := range r.recipes {
		if recipe.AppliesTo(signal) {
			out = append(out, recipe)
		}
	}
	return out
}

// Plan returns deterministic attempts for applicable recipes.
func (r Registry) Plan(signal Signal) Plan {
	recipes := r.Applicable(signal)
	attempts := make([]Attempt, 0, len(recipes))
	for _, recipe := range recipes {
		status := AttemptReady
		reason := "recipe is auto-safe for deterministic execution"
		if recipe.Safety == SafetyApprovalRequired || recipe.Destructive {
			status = AttemptSkippedApprovalRequired
			reason = "recipe requires explicit guardrail or operator approval"
		} else if recipe.Safety == SafetyOperatorRequired {
			status = AttemptSkippedOperatorRequired
			reason = "recipe requires operator or role-owned confirmation before mutation"
		}
		attempts = append(attempts, Attempt{
			RecipeID:   recipe.ID,
			Status:     status,
			Safety:     recipe.Safety,
			Reason:     reason,
			Commands:   append([]string(nil), recipe.Commands...),
			NextAction: recipe.NextAction,
		})
	}
	return Plan{Signal: signal, Attempts: attempts}
}

// AppliesTo reports whether a recipe matches a normalized signal.
func (r Recipe) AppliesTo(signal Signal) bool {
	message := strings.ToLower(signal.Message)
	for _, needle := range r.MessageContains {
		if strings.Contains(message, strings.ToLower(needle)) {
			return true
		}
	}
	if signal.Category == telemetry.CategoryUnknown && len(r.MessageContains) > 0 {
		return false
	}
	if matchesCategory(r.Categories, signal.Category) {
		return true
	}
	return false
}

func matchesCategory(categories []telemetry.FailureCategory, category telemetry.FailureCategory) bool {
	if category == "" {
		return false
	}
	for _, candidate := range categories {
		if candidate == category {
			return true
		}
	}
	return false
}

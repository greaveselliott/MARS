/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/tools-glossary.md
- docs/design-docs/harness-operating-model.md
- docs/features/F-005-agent-execution-runtime.md
*/
package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/greaveselliott/mars/internal/bundle"
	"github.com/greaveselliott/mars/internal/personas"
	"github.com/greaveselliott/mars/internal/roleregistry"
	"gopkg.in/yaml.v3"
)

const personaCreateSchema = `{
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "role_key": { "type": "string", "description": "Role key in lower-kebab-case." },
    "title": { "type": "string", "description": "Human-readable persona title." },
    "scope": { "type": "string", "enum": ["universal", "foundation", "deployed"], "description": "Persona scope metadata." },
    "domain": { "type": "string", "description": "Canonical operating domain." },
    "mode": { "type": "string", "description": "Role mode in lower-kebab-case." },
    "category": { "type": "string", "description": "Persona category such as default, optional-advisory, or recovery." },
    "modus_operandi": { "type": "string", "description": "One-sentence operating style." },
    "priorities": { "type": "array", "items": { "type": "string" }, "description": "Ordered persona priorities." },
    "owns": { "type": "array", "items": { "type": "string" }, "description": "Responsibilities the persona owns." },
    "does_not_own": { "type": "array", "items": { "type": "string" }, "description": "Responsibilities outside the persona boundary." },
    "best_feedback_format": { "type": "array", "items": { "type": "string" }, "description": "Preferred feedback structure." },
    "feedback_i_need": { "type": "array", "items": { "type": "string" }, "description": "Details this persona needs from others." },
    "feedback_i_give": { "type": "array", "items": { "type": "string" }, "description": "Feedback this persona gives downstream or upstream." },
    "stop_conditions": { "type": "array", "items": { "type": "string" }, "description": "Conditions where the persona should stop or route elsewhere." },
    "orchestrator_handoff": { "type": "array", "items": { "type": "string" }, "description": "How this persona should hand off through Orchestrator." },
    "activate": { "type": "boolean", "description": "Whether to add the role to .harness/manifest.yaml." },
    "tools": { "type": "array", "items": { "type": "string" }, "description": "Optional tool allowlist when activate is true." },
    "model": { "type": "string", "description": "Optional model when activate is true." },
    "trust_level": { "type": "string", "description": "Optional trust level when activate is true." },
    "overwrite": { "type": "boolean", "description": "Overwrite generated persona manual or role prompt if they already exist." }
  },
  "required": ["role_key", "title", "scope", "domain", "mode", "category", "modus_operandi", "priorities", "owns", "does_not_own", "best_feedback_format", "feedback_i_need", "feedback_i_give", "stop_conditions", "orchestrator_handoff"]
}`

type personaCreateArgs struct {
	RoleKey             string   `json:"role_key"`
	Title               string   `json:"title"`
	Scope               string   `json:"scope"`
	Domain              string   `json:"domain"`
	Mode                string   `json:"mode"`
	Category            string   `json:"category"`
	ModusOperandi       string   `json:"modus_operandi"`
	Priorities          []string `json:"priorities"`
	Owns                []string `json:"owns"`
	DoesNotOwn          []string `json:"does_not_own"`
	BestFeedbackFormat  []string `json:"best_feedback_format"`
	FeedbackINeed       []string `json:"feedback_i_need"`
	FeedbackIGive       []string `json:"feedback_i_give"`
	StopConditions      []string `json:"stop_conditions"`
	OrchestratorHandoff []string `json:"orchestrator_handoff"`
	Activate            bool     `json:"activate"`
	Tools               []string `json:"tools"`
	Model               string   `json:"model"`
	TrustLevel          string   `json:"trust_level"`
	Overwrite           bool     `json:"overwrite"`
}

func registerPersonaCreate(r *Registry) error {
	return r.Register(
		"persona_create",
		"Scaffold a repo-local agent persona manual, role prompt, registry row, and optional manifest role.",
		json.RawMessage(personaCreateSchema),
		handlePersonaCreate,
	)
}

func handlePersonaCreate(_ context.Context, root Root, raw json.RawMessage) (ToolResult, error) {
	var args personaCreateArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return ToolResult{}, fmt.Errorf("persona_create: parse arguments: %w", err)
	}
	p, err := args.persona()
	if err != nil {
		return ToolResult{}, err
	}
	if err := personas.Validate(p); err != nil {
		return ToolResult{}, fmt.Errorf("persona_create: %w", err)
	}

	manualPath := filepath.ToSlash(filepath.Join(personas.ManualDir, p.RoleKey+".md"))
	promptPath := filepath.ToSlash(filepath.Join(".harness", "roles", p.RoleKey+".md"))
	if err := writePersonaFile(root, manualPath, personas.RenderManual(p), args.Overwrite); err != nil {
		return ToolResult{}, err
	}
	if err := writePersonaFile(root, promptPath, renderPersonaPrompt(p), args.Overwrite); err != nil {
		return ToolResult{}, err
	}
	if err := upsertPersonaRegistryRow(root, p, args.Scope); err != nil {
		return ToolResult{}, err
	}
	if args.Activate {
		if err := upsertManifestRole(root, p, args); err != nil {
			return ToolResult{}, err
		}
	}

	lines := []string{
		"persona created",
		"manual: " + manualPath,
		"prompt: " + promptPath,
		"registry: " + roleregistry.RegistryPath,
	}
	if args.Activate {
		lines = append(lines, "manifest: .harness/manifest.yaml")
	}
	if args.Scope == "foundation" {
		lines = append(lines, "follow-up: add the canonical foundation persona to internal/personas before treating it as a foundation default")
	}
	return ToolResult{Output: strings.Join(lines, "\n")}, nil
}

func (a personaCreateArgs) persona() (personas.Persona, error) {
	scope := strings.TrimSpace(a.Scope)
	switch scope {
	case "universal", "foundation", "deployed":
	default:
		return personas.Persona{}, fmt.Errorf("persona_create: scope must be universal, foundation, or deployed")
	}
	return personas.Persona{
		RoleKey:             strings.TrimSpace(a.RoleKey),
		Title:               strings.TrimSpace(a.Title),
		Domain:              strings.TrimSpace(a.Domain),
		Mode:                strings.TrimSpace(a.Mode),
		Category:            strings.TrimSpace(a.Category),
		ModusOperandi:       strings.TrimSpace(a.ModusOperandi),
		Priorities:          cleanStrings(a.Priorities),
		Owns:                cleanStrings(a.Owns),
		DoesNotOwn:          cleanStrings(a.DoesNotOwn),
		BestFeedbackFormat:  cleanStrings(a.BestFeedbackFormat),
		FeedbackINeed:       cleanStrings(a.FeedbackINeed),
		FeedbackIGive:       cleanStrings(a.FeedbackIGive),
		StopConditions:      cleanStrings(a.StopConditions),
		OrchestratorHandoff: cleanStrings(a.OrchestratorHandoff),
	}, nil
}

func renderPersonaPrompt(p personas.Persona) string {
	return fmt.Sprintf(`# %s

%s
## Role

This role was scaffolded by persona_create. Keep this prompt aligned with
%s/%s.md and update the repo-specific operating instructions below.

## Prompt

Describe the role-specific workflow, required reads, evidence expectations, and
dispatch disposition rules here before activating the role.
`, p.Title, personas.RenderPromptManual(p), personas.ManualDir, p.RoleKey)
}

func writePersonaFile(root Root, rel, content string, overwrite bool) error {
	exists := false
	if _, err := root.RepoFS().Stat(rel); err == nil {
		exists = true
		if !overwrite {
			return fmt.Errorf("persona_create: %s already exists; pass overwrite true to replace it", rel)
		}
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("persona_create: stat %s: %w", rel, err)
	}
	if exists {
		if err := atomicWriteRepositoryFile(root, rel, []byte(content), 0o644); err != nil {
			return fmt.Errorf("persona_create: write %s: %w", rel, err)
		}
		return nil
	}
	if err := createExclusiveRepositoryFile(root, rel, []byte(content), 0o644); err != nil {
		return fmt.Errorf("persona_create: write %s: %w", rel, err)
	}
	return nil
}

func upsertPersonaRegistryRow(root Root, p personas.Persona, scope string) error {
	content := ""
	exists := false
	if data, err := root.RepoFS().ReadFile(roleregistry.RegistryPath); err == nil {
		exists = true
		content = string(data)
	} else if errors.Is(err, os.ErrNotExist) {
		content = roleregistry.DefaultMarkdown()
	} else {
		return fmt.Errorf("persona_create: read %s: %w", roleregistry.RegistryPath, err)
	}
	if registryContainsRole(content, p.RoleKey) {
		return nil
	}
	row := fmt.Sprintf("| `%s` | custom | %s | `%s` | persona_create `%s` scope | dispatch-only | none yet | target-owned persona trust | persona manual, feedback contract, and stop conditions | reasoning | persona clarity and handoff quality | return disposition to orchestrator with explicit handoff or feedback |\n",
		p.RoleKey, p.Domain, p.Mode, scope)
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += row
	var err error
	if exists {
		err = atomicWriteRepositoryFile(root, roleregistry.RegistryPath, []byte(content), 0o644)
	} else {
		err = createExclusiveRepositoryFile(root, roleregistry.RegistryPath, []byte(content), 0o644)
	}
	if err != nil {
		return fmt.Errorf("persona_create: write %s: %w", roleregistry.RegistryPath, err)
	}
	return nil
}

func upsertManifestRole(root Root, p personas.Persona, args personaCreateArgs) error {
	const manifestPath = ".harness/manifest.yaml"
	data, err := root.RepoFS().ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("persona_create: read .harness/manifest.yaml: %w", err)
	}
	var manifest bundle.Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("persona_create: parse .harness/manifest.yaml: %w", err)
	}
	if manifest.Roles == nil {
		manifest.Roles = map[string]bundle.RoleConfig{}
	}
	if _, exists := manifest.Roles[p.RoleKey]; exists && !args.Overwrite {
		return nil
	}
	tools := cleanStrings(args.Tools)
	if len(tools) == 0 {
		tools = []string{"file_read", "grep", "record_decision", "job_disposition_record"}
	}
	model := strings.TrimSpace(args.Model)
	if model == "" {
		model = "reasoning"
	}
	trust := strings.TrimSpace(args.TrustLevel)
	if trust == "" {
		trust = "contributor"
	}
	manifest.Roles[p.RoleKey] = bundle.RoleConfig{
		Prompt:     "roles/" + p.RoleKey + ".md",
		Domain:     p.Domain,
		Mode:       p.Mode,
		Model:      model,
		TrustLevel: trust,
		Tools:      tools,
	}
	out, err := yaml.Marshal(&manifest)
	if err != nil {
		return fmt.Errorf("persona_create: render .harness/manifest.yaml: %w", err)
	}
	if err := atomicWriteRepositoryFile(root, manifestPath, out, 0o644); err != nil {
		return fmt.Errorf("persona_create: write .harness/manifest.yaml: %w", err)
	}
	return nil
}

func registryContainsRole(content, roleKey string) bool {
	return strings.Contains(content, "| `"+roleKey+"` |") || strings.Contains(content, "| "+roleKey+" |")
}

func cleanStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

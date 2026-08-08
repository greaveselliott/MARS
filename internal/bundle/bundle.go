/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/context-efficiency.md
- docs/design-docs/documentation-sync-architecture.md
- docs/design-docs/role-customization.md
- docs/features/F-004-target-harness-lifecycle.md
- docs/features/F-005-agent-execution-runtime.md
- docs/features/F-019-typescript-monorepo-docsync.md
*/
package bundle

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/greaveselliott/mars/internal/guardrails"
	"github.com/greaveselliott/mars/internal/trust"
	"gopkg.in/yaml.v3"
)

const harnessDir = ".harness"
const manifestFile = "manifest.yaml"

// Manifest is the .harness/manifest.yaml structure.
type Manifest struct {
	Name              string                `yaml:"name"`
	Description       string                `yaml:"description"`
	OrchestrationMode string                `yaml:"orchestration_mode"`
	DocSync           DocSyncConfig         `yaml:"docsync"`
	Roles             map[string]RoleConfig `yaml:"roles"`
}

// DocSyncConfig selects authored source files for the documentation-sync
// operating model. Empty fields use the built-in defaults in internal/docsync.
type DocSyncConfig struct {
	IncludeRoots      []string `yaml:"include_roots"`
	IncludeExtensions []string `yaml:"include_extensions"`
	ExcludeGlobs      []string `yaml:"exclude_globs"`
}

// LoadDocSyncConfig reads only the optional DocSync selection from a target
// manifest. It does not require roles so audit configuration can be validated
// independently before the full harness is loaded.
func LoadDocSyncConfig(repoRoot string) (DocSyncConfig, bool, error) {
	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" {
		return DocSyncConfig{}, false, fmt.Errorf("bundle: repo root path is empty — pass --repo <path> to specify the target repository")
	}
	path := filepath.Join(repoRoot, harnessDir, manifestFile)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return DocSyncConfig{}, false, nil
	}
	if err != nil {
		return DocSyncConfig{}, false, fmt.Errorf("bundle: read %s: %w", path, err)
	}
	var manifest struct {
		DocSync DocSyncConfig `yaml:"docsync"`
	}
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return DocSyncConfig{}, false, fmt.Errorf("bundle: parse %s: %w — check YAML syntax", path, err)
	}
	return manifest.DocSync, true, nil
}

// RoleConfig defines a single role's configuration within a bundle.
type RoleConfig struct {
	Prompt      string   `yaml:"prompt"`
	Domain      string   `yaml:"domain"`
	Mode        string   `yaml:"mode"`
	Model       string   `yaml:"model"`
	TrustLevel  string   `yaml:"trust_level"`
	Tools       []string `yaml:"tools"`
	Guardrails  []string `yaml:"guardrails"`
	Knowledge   []string `yaml:"knowledge"`
	Triggers    []string `yaml:"triggers"`
	Then        []string `yaml:"then"`
	IdleThen    []string `yaml:"idle_then"`
	Schedule    string   `yaml:"schedule"`
	MaxTurns    int      `yaml:"max_turns"`
	ContextSize int      `yaml:"context_size"`
}

// Load reads .harness/manifest.yaml from repoRoot.
func Load(repoRoot string) (*Manifest, error) {
	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" {
		return nil, fmt.Errorf("bundle: repo root path is empty — pass --repo <path> to specify the target repository")
	}

	dir := filepath.Join(repoRoot, harnessDir)
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("bundle: missing %s/ directory in %s — run `mars init` to create one", harnessDir, repoRoot)
	}
	if err != nil {
		return nil, fmt.Errorf("bundle: stat %s: %w", dir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("bundle: %s exists but is not a directory — remove it and run `mars init`", dir)
	}

	path := filepath.Join(dir, manifestFile)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("bundle: missing %s in %s — run `mars init` to scaffold the bundle", manifestFile, dir)
	}
	if err != nil {
		return nil, fmt.Errorf("bundle: read %s: %w", path, err)
	}

	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("bundle: parse %s: %w — check YAML syntax", path, err)
	}

	if m.Name == "" {
		return nil, fmt.Errorf("bundle: manifest %s is missing required field 'name'", path)
	}
	if len(m.Roles) == 0 {
		return nil, fmt.Errorf("bundle: manifest %s defines no roles — add at least one role under the 'roles' key", path)
	}
	switch strings.TrimSpace(m.OrchestrationMode) {
	case "", "legacy", "dispatch":
	default:
		return nil, fmt.Errorf("bundle: manifest %s has invalid orchestration_mode %q — use legacy or dispatch", path, m.OrchestrationMode)
	}

	for name, role := range m.Roles {
		if strings.TrimSpace(role.Prompt) == "" {
			return nil, fmt.Errorf("bundle: role %q has no prompt path — set 'prompt' to a file relative to %s/", name, harnessDir)
		}
		if rawTrust := strings.TrimSpace(role.TrustLevel); rawTrust != "" {
			if _, ok := trust.ParseLevel(rawTrust); !ok {
				return nil, fmt.Errorf("bundle: role %q has invalid trust_level %q — use observer, contributor, or autonomous", name, role.TrustLevel)
			}
		}
		for _, target := range role.Then {
			if _, ok := m.Roles[target]; !ok {
				return nil, fmt.Errorf("bundle: role %q chains to %q via 'then' but %q is not defined in the manifest", name, target, target)
			}
		}
		for _, target := range role.IdleThen {
			if _, ok := m.Roles[target]; !ok {
				return nil, fmt.Errorf("bundle: role %q chains to %q via 'idle_then' but %q is not defined in the manifest", name, target, target)
			}
		}
		if s := strings.TrimSpace(role.Schedule); s != "" {
			if !isValidSchedule(s) {
				return nil, fmt.Errorf("bundle: role %q has invalid schedule %q — use a named preset (hourly, daily, weekly, monthly) or a 5-field cron expression", name, s)
			}
		}
	}

	return &m, nil
}

// DispatchMode reports whether the manifest has opted into disposition-driven routing.
func (m Manifest) DispatchMode() bool {
	return strings.TrimSpace(m.OrchestrationMode) == "dispatch"
}

// DisplayHandoff returns the role handoff label shown to operators. Dispatch
// mode routes terminal dispositions through the Orchestrator rather than
// exposing legacy role-to-role chains as the next hop.
func (m Manifest) DisplayHandoff(roleName string) []string {
	role, ok := m.Roles[roleName]
	if !ok {
		return nil
	}
	if m.DispatchMode() {
		if roleName == "orchestrator" {
			return []string{"selected role"}
		}
		if _, ok := m.Roles["orchestrator"]; ok {
			return []string{"orchestrator"}
		}
		return nil
	}
	return role.Then
}

var schedulePresets = map[string]bool{
	"hourly":  true,
	"daily":   true,
	"weekly":  true,
	"monthly": true,
}

func isValidSchedule(s string) bool {
	if schedulePresets[s] {
		return true
	}
	fields := strings.Fields(s)
	return len(fields) == 5
}

// RolePrompt reads the prompt file for a named role.
func (m *Manifest) RolePrompt(repoRoot, roleName string) (string, error) {
	role, ok := m.Roles[roleName]
	if !ok {
		available := make([]string, 0, len(m.Roles))
		for name := range m.Roles {
			available = append(available, name)
		}
		return "", fmt.Errorf("bundle: role %q not found in manifest; available roles: %s", roleName, strings.Join(available, ", "))
	}

	promptPath := filepath.Join(repoRoot, harnessDir, role.Prompt)
	data, err := os.ReadFile(promptPath)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("bundle: prompt file %q for role %q not found — create %s", role.Prompt, roleName, promptPath)
	}
	if err != nil {
		return "", fmt.Errorf("bundle: read prompt for role %q: %w", roleName, err)
	}

	content := strings.TrimSpace(string(data))
	if content == "" {
		return "", fmt.Errorf("bundle: prompt file %q for role %q is empty — add role instructions", role.Prompt, roleName)
	}

	return content, nil
}

type guardrailFile struct {
	Rules []guardrails.Rule `yaml:"rules"`
}

// LoadGuardrails reads the guardrail YAML files referenced by a role.
func (m *Manifest) LoadGuardrails(repoRoot, roleName string) ([]guardrails.Rule, error) {
	role, ok := m.Roles[roleName]
	if !ok {
		return nil, fmt.Errorf("bundle: role %q not found in manifest", roleName)
	}
	var rules []guardrails.Rule
	for _, ref := range role.Guardrails {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		path, err := resolveHarnessPath(repoRoot, ref)
		if err != nil {
			return nil, err
		}
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("bundle: guardrail file %q for role %q not found — create %s", ref, roleName, path)
		}
		if err != nil {
			return nil, fmt.Errorf("bundle: read guardrail file %q: %w", ref, err)
		}
		var gf guardrailFile
		if err := yaml.Unmarshal(data, &gf); err != nil {
			return nil, fmt.Errorf("bundle: parse guardrail file %q: %w", ref, err)
		}
		for _, r := range gf.Rules {
			if strings.TrimSpace(r.Scope) == "" {
				r.Scope = "global"
			}
			if r.Severity == "" {
				r.Severity = guardrails.SeverityAdvisory
			}
			if r.CreatedAt.IsZero() {
				r.CreatedAt = time.Now().UTC()
			}
			rules = append(rules, r)
		}
	}
	return rules, nil
}

// KnowledgeRoute is the manifest-level route format read from knowledge files.
type KnowledgeRoute struct {
	When  string `yaml:"when"`
	Paths string `yaml:"paths"`
}

type knowledgeFile struct {
	Routes []KnowledgeRoute `yaml:"routes"`
}

// LoadKnowledgeRoutes reads knowledge-route files referenced by a role.
func (m *Manifest) LoadKnowledgeRoutes(repoRoot, roleName string) ([]KnowledgeRoute, error) {
	role, ok := m.Roles[roleName]
	if !ok {
		return nil, fmt.Errorf("bundle: role %q not found in manifest", roleName)
	}
	var routes []KnowledgeRoute
	for _, ref := range role.Knowledge {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		path, err := resolveHarnessPath(repoRoot, ref)
		if err != nil {
			return nil, err
		}
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("bundle: knowledge file %q for role %q not found — create %s", ref, roleName, path)
		}
		if err != nil {
			return nil, fmt.Errorf("bundle: read knowledge file %q: %w", ref, err)
		}
		var kf knowledgeFile
		if err := yaml.Unmarshal(data, &kf); err == nil && len(kf.Routes) > 0 {
			routes = append(routes, kf.Routes...)
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "-"))
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			routes = append(routes, KnowledgeRoute{When: filepath.Base(ref), Paths: line})
		}
	}
	return routes, nil
}

func resolveHarnessPath(repoRoot, ref string) (string, error) {
	clean := filepath.Clean(ref)
	if filepath.IsAbs(clean) {
		return "", fmt.Errorf("bundle: reference %q must be relative to .harness/", ref)
	}
	path := filepath.Join(repoRoot, harnessDir, clean)
	rel, err := filepath.Rel(filepath.Join(repoRoot, harnessDir), path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("bundle: reference %q escapes .harness/", ref)
	}
	return path, nil
}

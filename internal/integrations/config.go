/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/board-driven-integrations.md
- docs/features/F-013-board-driven-integrations.md
*/
package integrations

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const integrationsPath = ".harness/integrations.yaml"

const (
	FlowProfileCEOLed      = "ceo-led"
	FlowProfileBoardDriven = "board-driven"

	DeliveryModeTrunk       = "trunk"
	DeliveryModePullRequest = "pull_request"

	SectionJIRA     = "jira"
	SectionFigma    = "figma"
	SectionDelivery = "delivery"
)

// Config is the optional .harness/integrations.yaml v1 shape.
type Config struct {
	Version       int                 `yaml:"version" json:"version"`
	FlowProfile   string              `yaml:"flow_profile" json:"flow_profile"`
	Ingestion     IngestionConfig     `yaml:"ingestion,omitempty" json:"ingestion,omitempty"`
	DesignSources DesignSourcesConfig `yaml:"design_sources,omitempty" json:"design_sources,omitempty"`
	Delivery      DeliveryConfig      `yaml:"delivery,omitempty" json:"delivery,omitempty"`
}

type IngestionConfig struct {
	JIRA JIRAConfig `yaml:"jira,omitempty" json:"jira,omitempty"`
}

type JIRAConfig struct {
	Enabled          bool                 `yaml:"enabled" json:"enabled"`
	BaseURL          string               `yaml:"base_url,omitempty" json:"base_url,omitempty"`
	Auth             JIRAAuthConfig       `yaml:"auth,omitempty" json:"auth,omitempty"`
	WebhookSecretEnv string               `yaml:"webhook_secret_env,omitempty" json:"webhook_secret_env,omitempty"`
	PollInterval     string               `yaml:"poll_interval,omitempty" json:"poll_interval,omitempty"`
	JQL              string               `yaml:"jql,omitempty" json:"jql,omitempty"`
	ProjectRepoMap   []ProjectRepoMapping `yaml:"project_repo_map,omitempty" json:"project_repo_map,omitempty"`
	Scope            JIRAScopeConfig      `yaml:"scope,omitempty" json:"scope,omitempty"`
	Fields           JIRAFieldsConfig     `yaml:"fields,omitempty" json:"fields,omitempty"`
	Prioritisation   PrioritisationConfig `yaml:"prioritisation,omitempty" json:"prioritisation,omitempty"`
}

type JIRAAuthConfig struct {
	EmailEnv    string `yaml:"email_env,omitempty" json:"email_env,omitempty"`
	APITokenEnv string `yaml:"api_token_env,omitempty" json:"api_token_env,omitempty"`
}

type ProjectRepoMapping struct {
	Project string `yaml:"project" json:"project"`
	Repo    string `yaml:"repo" json:"repo"`
}

type JIRAScopeConfig struct {
	AllowedWorkspaces []string `yaml:"allowed_workspaces,omitempty" json:"allowed_workspaces,omitempty"`
	RequiredLabels    []string `yaml:"required_labels,omitempty" json:"required_labels,omitempty"`
}

type JIRAFieldsConfig struct {
	Sprint      string `yaml:"sprint,omitempty" json:"sprint,omitempty"`
	Rank        string `yaml:"rank,omitempty" json:"rank,omitempty"`
	EpicLink    string `yaml:"epic_link,omitempty" json:"epic_link,omitempty"`
	StoryPoints string `yaml:"story_points,omitempty" json:"story_points,omitempty"`
}

type PrioritisationConfig struct {
	Scope            string   `yaml:"scope,omitempty" json:"scope,omitempty"`
	ReadyStatuses    []string `yaml:"ready_statuses,omitempty" json:"ready_statuses,omitempty"`
	Order            []string `yaml:"order,omitempty" json:"order,omitempty"`
	RespectBlockedBy bool     `yaml:"respect_blocked_by" json:"respect_blocked_by"`
}

type DesignSourcesConfig struct {
	Figma FigmaConfig `yaml:"figma,omitempty" json:"figma,omitempty"`
}

type FigmaConfig struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	TokenEnv string `yaml:"token_env,omitempty" json:"token_env,omitempty"`
	BaseURL  string `yaml:"base_url,omitempty" json:"base_url,omitempty"`
}

type DeliveryConfig struct {
	Enabled       bool   `yaml:"enabled" json:"enabled"`
	Mode          string `yaml:"mode,omitempty" json:"mode,omitempty"`
	BranchPattern string `yaml:"branch_pattern,omitempty" json:"branch_pattern,omitempty"`
	MinTrust      string `yaml:"min_trust,omitempty" json:"min_trust,omitempty"`
}

// Load reads optional repo integration configuration. Missing config is disabled state.
func Load(repoRoot string) (Config, error) {
	path := filepath.Join(strings.TrimSpace(repoRoot), integrationsPath)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Defaults(), nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("integrations: read %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("integrations: parse %s: %w", path, err)
	}
	return normalize(cfg), nil
}

func Defaults() Config {
	return normalize(Config{Version: 1})
}

func Path() string {
	return integrationsPath
}

func normalize(cfg Config) Config {
	if cfg.Version == 0 {
		cfg.Version = 1
	}
	cfg.FlowProfile = normalizeFlowProfile(cfg.FlowProfile)
	cfg.Ingestion.JIRA.BaseURL = strings.TrimSpace(cfg.Ingestion.JIRA.BaseURL)
	cfg.Ingestion.JIRA.Auth.EmailEnv = strings.TrimSpace(cfg.Ingestion.JIRA.Auth.EmailEnv)
	cfg.Ingestion.JIRA.Auth.APITokenEnv = strings.TrimSpace(cfg.Ingestion.JIRA.Auth.APITokenEnv)
	cfg.Ingestion.JIRA.WebhookSecretEnv = strings.TrimSpace(cfg.Ingestion.JIRA.WebhookSecretEnv)
	cfg.Ingestion.JIRA.PollInterval = strings.TrimSpace(cfg.Ingestion.JIRA.PollInterval)
	cfg.Ingestion.JIRA.JQL = strings.TrimSpace(cfg.Ingestion.JIRA.JQL)
	for i := range cfg.Ingestion.JIRA.ProjectRepoMap {
		cfg.Ingestion.JIRA.ProjectRepoMap[i].Project = strings.TrimSpace(cfg.Ingestion.JIRA.ProjectRepoMap[i].Project)
		cfg.Ingestion.JIRA.ProjectRepoMap[i].Repo = strings.TrimSpace(cfg.Ingestion.JIRA.ProjectRepoMap[i].Repo)
	}
	cfg.Ingestion.JIRA.Scope.AllowedWorkspaces = cleanStringList(cfg.Ingestion.JIRA.Scope.AllowedWorkspaces)
	cfg.Ingestion.JIRA.Scope.RequiredLabels = cleanStringList(cfg.Ingestion.JIRA.Scope.RequiredLabels)
	cfg.Ingestion.JIRA.Fields.Sprint = strings.TrimSpace(cfg.Ingestion.JIRA.Fields.Sprint)
	cfg.Ingestion.JIRA.Fields.Rank = strings.TrimSpace(cfg.Ingestion.JIRA.Fields.Rank)
	cfg.Ingestion.JIRA.Fields.EpicLink = strings.TrimSpace(cfg.Ingestion.JIRA.Fields.EpicLink)
	cfg.Ingestion.JIRA.Fields.StoryPoints = strings.TrimSpace(cfg.Ingestion.JIRA.Fields.StoryPoints)
	cfg.Ingestion.JIRA.Prioritisation.Scope = defaultString(cfg.Ingestion.JIRA.Prioritisation.Scope, "active_sprint")
	if len(cfg.Ingestion.JIRA.Prioritisation.Order) == 0 {
		cfg.Ingestion.JIRA.Prioritisation.Order = []string{"priority", "rank", "age"}
	}
	if len(cfg.Ingestion.JIRA.Prioritisation.ReadyStatuses) == 0 {
		cfg.Ingestion.JIRA.Prioritisation.ReadyStatuses = []string{"To Do", "Ready for Dev", "Selected for Development"}
	}
	cfg.DesignSources.Figma.TokenEnv = strings.TrimSpace(cfg.DesignSources.Figma.TokenEnv)
	cfg.DesignSources.Figma.BaseURL = defaultString(cfg.DesignSources.Figma.BaseURL, "https://api.figma.com")
	cfg.Delivery.Mode = defaultString(cfg.Delivery.Mode, DeliveryModeTrunk)
	cfg.Delivery.BranchPattern = defaultString(cfg.Delivery.BranchPattern, "mars/{ticket}-{slug}")
	cfg.Delivery.MinTrust = defaultString(cfg.Delivery.MinTrust, "contributor")
	return cfg
}

func normalizeFlowProfile(profile string) string {
	switch strings.ToLower(strings.TrimSpace(profile)) {
	case FlowProfileCEOLed:
		return FlowProfileCEOLed
	case FlowProfileBoardDriven:
		return FlowProfileBoardDriven
	default:
		return FlowProfileCEOLed
	}
}

func defaultString(s, fallback string) string {
	if trimmed := strings.TrimSpace(s); trimmed != "" {
		return trimmed
	}
	return fallback
}

func cleanStringList(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
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

func (c Config) BoardDriven() bool {
	return normalizeFlowProfile(c.FlowProfile) == FlowProfileBoardDriven
}

func (c Config) JiraEnabled() bool {
	return c.JIRAEnabled()
}

func (c Config) JIRAEnabled() bool {
	return c.BoardDriven() && c.Ingestion.JIRA.Enabled
}

func (c Config) FigmaEnabled() bool {
	return c.BoardDriven() && c.DesignSources.Figma.Enabled
}

func (c Config) PullRequestDelivery() bool {
	return c.BoardDriven() && strings.EqualFold(strings.TrimSpace(c.Delivery.Mode), DeliveryModePullRequest)
}

func (c Config) DeliveryEnabled() bool {
	return c.BoardDriven() && (c.Delivery.Enabled || c.PullRequestDelivery())
}

func (c Config) EnabledSections() []string {
	var sections []string
	if c.JIRAEnabled() {
		sections = append(sections, SectionJIRA)
	}
	if c.FigmaEnabled() {
		sections = append(sections, SectionFigma)
	}
	if c.DeliveryEnabled() {
		sections = append(sections, SectionDelivery)
	}
	return sections
}

func (c Config) SectionEnabled(section string) bool {
	switch strings.ToLower(strings.TrimSpace(section)) {
	case SectionJIRA:
		return c.JIRAEnabled()
	case SectionFigma:
		return c.FigmaEnabled()
	case SectionDelivery:
		return c.DeliveryEnabled()
	default:
		return false
	}
}

func (c Config) SuppressesSchedule(role string) bool {
	if !c.BoardDriven() {
		return false
	}
	switch strings.TrimSpace(role) {
	case "ceo", "coo", "head-of-strategy", "cto-weekly":
		return true
	default:
		return false
	}
}

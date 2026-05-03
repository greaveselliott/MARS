package roleregistry

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/greaveselliott/mars-harness/internal/bundle"
)

// RegistryPath is the checked-in role inventory path for source and target repos.
const RegistryPath = "docs/roles/ROLES.md"

// Entry describes one markdown registry row.
type Entry struct {
	Role               string
	Origin             string
	Domain             string
	Mode               string
	TriggerSources     string
	Schedule           string
	Tools              string
	TrustLevel         string
	Guardrails         string
	ModelRouting       string
	ScoringSignals     string
	EscalationBehavior string
}

// Issue describes one actionable registry consistency problem.
type Issue struct {
	Path    string `json:"path"`
	Message string `json:"message"`
	Fix     string `json:"fix"`
}

// Report summarizes role-registry consistency for a repo.
type Report struct {
	Issues []Issue `json:"issues"`
}

// OK returns true when the registry matches the manifest.
func (r Report) OK() bool {
	return len(r.Issues) == 0
}

// Summary returns a short operator-facing health summary.
func (r Report) Summary() string {
	if r.OK() {
		return "role registry matches manifest"
	}
	if len(r.Issues) == 1 {
		return "role-registry issue: " + r.Issues[0].Message
	}
	return fmt.Sprintf("role-registry found %d issues: %s", len(r.Issues), r.Issues[0].Message)
}

// Remediation returns the first concrete repair instruction.
func (r Report) Remediation() string {
	for _, issue := range r.Issues {
		if strings.TrimSpace(issue.Fix) != "" {
			return issue.Fix
		}
	}
	return "update docs/roles/ROLES.md to match .harness/manifest.yaml, then run 'mars-harness doctor --repo <path>'"
}

// CheckRepo compares a target repo role registry with its manifest.
func CheckRepo(repoPath string) (Report, error) {
	if strings.TrimSpace(repoPath) == "" {
		return Report{}, fmt.Errorf("role registry: repo path is empty")
	}
	absRepo, err := filepath.Abs(repoPath)
	if err != nil {
		return Report{}, fmt.Errorf("role registry: resolve repo path: %w", err)
	}

	manifest, err := bundle.Load(absRepo)
	if err != nil {
		return Report{}, fmt.Errorf("role registry: load manifest: %w", err)
	}

	registryAbs := filepath.Join(absRepo, filepath.FromSlash(RegistryPath))
	data, err := os.ReadFile(registryAbs)
	if err != nil {
		if os.IsNotExist(err) {
			return Report{Issues: []Issue{{
				Path:    RegistryPath,
				Message: "role registry is missing",
				Fix:     "create docs/roles/ROLES.md with one row per .harness/manifest.yaml role; mark target-specific roles with origin `custom`",
			}}}, nil
		}
		return Report{}, fmt.Errorf("role registry: read %s: %w", RegistryPath, err)
	}

	entries, err := ParseMarkdown(data)
	if err != nil {
		return Report{}, err
	}
	report := checkEntries(manifest.Roles, entries)
	sort.Slice(report.Issues, func(i, j int) bool {
		if report.Issues[i].Path == report.Issues[j].Path {
			return report.Issues[i].Message < report.Issues[j].Message
		}
		return report.Issues[i].Path < report.Issues[j].Path
	})
	return report, nil
}

// ParseMarkdown reads the first role-registry table in a markdown document.
func ParseMarkdown(data []byte) (map[string]Entry, error) {
	lines := strings.Split(string(data), "\n")
	header := map[string]int{}
	entries := map[string]Entry{}

	for _, line := range lines {
		cells, ok := parseTableLine(line)
		if !ok {
			continue
		}
		if isSeparatorRow(cells) {
			continue
		}
		if len(header) == 0 {
			maybeHeader := indexHeader(cells)
			if hasRequiredColumns(maybeHeader) {
				header = maybeHeader
			}
			continue
		}
		entry, ok := entryFromCells(cells, header)
		if !ok {
			continue
		}
		role := normalizeKey(entry.Role)
		if role == "" {
			continue
		}
		entry.Role = role
		entries[role] = entry
	}

	if len(header) == 0 {
		return nil, fmt.Errorf("role registry: %s must contain a markdown table with Role, Origin, Domain, Mode, Trigger sources, Schedule, Tools, Trust level, Guardrails, Model routing, Scoring signals, and Escalation behavior columns", RegistryPath)
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("role registry: %s contains no role rows", RegistryPath)
	}
	return entries, nil
}

// DefaultMarkdown returns the generated registry for starter harness roles.
func DefaultMarkdown() string {
	var b strings.Builder
	b.WriteString(strings.Join([]string{
		"# Role Registry",
		"",
		"**Status:** Seed",
		"**Updated:** 2026-05-03",
		"**Owner:** Project maintainers",
		"**Mirrors:** `.harness/manifest.yaml`, `docs/design-docs/harness-operating-model.md`",
		"",
		"## Purpose",
		"",
		"This checked registry is the compact inventory of default autonomous roles. It",
		"exists so humans and agents can see role domains, modes, triggers, tools,",
		"guardrails, trust, model routing, scoring signals, and escalation behavior in",
		"one repo-owned artifact.",
		"",
		"Read `docs/design-docs/harness-operating-model.md` for the domain and mode",
		"contract, `docs/design-docs/tools-glossary.md` before changing tool allowlists,",
		"and `.harness/manifest.yaml` for executable runtime configuration.",
		"",
		"## Registry Rules",
		"",
		"- `Origin` is `default` for roles generated by Mars Harness and `custom` for target-owned roles.",
		"- Every role in `.harness/manifest.yaml` should have one row here.",
		"- Custom target roles should be added with `Origin` set to `custom` and should not be treated as missing source defaults.",
		"- Optional GitHub webhook triggers are explicit repair inputs. Schedule and chain triggers remain the default delivery model.",
		"- After changing roles, run `mars-harness doctor --repo .` to check registry health.",
		"",
		"## Default Roles",
		"",
	}, "\n"))
	b.WriteString("\n")
	b.WriteString("| Role | Origin | Domain | Mode | Trigger sources | Schedule | Tools | Trust level | Guardrails | Model routing | Scoring signals | Escalation behavior |\n")
	b.WriteString("| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |\n")
	for _, entry := range DefaultEntries() {
		b.WriteString(formatRow(entry))
	}
	return b.String()
}

// DefaultEntries returns a copy of the generated starter role registry.
func DefaultEntries() []Entry {
	out := make([]Entry, len(defaultEntries))
	copy(out, defaultEntries)
	return out
}

var defaultEntries = []Entry{
	{
		Role:               "ceo",
		Origin:             "default",
		Domain:             "planner",
		Mode:               "strategy",
		TriggerSources:     "schedule; chain to cto-weekly",
		Schedule:           "0 20 * * 0",
		Tools:              "file_read, file_write, shell_exec, mars_harness_cli, grep, record_decision, job_disposition_record, harness_doctrine_sync, task_trace_summarize, git_status, git_commit, git_push",
		TrustLevel:         "progressive planner write with release and git gates",
		Guardrails:         "goal, plan, ticket, trust, and git discipline",
		ModelRouting:       "reasoning",
		ScoringSignals:     "goal quality, scenario priority, plan freshness, downstream ticket success",
		EscalationBehavior: "chain to cto-weekly; record blockers in goals, plans, or tickets",
	},
	{
		Role:               "coo",
		Origin:             "default",
		Domain:             "planner",
		Mode:               "ticket-breakdown",
		TriggerSources:     "chain from cto-weekly; chain to engineer",
		Schedule:           "chain-only",
		Tools:              "file_read, file_write, file_search, shell_exec, mars_harness_cli, grep, record_decision, ticket_create, job_disposition_record, task_trace_summarize, git_status, git_commit, git_push",
		TrustLevel:         "progressive planner write with ticket dedupe gates",
		Guardrails:         "ticket metadata, in-progress drain, dedupe, trust, and git discipline",
		ModelRouting:       "reasoning",
		ScoringSignals:     "ticket clarity, dedupe accuracy, scenario alignment, engineer unblock rate",
		EscalationBehavior: "chain to engineer; create blocker tickets when scope is unclear",
	},
	{
		Role:               "cto-weekly",
		Origin:             "default",
		Domain:             "planner",
		Mode:               "architecture-planning",
		TriggerSources:     "schedule; chain from ceo; chain to coo",
		Schedule:           "0 21 * * 0",
		Tools:              "file_read, file_write, shell_exec, mars_harness_cli, grep, record_decision, job_disposition_record, architecture_audit, harness_doctrine_sync, tool_creation_guard, tool_inventory_audit, git_status, git_diff, git_commit, git_push",
		TrustLevel:         "progressive architecture write with doctrine and tool-policy gates",
		Guardrails:         "architecture rationale, doctrine sync, tool policy, trust, and git discipline",
		ModelRouting:       "reasoning",
		ScoringSignals:     "architecture fit, decision quality, plan coherence, audit finding closure",
		EscalationBehavior: "chain to coo; record design blockers before ticket creation",
	},
	{
		Role:               "engineer",
		Origin:             "default",
		Domain:             "engineer",
		Mode:               "ticket-delivery",
		TriggerSources:     "schedule; chain from coo; chain to qa, engineer, dogfood; idle chain to ceo and janitor; orchestrator survey for eligible tickets, intervention debt, and dogfood failures",
		Schedule:           "0 0,6,12,18 * * 1-5",
		Tools:              "file_read, file_write, shell_exec, mars_harness_cli, grep, record_decision, tool_create, task_trace_summarize, git_status, git_diff, git_commit, git_push, job_disposition_record",
		TrustLevel:         "progressive engineering write with ticket, test, release, and git gates",
		Guardrails:         "blast-radius containment, tests, ticket evidence, in-progress drain, release versioning, and git discipline",
		ModelRouting:       "coding",
		ScoringSignals:     "test pass rate, ticket completion evidence, blocker metadata quality, regression rate, review rework",
		EscalationBehavior: "chain to qa and dogfood; return blocked tickets with blocker, blocked_by, trace_id, and next_action metadata",
	},
	{
		Role:               "qa",
		Origin:             "default",
		Domain:             "reviewer",
		Mode:               "quality-review",
		TriggerSources:     "chain from engineer; chain to security",
		Schedule:           "chain-only",
		Tools:              "file_read, grep, record_decision, job_disposition_record, architecture_audit, harness_doctrine_sync, tool_creation_guard, tool_inventory_audit",
		TrustLevel:         "reviewer read-only by default",
		Guardrails:         "evidence gate, BDD contracts, doctrine sync, and tool policy",
		ModelRouting:       "fast",
		ScoringSignals:     "defect detection, evidence accuracy, false approval rate, reopened tickets",
		EscalationBehavior: "chain to security; record findings instead of hiding incomplete work",
	},
	{
		Role:               "security",
		Origin:             "default",
		Domain:             "reviewer",
		Mode:               "security-review",
		TriggerSources:     "schedule; chain from qa; chain to dependency-manager",
		Schedule:           "0 22 * * 0",
		Tools:              "file_read, file_write, shell_exec, mars_harness_cli, grep, record_decision, job_disposition_record, git_status, git_commit, git_push",
		TrustLevel:         "progressive reviewer write for bounded remediation",
		Guardrails:         "security posture, blast-radius containment, trust, and git discipline",
		ModelRouting:       "reasoning",
		ScoringSignals:     "security finding validity, remediation success, dependency risk reduction",
		EscalationBehavior: "chain to dependency-manager; record unresolved risk as tickets",
	},
	{
		Role:               "dependency-manager",
		Origin:             "default",
		Domain:             "maintainer",
		Mode:               "dependency-maintenance",
		TriggerSources:     "schedule; chain from security",
		Schedule:           "0 23 * * 0",
		Tools:              "file_read, file_write, shell_exec, mars_harness_cli, grep, record_decision, job_disposition_record, git_status, git_commit, git_push",
		TrustLevel:         "progressive maintainer write with dependency and test gates",
		Guardrails:         "dependency scope, tests, trust, release versioning, and git discipline",
		ModelRouting:       "fast",
		ScoringSignals:     "update success, test pass rate, stale dependency reduction, rollback rate",
		EscalationBehavior: "record blocked upgrades with package, version, and failing command",
	},
	{
		Role:               "release-manager",
		Origin:             "default",
		Domain:             "maintainer",
		Mode:               "release-management",
		TriggerSources:     "schedule",
		Schedule:           "0 8 * * 1",
		Tools:              "file_read, file_write, shell_exec, mars_harness_cli, grep, record_decision, job_disposition_record, release_orchestrate, github_release_status, git_release_guard, git_status, git_diff, git_commit, git_push",
		TrustLevel:         "progressive release write with version, tag, asset, and git gates",
		Guardrails:         "semantic versioning, changelog, release assets, trust, and git discipline",
		ModelRouting:       "reasoning",
		ScoringSignals:     "release note accuracy, tag health, asset verification, release blocker closure",
		EscalationBehavior: "record release blockers explicitly; do not claim notes-only releases complete",
	},
	{
		Role:               "dogfood",
		Origin:             "default",
		Domain:             "end-to-end-tester",
		Mode:               "dogfood-validation",
		TriggerSources:     "schedule; chain from engineer",
		Schedule:           "0 10 * * 1-5",
		Tools:              "file_read, file_write, shell_exec, mars_harness_cli, grep, record_decision, tool_create, task_trace_summarize, git_status, git_diff, git_commit, git_push, job_disposition_record",
		TrustLevel:         "progressive tester write for bounded evidence and fixes",
		Guardrails:         "real command evidence, blast-radius containment, trust, and git discipline",
		ModelRouting:       "coding",
		ScoringSignals:     "setup success, E2E pass rate, reproduced failures, intervention debt quality",
		EscalationBehavior: "create or update intervention-debt tickets for repeated harness failures",
	},
	{
		Role:               "pipeline-fixer",
		Origin:             "default",
		Domain:             "engineer",
		Mode:               "pipeline-repair",
		TriggerSources:     `optional GitHub workflow_run.conclusion == "failure"; orchestrator survey for failed checks; chain to qa`,
		Schedule:           "event/survey-only",
		Tools:              "file_read, file_write, shell_exec, mars_harness_cli, grep, record_decision, job_disposition_record, architecture_audit, harness_doctrine_sync, tool_creation_guard, tool_inventory_audit, git_status, git_diff, git_commit, git_push",
		TrustLevel:         "progressive engineering write for bounded repair",
		Guardrails:         "pipeline scope, no recursive recovery, tests, trust, and git discipline",
		ModelRouting:       "coding",
		ScoringSignals:     "repair success, repeated failure suppression, CI recovery time, regression rate",
		EscalationBehavior: "chain to qa; record deterministic remediation when recovery repeats",
	},
	{
		Role:               "orchestrator",
		Origin:             "default",
		Domain:             "orchestrator",
		Mode:               "dispatch-routing",
		TriggerSources:     "dispatch fallback",
		Schedule:           "dispatch-only",
		Tools:              "file_read, grep, record_decision, job_disposition_record, task_trace_summarize, git_status, git_diff",
		TrustLevel:         "observer by default with disposition write",
		Guardrails:         "dispatch loop guard, manifest role validation, ticket truth, and trace discipline",
		ModelRouting:       "reasoning",
		ScoringSignals:     "ambiguous route resolution, loop prevention, blocked-work clarity",
		EscalationBehavior: "route ambiguous or repeated dispatch decisions to a valid manifest role or stop with a recorded reason",
	},
	{
		Role:               "janitor",
		Origin:             "default",
		Domain:             "orchestrator",
		Mode:               "ticket-hygiene",
		TriggerSources:     "schedule; ticket.stale_in_progress; idle chain from engineer; orchestrator survey for stale, blocked, and no-op ticket state",
		Schedule:           "0 7 * * *",
		Tools:              "file_read, file_write, shell_exec, mars_harness_cli, grep, record_decision, job_disposition_record, git_status, git_diff, git_commit, git_push",
		TrustLevel:         "progressive orchestrator write with ticket-state gates",
		Guardrails:         "ticket lifecycle, active-plan hygiene, stale in-progress detection, trust, and git discipline",
		ModelRouting:       "fast",
		ScoringSignals:     "stale ticket reduction, active-plan cleanliness, queue recovery, duplicate cleanup",
		EscalationBehavior: "return misleading state to clear tickets or create focused intervention debt",
	},
}

func checkEntries(roles map[string]bundle.RoleConfig, entries map[string]Entry) Report {
	var report Report
	for roleName, role := range roles {
		entry, ok := entries[roleName]
		if !ok {
			report.add(RegistryPath,
				fmt.Sprintf("manifest role `%s` is missing from the role registry", roleName),
				fmt.Sprintf("add a docs/roles/ROLES.md row for `%s`; use origin `custom` if this is target-specific", roleName))
			continue
		}
		report.checkRequiredFields(roleName, entry)
		if role.Domain != "" && normalizeComparable(entry.Domain) != normalizeComparable(role.Domain) {
			report.add(RegistryPath,
				fmt.Sprintf("role `%s` registry domain %q does not match manifest domain %q", roleName, entry.Domain, role.Domain),
				fmt.Sprintf("update `%s` domain in docs/roles/ROLES.md or .harness/manifest.yaml", roleName))
		}
		if role.Mode != "" && normalizeComparable(entry.Mode) != normalizeComparable(role.Mode) {
			report.add(RegistryPath,
				fmt.Sprintf("role `%s` registry mode %q does not match manifest mode %q", roleName, entry.Mode, role.Mode),
				fmt.Sprintf("update `%s` mode in docs/roles/ROLES.md or .harness/manifest.yaml", roleName))
		}
		if role.Model != "" && normalizeComparable(entry.ModelRouting) != normalizeComparable(role.Model) {
			report.add(RegistryPath,
				fmt.Sprintf("role `%s` registry model routing %q does not match manifest model %q", roleName, entry.ModelRouting, role.Model),
				fmt.Sprintf("update `%s` model routing in docs/roles/ROLES.md or .harness/manifest.yaml", roleName))
		}
		if role.Schedule != "" && !containsComparable(entry.Schedule, role.Schedule) {
			report.add(RegistryPath,
				fmt.Sprintf("role `%s` registry schedule does not include manifest schedule %q", roleName, role.Schedule),
				fmt.Sprintf("add `%s` to the `%s` schedule cell in docs/roles/ROLES.md", role.Schedule, roleName))
		}
		for _, trigger := range role.Triggers {
			if !containsComparable(entry.TriggerSources, trigger) {
				report.add(RegistryPath,
					fmt.Sprintf("role `%s` registry trigger sources do not include manifest trigger %q", roleName, trigger),
					fmt.Sprintf("add `%s` to the `%s` trigger sources cell in docs/roles/ROLES.md", trigger, roleName))
			}
			if strings.Contains(trigger, "workflow_run") && !containsComparable(entry.TriggerSources, "optional") {
				report.add(RegistryPath,
					fmt.Sprintf("role `%s` GitHub workflow trigger must be marked optional", roleName),
					fmt.Sprintf("mark `%s` trigger sources as optional in docs/roles/ROLES.md", roleName))
			}
		}
		for _, tool := range role.Tools {
			if !containsListItem(entry.Tools, tool) {
				report.add(RegistryPath,
					fmt.Sprintf("role `%s` registry tools do not include manifest tool `%s`", roleName, tool),
					fmt.Sprintf("add `%s` to the `%s` tools cell in docs/roles/ROLES.md", tool, roleName))
			}
		}
	}

	for roleName, entry := range entries {
		if _, ok := roles[roleName]; ok {
			continue
		}
		report.add(RegistryPath,
			fmt.Sprintf("registry role `%s` is not defined in .harness/manifest.yaml", roleName),
			fmt.Sprintf("add `%s` to .harness/manifest.yaml or remove the stale docs/roles/ROLES.md row", roleName))
		if normalizeComparable(entry.Origin) != "custom" {
			report.add(RegistryPath,
				fmt.Sprintf("registry role `%s` is absent from manifest and is not marked custom", roleName),
				fmt.Sprintf("mark `%s` origin as `custom` only if it is target-owned and defined in the manifest", roleName))
		}
	}
	return report
}

func (r *Report) checkRequiredFields(roleName string, entry Entry) {
	required := []struct {
		label string
		value string
	}{
		{"origin", entry.Origin},
		{"domain", entry.Domain},
		{"mode", entry.Mode},
		{"trigger sources", entry.TriggerSources},
		{"schedule", entry.Schedule},
		{"tools", entry.Tools},
		{"trust level", entry.TrustLevel},
		{"guardrails", entry.Guardrails},
		{"model routing", entry.ModelRouting},
		{"scoring signals", entry.ScoringSignals},
		{"escalation behavior", entry.EscalationBehavior},
	}
	for _, field := range required {
		if strings.TrimSpace(field.value) == "" {
			r.add(RegistryPath,
				fmt.Sprintf("role `%s` registry row is missing %s", roleName, field.label),
				fmt.Sprintf("fill the `%s` %s cell in docs/roles/ROLES.md", roleName, field.label))
		}
	}
}

func (r *Report) add(path, message, fix string) {
	r.Issues = append(r.Issues, Issue{Path: filepath.ToSlash(path), Message: message, Fix: fix})
}

func formatRow(entry Entry) string {
	cells := []string{
		codeCell(entry.Role),
		entry.Origin,
		entry.Domain,
		codeCell(entry.Mode),
		codeInlineRefs(entry.TriggerSources),
		codeInlineRefs(entry.Schedule),
		entry.Tools,
		entry.TrustLevel,
		entry.Guardrails,
		entry.ModelRouting,
		entry.ScoringSignals,
		entry.EscalationBehavior,
	}
	return "| " + strings.Join(cells, " | ") + " |\n"
}

func codeCell(value string) string {
	if value == "" {
		return ""
	}
	return "`" + value + "`"
}

func codeInlineRefs(value string) string {
	if value == "" || value == "chain-only" || value == "event-only" {
		return value
	}
	return strings.ReplaceAll(value, "`", "")
}

func parseTableLine(line string) ([]string, bool) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "|") || !strings.HasSuffix(line, "|") {
		return nil, false
	}
	trimmed := strings.Trim(line, "|")
	parts := strings.Split(trimmed, "|")
	cells := make([]string, 0, len(parts))
	for _, part := range parts {
		cells = append(cells, strings.TrimSpace(part))
	}
	return cells, true
}

func isSeparatorRow(cells []string) bool {
	if len(cells) == 0 {
		return false
	}
	for _, cell := range cells {
		trimmed := strings.TrimSpace(cell)
		if trimmed == "" {
			return false
		}
		for _, r := range trimmed {
			if r != '-' && r != ':' {
				return false
			}
		}
	}
	return true
}

func indexHeader(cells []string) map[string]int {
	index := map[string]int{}
	for i, cell := range cells {
		index[normalizeHeader(cell)] = i
	}
	return index
}

func hasRequiredColumns(header map[string]int) bool {
	for _, column := range []string{
		"role",
		"origin",
		"domain",
		"mode",
		"trigger sources",
		"schedule",
		"tools",
		"trust level",
		"guardrails",
		"model routing",
		"scoring signals",
		"escalation behavior",
	} {
		if _, ok := header[column]; !ok {
			return false
		}
	}
	return true
}

func entryFromCells(cells []string, header map[string]int) (Entry, bool) {
	get := func(column string) string {
		i, ok := header[column]
		if !ok || i >= len(cells) {
			return ""
		}
		return cleanCell(cells[i])
	}
	entry := Entry{
		Role:               get("role"),
		Origin:             get("origin"),
		Domain:             get("domain"),
		Mode:               get("mode"),
		TriggerSources:     get("trigger sources"),
		Schedule:           get("schedule"),
		Tools:              get("tools"),
		TrustLevel:         get("trust level"),
		Guardrails:         get("guardrails"),
		ModelRouting:       get("model routing"),
		ScoringSignals:     get("scoring signals"),
		EscalationBehavior: get("escalation behavior"),
	}
	return entry, strings.TrimSpace(entry.Role) != ""
}

func normalizeHeader(cell string) string {
	return strings.Join(strings.Fields(strings.ToLower(cleanCell(cell))), " ")
}

func cleanCell(cell string) string {
	replacer := strings.NewReplacer("`", "", "**", "", "<br>", " ", "<br/>", " ", "<br />", " ")
	return strings.TrimSpace(replacer.Replace(cell))
}

func normalizeKey(value string) string {
	return strings.ToLower(normalizeComparable(value))
}

func normalizeComparable(value string) string {
	cleaned := strings.ToLower(cleanCell(value))
	cleaned = strings.ReplaceAll(cleaned, "&quot;", `"`)
	return strings.Join(strings.Fields(cleaned), " ")
}

func containsComparable(cell, needle string) bool {
	return strings.Contains(normalizeComparable(cell), normalizeComparable(needle))
}

func containsListItem(cell, item string) bool {
	item = normalizeComparable(item)
	for _, token := range strings.FieldsFunc(normalizeComparable(cell), func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\t' || r == '\n'
	}) {
		if token == item {
			return true
		}
	}
	return false
}

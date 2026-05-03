package roleregistry_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/greaveselliott/mars-harness/internal/roleregistry"
	"github.com/greaveselliott/mars-harness/internal/scanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckRepoPassesGeneratedRegistry(t *testing.T) {
	t.Parallel()
	repo := initRegistryRepo(t)

	report, err := roleregistry.CheckRepo(repo)
	require.NoError(t, err)
	assert.True(t, report.OK(), "unexpected registry issues: %+v", report.Issues)
}

func TestCheckRepoReportsMissingManifestRole(t *testing.T) {
	t.Parallel()
	repo := initRegistryRepo(t)
	registryPath := filepath.Join(repo, filepath.FromSlash(roleregistry.RegistryPath))
	data, err := os.ReadFile(registryPath)
	require.NoError(t, err)
	lines := strings.Split(string(data), "\n")
	filtered := lines[:0]
	for _, line := range lines {
		if strings.HasPrefix(line, "| `qa` |") {
			continue
		}
		filtered = append(filtered, line)
	}
	require.NoError(t, os.WriteFile(registryPath, []byte(strings.Join(filtered, "\n")), 0o644))

	report, err := roleregistry.CheckRepo(repo)
	require.NoError(t, err)
	require.False(t, report.OK())
	assert.Contains(t, report.Summary(), "manifest role `qa` is missing")
	assert.Contains(t, report.Remediation(), "docs/roles/ROLES.md")
}

func TestCheckRepoAllowsRegisteredCustomRole(t *testing.T) {
	t.Parallel()
	repo := initRegistryRepo(t)
	addCustomRoleToManifest(t, repo)
	appendRegistryRow(t, repo, "| `analyst` | custom | planner | `custom-analysis` | manual chain | chain-only | file_read | target-owned guarded read | target policy and trust gates | fast | analysis quality | record blockers in tickets |\n")

	report, err := roleregistry.CheckRepo(repo)
	require.NoError(t, err)
	assert.True(t, report.OK(), "unexpected registry issues: %+v", report.Issues)
}

func TestCheckRepoClassifiesUnregisteredCustomRole(t *testing.T) {
	t.Parallel()
	repo := initRegistryRepo(t)
	addCustomRoleToManifest(t, repo)

	report, err := roleregistry.CheckRepo(repo)
	require.NoError(t, err)
	require.False(t, report.OK())
	assert.Contains(t, report.Summary(), "manifest role `analyst` is missing")
	assert.Contains(t, report.Remediation(), "origin `custom`")
}

func TestCheckRepoRequiresOptionalGitHubTriggers(t *testing.T) {
	t.Parallel()
	repo := initRegistryRepo(t)
	registryPath := filepath.Join(repo, filepath.FromSlash(roleregistry.RegistryPath))
	data, err := os.ReadFile(registryPath)
	require.NoError(t, err)
	updated := strings.ReplaceAll(string(data), "optional GitHub workflow_run.conclusion", "GitHub workflow_run.conclusion")
	require.NoError(t, os.WriteFile(registryPath, []byte(updated), 0o644))

	report, err := roleregistry.CheckRepo(repo)
	require.NoError(t, err)
	require.False(t, report.OK())
	assert.Contains(t, report.Summary(), "GitHub workflow trigger must be marked optional")
}

func initRegistryRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(repo, ".git"), 0o755))
	require.NoError(t, scanner.Init(repo, false))
	return repo
}

func addCustomRoleToManifest(t *testing.T, repo string) {
	t.Helper()
	manifestPath := filepath.Join(repo, ".harness", "manifest.yaml")
	f, err := os.OpenFile(manifestPath, os.O_APPEND|os.O_WRONLY, 0o644)
	require.NoError(t, err)
	defer f.Close()
	_, err = f.WriteString(`

  analyst:
    prompt: roles/analyst.md
    domain: planner
    mode: custom-analysis
    model: fast
    tools: [file_read]
`)
	require.NoError(t, err)
}

func appendRegistryRow(t *testing.T, repo, row string) {
	t.Helper()
	registryPath := filepath.Join(repo, filepath.FromSlash(roleregistry.RegistryPath))
	f, err := os.OpenFile(registryPath, os.O_APPEND|os.O_WRONLY, 0o644)
	require.NoError(t, err)
	defer f.Close()
	_, err = f.WriteString(row)
	require.NoError(t, err)
}

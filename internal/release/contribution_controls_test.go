/*
MarsDocSync:
docs:
- CONTRIBUTING.md
- docs/design-docs/code-documentation-map.md
- docs/design-docs/release-versioning.md
- docs/features/F-009-release-update-lifecycle.md
- docs/features/F-017-open-source-publication.md
*/
package release

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type dependabotContract struct {
	Version int                `yaml:"version"`
	Updates []dependabotUpdate `yaml:"updates"`
}

type dependabotUpdate struct {
	PackageEcosystem      string             `yaml:"package-ecosystem"`
	Directory             string             `yaml:"directory"`
	Schedule              dependabotSchedule `yaml:"schedule"`
	OpenPullRequestsLimit int                `yaml:"open-pull-requests-limit"`
	Labels                []string           `yaml:"labels"`
}

type dependabotSchedule struct {
	Interval string `yaml:"interval"`
	Day      string `yaml:"day"`
	Time     string `yaml:"time"`
	Timezone string `yaml:"timezone"`
}

type rulesetContract struct {
	Name         string         `json:"name"`
	Target       string         `json:"target"`
	Enforcement  string         `json:"enforcement"`
	BypassActors []rulesetActor `json:"bypass_actors"`
	Conditions   struct {
		RefName struct {
			Include []string `json:"include"`
			Exclude []string `json:"exclude"`
		} `json:"ref_name"`
	} `json:"conditions"`
	Rules []rulesetRule `json:"rules"`
}

type rulesetActor struct {
	ActorID    int    `json:"actor_id"`
	ActorType  string `json:"actor_type"`
	BypassMode string `json:"bypass_mode"`
}

type rulesetRule struct {
	Type       string          `json:"type"`
	Parameters json.RawMessage `json:"parameters,omitempty"`
}

type pullRequestRuleParameters struct {
	AllowedMergeMethods                        []string `json:"allowed_merge_methods"`
	DismissStaleReviewsOnPush                  bool     `json:"dismiss_stale_reviews_on_push"`
	RequireCodeOwnerReview                     bool     `json:"require_code_owner_review"`
	RequireExtraApprovalForUnattributedChanges bool     `json:"require_extra_approval_for_unattributed_changes"`
	RequireLastPushApproval                    bool     `json:"require_last_push_approval"`
	RequiredApprovingReviewCount               int      `json:"required_approving_review_count"`
	RequiredReviewThreadResolution             bool     `json:"required_review_thread_resolution"`
}

type statusCheckRuleParameters struct {
	DoNotEnforceOnCreate             bool                  `json:"do_not_enforce_on_create"`
	RequiredStatusChecks             []requiredStatusCheck `json:"required_status_checks"`
	StrictRequiredStatusChecksPolicy bool                  `json:"strict_required_status_checks_policy"`
}

type requiredStatusCheck struct {
	Context string `json:"context"`
}

func TestContributionWorkflowIsForkSafeAndDCOOnly(t *testing.T) {
	t.Parallel()
	path := filepath.Join(releaseRepoRoot(t), ".github", "workflows", "contribution-policy.yml")
	raw := readStrictYAML(t, path, new(workflowContract))
	var workflow workflowContract
	readStrictYAMLInto(t, raw, &workflow)

	require.Equal(t, "contribution-policy", workflow.Name)
	require.Equal(t, []string{"pull_request"}, sortedKeys(workflow.On))
	require.Equal(t, map[string]string{"contents": "read"}, workflow.Permissions)
	require.Equal(t, []string{"dco"}, sortedKeys(workflow.Jobs))
	job := workflow.Jobs["dco"]
	require.Equal(t, "DCO sign-off", job.Name)
	require.Equal(t, "ubuntu-24.04", job.RunsOn)
	require.Equal(t, 5, job.TimeoutMinutes)
	require.Empty(t, job.Permissions)
	require.Len(t, job.Steps, 2)
	require.Equal(t, "actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1", job.Steps[0].Uses)
	require.Equal(t, false, job.Steps[0].With["persist-credentials"])
	require.Equal(t, 0, job.Steps[0].With["fetch-depth"])
	require.Equal(t, "${{ github.event.pull_request.head.sha }}", job.Steps[0].With["ref"])
	require.Contains(t, job.Steps[1].Run, "scripts/check-dco.sh")

	text := strings.ToLower(string(raw))
	for _, forbidden := range []string{"pull_request_target", "secrets.", "${{ secrets", "id-token: write", "contents: write", "self-hosted", "environment:"} {
		require.NotContains(t, text, forbidden)
	}
}

func TestAllPullRequestWorkflowsAreReadOnlyAndAllActionsAreSHAAndOwnerPinned(t *testing.T) {
	t.Parallel()
	paths, err := filepath.Glob(filepath.Join(releaseRepoRoot(t), ".github", "workflows", "*.yml"))
	require.NoError(t, err)
	require.NotEmpty(t, paths)
	usesPattern := regexp.MustCompile(`(?m)^\s*uses:\s*([^\s#]+)`)
	fullGitHubSHA := regexp.MustCompile(`^actions/[A-Za-z0-9_./-]+@[0-9a-f]{40}$`)
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		require.NoError(t, err)
		text := string(raw)
		require.NotContains(t, text, "pull_request_target", path)
		for _, match := range usesPattern.FindAllStringSubmatch(text, -1) {
			require.Regexp(t, fullGitHubSHA, match[1], "%s uses an unapproved or mutable action reference", path)
		}
		if strings.Contains(text, "  pull_request:") {
			require.Contains(t, text, "permissions:\n  contents: read", path)
			lower := strings.ToLower(text)
			for _, forbidden := range []string{"${{ secrets", "secrets.", "id-token: write", "contents: write", "self-hosted", "environment:"} {
				require.NotContains(t, lower, forbidden, path)
			}
		}
	}
}

func TestDependabotCoversBothGoModulesAndPinnedActions(t *testing.T) {
	t.Parallel()
	path := filepath.Join(releaseRepoRoot(t), ".github", "dependabot.yml")
	raw := readStrictYAML(t, path, new(dependabotContract))
	var config dependabotContract
	readStrictYAMLInto(t, raw, &config)

	require.Equal(t, 2, config.Version)
	require.Len(t, config.Updates, 3)
	got := make([]string, 0, len(config.Updates))
	for _, update := range config.Updates {
		got = append(got, update.PackageEcosystem+":"+update.Directory)
		require.Equal(t, "weekly", update.Schedule.Interval)
		require.Equal(t, "monday", update.Schedule.Day)
		require.Equal(t, "Europe/London", update.Schedule.Timezone)
		require.Positive(t, update.OpenPullRequestsLimit)
		require.Equal(t, []string{"dependencies"}, update.Labels)
	}
	sort.Strings(got)
	require.Equal(t, []string{"github-actions:/", "gomod:/", "gomod:/tools/third-party-notices"}, got)
}

func TestMainRulesetMatchesMaintainerTrunkAndContributorPolicy(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile(filepath.Join(releaseRepoRoot(t), ".github", "rulesets", "main.json"))
	require.NoError(t, err)
	var ruleset rulesetContract
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	require.NoError(t, decoder.Decode(&ruleset))
	require.Equal(t, "Protect main and require contributor review", ruleset.Name)
	require.Equal(t, "branch", ruleset.Target)
	require.Equal(t, "active", ruleset.Enforcement)
	require.Equal(t, []rulesetActor{{ActorID: 5, ActorType: "RepositoryRole", BypassMode: "always"}}, ruleset.BypassActors)
	require.Equal(t, []string{"~DEFAULT_BRANCH"}, ruleset.Conditions.RefName.Include)
	require.Empty(t, ruleset.Conditions.RefName.Exclude)

	require.Len(t, ruleset.Rules, 4)
	require.Equal(t, "deletion", ruleset.Rules[0].Type)
	require.Equal(t, "non_fast_forward", ruleset.Rules[1].Type)
	var reviews pullRequestRuleParameters
	require.NoError(t, json.Unmarshal(ruleset.Rules[2].Parameters, &reviews))
	require.Equal(t, []string{"merge", "squash", "rebase"}, reviews.AllowedMergeMethods)
	require.True(t, reviews.DismissStaleReviewsOnPush)
	require.True(t, reviews.RequireCodeOwnerReview)
	require.True(t, reviews.RequireExtraApprovalForUnattributedChanges)
	require.True(t, reviews.RequireLastPushApproval)
	require.Equal(t, 1, reviews.RequiredApprovingReviewCount)
	require.True(t, reviews.RequiredReviewThreadResolution)

	var checks statusCheckRuleParameters
	require.NoError(t, json.Unmarshal(ruleset.Rules[3].Parameters, &checks))
	require.False(t, checks.DoNotEnforceOnCreate)
	require.True(t, checks.StrictRequiredStatusChecksPolicy)
	contexts := make([]string, 0, len(checks.RequiredStatusChecks))
	for _, check := range checks.RequiredStatusChecks {
		contexts = append(contexts, check.Context)
	}
	require.Equal(t, []string{"DCO sign-off", "below-minimum", "dependency-notices", "supported-source (1.25.13)", "supported-source (1.27.0)"}, contexts)
}

func TestCommunityFilesBindContributionAndSecurityBoundaries(t *testing.T) {
	t.Parallel()
	root := releaseRepoRoot(t)
	for _, path := range []string{
		"CODE_OF_CONDUCT.md", "CONTRIBUTING.md", "GOVERNANCE.md", "SECURITY.md", "SUPPORT.md",
		filepath.Join(".github", "CODEOWNERS"), filepath.Join(".github", "pull_request_template.md"),
		filepath.Join(".github", "ISSUE_TEMPLATE", "bug_report.yml"),
		filepath.Join(".github", "ISSUE_TEMPLATE", "feature_request.yml"),
		filepath.Join(".github", "ISSUE_TEMPLATE", "private_contact.yml"),
		filepath.Join(".github", "ISSUE_TEMPLATE", "config.yml"),
	} {
		require.FileExists(t, filepath.Join(root, path), path)
	}
	contributing, err := os.ReadFile(filepath.Join(root, "CONTRIBUTING.md"))
	require.NoError(t, err)
	contributingText := strings.Join(strings.Fields(string(contributing)), " ")
	for _, required := range []string{"fork and pull request", "git commit --signoff", "CODEOWNERS approval", "receive no secrets, write token, OIDC, or release authority"} {
		require.Contains(t, contributingText, required)
	}
	security, err := os.ReadFile(filepath.Join(root, "SECURITY.md"))
	require.NoError(t, err)
	require.Contains(t, string(security), "/security/advisories/new")
	require.NotContains(t, strings.ToLower(string(security)), "open a public issue for a suspected vulnerability")
	codeowners, err := os.ReadFile(filepath.Join(root, ".github", "CODEOWNERS"))
	require.NoError(t, err)
	require.Contains(t, string(codeowners), "* @greaveselliott")
}

func TestDCOCheckAcceptsAuthorSignoffAndRejectsMissingOrMismatchedSignoff(t *testing.T) {
	t.Parallel()
	script := filepath.Join(releaseRepoRoot(t), "scripts", "check-dco.sh")

	t.Run("accepted", func(t *testing.T) {
		dir, base := newDCOTestRepo(t)
		writeDCOFixture(t, dir, "signed.txt", "signed")
		runDCOGit(t, dir, "add", "signed.txt")
		runDCOGit(t, dir, "commit", "-m", "feat: signed\n\nSigned-off-by: Test User <test@example.com>")
		head := strings.TrimSpace(runDCOGit(t, dir, "rev-parse", "HEAD"))
		cmd := exec.Command(script, base, head)
		cmd.Dir = dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, string(output))
		require.Contains(t, string(output), "1 non-merge commit(s)")
	})

	for name, message := range map[string]string{
		"missing":    "feat: unsigned",
		"mismatched": "feat: wrong signer\n\nSigned-off-by: Other User <other@example.com>",
	} {
		t.Run(name, func(t *testing.T) {
			dir, base := newDCOTestRepo(t)
			writeDCOFixture(t, dir, "candidate.txt", name)
			runDCOGit(t, dir, "add", "candidate.txt")
			runDCOGit(t, dir, "commit", "-m", message)
			head := strings.TrimSpace(runDCOGit(t, dir, "rev-parse", "HEAD"))
			cmd := exec.Command(script, base, head)
			cmd.Dir = dir
			output, err := cmd.CombinedOutput()
			require.Error(t, err)
			require.Contains(t, string(output), "lacks a Signed-off-by trailer matching author email")
		})
	}
}

func TestDCOCheckAcceptsOnlyAuthenticatedDependabotIdentity(t *testing.T) {
	t.Parallel()
	script := filepath.Join(releaseRepoRoot(t), "scripts", "check-dco.sh")
	const botMessage = "build(deps): update dependency\n\nSigned-off-by: dependabot[bot] <support@github.com>"

	t.Run("accepted", func(t *testing.T) {
		dir, base := newDCOTestRepo(t)
		writeDCOFixture(t, dir, "dependency.txt", "updated")
		runDCOGit(t, dir, "add", "dependency.txt")
		runDCOGitWithEnv(t, dir, []string{
			"GIT_AUTHOR_NAME=dependabot[bot]",
			"GIT_AUTHOR_EMAIL=49699333+dependabot[bot]@users.noreply.github.com",
			"GIT_COMMITTER_NAME=GitHub",
			"GIT_COMMITTER_EMAIL=noreply@github.com",
		}, "commit", "-m", botMessage)
		head := strings.TrimSpace(runDCOGit(t, dir, "rev-parse", "HEAD"))
		cmd := exec.Command(script, base, head)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GITHUB_ACTOR=dependabot[bot]")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, string(output))
		require.Contains(t, string(output), "satisfy the DCO policy")
	})

	for _, fixture := range []struct {
		name       string
		authorName string
		authorMail string
		message    string
	}{
		{name: "spoofed author", authorName: "Test User", authorMail: "test@example.com", message: botMessage},
		{name: "wrong trailer", authorName: "dependabot[bot]", authorMail: "49699333+dependabot[bot]@users.noreply.github.com", message: "build(deps): update dependency\n\nSigned-off-by: Other Bot <other@example.com>"},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			dir, base := newDCOTestRepo(t)
			writeDCOFixture(t, dir, "dependency.txt", fixture.name)
			runDCOGit(t, dir, "add", "dependency.txt")
			runDCOGitWithEnv(t, dir, []string{
				"GIT_AUTHOR_NAME=" + fixture.authorName,
				"GIT_AUTHOR_EMAIL=" + fixture.authorMail,
				"GIT_COMMITTER_NAME=GitHub",
				"GIT_COMMITTER_EMAIL=noreply@github.com",
			}, "commit", "-m", fixture.message)
			head := strings.TrimSpace(runDCOGit(t, dir, "rev-parse", "HEAD"))
			cmd := exec.Command(script, base, head)
			cmd.Dir = dir
			cmd.Env = append(os.Environ(), "GITHUB_ACTOR=dependabot[bot]")
			output, err := cmd.CombinedOutput()
			require.Error(t, err)
			require.Contains(t, string(output), "lacks a Signed-off-by trailer matching author email")
		})
	}
}

func newDCOTestRepo(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	runDCOGit(t, dir, "init", "-b", "main")
	runDCOGit(t, dir, "config", "user.name", "Test User")
	runDCOGit(t, dir, "config", "user.email", "test@example.com")
	writeDCOFixture(t, dir, "base.txt", "base")
	runDCOGit(t, dir, "add", "base.txt")
	runDCOGit(t, dir, "commit", "-m", "chore: base")
	return dir, strings.TrimSpace(runDCOGit(t, dir, "rev-parse", "HEAD"))
}

func writeDCOFixture(t *testing.T, dir, name, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600))
}

func runDCOGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s: %s", strings.Join(args, " "), string(output))
	return string(output)
}

func runDCOGitWithEnv(t *testing.T, dir string, env []string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s: %s", strings.Join(args, " "), string(output))
	return string(output)
}

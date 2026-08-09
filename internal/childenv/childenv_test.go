/*
MarsDocSync:
docs:
- docs/configuration-reference.html
- docs/design-docs/code-documentation-map.md
- docs/features/F-005-agent-execution-runtime.md
- docs/product-specs/product-surface.md
*/
package childenv

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFilterPreservesOrdinaryVariablesAndRemovesSensitiveNames(t *testing.T) {
	parent := []string{
		"PATH=/usr/bin:/bin", "HOME=/home/mars", "TMPDIR=/tmp/mars", "LANG=en_GB.UTF-8",
		"LC_ALL=C", "GOCACHE=/cache/go", "PROJECT_MODE=validation", "AUTHOR_NAME=octavia",
		"MARS_WEBHOOK_SECRET=mars-secret", "GITHUB_TOKEN=github-secret", "GH_TOKEN=gh-secret",
		"AWS_ACCESS_KEY_ID=aws-secret", "AZURE_CLIENT_SECRET=azure-secret",
		"GOOGLE_APPLICATION_CREDENTIALS=/tmp/google.json", "OPENAI_API_KEY=openai-secret",
		"AUTHORIZATION=Bearer secret", "BASIC_AUTH=basic-secret", "SSH_AUTH_SOCK=/tmp/agent.sock", "NPM_TOKEN=npm-secret",
		"DB_PASSWORD=db-secret", "SERVICE_API_KEY=api-secret", "PRIVATE_KEY=private-secret",
		"CREDENTIAL_FILE=/tmp/credentials", "GIT_ASKPASS=/tmp/askpass",
		"XAI_ENDPOINT=https://xai.invalid", "DEEPSEEK_ENDPOINT=https://deepseek.invalid",
		"GROQ_REGION=test",
	}

	got, err := Filter(parent)
	require.NoError(t, err)
	require.Equal(t, parent[:8], got)
}

func TestFilterOwnerAllowlistRestoresNamedVariablesButNeverPropagatesControl(t *testing.T) {
	parent := []string{
		"PATH=/usr/bin:/bin",
		"SSH_AUTH_SOCK=/tmp/agent.sock",
		"JIRA_API_TOKEN=jira-secret",
		AllowlistVariable + "=SSH_AUTH_SOCK, JIRA_API_TOKEN," + AllowlistVariable,
	}

	got, err := Filter(parent)
	require.NoError(t, err)
	require.Equal(t, parent[:3], got)
}

func TestFilterRejectsInvalidOwnerAllowlistName(t *testing.T) {
	_, err := Filter([]string{AllowlistVariable + "=GOOD_NAME,bad-name"})
	require.ErrorContains(t, err, "invalid variable name \"bad-name\"")
}

func TestApplyWithReplacesExplicitOverride(t *testing.T) {
	t.Setenv("GOBIN", "/old")
	cmd := exec.Command("true")
	require.NoError(t, ApplyWith(cmd, "GOBIN=/new"))
	require.Contains(t, cmd.Env, "GOBIN=/new")
	require.NotContains(t, cmd.Env, "GOBIN=/old")
}

/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/pipeline-engine.md
- docs/design-docs/orchestrated-organization-layer.md
- docs/features/F-006-queue-and-orchestration.md
*/
package serve

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	gh "github.com/greaveselliott/mars/internal/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeTriggerRepo(t *testing.T, remote string, roles map[string][]string) RepoRecord {
	t.Helper()
	dir := t.TempDir()
	harnessDir := filepath.Join(dir, ".harness")
	require.NoError(t, os.MkdirAll(harnessDir, 0o755))

	yamlContent := "name: test-repo\nroles:\n"
	for role, triggers := range roles {
		yamlContent += "  " + role + ":\n"
		yamlContent += "    prompt: prompts/" + role + ".md\n"
		if len(triggers) > 0 {
			yamlContent += "    triggers:\n"
			for _, trig := range triggers {
				yamlContent += "      - " + trig + "\n"
			}
		}
	}
	require.NoError(t, os.WriteFile(filepath.Join(harnessDir, "manifest.yaml"), []byte(yamlContent), 0o644))

	return RepoRecord{
		ID:     "repo-" + remote,
		Path:   dir,
		Remote: remote,
		Branch: "main",
	}
}

func TestTriggerRouter_Match(t *testing.T) {
	t.Parallel()

	repo := makeTriggerRepo(t, "owner/app", map[string][]string{
		"reviewer":  {"pull_request.opened", "pull_request.synchronize"},
		"ci-fixer":  {`workflow_run.conclusion == "failure"`},
		"deployer":  {"push"},
		"manual":    {"workflow_dispatch"},
		"merger":    {"pull_request.merged"},
		"scheduler": {"schedule.daily"},
		"triager":   {"ticket.created"},
		"monitor":   {"alert.firing"},
	})

	router := NewTriggerRouter()
	require.NoError(t, router.Rebuild([]RepoRecord{repo}))

	tests := []struct {
		name      string
		event     gh.Event
		wantRoles []string
	}{
		{
			name: "pull_request.opened matches reviewer",
			event: gh.Event{
				Type:   "pull_request",
				Action: "opened",
				Repo:   "owner/app",
			},
			wantRoles: []string{"reviewer"},
		},
		{
			name: "pull_request.synchronize matches reviewer",
			event: gh.Event{
				Type:   "pull_request",
				Action: "synchronize",
				Repo:   "owner/app",
			},
			wantRoles: []string{"reviewer"},
		},
		{
			name: "pull_request.merged matches on closed+merged payload",
			event: gh.Event{
				Type:    "pull_request",
				Action:  "closed",
				Repo:    "owner/app",
				Payload: json.RawMessage(`{"pull_request":{"merged":true}}`),
			},
			wantRoles: []string{"merger"},
		},
		{
			name: "pull_request.closed without merged does NOT match merger",
			event: gh.Event{
				Type:    "pull_request",
				Action:  "closed",
				Repo:    "owner/app",
				Payload: json.RawMessage(`{"pull_request":{"merged":false}}`),
			},
			wantRoles: nil,
		},
		{
			name: "workflow_run failure matches ci-fixer",
			event: gh.Event{
				Type:    "workflow_run",
				Action:  "completed",
				Repo:    "owner/app",
				Payload: json.RawMessage(`{"conclusion":"failure"}`),
			},
			wantRoles: []string{"ci-fixer"},
		},
		{
			name: "workflow_run success does NOT match ci-fixer",
			event: gh.Event{
				Type:    "workflow_run",
				Action:  "completed",
				Repo:    "owner/app",
				Payload: json.RawMessage(`{"conclusion":"success"}`),
			},
			wantRoles: nil,
		},
		{
			name: "push matches deployer",
			event: gh.Event{
				Type: "push",
				Repo: "owner/app",
			},
			wantRoles: []string{"deployer"},
		},
		{
			name: "workflow_dispatch matches manual",
			event: gh.Event{
				Type: "workflow_dispatch",
				Repo: "owner/app",
			},
			wantRoles: []string{"manual"},
		},
		{
			name: "schedule triggers are NOT matched by webhook events",
			event: gh.Event{
				Type: "schedule",
				Repo: "owner/app",
			},
			wantRoles: nil,
		},
		{
			name: "event from different repo does NOT match",
			event: gh.Event{
				Type:   "pull_request",
				Action: "opened",
				Repo:   "other/repo",
			},
			wantRoles: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			matches := router.Match(tt.event)
			var gotRoles []string
			for _, m := range matches {
				gotRoles = append(gotRoles, m.Role)
			}
			if tt.wantRoles == nil {
				assert.Empty(t, gotRoles)
			} else {
				assert.ElementsMatch(t, tt.wantRoles, gotRoles)
			}
		})
	}
}

func TestTriggerRouter_Rebuild_skipsBadManifest(t *testing.T) {
	t.Parallel()

	goodRepo := makeTriggerRepo(t, "owner/good", map[string][]string{
		"ci-fixer": {"push"},
	})

	badDir := t.TempDir()
	badRepo := RepoRecord{ID: "bad", Path: badDir, Remote: "owner/bad", Branch: "main"}

	router := NewTriggerRouter()
	err := router.Rebuild([]RepoRecord{badRepo, goodRepo})
	require.NoError(t, err)

	matches := router.Match(gh.Event{Type: "push", Repo: "owner/good"})
	assert.Len(t, matches, 1)
	assert.Equal(t, "ci-fixer", matches[0].Role)
}

func TestTriggerRouter_EmptyRemote_matchesAll(t *testing.T) {
	t.Parallel()

	repo := makeTriggerRepo(t, "", map[string][]string{
		"deployer": {"push"},
	})

	router := NewTriggerRouter()
	require.NoError(t, router.Rebuild([]RepoRecord{repo}))

	matches := router.Match(gh.Event{Type: "push", Repo: "any/repo"})
	assert.Len(t, matches, 1)
	assert.Equal(t, "deployer", matches[0].Role)
}

func TestNewTriggerRouter_empty(t *testing.T) {
	t.Parallel()
	router := NewTriggerRouter()
	matches := router.Match(gh.Event{Type: "push"})
	assert.Empty(t, matches)
}

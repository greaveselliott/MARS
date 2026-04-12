package serve

import (
	"encoding/json"
	"log/slog"
	"strings"
	"sync"

	"github.com/greaveselliott/mars-harness/internal/bundle"
	gh "github.com/greaveselliott/mars-harness/internal/github"
)

// TriggerMatch represents a matched (repo, role) pair.
type TriggerMatch struct {
	RepoID   string
	RepoPath string
	Role     string
	Trigger  string
}

// TriggerRouter matches incoming events to roles across registered repos.
type TriggerRouter struct {
	mu    sync.RWMutex
	index []triggerEntry
}

type triggerEntry struct {
	repoID     string
	repoPath   string
	repoRemote string
	role       string
	trigger    string
}

// NewTriggerRouter creates an empty router.
func NewTriggerRouter() *TriggerRouter {
	return &TriggerRouter{}
}

// Rebuild reloads the trigger index from all registered repos.
// Loads each repo's .harness/manifest.yaml and extracts trigger rules.
func (tr *TriggerRouter) Rebuild(repos []RepoRecord) error {
	var entries []triggerEntry

	for _, repo := range repos {
		manifest, err := bundle.Load(repo.Path)
		if err != nil {
			slog.Warn("serve: skipping repo during trigger rebuild — manifest load failed",
				"repo_id", repo.ID, "path", repo.Path, "error", err)
			continue
		}

		for roleName, roleCfg := range manifest.Roles {
			for _, trig := range roleCfg.Triggers {
				entries = append(entries, triggerEntry{
					repoID:     repo.ID,
					repoPath:   repo.Path,
					repoRemote: repo.Remote,
					role:       roleName,
					trigger:    trig,
				})
			}
		}
	}

	tr.mu.Lock()
	tr.index = entries
	tr.mu.Unlock()

	slog.Info("serve: trigger index rebuilt", "entries", len(entries), "repos", len(repos))
	return nil
}

// Len returns the number of entries in the trigger index.
func (tr *TriggerRouter) Len() int {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	return len(tr.index)
}

// Match finds all (repo, role) pairs whose triggers match the given GitHub event.
// Only entries whose repo remote matches event.Repo are considered. If the entry
// has no remote set, it matches all events regardless of repo.
func (tr *TriggerRouter) Match(event gh.Event) []TriggerMatch {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	var matches []TriggerMatch
	for _, entry := range tr.index {
		if entry.repoRemote != "" && event.Repo != "" && event.Repo != entry.repoRemote {
			continue
		}
		if matchesTrigger(entry.trigger, event) {
			matches = append(matches, TriggerMatch{
				RepoID:   entry.repoID,
				RepoPath: entry.repoPath,
				Role:     entry.role,
				Trigger:  entry.trigger,
			})
		}
	}
	return matches
}

// matchesTrigger checks whether a single trigger rule matches the event.
func matchesTrigger(trigger string, event gh.Event) bool {
	if isNonWebhookTrigger(trigger) {
		return false
	}

	if isConditionalTrigger(trigger) {
		return matchConditional(trigger, event)
	}

	return matchSimple(trigger, event)
}

var nonWebhookPrefixes = []string{"schedule.", "ticket.", "alert."}

func isNonWebhookTrigger(trigger string) bool {
	for _, prefix := range nonWebhookPrefixes {
		if strings.HasPrefix(trigger, prefix) {
			return true
		}
	}
	return false
}

func isConditionalTrigger(trigger string) bool {
	return strings.Contains(trigger, "==")
}

// matchSimple handles triggers like "pull_request.opened", "push", "workflow_dispatch".
func matchSimple(trigger string, event gh.Event) bool {
	parts := strings.SplitN(trigger, ".", 2)
	eventType := parts[0]

	if eventType != event.Type {
		return false
	}

	if len(parts) == 1 {
		return true
	}

	action := parts[1]

	if action == "merged" {
		return event.Action == "closed" && payloadFieldEquals(event.Payload, "pull_request.merged", "true")
	}

	return event.Action == action
}

// matchConditional handles triggers like `workflow_run.conclusion == "failure"`.
func matchConditional(trigger string, event gh.Event) bool {
	eqIdx := strings.Index(trigger, "==")
	if eqIdx < 0 {
		return false
	}

	lhs := strings.TrimSpace(trigger[:eqIdx])
	rhs := strings.TrimSpace(trigger[eqIdx+2:])
	rhs = strings.Trim(rhs, `"`)

	parts := strings.SplitN(lhs, ".", 2)
	eventType := parts[0]

	if eventType != event.Type {
		return false
	}

	if len(parts) < 2 {
		return false
	}

	fieldPath := parts[1]
	return payloadFieldEquals(event.Payload, fieldPath, rhs)
}

// payloadFieldEquals extracts a dotted field path from JSON and compares it to an expected value.
func payloadFieldEquals(payload json.RawMessage, fieldPath, expected string) bool {
	if len(payload) == 0 {
		return false
	}

	var data map[string]json.RawMessage
	if err := json.Unmarshal(payload, &data); err != nil {
		return false
	}

	fields := strings.Split(fieldPath, ".")
	return nestedFieldEquals(data, fields, expected)
}

func nestedFieldEquals(data map[string]json.RawMessage, fields []string, expected string) bool {
	if len(fields) == 0 {
		return false
	}

	raw, ok := data[fields[0]]
	if !ok {
		return false
	}

	if len(fields) == 1 {
		val := strings.Trim(string(raw), `"`)
		return val == expected
	}

	var nested map[string]json.RawMessage
	if err := json.Unmarshal(raw, &nested); err != nil {
		return false
	}
	return nestedFieldEquals(nested, fields[1:], expected)
}


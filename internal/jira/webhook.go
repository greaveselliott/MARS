/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/board-driven-integrations.md
- docs/features/F-013-board-driven-integrations.md
- docs/product-specs/product-surface.md
*/
package jira

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	defaultWebhookMaxBodySize = 2 << 20
	jiraSignatureHeader       = "X-Mars-Harness-Jira-Signature"
	jiraSignaturePrefix       = "sha256="
)

type WebhookConfig struct {
	Repositories func(context.Context) ([]Repository, error)
	EnvLookup    func(string) (string, bool)
	Logger       *slog.Logger
	MaxBodySize  int64
}

func WebhookHandler(cfg WebhookConfig) http.Handler {
	dedup := newDedupStore()
	if cfg.EnvLookup == nil {
		cfg.EnvLookup = os.LookupEnv
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.MaxBodySize <= 0 {
		cfg.MaxBodySize = defaultWebhookMaxBodySize
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "jira webhook endpoint accepts POST only", http.StatusMethodNotAllowed)
			return
		}
		repos, err := cfg.Repositories(r.Context())
		if err != nil {
			cfg.Logger.Warn("jira webhook: repository lookup failed", "err", err)
			http.Error(w, "jira webhook repository lookup failed", http.StatusInternalServerError)
			return
		}
		repos, err = resolveEnvBackedRepositories(repos, cfg.EnvLookup)
		if err != nil {
			cfg.Logger.Warn("jira webhook: env-backed config unavailable", "err", sanitizeError(err))
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		if len(enabledJIRARepos(repos)) == 0 {
			http.NotFound(w, r)
			return
		}
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, cfg.MaxBodySize))
		if err != nil {
			cfg.Logger.Warn("jira webhook: body read error", "err", err)
			http.Error(w, "jira webhook request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		raw, err := RawIssueFromWebhookPayload(body)
		if err != nil {
			cfg.Logger.Warn("jira webhook: invalid payload", "err", sanitizeError(err))
			http.Error(w, "invalid jira webhook payload", http.StatusBadRequest)
			return
		}
		secrets, err := webhookSecretsForIssue(repos, raw, cfg.EnvLookup)
		if err != nil {
			cfg.Logger.Warn("jira webhook: secret unavailable", "err", err)
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		if !verifyAnySignature(secrets, body, r.Header.Get(jiraSignatureHeader)) {
			cfg.Logger.Warn("jira webhook: invalid signature")
			http.Error(w, "invalid jira webhook signature", http.StatusBadRequest)
			return
		}
		deliveryID := deliveryID(r.Header, body)
		if dedup.seen(deliveryID) {
			cfg.Logger.Info("jira webhook: duplicate delivery ignored", "delivery_id", deliveryID)
			http.Error(w, "duplicate jira webhook delivery", http.StatusConflict)
			return
		}
		result, err := MirrorRawIssue(r.Context(), repos, raw)
		if err != nil {
			cfg.Logger.Warn("jira webhook: mirror failed", "jira_key", raw.Key, "err", sanitizeError(err))
			http.Error(w, "jira webhook mirror failed", http.StatusInternalServerError)
			return
		}
		cfg.Logger.Info("jira webhook: processed",
			"jira_key", raw.Key,
			"project", raw.Project,
			"status", result.Status,
			"reason", result.Reason,
			"ticket", result.TicketPath,
		)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(result)
	})
}

func webhookSecretsForIssue(repos []Repository, raw RawIssue, lookup func(string) (string, bool)) ([]string, error) {
	resolved, result := resolveRepository(repos, raw.Key, raw.Project)
	if result.Status == "" {
		return webhookSecretForRepo(resolved, lookup)
	}
	return webhookSecrets(repos, lookup)
}

func webhookSecretForRepo(repo Repository, lookup func(string) (string, bool)) ([]string, error) {
	envName := strings.TrimSpace(repo.Config.Ingestion.JIRA.WebhookSecretEnv)
	if envName == "" {
		return nil, fmt.Errorf("jira webhook secret unavailable - configure webhook_secret_env")
	}
	value, ok := lookup(envName)
	if !ok || strings.TrimSpace(value) == "" {
		return nil, fmt.Errorf("jira webhook secret unavailable - set env var %s", envName)
	}
	return []string{value}, nil
}

func webhookSecrets(repos []Repository, lookup func(string) (string, bool)) ([]string, error) {
	var out []string
	var missing []string
	for _, repo := range repos {
		if !repo.Config.JIRAEnabled() {
			continue
		}
		envName := strings.TrimSpace(repo.Config.Ingestion.JIRA.WebhookSecretEnv)
		if envName == "" {
			missing = append(missing, "webhook_secret_env")
			continue
		}
		value, ok := lookup(envName)
		if !ok || strings.TrimSpace(value) == "" {
			missing = append(missing, envName)
			continue
		}
		out = append(out, value)
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("jira webhook secret unavailable - set env var %s", strings.Join(uniqueStrings(missing), ", "))
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("jira webhook secret unavailable - configure webhook_secret_env")
	}
	return out, nil
}

func verifyAnySignature(secrets []string, body []byte, signature string) bool {
	if !strings.HasPrefix(signature, jiraSignaturePrefix) {
		return false
	}
	got, err := hex.DecodeString(strings.TrimPrefix(signature, jiraSignaturePrefix))
	if err != nil {
		return false
	}
	for _, secret := range secrets {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		if hmac.Equal(got, mac.Sum(nil)) {
			return true
		}
	}
	return false
}

func SignWebhookPayloadForTest(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return jiraSignaturePrefix + hex.EncodeToString(mac.Sum(nil))
}

func deliveryID(header http.Header, body []byte) string {
	for _, key := range []string{"X-Atlassian-Webhook-Identifier", "X-Mars-Harness-Delivery", "X-Request-Id"} {
		if value := strings.TrimSpace(header.Get(key)); value != "" {
			return value
		}
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

type dedupStore struct {
	mu      sync.Mutex
	entries map[string]time.Time
}

func newDedupStore() *dedupStore {
	return &dedupStore{entries: map[string]time.Time{}}
}

func (d *dedupStore) seen(id string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.entries[id]; ok {
		return true
	}
	d.entries[id] = time.Now()
	return false
}

func sanitizeError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s", redactSensitiveText(err.Error()))
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	var out []string
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

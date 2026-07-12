/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/github-app-integration.md
- docs/features/F-006-queue-and-orchestration.md
- docs/features/F-011-optional-github-integration.md
- docs/features/F-017-open-source-publication.md
- docs/product-specs/product-surface.md
*/
package github

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const testSecret = "0123456789abcdef0123456789abcdef"

func sign(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func webhookRequest(eventType, deliveryID string, payload []byte, secret string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
	req.Header.Set("X-GitHub-Event", eventType)
	req.Header.Set("X-GitHub-Delivery", deliveryID)
	if secret != "" {
		req.Header.Set("X-Hub-Signature-256", sign(secret, payload))
	}
	return req
}

func enabledConfig() WebhookConfig {
	return WebhookConfig{
		Secret:          testSecret,
		AllowedActorIDs: []int64{42},
		HasRepositories: func(context.Context) (bool, error) { return true, nil },
		ResolveRepository: func(_ context.Context, repo string) (RepositoryPolicy, bool, error) {
			if repo != "owner/repo" {
				return RepositoryPolicy{}, false, nil
			}
			return RepositoryPolicy{Repository: "owner/repo", Branch: "main"}, true, nil
		},
	}
}

func eventFixtures() map[string][]byte {
	return map[string][]byte{
		"push":         []byte(`{"ref":"refs/heads/main","sender":{"id":42,"login":"trusted"},"repository":{"full_name":"Owner/Repo"}}`),
		"workflow_run": []byte(`{"action":"completed","sender":{"id":999},"repository":{"full_name":"owner/repo"},"workflow_run":{"actor":{"id":42,"login":"trusted"},"head_repository":{"full_name":"owner/repo"},"head_branch":"main","conclusion":"failure"}}`),
		"pull_request": []byte(`{"action":"opened","sender":{"id":42,"login":"trusted"},"repository":{"full_name":"owner/repo"},"pull_request":{"base":{"ref":"main","repo":{"full_name":"owner/repo"}},"head":{"ref":"feature","repo":{"full_name":"owner/repo","fork":false}}}}`),
		"check_suite":  []byte(`{"action":"completed","sender":{"id":42,"login":"trusted"},"repository":{"id":1001,"full_name":"owner/repo","private":true},"check_suite":{"id":2002,"head_branch":"main","head_sha":"redacted-sha","conclusion":"failure","latest_check_runs_count":1}}`),
		"merge_group":  []byte(`{"action":"checks_requested","sender":{"id":42,"login":"trusted"},"repository":{"full_name":"owner/repo"},"merge_group":{"base_ref":"refs/heads/main","head_ref":"refs/heads/gh-readonly-queue/main/pr-1"}}`),
	}
}

func TestWebhookHandlerAuthorizedEventFixtures(t *testing.T) {
	t.Parallel()
	for eventType, body := range eventFixtures() {
		eventType, body := eventType, body
		t.Run(eventType, func(t *testing.T) {
			t.Parallel()
			var received Event
			h := WebhookHandler(enabledConfig(), func(_ context.Context, event Event) error {
				received = event
				return nil
			})
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, webhookRequest(eventType, "delivery-"+eventType, body, testSecret))
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			require.Equal(t, int64(42), received.ActorID)
			require.Equal(t, "owner/repo", received.Repo)
			require.Equal(t, "main", received.Branch)
			require.Len(t, received.BodySHA, 64)
		})
	}
}

func TestWebhookHandlerDisabledPolicyReturns503(t *testing.T) {
	t.Parallel()
	body := eventFixtures()["push"]
	for name, cfg := range map[string]WebhookConfig{
		"missing secret":       {},
		"short secret":         {Secret: "short", AllowedActorIDs: []int64{42}},
		"missing actors":       {Secret: testSecret},
		"missing repositories": {Secret: testSecret, AllowedActorIDs: []int64{42}, HasRepositories: func(context.Context) (bool, error) { return false, nil }, ResolveRepository: enabledConfig().ResolveRepository},
	} {
		name, cfg := name, cfg
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var called atomic.Bool
			h := WebhookHandler(cfg, func(context.Context, Event) error { called.Store(true); return nil })
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, webhookRequest("push", "disabled", body, testSecret))
			require.Equal(t, http.StatusServiceUnavailable, rec.Code)
			require.False(t, called.Load())
		})
	}
}

func TestWebhookHandlerInvalidDirectActorPoliciesFailClosed(t *testing.T) {
	t.Parallel()
	body := eventFixtures()["push"]
	overLimit := make([]int64, MaxWebhookActorIDs+1)
	for i := range overLimit {
		overLimit[i] = int64(i + 1)
	}
	invalid := []WebhookConfig{enabledConfig(), enabledConfig()}
	invalid[0].AllowedActorIDs = []int64{42, 0}
	invalid[1].AllowedActorIDs = overLimit
	for i, cfg := range invalid {
		rec := httptest.NewRecorder()
		WebhookHandler(cfg, func(context.Context, Event) error { t.Fatal("unexpected dispatch"); return nil }).ServeHTTP(rec, webhookRequest("push", fmt.Sprintf("invalid-policy-%d", i), body, testSecret))
		require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	}

	cfg := enabledConfig()
	cfg.AllowedActorIDs = []int64{42, 42}
	rec := httptest.NewRecorder()
	WebhookHandler(cfg, func(context.Context, Event) error { return nil }).ServeHTTP(rec, webhookRequest("push", "deduped-policy", body, testSecret))
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestWebhookHandlerTransportAndMethodFailures(t *testing.T) {
	t.Parallel()
	cfg := enabledConfig()
	body := eventFixtures()["push"]
	for name, tc := range map[string]struct {
		req    *http.Request
		status int
	}{
		"invalid signature": {webhookRequest("push", "bad-sig", body, "wrong-secret-that-is-still-long-enough"), http.StatusUnauthorized},
		"missing signature": {webhookRequest("push", "no-sig", body, ""), http.StatusUnauthorized},
		"missing delivery":  {webhookRequest("push", "", body, testSecret), http.StatusBadRequest},
		"missing event":     {webhookRequest("", "no-event", body, testSecret), http.StatusBadRequest},
		"wrong method":      {httptest.NewRequest(http.MethodGet, "/webhook", nil), http.StatusMethodNotAllowed},
	} {
		name, req, status := name, tc.req, tc.status
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			h := WebhookHandler(cfg, func(context.Context, Event) error { t.Fatal("unexpected dispatch"); return nil })
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			require.Equal(t, status, rec.Code)
		})
	}
}

func TestWebhookHandlerOversizedBodyReturns413(t *testing.T) {
	t.Parallel()
	cfg := enabledConfig()
	cfg.MaxBodySize = 32
	body := bytes.Repeat([]byte("x"), 64)
	rec := httptest.NewRecorder()
	WebhookHandler(cfg, func(context.Context, Event) error { t.Fatal("unexpected dispatch"); return nil }).ServeHTTP(rec, webhookRequest("push", "large", body, testSecret))
	require.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
}

func TestWebhookHandlerUnsupportedAndIssueCommentNeverDispatch(t *testing.T) {
	t.Parallel()
	cfg := enabledConfig()
	for _, eventType := range []string{"deployment_status", "issue_comment"} {
		rec := httptest.NewRecorder()
		WebhookHandler(cfg, func(context.Context, Event) error { t.Fatal("unexpected dispatch"); return nil }).ServeHTTP(rec, webhookRequest(eventType, eventType, []byte(`{}`), testSecret))
		require.Equal(t, http.StatusAccepted, rec.Code)
	}
}

func TestWebhookHandlerAuthorizationFailuresDoNotPoisonReplay(t *testing.T) {
	t.Parallel()
	cfg := enabledConfig()
	fixtures := map[string][]byte{
		"actor":  []byte(`{"ref":"refs/heads/main","sender":{"id":7},"repository":{"full_name":"owner/repo"}}`),
		"repo":   []byte(`{"ref":"refs/heads/main","sender":{"id":42},"repository":{"full_name":"owner/other"}}`),
		"branch": []byte(`{"ref":"refs/heads/dev","sender":{"id":42},"repository":{"full_name":"owner/repo"}}`),
		"fork":   []byte(`{"action":"opened","sender":{"id":42},"repository":{"full_name":"owner/repo"},"pull_request":{"base":{"ref":"main","repo":{"full_name":"owner/repo"}},"head":{"ref":"feature","repo":{"full_name":"fork/repo","fork":true}}}}`),
	}
	for name, body := range fixtures {
		name, body := name, body
		t.Run(name, func(t *testing.T) {
			var calls int
			h := WebhookHandler(cfg, func(context.Context, Event) error { calls++; return nil })
			for i := 0; i < 2; i++ {
				rec := httptest.NewRecorder()
				eventType := "push"
				if name == "fork" {
					eventType = "pull_request"
				}
				h.ServeHTTP(rec, webhookRequest(eventType, "same-rejected-id", body, testSecret))
				require.Equal(t, http.StatusAccepted, rec.Code)
			}
			require.Zero(t, calls)
		})
	}
}

func TestWebhookHandlerMalformedEventMetadataReturns400(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		event string
		body  []byte
	}{
		"push":         {"push", []byte(`{"ref":"main","sender":{"id":42},"repository":{"full_name":"owner/repo"}}`)},
		"workflow_run": {"workflow_run", []byte(`{"action":"completed","workflow_run":{"actor":{"id":42},"head_repository":{},"head_branch":"main"}}`)},
		"pull_request": {"pull_request", []byte(`{"action":"opened","sender":{"id":42},"pull_request":{"base":{"ref":"main","repo":{"full_name":"owner/repo"}},"head":{"ref":"","repo":{"full_name":"owner/repo","fork":false}}}}`)},
		"check_suite":  {"check_suite", []byte(`{"action":"","sender":{"id":42},"repository":{"full_name":"owner/repo"},"check_suite":{"head_branch":"main"}}`)},
		"merge_group":  {"merge_group", []byte(`{"action":"","sender":{"id":42},"repository":{"full_name":"owner/repo"},"merge_group":{"base_ref":"refs/heads/main"}}`)},
	} {
		rec := httptest.NewRecorder()
		WebhookHandler(enabledConfig(), func(context.Context, Event) error { t.Fatal("unexpected dispatch"); return nil }).ServeHTTP(rec, webhookRequest(tc.event, name, tc.body, testSecret))
		require.Equal(t, http.StatusBadRequest, rec.Code)
	}
}

func TestWebhookHandlerRequiredActionValidationMatrix(t *testing.T) {
	t.Parallel()
	for _, eventType := range []string{"workflow_run", "pull_request", "check_suite", "merge_group"} {
		eventType := eventType
		for _, action := range []string{"", " ", "bad\naction", strings.Repeat("x", maxEventType+1)} {
			action := action
			t.Run(eventType+fmt.Sprintf("_%q", action), func(t *testing.T) {
				t.Parallel()
				var payload map[string]any
				require.NoError(t, json.Unmarshal(eventFixtures()[eventType], &payload))
				payload["action"] = action
				body, err := json.Marshal(payload)
				require.NoError(t, err)
				rec := httptest.NewRecorder()
				WebhookHandler(enabledConfig(), func(context.Context, Event) error { t.Fatal("unexpected dispatch"); return nil }).ServeHTTP(rec, webhookRequest(eventType, "action-"+eventType+fmt.Sprint(len(action)), body, testSecret))
				require.Equal(t, http.StatusBadRequest, rec.Code)
			})
		}
	}
}

func TestWebhookHandlerBoundedNonCompletedWorkflowActionIsUnauthorized(t *testing.T) {
	t.Parallel()
	var payload map[string]any
	require.NoError(t, json.Unmarshal(eventFixtures()["workflow_run"], &payload))
	payload["action"] = "requested"
	body, err := json.Marshal(payload)
	require.NoError(t, err)
	rec := httptest.NewRecorder()
	WebhookHandler(enabledConfig(), func(context.Context, Event) error { t.Fatal("unexpected dispatch"); return nil }).ServeHTTP(rec, webhookRequest("workflow_run", "workflow-requested", body, testSecret))
	require.Equal(t, http.StatusAccepted, rec.Code)
}

func TestWebhookHandlerReplayBindsDeliveryAndBodyAndIsConcurrent(t *testing.T) {
	t.Parallel()
	var calls atomic.Int64
	h := WebhookHandler(enabledConfig(), func(context.Context, Event) error {
		calls.Add(1)
		time.Sleep(10 * time.Millisecond)
		return nil
	})
	body := eventFixtures()["push"]
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, webhookRequest("push", "concurrent", body, testSecret))
			require.Equal(t, http.StatusOK, rec.Code)
		}()
	}
	wg.Wait()
	require.Equal(t, int64(1), calls.Load())

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, webhookRequest("push", "changed-delivery", body, testSecret))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(1), calls.Load(), "same signed body with changed delivery must not dispatch twice")
}

func TestWebhookHandlerCallbackFailureReleasesReplay(t *testing.T) {
	t.Parallel()
	var calls atomic.Int64
	h := WebhookHandler(enabledConfig(), func(context.Context, Event) error {
		if calls.Add(1) == 1 {
			return fmt.Errorf("queue unavailable")
		}
		return nil
	})
	body := eventFixtures()["push"]
	first := httptest.NewRecorder()
	h.ServeHTTP(first, webhookRequest("push", "retryable", body, testSecret))
	require.Equal(t, http.StatusInternalServerError, first.Code)
	second := httptest.NewRecorder()
	h.ServeHTTP(second, webhookRequest("push", "retryable", body, testSecret))
	require.Equal(t, http.StatusOK, second.Code)
	require.Equal(t, int64(2), calls.Load())
}

func TestReplayStoreCapAndTTL(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	store := newReplayStore(2, time.Hour)
	dup, full := store.claim("a", "sha-a", now)
	require.False(t, dup)
	require.False(t, full)
	store.commit("a")
	dup, full = store.claim("b", "sha-b", now)
	require.False(t, dup)
	require.False(t, full)
	store.commit("b")
	dup, full = store.claim("c", "sha-c", now)
	require.False(t, dup)
	require.True(t, full)
	dup, full = store.claim("c", "sha-c", now.Add(2*time.Hour))
	require.False(t, dup)
	require.False(t, full)
	require.Equal(t, 1, store.size())
}

func TestHealthHandlerMethods(t *testing.T) {
	t.Parallel()
	for _, method := range []string{http.MethodGet, http.MethodHead} {
		rec := httptest.NewRecorder()
		HealthHandler().ServeHTTP(rec, httptest.NewRequest(method, "/healthz", nil))
		require.Equal(t, http.StatusOK, rec.Code)
	}
	rec := httptest.NewRecorder()
	HealthHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/healthz", nil))
	require.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestVerifySignature(t *testing.T) {
	t.Parallel()
	body := []byte(`{"ok":true}`)
	require.True(t, verifySignature(testSecret, body, sign(testSecret, body)))
	require.False(t, verifySignature(testSecret, body, "sha256=not-hex"))
	require.False(t, verifySignature(testSecret, body, sign("wrong", body)))
}

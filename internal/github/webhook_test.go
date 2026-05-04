/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/features/F-011-optional-github-integration.md
- docs/product-specs/product-surface.md
*/
package github

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const testSecret = "test-webhook-secret-42"

func sign(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func webhookRequest(t *testing.T, eventType, deliveryID string, payload []byte, secret string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
	req.Header.Set("X-GitHub-Event", eventType)
	req.Header.Set("X-GitHub-Delivery", deliveryID)
	if secret != "" {
		req.Header.Set("X-Hub-Signature-256", sign(secret, payload))
	}
	return req
}

func samplePayload(action, repo string) []byte {
	p := map[string]any{
		"action": action,
		"repository": map[string]any{
			"full_name": repo,
		},
	}
	b, _ := json.Marshal(p)
	return b
}

// --- HMAC validation ---

func TestWebhookHandler_validSignature(t *testing.T) {
	t.Parallel()
	var received Event
	handler := WebhookHandler(WebhookConfig{Secret: testSecret}, func(e Event) {
		received = e
	})

	body := samplePayload("opened", "org/repo")
	req := webhookRequest(t, "pull_request", "delivery-1", body, testSecret)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "delivery-1", received.ID)
	require.Equal(t, "pull_request", received.Type)
	require.Equal(t, "opened", received.Action)
	require.Equal(t, "org/repo", received.Repo)
}

func TestWebhookHandler_invalidSignature(t *testing.T) {
	t.Parallel()
	handler := WebhookHandler(WebhookConfig{Secret: testSecret}, func(e Event) {
		t.Fatal("should not be called for invalid signature")
	})

	body := samplePayload("opened", "org/repo")
	req := webhookRequest(t, "pull_request", "delivery-2", body, "wrong-secret")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "invalid webhook signature")
}

func TestWebhookHandler_tamperedBody(t *testing.T) {
	t.Parallel()
	handler := WebhookHandler(WebhookConfig{Secret: testSecret}, func(e Event) {
		t.Fatal("should not be called for tampered body")
	})

	original := samplePayload("opened", "org/repo")
	tampered := append(original, []byte(`,"extra":"data"}`)...)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(tampered))
	req.Header.Set("X-GitHub-Event", "pull_request")
	req.Header.Set("X-GitHub-Delivery", "delivery-tamper")
	req.Header.Set("X-Hub-Signature-256", sign(testSecret, original))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestWebhookHandler_missingSignatureHeader(t *testing.T) {
	t.Parallel()
	handler := WebhookHandler(WebhookConfig{Secret: testSecret}, func(e Event) {
		t.Fatal("should not be called")
	})

	body := samplePayload("opened", "org/repo")
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", "pull_request")
	req.Header.Set("X-GitHub-Delivery", "delivery-nosig")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestWebhookHandler_noSecretConfigured(t *testing.T) {
	t.Parallel()
	var called bool
	handler := WebhookHandler(WebhookConfig{}, func(e Event) {
		called = true
	})

	body := samplePayload("opened", "org/repo")
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", "push")
	req.Header.Set("X-GitHub-Delivery", "delivery-nosec")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, called, "handler should process when no secret is configured")
}

// --- Deduplication ---

func TestWebhookHandler_dedup(t *testing.T) {
	t.Parallel()
	var callCount int
	handler := WebhookHandler(WebhookConfig{Secret: testSecret}, func(e Event) {
		callCount++
	})

	body := samplePayload("opened", "org/repo")

	for i := 0; i < 3; i++ {
		req := webhookRequest(t, "pull_request", "delivery-dup", body, testSecret)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if i == 0 {
			require.Equal(t, http.StatusOK, rec.Code)
		} else {
			require.Equal(t, http.StatusConflict, rec.Code)
		}
	}
	require.Equal(t, 1, callCount, "handler should only be called once for duplicate deliveries")
}

func TestDedupStore_seenAndSweep(t *testing.T) {
	t.Parallel()
	ds := &dedupStore{entries: make(map[string]time.Time)}

	require.False(t, ds.seen("a"))
	require.True(t, ds.seen("a"))
	require.False(t, ds.seen("b"))

	ds.mu.Lock()
	ds.entries["old"] = time.Now().Add(-2 * dedupTTL)
	ds.mu.Unlock()

	ds.sweep()

	ds.mu.Lock()
	_, hasOld := ds.entries["old"]
	_, hasA := ds.entries["a"]
	ds.mu.Unlock()

	require.False(t, hasOld, "expired entries should be swept")
	require.True(t, hasA, "recent entries should survive sweep")
}

// --- Supported / unsupported events ---

func TestWebhookHandler_allSupportedEvents(t *testing.T) {
	t.Parallel()
	events := []string{"push", "pull_request", "check_suite", "workflow_run", "merge_group", "issue_comment"}

	for _, evt := range events {
		evt := evt
		t.Run(evt, func(t *testing.T) {
			t.Parallel()
			var received bool
			handler := WebhookHandler(WebhookConfig{Secret: testSecret}, func(e Event) {
				received = true
				require.Equal(t, evt, e.Type)
			})

			body := samplePayload("completed", "org/repo")
			req := webhookRequest(t, evt, fmt.Sprintf("del-%s", evt), body, testSecret)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			require.Equal(t, http.StatusOK, rec.Code)
			require.True(t, received)
		})
	}
}

func TestWebhookHandler_unknownEventReturns202(t *testing.T) {
	t.Parallel()
	handler := WebhookHandler(WebhookConfig{Secret: testSecret}, func(e Event) {
		t.Fatal("should not be called for unknown events")
	})

	body := samplePayload("completed", "org/repo")
	req := webhookRequest(t, "deployment_status", "delivery-unknown", body, testSecret)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusAccepted, rec.Code)
	require.Contains(t, rec.Body.String(), "not handled")
}

// --- Oversized body ---

func TestWebhookHandler_oversizedBody(t *testing.T) {
	t.Parallel()
	handler := WebhookHandler(WebhookConfig{Secret: testSecret, MaxBodySize: 100}, func(e Event) {
		t.Fatal("should not be called for oversized body")
	})

	bigBody := bytes.Repeat([]byte("x"), 200)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(bigBody))
	req.Header.Set("X-GitHub-Event", "push")
	req.Header.Set("X-GitHub-Delivery", "delivery-big")
	req.Header.Set("X-Hub-Signature-256", sign(testSecret, bigBody))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
}

// --- Missing headers ---

func TestWebhookHandler_missingDeliveryHeader(t *testing.T) {
	t.Parallel()
	handler := WebhookHandler(WebhookConfig{}, func(e Event) {
		t.Fatal("should not be called")
	})

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader("{}"))
	req.Header.Set("X-GitHub-Event", "push")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "X-GitHub-Delivery")
}

func TestWebhookHandler_missingEventHeader(t *testing.T) {
	t.Parallel()
	handler := WebhookHandler(WebhookConfig{}, func(e Event) {
		t.Fatal("should not be called")
	})

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader("{}"))
	req.Header.Set("X-GitHub-Delivery", "del-1")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "X-GitHub-Event")
}

func TestWebhookHandler_wrongMethod(t *testing.T) {
	t.Parallel()
	handler := WebhookHandler(WebhookConfig{}, func(e Event) {
		t.Fatal("should not be called")
	})

	req := httptest.NewRequest(http.MethodGet, "/webhook", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

// --- Health endpoint ---

func TestHealthHandler(t *testing.T) {
	t.Parallel()
	handler := HealthHandler()

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"status":"ok"`)
	require.Contains(t, rec.Header().Get("Content-Type"), "application/json")
}

// --- Payload parsing ---

func TestExtractEventMeta(t *testing.T) {
	t.Parallel()
	body := samplePayload("synchronize", "owner/name")
	action, repo := extractEventMeta(body)
	require.Equal(t, "synchronize", action)
	require.Equal(t, "owner/name", repo)
}

func TestExtractEventMeta_malformedJSON(t *testing.T) {
	t.Parallel()
	action, repo := extractEventMeta([]byte("not json"))
	require.Empty(t, action)
	require.Empty(t, repo)
}

// --- Concurrency ---

func TestWebhookHandler_concurrentRequests(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var events []Event
	handler := WebhookHandler(WebhookConfig{Secret: testSecret}, func(e Event) {
		mu.Lock()
		events = append(events, e)
		mu.Unlock()
	})

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			body := samplePayload("opened", "org/repo")
			deliveryID := fmt.Sprintf("concurrent-%d", i)

			req, _ := http.NewRequest(http.MethodPost, srv.URL+"/webhook", bytes.NewReader(body))
			req.Header.Set("X-GitHub-Event", "pull_request")
			req.Header.Set("X-GitHub-Delivery", deliveryID)
			req.Header.Set("X-Hub-Signature-256", sign(testSecret, body))

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return
			}
			resp.Body.Close()
		}(i)
	}
	wg.Wait()

	mu.Lock()
	require.Len(t, events, 20, "all unique deliveries should be processed")
	mu.Unlock()
}

// --- Verify signature helper ---

func TestVerifySignature_valid(t *testing.T) {
	t.Parallel()
	body := []byte(`{"action":"opened"}`)
	sig := sign(testSecret, body)
	require.True(t, verifySignature(testSecret, body, sig))
}

func TestVerifySignature_invalidHex(t *testing.T) {
	t.Parallel()
	require.False(t, verifySignature(testSecret, []byte("x"), "sha256=not-hex!!"))
}

func TestVerifySignature_noPrefix(t *testing.T) {
	t.Parallel()
	require.False(t, verifySignature(testSecret, []byte("x"), "md5=abc"))
}

func TestVerifySignature_wrongSecret(t *testing.T) {
	t.Parallel()
	body := []byte(`{"test":true}`)
	sig := sign("correct-secret", body)
	require.False(t, verifySignature("wrong-secret", body, sig))
}

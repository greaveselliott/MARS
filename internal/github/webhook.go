package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	defaultMaxBodySize = 10 << 20 // 10 MB
	dedupTTL           = 1 * time.Hour
	dedupSweepInterval = 10 * time.Minute
	signaturePrefix    = "sha256="
)

// WebhookConfig configures the receiver.
type WebhookConfig struct {
	Secret      string
	MaxBodySize int64
}

func (c WebhookConfig) maxBody() int64 {
	if c.MaxBodySize > 0 {
		return c.MaxBodySize
	}
	return defaultMaxBodySize
}

var supportedEvents = map[string]bool{
	"push":          true,
	"pull_request":  true,
	"check_suite":   true,
	"workflow_run":  true,
	"merge_group":   true,
	"issue_comment": true,
}

// dedupStore tracks recently seen delivery IDs to prevent reprocessing.
type dedupStore struct {
	mu      sync.Mutex
	entries map[string]time.Time
}

func newDedupStore() *dedupStore {
	ds := &dedupStore{entries: make(map[string]time.Time)}
	go ds.sweepLoop()
	return ds
}

// seen returns true if the ID was already recorded (duplicate). Otherwise records it.
func (d *dedupStore) seen(id string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.entries[id]; ok {
		return true
	}
	d.entries[id] = time.Now()
	return false
}

func (d *dedupStore) sweepLoop() {
	ticker := time.NewTicker(dedupSweepInterval)
	defer ticker.Stop()
	for range ticker.C {
		d.sweep()
	}
}

func (d *dedupStore) sweep() {
	d.mu.Lock()
	defer d.mu.Unlock()
	cutoff := time.Now().Add(-dedupTTL)
	for id, ts := range d.entries {
		if ts.Before(cutoff) {
			delete(d.entries, id)
		}
	}
}

// WebhookHandler returns an http.Handler that validates and normalizes GitHub webhooks.
// Valid events are dispatched to onEvent. The handler responds with:
//   - 200 for processed events
//   - 202 for unsupported event types (logged, not an error)
//   - 400 for missing headers or invalid signatures
//   - 413 for oversized payloads
//   - 409 for duplicate delivery IDs
func WebhookHandler(cfg WebhookConfig, onEvent func(Event)) http.Handler {
	dedup := newDedupStore()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "webhook endpoint accepts POST only", http.StatusMethodNotAllowed)
			return
		}

		deliveryID := r.Header.Get("X-GitHub-Delivery")
		if deliveryID == "" {
			http.Error(w, "missing X-GitHub-Delivery header — this endpoint expects GitHub webhook requests", http.StatusBadRequest)
			return
		}

		eventType := r.Header.Get("X-GitHub-Event")
		if eventType == "" {
			http.Error(w, "missing X-GitHub-Event header", http.StatusBadRequest)
			return
		}

		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, cfg.maxBody()))
		if err != nil {
			slog.Warn("github webhook: body read error", "delivery_id", deliveryID, "error", err)
			http.Error(w, "request body too large — maximum is 10MB", http.StatusRequestEntityTooLarge)
			return
		}

		if cfg.Secret != "" {
			sig := r.Header.Get("X-Hub-Signature-256")
			if !verifySignature(cfg.Secret, body, sig) {
				slog.Warn("github webhook: invalid signature", "delivery_id", deliveryID)
				http.Error(w, "invalid webhook signature — verify the webhook secret matches your GitHub App configuration", http.StatusBadRequest)
				return
			}
		}

		if dedup.seen(deliveryID) {
			slog.Info("github webhook: duplicate delivery ignored", "delivery_id", deliveryID)
			http.Error(w, "duplicate delivery", http.StatusConflict)
			return
		}

		if !supportedEvents[eventType] {
			slog.Info("github webhook: ignored unsupported event type", "type", eventType, "delivery_id", deliveryID)
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprintf(w, "event type %q is not handled", eventType)
			return
		}

		action, repo := extractEventMeta(body)

		event := Event{
			ID:       deliveryID,
			Type:     eventType,
			Action:   action,
			Repo:     repo,
			Payload:  json.RawMessage(body),
			Received: time.Now(),
		}

		onEvent(event)

		slog.Info("github webhook: processed",
			"type", eventType,
			"action", action,
			"repo", repo,
			"delivery_id", deliveryID,
		)
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})
}

// HealthHandler returns a simple health check handler.
func HealthHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok"}`)
	})
}

// verifySignature checks the HMAC-SHA256 signature from the X-Hub-Signature-256 header.
func verifySignature(secret string, body []byte, signature string) bool {
	if !strings.HasPrefix(signature, signaturePrefix) {
		return false
	}
	sigHex := strings.TrimPrefix(signature, signaturePrefix)
	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := mac.Sum(nil)

	return hmac.Equal(sigBytes, expected)
}

// extractEventMeta pulls the "action" and "repository.full_name" from a webhook payload.
func extractEventMeta(body []byte) (action, repo string) {
	var meta struct {
		Action string `json:"action"`
		Repo   struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
	}
	if err := json.Unmarshal(body, &meta); err == nil {
		action = meta.Action
		repo = meta.Repo.FullName
	}
	return
}

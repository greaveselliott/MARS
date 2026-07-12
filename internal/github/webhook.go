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
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode"
)

const (
	defaultMaxBodySize = 2 << 20 // 2 MiB
	defaultReplayLimit = 10_000
	defaultReplayTTL   = 24 * time.Hour
	signaturePrefix    = "sha256="
	minWebhookSecret   = 32
	maxDeliveryID      = 128
	maxEventType       = 64
)

// RepositoryPolicy is the exact registered repository boundary for an event.
type RepositoryPolicy struct {
	Repository string
	Branch     string
}

// WebhookConfig configures fail-closed GitHub webhook ingress.
type WebhookConfig struct {
	Secret            string
	AllowedActorIDs   []int64
	HasRepositories   func(context.Context) (bool, error)
	ResolveRepository func(context.Context, string) (RepositoryPolicy, bool, error)
	MaxBodySize       int64
	ReplayLimit       int
	ReplayTTL         time.Duration
	Now               func() time.Time
}

func (c WebhookConfig) maxBody() int64 {
	if c.MaxBodySize > 0 {
		return c.MaxBodySize
	}
	return defaultMaxBodySize
}

func (c WebhookConfig) now() time.Time {
	if c.Now != nil {
		return c.Now().UTC()
	}
	return time.Now().UTC()
}

func (c WebhookConfig) replayLimit() int {
	if c.ReplayLimit > 0 {
		return c.ReplayLimit
	}
	return defaultReplayLimit
}

func (c WebhookConfig) replayTTL() time.Duration {
	if c.ReplayTTL > 0 {
		return c.ReplayTTL
	}
	return defaultReplayTTL
}

func (c WebhookConfig) allowedActors() (map[int64]struct{}, error) {
	ids, err := NormalizeWebhookActorIDs(c.AllowedActorIDs)
	if err != nil {
		return nil, err
	}
	out := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		out[id] = struct{}{}
	}
	return out, nil
}

var supportedEvents = map[string]bool{
	"push":         true,
	"pull_request": true,
	"check_suite":  true,
	"workflow_run": true,
	"merge_group":  true,
}

type replayEntry struct {
	delivery string
	digest   string
	seenAt   time.Time
	pending  bool
}

// replayStore binds both the delivery ID and exact body digest. Entries are
// reserved before dispatch, committed only on success, and rolled back on
// callback failure so transient queue errors are retryable.
type replayStore struct {
	mu         sync.Mutex
	byDelivery map[string]*replayEntry
	byDigest   map[string]*replayEntry
	limit      int
	ttl        time.Duration
}

func newReplayStore(limit int, ttl time.Duration) *replayStore {
	return &replayStore{
		byDelivery: make(map[string]*replayEntry),
		byDigest:   make(map[string]*replayEntry),
		limit:      limit,
		ttl:        ttl,
	}
}

func (s *replayStore) claim(delivery, digest string, now time.Time) (duplicate bool, full bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sweepLocked(now)
	if _, ok := s.byDelivery[delivery]; ok {
		return true, false
	}
	if _, ok := s.byDigest[digest]; ok {
		return true, false
	}
	if len(s.byDelivery) >= s.limit {
		return false, true
	}
	entry := &replayEntry{delivery: delivery, digest: digest, seenAt: now, pending: true}
	s.byDelivery[delivery] = entry
	s.byDigest[digest] = entry
	return false, false
}

func (s *replayStore) commit(delivery string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if entry := s.byDelivery[delivery]; entry != nil {
		entry.pending = false
	}
}

func (s *replayStore) release(delivery string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry := s.byDelivery[delivery]
	if entry == nil {
		return
	}
	delete(s.byDelivery, entry.delivery)
	delete(s.byDigest, entry.digest)
}

func (s *replayStore) sweepLocked(now time.Time) {
	cutoff := now.Add(-s.ttl)
	for delivery, entry := range s.byDelivery {
		if !entry.pending && entry.seenAt.Before(cutoff) {
			delete(s.byDelivery, delivery)
			delete(s.byDigest, entry.digest)
		}
	}
}

func (s *replayStore) size() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.byDelivery)
}

type eventPayload struct {
	Action string `json:"action"`
	Sender struct {
		ID    int64  `json:"id"`
		Login string `json:"login"`
	} `json:"sender"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	Ref         string `json:"ref"`
	WorkflowRun struct {
		Actor struct {
			ID    int64  `json:"id"`
			Login string `json:"login"`
		} `json:"actor"`
		HeadRepository struct {
			FullName string `json:"full_name"`
		} `json:"head_repository"`
		HeadBranch string `json:"head_branch"`
	} `json:"workflow_run"`
	PullRequest struct {
		Base struct {
			Ref  string `json:"ref"`
			Repo struct {
				FullName string `json:"full_name"`
			} `json:"repo"`
		} `json:"base"`
		Head struct {
			Ref  string `json:"ref"`
			Repo struct {
				FullName string `json:"full_name"`
				Fork     bool   `json:"fork"`
			} `json:"repo"`
		} `json:"head"`
	} `json:"pull_request"`
	CheckSuite struct {
		HeadBranch string `json:"head_branch"`
	} `json:"check_suite"`
	MergeGroup struct {
		BaseRef string `json:"base_ref"`
	} `json:"merge_group"`
}

type authorizationError struct {
	malformed bool
	reason    string
}

func (e *authorizationError) Error() string { return e.reason }

// WebhookHandler validates, authorizes, and normalizes GitHub webhooks before
// invoking onEvent. The callback must durably accept the event before returning
// nil; callback errors release the in-memory replay reservation.
func WebhookHandler(cfg WebhookConfig, onEvent func(context.Context, Event) error) http.Handler {
	replay := newReplayStore(cfg.replayLimit(), cfg.replayTTL())
	allowedActors, actorPolicyErr := cfg.allowedActors()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "webhook endpoint accepts POST only", http.StatusMethodNotAllowed)
			return
		}
		if actorPolicyErr != nil || len(cfg.Secret) < minWebhookSecret || len(allowedActors) == 0 || cfg.HasRepositories == nil || cfg.ResolveRepository == nil {
			http.Error(w, "GitHub webhook ingress is disabled; configure a webhook secret of at least 32 bytes, trusted numeric actor IDs, and an exact registered owner/repo remote", http.StatusServiceUnavailable)
			return
		}
		hasRepos, err := cfg.HasRepositories(r.Context())
		if err != nil {
			slog.Warn("github webhook: repository policy unavailable", "error", err)
			http.Error(w, "GitHub webhook repository policy is temporarily unavailable", http.StatusServiceUnavailable)
			return
		}
		if !hasRepos {
			http.Error(w, "GitHub webhook ingress is disabled; register an exact owner/repo remote first", http.StatusServiceUnavailable)
			return
		}

		deliveryID := r.Header.Get("X-GitHub-Delivery")
		if !validBoundedToken(deliveryID, maxDeliveryID) {
			http.Error(w, "missing or invalid X-GitHub-Delivery header", http.StatusBadRequest)
			return
		}
		eventType := r.Header.Get("X-GitHub-Event")
		if !validBoundedToken(eventType, maxEventType) {
			http.Error(w, "missing or invalid X-GitHub-Event header", http.StatusBadRequest)
			return
		}

		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, cfg.maxBody()))
		if err != nil {
			http.Error(w, fmt.Sprintf("request body too large; maximum is %d bytes", cfg.maxBody()), http.StatusRequestEntityTooLarge)
			return
		}
		if !verifySignature(cfg.Secret, body, r.Header.Get("X-Hub-Signature-256")) {
			http.Error(w, "invalid webhook signature; verify the configured environment or owner-only GitHub App webhook secret", http.StatusUnauthorized)
			return
		}
		if eventType == "issue_comment" || !supportedEvents[eventType] {
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, "event accepted without autonomous dispatch")
			return
		}

		event, authErr := authorizeEvent(r.Context(), cfg, allowedActors, eventType, deliveryID, body)
		if authErr != nil {
			var denied *authorizationError
			if errors.As(authErr, &denied) && !denied.malformed {
				slog.Info("github webhook: event not authorized", "type", eventType, "reason", bounded(denied.reason, 96))
				w.WriteHeader(http.StatusAccepted)
				fmt.Fprint(w, "event accepted without autonomous dispatch")
				return
			}
			http.Error(w, "malformed GitHub webhook metadata", http.StatusBadRequest)
			return
		}

		duplicate, full := replay.claim(event.ID, event.BodySHA, cfg.now())
		if duplicate {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "duplicate accepted")
			return
		}
		if full {
			http.Error(w, "GitHub webhook replay capacity is temporarily full; retry later", http.StatusServiceUnavailable)
			return
		}
		if err := onEvent(r.Context(), event); err != nil {
			replay.release(event.ID)
			slog.Error("github webhook: durable dispatch failed", "type", event.Type, "delivery_id", bounded(event.ID, maxDeliveryID), "error", err)
			http.Error(w, "GitHub webhook could not be durably queued; retry the delivery", http.StatusInternalServerError)
			return
		}
		replay.commit(event.ID)

		slog.Info("github webhook: processed", "type", event.Type, "action", bounded(event.Action, maxEventType), "repo", bounded(event.Repo, MaxRepositoryName), "branch", bounded(event.Branch, MaxBranchName), "actor_id", event.ActorID, "delivery_id", bounded(event.ID, maxDeliveryID))
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})
}

func authorizeEvent(ctx context.Context, cfg WebhookConfig, actors map[int64]struct{}, eventType, deliveryID string, body []byte) (Event, error) {
	var payload eventPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return Event{}, &authorizationError{malformed: true, reason: "invalid JSON"}
	}
	action := payload.Action
	if eventType != "push" && !validBoundedToken(action, maxEventType) {
		return Event{}, &authorizationError{malformed: true, reason: eventType + " action is missing or invalid"}
	}
	actorID, actor, repo, branch, fork := payload.Sender.ID, payload.Sender.Login, payload.Repository.FullName, "", false
	switch eventType {
	case "push":
		branch = strings.TrimPrefix(payload.Ref, "refs/heads/")
		if payload.Ref == branch {
			return Event{}, &authorizationError{malformed: true, reason: "push ref is not refs/heads/<branch>"}
		}
	case "workflow_run":
		if action != "completed" {
			return Event{}, &authorizationError{reason: "workflow_run action is not completed"}
		}
		actorID, actor = payload.WorkflowRun.Actor.ID, payload.WorkflowRun.Actor.Login
		repo, branch = payload.WorkflowRun.HeadRepository.FullName, payload.WorkflowRun.HeadBranch
	case "pull_request":
		if payload.PullRequest.Base.Repo.FullName == "" || payload.PullRequest.Base.Ref == "" ||
			payload.PullRequest.Head.Repo.FullName == "" || payload.PullRequest.Head.Ref == "" {
			return Event{}, &authorizationError{malformed: true, reason: "pull_request action/base/head metadata is incomplete"}
		}
		repo, branch = payload.PullRequest.Base.Repo.FullName, payload.PullRequest.Base.Ref
		headRepo, headErr := NormalizeRepository(payload.PullRequest.Head.Repo.FullName)
		baseRepo, baseErr := NormalizeRepository(repo)
		if headErr != nil || baseErr != nil || ValidateBranch(payload.PullRequest.Head.Ref) != nil {
			return Event{}, &authorizationError{malformed: true, reason: "pull_request repository or branch metadata is invalid"}
		}
		fork = payload.PullRequest.Head.Repo.Fork || headRepo != baseRepo
	case "check_suite":
		repo, branch = payload.Repository.FullName, payload.CheckSuite.HeadBranch
	case "merge_group":
		branch = strings.TrimPrefix(payload.MergeGroup.BaseRef, "refs/heads/")
		if payload.MergeGroup.BaseRef == branch {
			return Event{}, &authorizationError{malformed: true, reason: "merge_group base_ref is not refs/heads/<branch>"}
		}
	}

	normalizedRepo, repoErr := NormalizeRepository(repo)
	branchErr := ValidateBranch(branch)
	if actorID <= 0 || repoErr != nil || branchErr != nil || len(action) > maxEventType {
		return Event{}, &authorizationError{malformed: true, reason: "required actor/repository/branch metadata is missing or oversized"}
	}
	repo = normalizedRepo
	if _, ok := actors[actorID]; !ok {
		return Event{}, &authorizationError{reason: "actor ID is not trusted"}
	}
	policy, found, err := cfg.ResolveRepository(ctx, repo)
	if err != nil {
		return Event{}, fmt.Errorf("resolve registered repository: %w", err)
	}
	if !found {
		return Event{}, &authorizationError{reason: "repository is not registered"}
	}
	policyRepo, policyRepoErr := NormalizeRepository(policy.Repository)
	if policyRepoErr != nil || policyRepo != repo {
		return Event{}, &authorizationError{reason: "repository does not match registered remote"}
	}
	if policy.Branch != branch {
		return Event{}, &authorizationError{reason: "branch does not match registered branch"}
	}
	if fork {
		return Event{}, &authorizationError{reason: "fork-derived event is not authorized"}
	}
	digest := sha256.Sum256(body)
	return Event{
		ID: deliveryID, BodySHA: hex.EncodeToString(digest[:]), Type: eventType,
		Action: action, ActorID: actorID, Actor: bounded(actor, 96), Repo: repo,
		Branch: branch, Fork: fork, Payload: json.RawMessage(body), Received: cfg.now(),
	}, nil
}

// HealthHandler returns a method-bounded health check handler.
func HealthHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "health endpoint accepts GET or HEAD only", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if r.Method == http.MethodGet {
			fmt.Fprint(w, `{"status":"ok"}`)
		}
	})
}

func verifySignature(secret string, body []byte, signature string) bool {
	if !strings.HasPrefix(signature, signaturePrefix) {
		return false
	}
	sigBytes, err := hex.DecodeString(strings.TrimPrefix(signature, signaturePrefix))
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return hmac.Equal(sigBytes, mac.Sum(nil))
}

func bounded(value string, limit int) string {
	value = strings.TrimSpace(value)
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return '?'
		}
		return r
	}, value)
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}

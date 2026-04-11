package github

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func testKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return key
}

// --- NewClient validation ---

func TestNewClient_requiresPATToken(t *testing.T) {
	t.Parallel()
	_, err := NewClient(ClientConfig{Mode: AuthPAT})
	require.Error(t, err)
	require.Contains(t, err.Error(), "Token is required")
}

func TestNewClient_requiresAppFields(t *testing.T) {
	t.Parallel()
	key := testKey(t)

	_, err := NewClient(ClientConfig{Mode: AuthApp})
	require.ErrorContains(t, err, "AppID is required")

	_, err = NewClient(ClientConfig{Mode: AuthApp, AppID: 1})
	require.ErrorContains(t, err, "PrivateKey is required")

	_, err = NewClient(ClientConfig{Mode: AuthApp, AppID: 1, PrivateKey: key})
	require.ErrorContains(t, err, "InstallID is required")
}

func TestNewClient_rejectsUnknownMode(t *testing.T) {
	t.Parallel()
	_, err := NewClient(ClientConfig{Mode: "magic"})
	require.ErrorContains(t, err, "unknown auth mode")
}

func TestNewClient_PAT_happyPath(t *testing.T) {
	t.Parallel()
	c, err := NewClient(ClientConfig{Mode: AuthPAT, Token: "ghp_test123"})
	require.NoError(t, err)
	require.Equal(t, defaultBaseURL, c.baseURL)
}

func TestNewClient_defaultsBaseURL(t *testing.T) {
	t.Parallel()
	c, err := NewClient(ClientConfig{Mode: AuthPAT, Token: "t", BaseURL: "https://ghe.corp.com/api/v3/"})
	require.NoError(t, err)
	require.Equal(t, "https://ghe.corp.com/api/v3", c.baseURL)
}

// --- JWT generation ---

func TestGenerateJWT_validStructure(t *testing.T) {
	t.Parallel()
	key := testKey(t)
	c := &Client{cfg: ClientConfig{AppID: 42, PrivateKey: key}}

	jwt, err := c.generateJWT()
	require.NoError(t, err)

	parts := strings.Split(jwt, ".")
	require.Len(t, parts, 3, "JWT must have 3 dot-separated parts")

	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	require.NoError(t, err)
	require.Contains(t, string(headerBytes), `"alg":"RS256"`)

	claimsBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	var claims map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(claimsBytes, &claims))
	require.Contains(t, string(claims["iss"]), "42")

	sigBytes, err := base64.RawURLEncoding.DecodeString(parts[2])
	require.NoError(t, err)

	sigInput := parts[0] + "." + parts[1]
	hashed := sha256.Sum256([]byte(sigInput))
	err = rsa.VerifyPKCS1v15(&key.PublicKey, crypto.SHA256, hashed[:], sigBytes)
	require.NoError(t, err, "JWT signature must verify with the public key")
}

// --- Token refresh for App mode ---

func TestClient_AppMode_tokenRefresh(t *testing.T) {
	t.Parallel()
	key := testKey(t)

	var tokenRequests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/access_tokens") {
			tokenRequests.Add(1)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(installationToken{
				Token:     "ghs_freshtoken",
				ExpiresAt: time.Now().Add(1 * time.Hour),
			})
			return
		}
		if strings.HasSuffix(r.URL.Path, "/comments") {
			require.Equal(t, "Bearer ghs_freshtoken", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusCreated)
			return
		}
		http.Error(w, "not found", 404)
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(ClientConfig{
		Mode:       AuthApp,
		BaseURL:    srv.URL,
		AppID:      1,
		PrivateKey: key,
		InstallID:  99,
		HTTPClient: srv.Client(),
	})
	require.NoError(t, err)

	err = c.PostComment(context.Background(), "o", "r", 1, "hello")
	require.NoError(t, err)
	require.EqualValues(t, 1, tokenRequests.Load())

	// Second call should reuse the cached token.
	err = c.PostComment(context.Background(), "o", "r", 1, "hello again")
	require.NoError(t, err)
	require.EqualValues(t, 1, tokenRequests.Load(), "should reuse cached token")
}

func TestClient_AppMode_tokenRefreshOnExpiry(t *testing.T) {
	t.Parallel()
	key := testKey(t)

	var tokenRequests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/access_tokens") {
			tokenRequests.Add(1)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(installationToken{
				Token:     "ghs_new",
				ExpiresAt: time.Now().Add(30 * time.Second), // expires within buffer
			})
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(ClientConfig{
		Mode:       AuthApp,
		BaseURL:    srv.URL,
		AppID:      1,
		PrivateKey: key,
		InstallID:  99,
		HTTPClient: srv.Client(),
	})
	require.NoError(t, err)

	err = c.PostComment(context.Background(), "o", "r", 1, "first")
	require.NoError(t, err)
	require.EqualValues(t, 1, tokenRequests.Load())

	// Token expires within 60s buffer, so next call should refresh.
	err = c.PostComment(context.Background(), "o", "r", 1, "second")
	require.NoError(t, err)
	require.EqualValues(t, 2, tokenRequests.Load(), "should refresh near-expired token")
}

// --- Rate limiting ---

func TestClient_rateLimitRetryAfter(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			w.Header().Set("Retry-After", "0")
			http.Error(w, `{"message":"rate limit"}`, http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(ClientConfig{
		Mode:       AuthPAT,
		Token:      "ghp_test",
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
		MaxRetries: 3,
	})
	require.NoError(t, err)

	err = c.PostComment(context.Background(), "o", "r", 1, "hi")
	require.NoError(t, err)
	require.EqualValues(t, 2, calls.Load())
}

func TestClient_rateLimitForbidden(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", "0")
			http.Error(w, `{"message":"API rate limit exceeded"}`, http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(ClientConfig{
		Mode:       AuthPAT,
		Token:      "ghp_test",
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
		MaxRetries: 3,
	})
	require.NoError(t, err)

	err = c.PostComment(context.Background(), "o", "r", 1, "hi")
	require.NoError(t, err)
	require.EqualValues(t, 2, calls.Load())
}

func TestClient_serverErrorRetries(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n < 3 {
			http.Error(w, "bad gateway", http.StatusBadGateway)
			return
		}
		_ = json.NewEncoder(w).Encode(PullRequest{Number: 42, HTMLURL: "https://github.com/o/r/pull/42"})
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(ClientConfig{
		Mode:       AuthPAT,
		Token:      "ghp_test",
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
		MaxRetries: 3,
	})
	require.NoError(t, err)

	pr, err := c.CreatePR(context.Background(), "o", "r", "title", "body", "feature", "main")
	require.NoError(t, err)
	require.Equal(t, 42, pr.Number)
}

// --- CreatePR ---

func TestCreatePR_sendsCorrectPayload(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/repos/myorg/myrepo/pulls", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "application/vnd.github+json", r.Header.Get("Accept"))

		b, _ := io.ReadAll(r.Body)
		var body map[string]string
		require.NoError(t, json.Unmarshal(b, &body))
		require.Equal(t, "Fix bug", body["title"])
		require.Equal(t, "Fixes #1", body["body"])
		require.Equal(t, "fix-branch", body["head"])
		require.Equal(t, "main", body["base"])

		_ = json.NewEncoder(w).Encode(PullRequest{Number: 7, HTMLURL: "https://github.com/myorg/myrepo/pull/7"})
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(ClientConfig{Mode: AuthPAT, Token: "t", BaseURL: srv.URL, HTTPClient: srv.Client()})
	require.NoError(t, err)

	pr, err := c.CreatePR(context.Background(), "myorg", "myrepo", "Fix bug", "Fixes #1", "fix-branch", "main")
	require.NoError(t, err)
	require.Equal(t, 7, pr.Number)
}

// --- CreateCheckRun / UpdateCheckRun ---

func TestCreateCheckRun_happyPath(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/repos/o/r/check-runs", r.URL.Path)
		b, _ := io.ReadAll(r.Body)
		var body map[string]string
		require.NoError(t, json.Unmarshal(b, &body))
		require.Equal(t, "mars-harness", body["name"])
		require.Equal(t, "abc1234", body["head_sha"])
		require.Equal(t, "in_progress", body["status"])

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(CheckRun{ID: 100, Name: "mars-harness", Status: "in_progress"})
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(ClientConfig{Mode: AuthPAT, Token: "t", BaseURL: srv.URL, HTTPClient: srv.Client()})
	require.NoError(t, err)

	cr, err := c.CreateCheckRun(context.Background(), "o", "r", "mars-harness", "abc1234", CheckInProgress)
	require.NoError(t, err)
	require.Equal(t, int64(100), cr.ID)
}

func TestUpdateCheckRun_sendsConclusion(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/repos/o/r/check-runs/100", r.URL.Path)
		require.Equal(t, http.MethodPatch, r.Method)

		b, _ := io.ReadAll(r.Body)
		var body map[string]string
		require.NoError(t, json.Unmarshal(b, &body))
		require.Equal(t, "completed", body["status"])
		require.Equal(t, "success", body["conclusion"])

		_ = json.NewEncoder(w).Encode(CheckRun{ID: 100, Status: "completed", Conclusion: "success"})
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(ClientConfig{Mode: AuthPAT, Token: "t", BaseURL: srv.URL, HTTPClient: srv.Client()})
	require.NoError(t, err)

	err = c.UpdateCheckRun(context.Background(), "o", "r", 100, CheckCompleted, "success")
	require.NoError(t, err)
}

// --- PostComment ---

func TestPostComment_sendsBody(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/repos/o/r/issues/5/comments", r.URL.Path)
		b, _ := io.ReadAll(r.Body)
		var body map[string]string
		require.NoError(t, json.Unmarshal(b, &body))
		require.Equal(t, "LGTM", body["body"])
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(ClientConfig{Mode: AuthPAT, Token: "t", BaseURL: srv.URL, HTTPClient: srv.Client()})
	require.NoError(t, err)

	err = c.PostComment(context.Background(), "o", "r", 5, "LGTM")
	require.NoError(t, err)
}

// --- Error handling ---

func TestClient_unexpectedStatus(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"Validation Failed"}`, http.StatusUnprocessableEntity)
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(ClientConfig{Mode: AuthPAT, Token: "t", BaseURL: srv.URL, HTTPClient: srv.Client()})
	require.NoError(t, err)

	_, err = c.CreatePR(context.Background(), "o", "r", "t", "b", "h", "base")
	require.Error(t, err)
	require.Contains(t, err.Error(), "Validation Failed")
}

func TestClient_contextCancellation(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(ClientConfig{
		Mode:       AuthPAT,
		Token:      "t",
		BaseURL:    srv.URL,
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
		MaxRetries: 1,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err = c.PostComment(ctx, "o", "r", 1, "hi")
	require.Error(t, err)
}

// --- PEM parsing helper for tests ---

func TestPEMRoundtrip(t *testing.T) {
	t.Parallel()
	key := testKey(t)

	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})

	block, _ := pem.Decode(pemBytes)
	require.NotNil(t, block)
	parsed, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	require.NoError(t, err)

	require.True(t, key.D.Cmp(parsed.D) == 0, "keys must round-trip through PEM")
}

// --- Helpers ---

func TestRateLimitWait_retryAfterHeader(t *testing.T) {
	t.Parallel()
	resp := &http.Response{Header: http.Header{}}
	resp.Header.Set("Retry-After", "5")
	require.Equal(t, 5*time.Second, rateLimitWait(resp))
}

func TestRateLimitWait_resetHeader(t *testing.T) {
	t.Parallel()
	future := time.Now().Add(10 * time.Second).Unix()
	resp := &http.Response{Header: http.Header{}}
	resp.Header.Set("X-RateLimit-Reset", big.NewInt(future).String())
	wait := rateLimitWait(resp)
	require.Greater(t, wait, 5*time.Second)
	require.LessOrEqual(t, wait, 11*time.Second)
}

func TestRateLimitWait_noHeaders(t *testing.T) {
	t.Parallel()
	resp := &http.Response{Header: http.Header{}}
	require.Equal(t, rateLimitSleepFloor, rateLimitWait(resp))
}

func TestBackoff_increases(t *testing.T) {
	t.Parallel()
	require.Less(t, backoff(0), backoff(1))
	require.Less(t, backoff(1), backoff(2))
	require.LessOrEqual(t, backoff(100), 5001*time.Millisecond)
}

/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/dashboard.md
- docs/features/F-010-dashboard-control-plane.md
- docs/features/F-017-open-source-publication.md
*/
package dashboard

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func securityRequest(method, path, host, origin, body string) *http.Request {
	r := httptest.NewRequest(method, "http://"+host+path, strings.NewReader(body))
	r.Host = host
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	return r
}

func loginSession(t *testing.T, h http.Handler, host, origin string, prior *http.Cookie) (*http.Cookie, string) {
	t.Helper()
	r := securityRequest(http.MethodPost, "/api/login", host, origin, `{"secret":"`+testControlSecret+`"}`)
	if prior != nil {
		r.AddCookie(prior)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("login status = %d, body=%s", w.Code, w.Body.String())
	}
	var response struct {
		CSRF string `json:"csrf_token"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	result := w.Result()
	cookies := result.Cookies()
	if len(cookies) != 1 {
		t.Fatalf("login cookies = %d, want 1", len(cookies))
	}
	return cookies[0], response.CSRF
}

func TestSecurityConfigurationFailsClosed(t *testing.T) {
	t.Parallel()
	if _, err := New(Config{ControlSecret: "short"}); err == nil || !strings.Contains(err.Error(), "shorter than 32") {
		t.Fatalf("short secret error = %v", err)
	}
	badOrigins := []string{
		"http://proxy.example", "https://*.example", "https://user@proxy.example",
		"https://proxy.example/path", "https://proxy.example?q=1", "https://proxy.example#fragment",
	}
	for _, origin := range badOrigins {
		if _, err := New(Config{ControlSecret: testControlSecret, TrustedOrigin: origin}); err == nil {
			t.Errorf("trusted origin %q unexpectedly accepted", origin)
		}
	}
	if _, err := New(Config{TrustedOrigin: "https://proxy.example"}); err == nil || !strings.Contains(err.Error(), "requires MARS_DASHBOARD_CONTROL_SECRET") {
		t.Fatalf("remote without secret error = %v", err)
	}
}

func TestMissingSecretPreservesObserverAndDisablesMutation(t *testing.T) {
	called := false
	d, err := New(Config{Controls: ControlCallbacks{Pause: func() { called = true }}})
	if err != nil {
		t.Fatal(err)
	}
	read := httptest.NewRecorder()
	d.Handler().ServeHTTP(read, securityRequest(http.MethodGet, "/pipeline", "127.0.0.1:9090", "", ""))
	if read.Code != http.StatusOK {
		t.Fatalf("observer read status = %d", read.Code)
	}
	locked := httptest.NewRecorder()
	d.Handler().ServeHTTP(locked, securityRequest(http.MethodGet, "/api/repos", "127.0.0.1:9090", "", ""))
	if locked.Code != http.StatusServiceUnavailable {
		t.Fatalf("privileged observer status = %d, want 503", locked.Code)
	}
	mutation := httptest.NewRecorder()
	d.Handler().ServeHTTP(mutation, securityRequest(http.MethodPost, "/api/pause", "127.0.0.1:9090", "http://127.0.0.1:9090", ""))
	if mutation.Code != http.StatusServiceUnavailable || called {
		t.Fatalf("mutation status=%d called=%v", mutation.Code, called)
	}
}

func TestAuthenticatedMutationRequiresHostOriginSessionAndCSRF(t *testing.T) {
	called := 0
	d, err := New(Config{ControlSecret: testControlSecret, Controls: ControlCallbacks{Pause: func() { called++ }}})
	if err != nil {
		t.Fatal(err)
	}
	h := d.Handler()
	cookie, csrf := loginSession(t, h, "127.0.0.1:9090", "http://127.0.0.1:9090", nil)
	if !cookie.HttpOnly || cookie.Path != "/" || cookie.SameSite != http.SameSiteStrictMode || cookie.Secure {
		t.Fatalf("local cookie flags = %+v", cookie)
	}
	tests := []struct {
		name, host, origin, csrf string
		cookie                   *http.Cookie
		want                     int
	}{
		{"valid", "127.0.0.1:9090", "http://127.0.0.1:9090", csrf, cookie, http.StatusOK},
		{"host", "attacker.example", "http://attacker.example", csrf, cookie, http.StatusBadRequest},
		{"origin", "127.0.0.1:9090", "https://attacker.example", csrf, cookie, http.StatusForbidden},
		{"missing-origin", "127.0.0.1:9090", "", csrf, cookie, http.StatusForbidden},
		{"missing-session", "127.0.0.1:9090", "http://127.0.0.1:9090", csrf, nil, http.StatusUnauthorized},
		{"wrong-csrf", "127.0.0.1:9090", "http://127.0.0.1:9090", "wrong", cookie, http.StatusForbidden},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := securityRequest(http.MethodPost, "/api/pause", tc.host, tc.origin, "")
			if tc.cookie != nil {
				r.AddCookie(tc.cookie)
			}
			if tc.csrf != "" {
				r.Header.Set("X-MARS-CSRF-Token", tc.csrf)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tc.want {
				t.Fatalf("status=%d want=%d body=%s", w.Code, tc.want, w.Body.String())
			}
		})
	}
	if called != 1 {
		t.Fatalf("callback count = %d, want 1", called)
	}
}

func TestLoginRejectsDifferentLengthSecretsWithoutDisclosure(t *testing.T) {
	d, err := New(Config{ControlSecret: testControlSecret})
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"x", strings.Repeat("x", 100)} {
		r := securityRequest(http.MethodPost, "/api/login", "127.0.0.1:9090", "http://127.0.0.1:9090", `{"secret":"`+secret+`"}`)
		w := httptest.NewRecorder()
		d.Handler().ServeHTTP(w, r)
		if w.Code != http.StatusUnauthorized || strings.Contains(w.Body.String(), secret) {
			t.Fatalf("invalid secret length=%d status=%d body=%q", len(secret), w.Code, w.Body.String())
		}
	}
}

func TestSessionRotationExpiryAndCrossSessionCSRF(t *testing.T) {
	d, err := New(Config{ControlSecret: testControlSecret, Controls: ControlCallbacks{Pause: func() {}}})
	if err != nil {
		t.Fatal(err)
	}
	g := d.handler.(*securityGateway)
	firstCookie, firstCSRF := loginSession(t, g, "127.0.0.1:9090", "http://127.0.0.1:9090", nil)
	secondCookie, secondCSRF := loginSession(t, g, "127.0.0.1:9090", "http://127.0.0.1:9090", firstCookie)
	if firstCookie.Value == secondCookie.Value || firstCSRF == secondCSRF {
		t.Fatal("login did not rotate session material")
	}
	for _, attempt := range []struct {
		cookie *http.Cookie
		csrf   string
	}{{firstCookie, firstCSRF}, {secondCookie, firstCSRF}} {
		r := securityRequest(http.MethodPost, "/api/pause", "127.0.0.1:9090", "http://127.0.0.1:9090", "")
		r.AddCookie(attempt.cookie)
		r.Header.Set("X-MARS-CSRF-Token", attempt.csrf)
		w := httptest.NewRecorder()
		g.ServeHTTP(w, r)
		if w.Code != http.StatusUnauthorized && w.Code != http.StatusForbidden {
			t.Fatalf("stale/cross-session status = %d", w.Code)
		}
	}
	g.now = func() time.Time { return time.Now().Add(sessionIdleLifetime + time.Minute) }
	r := securityRequest(http.MethodPost, "/api/pause", "127.0.0.1:9090", "http://127.0.0.1:9090", "")
	r.AddCookie(secondCookie)
	r.Header.Set("X-MARS-CSRF-Token", secondCSRF)
	w := httptest.NewRecorder()
	g.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expired session status = %d", w.Code)
	}
}

func TestStrictControlJSONAndMethods(t *testing.T) {
	called := 0
	d, err := New(Config{ControlSecret: testControlSecret, Controls: ControlCallbacks{Scan: func(_ context.Context, _ string) error { called++; return nil }}})
	if err != nil {
		t.Fatal(err)
	}
	cookie, csrf := loginSession(t, d.Handler(), "127.0.0.1:9090", "http://127.0.0.1:9090", nil)
	for _, body := range []string{`{"repo_id":"r1","unknown":true}`, `{"repo_id":"r1"}{}`, `{"repo_id":"bad value"}`} {
		r := securityRequest(http.MethodPost, "/api/scan", "127.0.0.1:9090", "http://127.0.0.1:9090", body)
		r.AddCookie(cookie)
		r.Header.Set("X-MARS-CSRF-Token", csrf)
		w := httptest.NewRecorder()
		d.Handler().ServeHTTP(w, r)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("body %q status=%d", body, w.Code)
		}
	}
	method := httptest.NewRecorder()
	d.Handler().ServeHTTP(method, securityRequest(http.MethodOptions, "/api/status", "127.0.0.1:9090", "http://attacker.example", ""))
	if method.Code != http.StatusMethodNotAllowed || method.Header().Get("Allow") != "GET, HEAD" {
		t.Fatalf("OPTIONS status=%d allow=%q", method.Code, method.Header().Get("Allow"))
	}
	if called != 0 {
		t.Fatalf("invalid requests invoked callback %d times", called)
	}
	headEvents := securityRequest(http.MethodHead, "/api/events", "127.0.0.1:9090", "", "")
	headEvents.AddCookie(cookie)
	w := httptest.NewRecorder()
	d.Handler().ServeHTTP(w, headEvents)
	if w.Code != http.StatusMethodNotAllowed || w.Header().Get("Allow") != "GET" {
		t.Fatalf("HEAD events status=%d allow=%q", w.Code, w.Header().Get("Allow"))
	}
}

func TestRemoteOriginRequiresAuthAndIgnoresForwardedAuthority(t *testing.T) {
	d, err := New(Config{ControlSecret: testControlSecret, TrustedOrigin: "https://proxy.example"})
	if err != nil {
		t.Fatal(err)
	}
	unauth := securityRequest(http.MethodGet, "/pipeline", "proxy.example", "", "")
	unauth.Header.Set("Forwarded", "host=127.0.0.1:9090;proto=http")
	unauth.Header.Set("X-Forwarded-Host", "127.0.0.1:9090")
	w := httptest.NewRecorder()
	d.Handler().ServeHTTP(w, unauth)
	if w.Code != http.StatusFound || w.Header().Get("Location") != "/login" {
		t.Fatalf("remote anonymous page status = %d location=%q", w.Code, w.Header().Get("Location"))
	}
	for _, path := range []string{"/login", "/static/login.js", "/static/style.css"} {
		loginShell := httptest.NewRecorder()
		d.Handler().ServeHTTP(loginShell, securityRequest(http.MethodGet, path, "proxy.example", "", ""))
		if loginShell.Code != http.StatusOK {
			t.Fatalf("remote login shell %s status=%d", path, loginShell.Code)
		}
	}
	locked := httptest.NewRecorder()
	d.Handler().ServeHTTP(locked, securityRequest(http.MethodGet, "/api/repos", "proxy.example", "", ""))
	if locked.Code != http.StatusUnauthorized {
		t.Fatalf("remote anonymous API status = %d", locked.Code)
	}
	cookie, _ := loginSession(t, d.Handler(), "proxy.example", "https://proxy.example", nil)
	if !cookie.Secure {
		t.Fatal("proxy session cookie is not Secure")
	}
	read := securityRequest(http.MethodGet, "/pipeline", "proxy.example", "", "")
	read.AddCookie(cookie)
	w = httptest.NewRecorder()
	d.Handler().ServeHTTP(w, read)
	if w.Code != http.StatusOK {
		t.Fatalf("remote authenticated read status = %d", w.Code)
	}
}

func TestSecurityHeadersAndVendoredAssets(t *testing.T) {
	d := newTestDashboard(t)
	for _, path := range []string{"/", "/login", "/pipeline", "/not-found", "/static/app.js"} {
		w := httptest.NewRecorder()
		d.Handler().ServeHTTP(w, securityRequest(http.MethodGet, path, "127.0.0.1:9090", "", ""))
		for _, header := range []string{"Content-Security-Policy", "X-Content-Type-Options", "Referrer-Policy", "X-Frame-Options", "Cross-Origin-Resource-Policy", "Permissions-Policy", "Cache-Control"} {
			if w.Header().Get(header) == "" {
				t.Errorf("%s missing %s", path, header)
			}
		}
		if csp := w.Header().Get("Content-Security-Policy"); strings.Contains(csp, "unsafe-inline") || strings.Contains(csp, "unsafe-eval") {
			t.Fatalf("unsafe CSP: %s", csp)
		}
	}
	configured, err := New(Config{ControlSecret: testControlSecret})
	if err != nil {
		t.Fatal(err)
	}
	configured.HandleFunc("/api/telemetry", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, http.StatusOK, map[string]bool{"ok": true}) })
	cookie, _ := loginSession(t, configured.Handler(), "127.0.0.1:9090", "http://127.0.0.1:9090", nil)
	for _, path := range []string{"/api/telemetry", "/api/events"} {
		ctx, cancel := context.WithCancel(context.Background())
		if path == "/api/events" {
			cancel()
		} else {
			defer cancel()
		}
		r := securityRequest(http.MethodGet, path, "127.0.0.1:9090", "", "").WithContext(ctx)
		r.AddCookie(cookie)
		w := httptest.NewRecorder()
		configured.Handler().ServeHTTP(w, r)
		assertSecurityHeaders(t, path, w.Header())
	}
	login := httptest.NewRecorder()
	configured.Handler().ServeHTTP(login, securityRequest(http.MethodPost, "/api/login", "127.0.0.1:9090", "http://127.0.0.1:9090", `{"secret":"wrong"}`))
	assertSecurityHeaders(t, "login error", login.Header())
	assets := map[string]string{
		"static/vendor/htmx-2.0.4.min.js":  "e209dda5c8235479f3166defc7750e1dbcd5a5c1808b7792fc2e6733768fb447",
		"static/vendor/chart-4.4.7.umd.js": "2812cb8825fdc57469eb2f7bb055e9429244e599920511ee477e828499b632cb",
	}
	for path, want := range assets {
		body, err := content.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(body)
		if got := hex.EncodeToString(sum[:]); got != want {
			t.Errorf("%s hash=%s want=%s", path, got, want)
		}
	}
	htmxLicense, err := content.ReadFile("static/vendor/htmx-LICENSE.txt")
	if err != nil || !strings.HasPrefix(string(htmxLicense), "Zero-Clause BSD") {
		t.Fatalf("htmx license metadata is not Zero-Clause BSD: err=%v", err)
	}
	assetMetadata, err := content.ReadFile("static/vendor/ASSETS.md")
	if err != nil || !strings.Contains(string(assetMetadata), "Zero-Clause BSD") || !strings.Contains(string(assetMetadata), "Chart.js 4.4.7") || !strings.Contains(string(assetMetadata), "MIT") {
		t.Fatalf("vendor asset metadata incomplete: err=%v", err)
	}
	for _, path := range []string{"static/app.js", "static/login.js", "templates/login.html", "templates/pipeline.html", "templates/orchestration.html", "templates/roles.html", "templates/throughput.html", "templates/debug.html", "templates/evolution.html"} {
		body, err := content.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		text := string(body)
		for _, forbidden := range []string{"innerHTML", "outerHTML", "insertAdjacentHTML", "document.write", "onclick=", "onchange=", "https://unpkg.com", "https://cdn.jsdelivr.net"} {
			if strings.Contains(text, forbidden) {
				t.Errorf("%s contains forbidden browser sink/source %q", path, forbidden)
			}
		}
		if strings.HasPrefix(path, "templates/") && path != "templates/login.html" {
			for _, required := range []string{`"allowEval":false`, `"allowScriptTags":false`, `"includeIndicatorStyles":false`, `"defaultSwapStyle":"textContent"`} {
				if !strings.Contains(text, required) {
					t.Errorf("%s missing fail-closed htmx setting %s", path, required)
				}
			}
		}
	}
}

func assertSecurityHeaders(t *testing.T, name string, header http.Header) {
	t.Helper()
	for _, key := range []string{"Content-Security-Policy", "X-Content-Type-Options", "Referrer-Policy", "X-Frame-Options", "Cross-Origin-Resource-Policy", "Permissions-Policy", "Cache-Control"} {
		if header.Get(key) == "" {
			t.Errorf("%s missing %s", name, key)
		}
	}
	if header.Get("Cache-Control") != "no-store" {
		t.Errorf("%s Cache-Control=%q, want no-store", name, header.Get("Cache-Control"))
	}
}

func TestStatusProjectionAndRedaction(t *testing.T) {
	d, err := New(Config{ControlSecret: testControlSecret, Controls: ControlCallbacks{Status: func() interface{} {
		return map[string]interface{}{"healthy": true, "repos": []map[string]string{{"id": "r1", "path": "/Users/alice/private/repo", "flow_profile": "default"}}}
	}}})
	if err != nil {
		t.Fatal(err)
	}
	minimal := httptest.NewRecorder()
	d.Handler().ServeHTTP(minimal, securityRequest(http.MethodGet, "/api/status", "127.0.0.1:9090", "", ""))
	if strings.Contains(minimal.Body.String(), "repo") || strings.Contains(minimal.Body.String(), "/Users/") {
		t.Fatalf("minimal status disclosed repo data: %s", minimal.Body.String())
	}
	cookie, _ := loginSession(t, d.Handler(), "127.0.0.1:9090", "http://127.0.0.1:9090", nil)
	authenticated := httptest.NewRecorder()
	r := securityRequest(http.MethodGet, "/api/status", "127.0.0.1:9090", "", "")
	r.AddCookie(cookie)
	d.Handler().ServeHTTP(authenticated, r)
	if strings.Contains(authenticated.Body.String(), "/Users/") || !strings.Contains(authenticated.Body.String(), `"path":"repo"`) {
		t.Fatalf("authenticated status path projection = %s", authenticated.Body.String())
	}
	for _, credential := range []string{
		"Authorization: Bearer TOKEN_VALUE",
		"Authorization: Basic dXNlcjpwYXNz",
		`Authorization: Digest username="Mufasa", realm="test", nonce="secret"`,
		`{"authorization":"Basic dXNlcjpwYXNz","other":"must also be conservatively removed"}`,
	} {
		redacted := RedactBrowserText(credential, 256)
		if strings.Contains(redacted, "TOKEN_VALUE") || strings.Contains(redacted, "dXNlcjpwYXNz") || strings.Contains(redacted, "Mufasa") || strings.Contains(redacted, "nonce") || !strings.Contains(redacted, "[redacted]") {
			t.Fatalf("authorization redaction = %q", redacted)
		}
	}
}

func TestSessionInvalidationClosesSSEOnLogoutRestartAndExpiry(t *testing.T) {
	for _, mode := range []string{"logout", "restart", "expiry"} {
		t.Run(mode, func(t *testing.T) {
			started := make(chan struct{})
			ended := make(chan struct{})
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/events" {
					close(started)
					<-r.Context().Done()
					close(ended)
					return
				}
				writeJSON(w, http.StatusOK, controlResponse{OK: true})
			})
			g, err := newSecurityGateway(next, testControlSecret, "")
			if err != nil {
				t.Fatal(err)
			}
			cookie, csrf := loginSession(t, g, "127.0.0.1:9090", "http://127.0.0.1:9090", nil)
			if mode == "expiry" {
				g.mu.Lock()
				g.sessions[cookie.Value].createdAt = time.Now().Add(-sessionMaxLifetime + 25*time.Millisecond)
				g.mu.Unlock()
			}
			eventReq := securityRequest(http.MethodGet, "/api/events", "127.0.0.1:9090", "", "")
			eventReq.AddCookie(cookie)
			go g.ServeHTTP(httptest.NewRecorder(), eventReq)
			select {
			case <-started:
			case <-time.After(time.Second):
				t.Fatal("SSE did not start")
			}
			if mode != "expiry" {
				path := "/api/logout"
				if mode == "restart" {
					path = "/api/restart"
				}
				r := securityRequest(http.MethodPost, path, "127.0.0.1:9090", "http://127.0.0.1:9090", "")
				r.AddCookie(cookie)
				r.Header.Set("X-MARS-CSRF-Token", csrf)
				w := httptest.NewRecorder()
				g.ServeHTTP(w, r)
				if w.Code != http.StatusOK {
					t.Fatalf("%s status=%d", mode, w.Code)
				}
			}
			select {
			case <-ended:
			case <-time.After(time.Second):
				t.Fatalf("SSE did not close after %s", mode)
			}
			check := securityRequest(http.MethodGet, "/api/repos", "127.0.0.1:9090", "", "")
			check.AddCookie(cookie)
			w := httptest.NewRecorder()
			g.ServeHTTP(w, check)
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("session survived %s: %d", mode, w.Code)
			}
		})
	}
}

func TestLoginSessionBootstrapAllowsMutationWithoutSecondSecret(t *testing.T) {
	called := 0
	d, err := New(Config{ControlSecret: testControlSecret, Controls: ControlCallbacks{Pause: func() { called++ }}})
	if err != nil {
		t.Fatal(err)
	}
	cookie, _ := loginSession(t, d.Handler(), "127.0.0.1:9090", "http://127.0.0.1:9090", nil)
	bootstrap := securityRequest(http.MethodGet, "/api/session", "127.0.0.1:9090", "", "")
	bootstrap.AddCookie(cookie)
	w := httptest.NewRecorder()
	d.Handler().ServeHTTP(w, bootstrap)
	if w.Code != http.StatusOK {
		t.Fatalf("session bootstrap status=%d body=%s", w.Code, w.Body.String())
	}
	var payload struct {
		CSRF string `json:"csrf_token"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil || payload.CSRF == "" {
		t.Fatalf("session bootstrap payload=%q err=%v", w.Body.String(), err)
	}
	mutation := securityRequest(http.MethodPost, "/api/pause", "127.0.0.1:9090", "http://127.0.0.1:9090", "")
	mutation.AddCookie(cookie)
	mutation.Header.Set("X-MARS-CSRF-Token", payload.CSRF)
	w = httptest.NewRecorder()
	d.Handler().ServeHTTP(w, mutation)
	if w.Code != http.StatusOK || called != 1 {
		t.Fatalf("post-bootstrap mutation status=%d called=%d", w.Code, called)
	}
	for _, path := range []string{"static/app.js", "static/login.js"} {
		body, err := content.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(body), "sessionStorage") || strings.Contains(string(body), "localStorage") {
			t.Fatalf("%s persists CSRF in browser storage", path)
		}
	}
}

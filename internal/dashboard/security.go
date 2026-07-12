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
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	controlSecretMinBytes = 32
	loginBodyLimit        = 4 << 10
	controlBodyLimit      = 64 << 10
	maxSessions           = 128
	sessionIdleLifetime   = 30 * time.Minute
	sessionMaxLifetime    = 8 * time.Hour
	maxSSEClients         = 64
	maxSSEPerSession      = 4
)

const sessionCookieName = "mars_dashboard_session"

var safeControlValue = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/@+-]{0,255}$`)

type proxyOrigin struct {
	origin    string
	authority string
}

func parseProxyOrigin(raw string) (*proxyOrigin, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("dashboard trusted origin must be one exact HTTPS origin such as https://mars.example.test; remove wildcards, credentials, paths, queries, and fragments")
	}
	if strings.ContainsAny(u.Host, "*\\") {
		return nil, fmt.Errorf("dashboard trusted origin must not contain wildcards")
	}
	if _, err := net.LookupPort("tcp", u.Port()); u.Port() != "" && err != nil {
		return nil, fmt.Errorf("dashboard trusted origin has an invalid port")
	}
	origin := "https://" + strings.ToLower(u.Host)
	return &proxyOrigin{origin: origin, authority: strings.ToLower(u.Host)}, nil
}

type session struct {
	id        string
	csrf      string
	createdAt time.Time
	lastSeen  time.Time
}

type rateWindow struct {
	started time.Time
	count   int
}

type securityContextKey struct{}

type securityGateway struct {
	next          http.Handler
	controlSecret []byte
	proxy         *proxyOrigin
	now           func() time.Time

	mu          sync.Mutex
	sessions    map[string]*session
	rates       map[string]rateWindow
	sseByClient map[string]int
	sseTotal    int
	streams     map[string]map[uint64]context.CancelFunc
	streamSeq   uint64
}

func newSecurityGateway(next http.Handler, controlSecret, trustedOrigin string) (*securityGateway, error) {
	secret := []byte(controlSecret)
	if len(secret) > 0 && len(secret) < controlSecretMinBytes {
		return nil, fmt.Errorf("dashboard control secret is configured but shorter than 32 bytes — set MARS_DASHBOARD_CONTROL_SECRET to a new random value of at least 32 bytes")
	}
	proxy, err := parseProxyOrigin(trustedOrigin)
	if err != nil {
		return nil, err
	}
	if proxy != nil && len(secret) == 0 {
		return nil, fmt.Errorf("dashboard trusted origin requires MARS_DASHBOARD_CONTROL_SECRET with at least 32 bytes; remote observation cannot be anonymous")
	}
	return &securityGateway{
		next:          next,
		controlSecret: append([]byte(nil), secret...),
		proxy:         proxy,
		now:           time.Now,
		sessions:      make(map[string]*session),
		rates:         make(map[string]rateWindow),
		sseByClient:   make(map[string]int),
		streams:       make(map[string]map[uint64]context.CancelFunc),
	}, nil
}

func (g *securityGateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	setSecurityHeaders(w.Header())
	if !g.allowedHost(r.Host) {
		writeGatewayError(w, http.StatusBadRequest, "dashboard request rejected: use the exact loopback dashboard address or the configured HTTPS proxy origin")
		return
	}
	remote := g.isProxyRequest(r.Host)
	activeSession := g.sessionForRequest(r)
	authenticated := activeSession != nil
	if remote && !authenticated && !isLoginShellPath(r.URL.Path) && r.URL.Path != "/api/login" {
		if r.Method == http.MethodGet && isDashboardPagePath(r.URL.Path) {
			http.Redirect(w, r, "/login", http.StatusFound)
		} else {
			writeGatewayError(w, http.StatusUnauthorized, "dashboard authentication required")
		}
		return
	}
	if !authenticated && isPrivilegedReadPath(r.URL.Path) {
		if len(g.controlSecret) == 0 {
			writeGatewayError(w, http.StatusServiceUnavailable, "dashboard data is locked; set MARS_DASHBOARD_CONTROL_SECRET to a random value of at least 32 bytes and restart MARS")
		} else {
			writeGatewayError(w, http.StatusUnauthorized, "dashboard authentication required")
		}
		return
	}
	if r.URL.Path == "/api/login" {
		g.handleLogin(w, r)
		return
	}
	if r.URL.Path == "/api/logout" {
		g.handleLogout(w, r)
		return
	}
	if r.URL.Path == "/api/session" {
		if r.Method != http.MethodGet {
			methodNotAllowed(w, "GET")
			return
		}
		if activeSession == nil {
			writeGatewayError(w, http.StatusUnauthorized, "dashboard authentication required")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"csrf_token": activeSession.csrf})
		return
	}
	if isMutationPath(r.URL.Path) {
		g.authorizeMutation(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		methodNotAllowed(w, "GET, HEAD")
		return
	}
	if r.URL.Path == "/api/events" {
		if r.Method != http.MethodGet {
			methodNotAllowed(w, "GET")
			return
		}
		key := "anonymous:" + clientAddress(r)
		if activeSession != nil {
			key = activeSession.id
		}
		if !g.allowRate("sse:"+key, 30, time.Minute) || !g.acquireSSE(key) {
			w.Header().Set("Retry-After", "10")
			writeGatewayError(w, http.StatusTooManyRequests, "dashboard event-stream connection limit reached; retry later")
			return
		}
		defer g.releaseSSE(key)
		ctx, cancel := context.WithCancel(r.Context())
		cleanup := g.registerStream(key, cancel)
		defer cleanup()
		if activeSession != nil {
			expires := activeSession.lastSeen.Add(sessionIdleLifetime)
			if absolute := activeSession.createdAt.Add(sessionMaxLifetime); absolute.Before(expires) {
				expires = absolute
			}
			timer := time.AfterFunc(max(time.Until(expires), time.Millisecond), cancel)
			defer timer.Stop()
		}
		r = r.WithContext(ctx)
	}
	r = r.WithContext(context.WithValue(r.Context(), securityContextKey{}, authenticated))
	g.next.ServeHTTP(w, r)
}

func setSecurityHeaders(h http.Header) {
	h.Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; font-src 'self'; object-src 'none'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Referrer-Policy", "no-referrer")
	h.Set("X-Frame-Options", "DENY")
	h.Set("Cross-Origin-Resource-Policy", "same-origin")
	h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()")
	h.Set("Cache-Control", "no-store")
}

func (g *securityGateway) allowedHost(hostport string) bool {
	if g.isProxyRequest(hostport) {
		return true
	}
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		host = hostport
	}
	host = strings.Trim(strings.TrimSpace(host), "[]")
	return strings.EqualFold(host, "localhost") || net.ParseIP(host) != nil && net.ParseIP(host).IsLoopback()
}

func (g *securityGateway) isProxyRequest(host string) bool {
	return g.proxy != nil && subtle.ConstantTimeCompare([]byte(strings.ToLower(strings.TrimSpace(host))), []byte(g.proxy.authority)) == 1
}

func (g *securityGateway) requestOrigin(r *http.Request) string {
	if g.isProxyRequest(r.Host) {
		return g.proxy.origin
	}
	return "http://" + strings.ToLower(r.Host)
}

func (g *securityGateway) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, "POST")
		return
	}
	if len(g.controlSecret) == 0 {
		writeGatewayError(w, http.StatusServiceUnavailable, "dashboard controls are disabled; set MARS_DASHBOARD_CONTROL_SECRET to a random value of at least 32 bytes and restart MARS")
		return
	}
	if !sameOrigin(r, g.requestOrigin(r)) {
		writeGatewayError(w, http.StatusForbidden, "dashboard login requires the exact same Origin as the request Host")
		return
	}
	if !g.allowRate("login:"+clientAddress(r), 10, time.Minute) {
		w.Header().Set("Retry-After", "60")
		writeGatewayError(w, http.StatusTooManyRequests, "too many dashboard login attempts; retry later")
		return
	}
	if !isJSON(r.Header.Get("Content-Type")) {
		writeGatewayError(w, http.StatusUnsupportedMediaType, "dashboard login requires Content-Type: application/json")
		return
	}
	var body struct {
		Secret string `json:"secret"`
	}
	if err := decodeStrictJSON(w, r, loginBodyLimit, &body); err != nil {
		writeGatewayError(w, http.StatusBadRequest, "invalid dashboard login request")
		return
	}
	wantDigest := sha256.Sum256(g.controlSecret)
	gotDigest := sha256.Sum256([]byte(body.Secret))
	if subtle.ConstantTimeCompare(gotDigest[:], wantDigest[:]) != 1 {
		writeGatewayError(w, http.StatusUnauthorized, "dashboard authentication failed")
		return
	}
	if existing, err := r.Cookie(sessionCookieName); err == nil {
		g.mu.Lock()
		delete(g.sessions, existing.Value)
		g.mu.Unlock()
		g.closeStreams(existing.Value)
	}
	s, err := g.createSession()
	if err != nil {
		writeGatewayError(w, http.StatusServiceUnavailable, "dashboard session unavailable; retry login")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    s.id,
		Path:     "/",
		HttpOnly: true,
		Secure:   g.isProxyRequest(r.Host),
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(sessionMaxLifetime.Seconds()),
	})
	writeJSON(w, http.StatusOK, map[string]string{"csrf_token": s.csrf})
}

func (g *securityGateway) handleLogout(w http.ResponseWriter, r *http.Request) {
	if !g.authorizeBrowserMutation(w, r, true) {
		return
	}
	if c, err := r.Cookie(sessionCookieName); err == nil {
		g.mu.Lock()
		delete(g.sessions, c.Value)
		g.mu.Unlock()
		g.closeStreams(c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Path: "/", HttpOnly: true, Secure: g.isProxyRequest(r.Host), SameSite: http.SameSiteStrictMode, MaxAge: -1})
	writeJSON(w, http.StatusOK, controlResponse{OK: true})
}

func (g *securityGateway) authorizeMutation(w http.ResponseWriter, r *http.Request) {
	bodyless := r.URL.Path != "/api/scan" && r.URL.Path != "/api/run-role"
	if !g.authorizeBrowserMutation(w, r, bodyless) {
		return
	}
	g.next.ServeHTTP(w, r)
	if r.URL.Path == "/api/restart" {
		g.invalidateAllSessions()
	}
}

func (g *securityGateway) authorizeBrowserMutation(w http.ResponseWriter, r *http.Request, bodyless bool) bool {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, "POST")
		return false
	}
	if len(g.controlSecret) == 0 {
		writeGatewayError(w, http.StatusServiceUnavailable, "dashboard controls are disabled; set MARS_DASHBOARD_CONTROL_SECRET to a random value of at least 32 bytes and restart MARS")
		return false
	}
	s := g.sessionForRequest(r)
	if s == nil {
		writeGatewayError(w, http.StatusUnauthorized, "dashboard authentication required")
		return false
	}
	if !sameOrigin(r, g.requestOrigin(r)) {
		writeGatewayError(w, http.StatusForbidden, "dashboard mutation requires the exact same Origin as the request Host")
		return false
	}
	if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-MARS-CSRF-Token")), []byte(s.csrf)) != 1 {
		writeGatewayError(w, http.StatusForbidden, "dashboard mutation requires the current session CSRF token")
		return false
	}
	if !g.allowRate("mutation:"+s.id, 120, time.Minute) {
		w.Header().Set("Retry-After", "30")
		writeGatewayError(w, http.StatusTooManyRequests, "dashboard mutation rate limit reached; retry later")
		return false
	}
	if bodyless && r.ContentLength > 0 {
		writeGatewayError(w, http.StatusBadRequest, "dashboard mutation does not accept a request body")
		return false
	}
	if !bodyless && !isJSON(r.Header.Get("Content-Type")) {
		writeGatewayError(w, http.StatusUnsupportedMediaType, "dashboard mutation requires Content-Type: application/json")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, controlBodyLimit)
	return true
}

func (g *securityGateway) createSession() (*session, error) {
	id, err := randomToken(32)
	if err != nil {
		return nil, err
	}
	csrf, err := randomToken(32)
	if err != nil {
		return nil, err
	}
	now := g.now()
	s := &session{id: id, csrf: csrf, createdAt: now, lastSeen: now}
	g.mu.Lock()
	g.pruneSessionsLocked(now)
	evicted := ""
	if len(g.sessions) >= maxSessions {
		var oldest string
		for id, candidate := range g.sessions {
			if oldest == "" || candidate.lastSeen.Before(g.sessions[oldest].lastSeen) {
				oldest = id
			}
		}
		delete(g.sessions, oldest)
		evicted = oldest
	}
	g.sessions[id] = s
	g.mu.Unlock()
	if evicted != "" {
		g.closeStreams(evicted)
	}
	return s, nil
}

func (g *securityGateway) sessionForRequest(r *http.Request) *session {
	c, err := r.Cookie(sessionCookieName)
	if err != nil || c.Value == "" {
		return nil
	}
	now := g.now()
	g.mu.Lock()
	defer g.mu.Unlock()
	g.pruneSessionsLocked(now)
	s := g.sessions[c.Value]
	if s == nil {
		return nil
	}
	s.lastSeen = now
	copy := *s
	return &copy
}

func (g *securityGateway) pruneSessionsLocked(now time.Time) {
	for id, s := range g.sessions {
		if now.Sub(s.lastSeen) > sessionIdleLifetime || now.Sub(s.createdAt) > sessionMaxLifetime {
			delete(g.sessions, id)
		}
	}
}

func (g *securityGateway) allowRate(key string, limit int, window time.Duration) bool {
	now := g.now()
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.rates) >= 1024 {
		for candidate, value := range g.rates {
			if now.Sub(value.started) >= time.Hour {
				delete(g.rates, candidate)
			}
		}
		if _, exists := g.rates[key]; !exists && len(g.rates) >= 1024 {
			return false
		}
	}
	r := g.rates[key]
	if r.started.IsZero() || now.Sub(r.started) >= window {
		r = rateWindow{started: now}
	}
	if r.count >= limit {
		return false
	}
	r.count++
	g.rates[key] = r
	return true
}

func clientAddress(r *http.Request) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}

func (g *securityGateway) acquireSSE(key string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.sseTotal >= maxSSEClients || g.sseByClient[key] >= maxSSEPerSession {
		return false
	}
	g.sseTotal++
	g.sseByClient[key]++
	return true
}

func (g *securityGateway) registerStream(key string, cancel context.CancelFunc) func() {
	g.mu.Lock()
	g.streamSeq++
	id := g.streamSeq
	if g.streams[key] == nil {
		g.streams[key] = make(map[uint64]context.CancelFunc)
	}
	g.streams[key][id] = cancel
	g.mu.Unlock()
	return func() {
		cancel()
		g.mu.Lock()
		if group := g.streams[key]; group != nil {
			delete(group, id)
			if len(group) == 0 {
				delete(g.streams, key)
			}
		}
		g.mu.Unlock()
	}
}

func (g *securityGateway) closeStreams(key string) {
	g.mu.Lock()
	group := g.streams[key]
	delete(g.streams, key)
	cancels := make([]context.CancelFunc, 0, len(group))
	for _, cancel := range group {
		cancels = append(cancels, cancel)
	}
	g.mu.Unlock()
	for _, cancel := range cancels {
		cancel()
	}
}

func (g *securityGateway) invalidateAllSessions() {
	g.mu.Lock()
	g.sessions = make(map[string]*session)
	groups := g.streams
	g.streams = make(map[string]map[uint64]context.CancelFunc)
	g.mu.Unlock()
	for _, group := range groups {
		for _, cancel := range group {
			cancel()
		}
	}
}

func (g *securityGateway) releaseSSE(key string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.sseTotal > 0 {
		g.sseTotal--
	}
	if g.sseByClient[key] <= 1 {
		delete(g.sseByClient, key)
	} else {
		g.sseByClient[key]--
	}
}

func sameOrigin(r *http.Request, want string) bool {
	got := strings.TrimSpace(r.Header.Get("Origin"))
	return got != "" && subtle.ConstantTimeCompare([]byte(strings.ToLower(got)), []byte(strings.ToLower(want))) == 1
}

func isMutationPath(path string) bool {
	switch path {
	case "/api/emergency-stop", "/api/pause", "/api/resume", "/api/stop", "/api/restart", "/api/scan", "/api/run-role":
		return true
	default:
		return false
	}
}

func isLoginShellPath(path string) bool {
	return path == "/login" || path == "/static/login.js" || path == "/static/style.css"
}

func isPrivilegedReadPath(path string) bool {
	switch path {
	case "/api/session", "/api/events", "/api/repos", "/api/repo-roles", "/api/telemetry", "/api/evolution", "/api/roles", "/api/quality-score", "/api/throughput", "/api/orchestration", "/api/orchestration/decisions":
		return true
	default:
		return false
	}
}

func requestAuthenticated(r *http.Request) bool {
	value, _ := r.Context().Value(securityContextKey{}).(bool)
	return value
}

func isDashboardPagePath(path string) bool {
	switch path {
	case "/", "/pipeline", "/orchestration", "/roles", "/throughput", "/debug", "/evolution":
		return true
	default:
		return false
	}
}

func isJSON(contentType string) bool {
	contentType = strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	return contentType == "application/json"
}

func decodeStrictJSON(w http.ResponseWriter, r *http.Request, limit int64, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		return fmt.Errorf("request body must contain exactly one JSON value")
	}
	return nil
}

func randomToken(bytes int) (string, error) {
	b := make([]byte, bytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func methodNotAllowed(w http.ResponseWriter, allow string) {
	w.Header().Set("Allow", allow)
	writeGatewayError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func writeGatewayError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, controlResponse{Error: message})
}

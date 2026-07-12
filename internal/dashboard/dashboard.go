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
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/greaveselliott/mars/internal/network"
)

//go:embed templates/*.html static/*
var content embed.FS

// ChainNode represents a single role in the pipeline visualization.
type ChainNode struct {
	Name   string
	Domain string
	Mode   string
	Active bool
}

// PipelineView describes how the dashboard should render role routing.
type PipelineView struct {
	Mode        string
	Description string
	Nodes       []ChainNode
}

// ControlCallbacks groups the server methods exposed through the
// dashboard's control API. Each field is optional — nil disables the
// corresponding endpoint.
type ControlCallbacks struct {
	Pause     func()
	Resume    func()
	Restart   func(ctx context.Context) error
	Stop      func(ctx context.Context) error
	Scan      func(ctx context.Context, repoID string) error
	RunRole   func(ctx context.Context, repoID, role string) error
	Status    func() interface{}
	ListRepos func() []RepoInfoDTO
	ListRoles func(repoID string) []string
	IsPaused  func() bool
}

// RepoInfoDTO is a lightweight repo descriptor for API responses.
type RepoInfoDTO struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

// Config controls the dashboard.
type Config struct {
	Addr             string // default "127.0.0.1:9090"
	ControlSecret    string // environment-only browser control credential
	TrustedOrigin    string // exact HTTPS reverse-proxy origin; listener remains loopback-only
	EmergencyStop    func() []error
	ChainProvider    func() []ChainNode // returns the live pipeline chain from manifest
	PipelineProvider func() PipelineView
	Controls         ControlCallbacks
}

const (
	eventBufferSize = 100
	maxEventBytes   = 4096
)

// Dashboard serves the embedded web UI.
type Dashboard struct {
	cfg        Config
	mux        *http.ServeMux
	handler    http.Handler
	tmpl       *template.Template
	sseClients map[chan string]struct{}
	eventBuf   []string
	mu         sync.RWMutex
}

// New creates a Dashboard, parses embedded templates, and wires routes.
func New(cfg Config) (*Dashboard, error) {
	if cfg.Addr == "" {
		cfg.Addr = "127.0.0.1:9090"
	}
	if err := network.ValidateLoopbackAddress("dashboard", cfg.Addr); err != nil {
		return nil, fmt.Errorf("dashboard: %w", err)
	}

	tmpl, err := template.ParseFS(content, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("dashboard: parse templates: %w", err)
	}

	d := &Dashboard{
		cfg:        cfg,
		mux:        http.NewServeMux(),
		tmpl:       tmpl,
		sseClients: make(map[chan string]struct{}),
	}

	d.routes()
	gateway, err := newSecurityGateway(d.mux, cfg.ControlSecret, cfg.TrustedOrigin)
	if err != nil {
		return nil, fmt.Errorf("dashboard: %w", err)
	}
	d.handler = gateway
	return d, nil
}

// Handler returns the HTTP handler for the dashboard.
func (d *Dashboard) Handler() http.Handler {
	return d.handler
}

// InvalidateSessions revokes every dashboard browser session and terminates
// active event streams. It is safe to call repeatedly during warm restart.
func (d *Dashboard) InvalidateSessions() {
	if gateway, ok := d.handler.(*securityGateway); ok {
		gateway.invalidateAllSessions()
	}
}

// HandleFunc registers an additional route on the dashboard mux.
func (d *Dashboard) HandleFunc(pattern string, handler http.HandlerFunc) {
	d.mux.HandleFunc(pattern, handler)
}

func (d *Dashboard) routes() {
	d.mux.HandleFunc("/", d.handleIndex)
	d.mux.HandleFunc("/login", d.handleLoginPage)
	d.mux.HandleFunc("/orchestration", d.handlePage("orchestration", "Orchestration"))
	d.mux.HandleFunc("/pipeline", d.handlePage("pipeline", "Pipeline Flow"))
	d.mux.HandleFunc("/roles", d.handlePage("roles", "Role Health"))
	d.mux.HandleFunc("/throughput", d.handlePage("throughput", "Throughput"))
	d.mux.HandleFunc("/debug", d.handlePage("debug", "Debug"))
	d.mux.HandleFunc("/evolution", d.handlePage("evolution", "Evolution History"))
	d.mux.HandleFunc("/api/events", d.handleSSE)
	d.mux.HandleFunc("/api/emergency-stop", d.handleEmergencyStop)
	d.mux.HandleFunc("/api/pause", d.handlePause)
	d.mux.HandleFunc("/api/resume", d.handleResume)
	d.mux.HandleFunc("/api/stop", d.handleStop)
	d.mux.HandleFunc("/api/restart", d.handleRestart)
	d.mux.HandleFunc("/api/scan", d.handleScan)
	d.mux.HandleFunc("/api/run-role", d.handleRunRole)
	d.mux.HandleFunc("/api/status", d.handleStatus)
	d.mux.HandleFunc("/api/repos", d.handleListRepos)
	d.mux.HandleFunc("/api/repo-roles", d.handleListRolesForRepo)

	staticFS, err := fs.Sub(content, "static")
	if err != nil {
		slog.Error("dashboard: failed to create static sub-FS", "error", err)
		return
	}
	d.mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
}

func (d *Dashboard) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := d.tmpl.ExecuteTemplate(w, "login.html", nil); err != nil {
		slog.Error("dashboard: render login template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (d *Dashboard) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/pipeline", http.StatusFound)
}

type pageData struct {
	Title      string
	ActiveNav  string
	Now        string
	ChainNodes []ChainNode
	Pipeline   PipelineView
}

func (d *Dashboard) handlePage(name, title string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := pageData{
			Title:     title,
			ActiveNav: name,
			Now:       time.Now().Format(time.RFC3339),
		}

		if name == "pipeline" && requestAuthenticated(r) {
			if d.cfg.PipelineProvider != nil {
				data.Pipeline = sanitizePipelineView(d.cfg.PipelineProvider())
				data.ChainNodes = data.Pipeline.Nodes
			} else if d.cfg.ChainProvider != nil {
				data.ChainNodes = sanitizeChainNodes(d.cfg.ChainProvider())
				data.Pipeline = PipelineView{Mode: "legacy", Nodes: data.ChainNodes}
			}
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := d.tmpl.ExecuteTemplate(w, name+".html", data); err != nil {
			slog.Error("dashboard: render template", "template", name, "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}
}

func sanitizePipelineView(view PipelineView) PipelineView {
	view.Mode = RedactBrowserText(view.Mode, 32)
	view.Description = RedactBrowserText(view.Description, 512)
	view.Nodes = sanitizeChainNodes(view.Nodes)
	return view
}

func sanitizeChainNodes(nodes []ChainNode) []ChainNode {
	if len(nodes) > 128 {
		nodes = nodes[:128]
	}
	result := make([]ChainNode, len(nodes))
	for i, item := range nodes {
		result[i] = ChainNode{
			Name: RedactBrowserText(item.Name, 128), Domain: RedactBrowserText(item.Domain, 128),
			Mode: RedactBrowserText(item.Mode, 128), Active: item.Active,
		}
	}
	return result
}

// handleSSE streams server-sent events to the client.
func (d *Dashboard) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ch := make(chan string, 64)

	d.mu.Lock()
	d.sseClients[ch] = struct{}{}
	replay := make([]string, len(d.eventBuf))
	copy(replay, d.eventBuf)
	d.mu.Unlock()

	defer func() {
		d.mu.Lock()
		delete(d.sseClients, ch)
		d.mu.Unlock()
	}()

	// Send initial keepalive so the client knows the connection is alive.
	fmt.Fprintf(w, ": keepalive\n\n")
	flusher.Flush()

	// Replay recent events so late-joining clients see current state.
	for _, msg := range replay {
		fmt.Fprintf(w, "data: %s\n\n", msg)
	}
	if len(replay) > 0 {
		flusher.Flush()
	}

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		}
	}
}

// BroadcastEvent sends an event to all connected SSE clients.
// eventType is used as a JSON field, data is the payload.
func (d *Dashboard) BroadcastEvent(eventType, data string) {
	eventType = sanitizeEventType(eventType)
	data = RedactBrowserText(data, maxEventBytes)
	payload, err := json.Marshal(map[string]string{
		"type": eventType,
		"data": data,
	})
	if err != nil {
		slog.Error("dashboard: marshal SSE event", "error", err)
		return
	}

	msg := string(payload)

	d.mu.Lock()
	d.eventBuf = append(d.eventBuf, msg)
	if len(d.eventBuf) > eventBufferSize {
		d.eventBuf = d.eventBuf[len(d.eventBuf)-eventBufferSize:]
	}
	clients := make([]chan string, 0, len(d.sseClients))
	for ch := range d.sseClients {
		clients = append(clients, ch)
	}
	d.mu.Unlock()

	for _, ch := range clients {
		select {
		case ch <- msg:
		default:
			slog.Warn("dashboard: SSE client buffer full, dropping event")
		}
	}
}

// Authorization credentials may contain spaces, commas, quoted Digest fields,
// or scheme-specific payloads. Redact the full remainder of the bounded line
// instead of trying to parse schemes and accidentally preserving credentials.
var authorizationBrowserValue = regexp.MustCompile(`(?im)(authorization(?:["']?\s*[:=]\s*["']?|\s+))[^\r\n]*`)
var sensitiveBrowserValue = regexp.MustCompile(`(?i)(token|secret|password|api[_-]?key|credential)(["' :=]+)[^\s,"'}]+`)
var personalBrowserPath = regexp.MustCompile(`(?i)(/Users/|/home/|[A-Z]:\\Users\\)[^/\\\s]+`)

func sanitizeEventType(value string) string {
	value = strings.TrimSpace(value)
	var b strings.Builder
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' {
			b.WriteRune(r)
		}
		if b.Len() >= 64 {
			break
		}
	}
	if b.Len() == 0 {
		return "event"
	}
	return b.String()
}

// RedactBrowserText creates a bounded browser DTO string without common
// credential values, control characters, or personal home-directory names.
func RedactBrowserText(value string, limit int) string {
	value = strings.Map(func(r rune) rune {
		if r < 0x20 && r != '\n' && r != '\t' || r == 0x7f {
			return -1
		}
		return r
	}, value)
	value = authorizationBrowserValue.ReplaceAllString(value, "${1}[redacted]")
	value = sensitiveBrowserValue.ReplaceAllString(value, "$1$2[redacted]")
	value = personalBrowserPath.ReplaceAllString(value, "${1}[redacted]")
	if len(value) > limit {
		value = value[:limit] + "…"
	}
	return value
}

type emergencyStopResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

func (d *Dashboard) handleEmergencyStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !requireEmptyBody(w, r) {
		return
	}

	w.Header().Set("Content-Type", "application/json")

	resp := emergencyStopResponse{OK: true}

	if d.cfg.EmergencyStop != nil {
		errs := d.cfg.EmergencyStop()
		if len(errs) > 0 {
			resp.OK = false
			resp.Error = "emergency stop did not complete; inspect the owner-only local command log"
		}
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("dashboard: encode emergency stop response", "error", err)
	}
}

// --- Control API handlers ---

type controlResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("dashboard: encode response", "error", err)
	}
}

func (d *Dashboard) requirePost(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return false
	}
	return true
}

func requireEmptyBody(w http.ResponseWriter, r *http.Request) bool {
	if r.Body == nil {
		return true
	}
	var one [1]byte
	n, _ := r.Body.Read(one[:])
	if n != 0 {
		writeJSON(w, http.StatusBadRequest, controlResponse{Error: "request body must be empty"})
		return false
	}
	return true
}

func (d *Dashboard) handlePause(w http.ResponseWriter, r *http.Request) {
	if !d.requirePost(w, r) {
		return
	}
	if !requireEmptyBody(w, r) {
		return
	}
	if d.cfg.Controls.Pause == nil {
		writeJSON(w, http.StatusNotImplemented, controlResponse{Error: "pause not available"})
		return
	}
	d.cfg.Controls.Pause()
	writeJSON(w, http.StatusOK, controlResponse{OK: true})
}

func (d *Dashboard) handleResume(w http.ResponseWriter, r *http.Request) {
	if !d.requirePost(w, r) {
		return
	}
	if !requireEmptyBody(w, r) {
		return
	}
	if d.cfg.Controls.Resume == nil {
		writeJSON(w, http.StatusNotImplemented, controlResponse{Error: "resume not available"})
		return
	}
	d.cfg.Controls.Resume()
	writeJSON(w, http.StatusOK, controlResponse{OK: true})
}

func (d *Dashboard) handleStop(w http.ResponseWriter, r *http.Request) {
	if !d.requirePost(w, r) {
		return
	}
	if !requireEmptyBody(w, r) {
		return
	}
	if d.cfg.Controls.Stop == nil {
		writeJSON(w, http.StatusNotImplemented, controlResponse{Error: "stop not available"})
		return
	}
	if err := d.cfg.Controls.Stop(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, controlResponse{Error: "stop failed; inspect the owner-only local command log"})
		return
	}
	writeJSON(w, http.StatusOK, controlResponse{OK: true})
}

func (d *Dashboard) handleRestart(w http.ResponseWriter, r *http.Request) {
	if !d.requirePost(w, r) {
		return
	}
	if !requireEmptyBody(w, r) {
		return
	}
	if d.cfg.Controls.Restart == nil {
		writeJSON(w, http.StatusNotImplemented, controlResponse{Error: "restart not available"})
		return
	}
	if err := d.cfg.Controls.Restart(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, controlResponse{Error: "restart failed; inspect the owner-only local command log"})
		return
	}
	writeJSON(w, http.StatusOK, controlResponse{OK: true})
}

func (d *Dashboard) handleScan(w http.ResponseWriter, r *http.Request) {
	if !d.requirePost(w, r) {
		return
	}
	if d.cfg.Controls.Scan == nil {
		writeJSON(w, http.StatusNotImplemented, controlResponse{Error: "scan not available"})
		return
	}
	var body struct {
		RepoID string `json:"repo_id"`
	}
	if err := decodeStrictJSON(w, r, controlBodyLimit, &body); err != nil || !safeControlValue.MatchString(body.RepoID) {
		writeJSON(w, http.StatusBadRequest, controlResponse{Error: "repo_id is required"})
		return
	}
	if err := d.cfg.Controls.Scan(r.Context(), body.RepoID); err != nil {
		writeJSON(w, http.StatusInternalServerError, controlResponse{Error: "scan failed; inspect the owner-only local command log"})
		return
	}
	writeJSON(w, http.StatusOK, controlResponse{OK: true})
}

func (d *Dashboard) handleRunRole(w http.ResponseWriter, r *http.Request) {
	if !d.requirePost(w, r) {
		return
	}
	if d.cfg.Controls.RunRole == nil {
		writeJSON(w, http.StatusNotImplemented, controlResponse{Error: "run-role not available"})
		return
	}
	var body struct {
		RepoID string `json:"repo_id"`
		Role   string `json:"role"`
	}
	if err := decodeStrictJSON(w, r, controlBodyLimit, &body); err != nil || !safeControlValue.MatchString(body.RepoID) || !safeControlValue.MatchString(body.Role) {
		writeJSON(w, http.StatusBadRequest, controlResponse{Error: "repo_id and role are required"})
		return
	}
	if err := d.cfg.Controls.RunRole(r.Context(), body.RepoID, body.Role); err != nil {
		writeJSON(w, http.StatusInternalServerError, controlResponse{Error: "run-role failed; inspect the owner-only local command log"})
		return
	}
	writeJSON(w, http.StatusOK, controlResponse{OK: true})
}

func (d *Dashboard) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if d.cfg.Controls.Status == nil {
		writeJSON(w, http.StatusNotImplemented, controlResponse{Error: "status not available"})
		return
	}
	writeJSON(w, http.StatusOK, projectStatus(d.cfg.Controls.Status(), requestAuthenticated(r)))
}

func projectStatus(value interface{}, includeRepos bool) interface{} {
	raw, err := json.Marshal(value)
	if err != nil {
		return map[string]interface{}{}
	}
	var source map[string]interface{}
	if err := json.Unmarshal(raw, &source); err != nil {
		return map[string]interface{}{}
	}
	result := make(map[string]interface{})
	for _, key := range []string{"healthy", "paused", "active_jobs", "uptime_secs"} {
		if item, ok := source[key]; ok {
			result[key] = item
		}
	}
	if includeRepos {
		if repos, ok := source["repos"].([]interface{}); ok {
			bounded := make([]interface{}, 0, min(len(repos), 256))
			for _, item := range repos[:min(len(repos), 256)] {
				repo, ok := item.(map[string]interface{})
				if !ok {
					continue
				}
				projected := map[string]interface{}{}
				if id, ok := repo["id"].(string); ok {
					projected["id"] = RedactBrowserText(id, 256)
				}
				if path, ok := repo["path"].(string); ok {
					projected["path"] = RedactBrowserText(filepath.Base(path), 128)
				}
				if profile, ok := repo["flow_profile"].(string); ok {
					projected["flow_profile"] = RedactBrowserText(profile, 128)
				}
				bounded = append(bounded, projected)
			}
			result["repos"] = bounded
		}
	}
	return result
}

func (d *Dashboard) handleListRepos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if d.cfg.Controls.ListRepos == nil {
		writeJSON(w, http.StatusOK, []interface{}{})
		return
	}
	repos := d.cfg.Controls.ListRepos()
	if len(repos) > 256 {
		repos = repos[:256]
	}
	for i := range repos {
		repos[i].ID = RedactBrowserText(repos[i].ID, 256)
		repos[i].Path = RedactBrowserText(filepath.Base(repos[i].Path), 128)
	}
	writeJSON(w, http.StatusOK, repos)
}

func (d *Dashboard) handleListRolesForRepo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if d.cfg.Controls.ListRoles == nil {
		writeJSON(w, http.StatusOK, []string{})
		return
	}
	repoID := r.URL.Query().Get("repo_id")
	if repoID == "" {
		writeJSON(w, http.StatusBadRequest, controlResponse{Error: "repo_id query param is required"})
		return
	}
	roles := d.cfg.Controls.ListRoles(repoID)
	if len(roles) > 256 {
		roles = roles[:256]
	}
	for i := range roles {
		roles[i] = RedactBrowserText(roles[i], 128)
	}
	writeJSON(w, http.StatusOK, roles)
}

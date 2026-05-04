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
	"sync"
	"time"
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
	Addr             string // default ":9090"
	EmergencyStop    func() []error
	ChainProvider    func() []ChainNode // returns the live pipeline chain from manifest
	PipelineProvider func() PipelineView
	Controls         ControlCallbacks
}

const eventBufferSize = 200

// Dashboard serves the embedded web UI.
type Dashboard struct {
	cfg        Config
	mux        *http.ServeMux
	tmpl       *template.Template
	sseClients map[chan string]struct{}
	eventBuf   []string
	mu         sync.RWMutex
}

// New creates a Dashboard, parses embedded templates, and wires routes.
func New(cfg Config) (*Dashboard, error) {
	if cfg.Addr == "" {
		cfg.Addr = ":9090"
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
	return d, nil
}

// Handler returns the HTTP handler for the dashboard.
func (d *Dashboard) Handler() http.Handler {
	return d.mux
}

// HandleFunc registers an additional route on the dashboard mux.
func (d *Dashboard) HandleFunc(pattern string, handler http.HandlerFunc) {
	d.mux.HandleFunc(pattern, handler)
}

func (d *Dashboard) routes() {
	d.mux.HandleFunc("/", d.handleIndex)
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

		if name == "pipeline" {
			if d.cfg.PipelineProvider != nil {
				data.Pipeline = d.cfg.PipelineProvider()
				data.ChainNodes = data.Pipeline.Nodes
			} else if d.cfg.ChainProvider != nil {
				data.ChainNodes = d.cfg.ChainProvider()
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

// handleSSE streams server-sent events to the client.
func (d *Dashboard) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
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

type emergencyStopResponse struct {
	OK     bool     `json:"ok"`
	Errors []string `json:"errors,omitempty"`
}

func (d *Dashboard) handleEmergencyStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	resp := emergencyStopResponse{OK: true}

	if d.cfg.EmergencyStop != nil {
		errs := d.cfg.EmergencyStop()
		if len(errs) > 0 {
			resp.OK = false
			for _, e := range errs {
				resp.Errors = append(resp.Errors, e.Error())
			}
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

func (d *Dashboard) handlePause(w http.ResponseWriter, r *http.Request) {
	if !d.requirePost(w, r) {
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
	if d.cfg.Controls.Stop == nil {
		writeJSON(w, http.StatusNotImplemented, controlResponse{Error: "stop not available"})
		return
	}
	if err := d.cfg.Controls.Stop(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, controlResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, controlResponse{OK: true})
}

func (d *Dashboard) handleRestart(w http.ResponseWriter, r *http.Request) {
	if !d.requirePost(w, r) {
		return
	}
	if d.cfg.Controls.Restart == nil {
		writeJSON(w, http.StatusNotImplemented, controlResponse{Error: "restart not available"})
		return
	}
	if err := d.cfg.Controls.Restart(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, controlResponse{Error: err.Error()})
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
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.RepoID == "" {
		writeJSON(w, http.StatusBadRequest, controlResponse{Error: "repo_id is required"})
		return
	}
	if err := d.cfg.Controls.Scan(r.Context(), body.RepoID); err != nil {
		writeJSON(w, http.StatusInternalServerError, controlResponse{Error: err.Error()})
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
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.RepoID == "" || body.Role == "" {
		writeJSON(w, http.StatusBadRequest, controlResponse{Error: "repo_id and role are required"})
		return
	}
	if err := d.cfg.Controls.RunRole(r.Context(), body.RepoID, body.Role); err != nil {
		writeJSON(w, http.StatusInternalServerError, controlResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, controlResponse{OK: true})
}

func (d *Dashboard) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if d.cfg.Controls.Status == nil {
		writeJSON(w, http.StatusNotImplemented, controlResponse{Error: "status not available"})
		return
	}
	writeJSON(w, http.StatusOK, d.cfg.Controls.Status())
}

func (d *Dashboard) handleListRepos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if d.cfg.Controls.ListRepos == nil {
		writeJSON(w, http.StatusOK, []interface{}{})
		return
	}
	writeJSON(w, http.StatusOK, d.cfg.Controls.ListRepos())
}

func (d *Dashboard) handleListRolesForRepo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
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
	writeJSON(w, http.StatusOK, d.cfg.Controls.ListRoles(repoID))
}

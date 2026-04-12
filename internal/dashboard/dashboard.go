package dashboard

import (
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

// Config controls the dashboard.
type Config struct {
	Addr          string       // default ":9090"
	EmergencyStop func() []error
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
	d.mux.HandleFunc("/pipeline", d.handlePage("pipeline", "Pipeline Flow"))
	d.mux.HandleFunc("/roles", d.handlePage("roles", "Role Health"))
	d.mux.HandleFunc("/throughput", d.handlePage("throughput", "Throughput"))
	d.mux.HandleFunc("/debug", d.handlePage("debug", "Debug"))
	d.mux.HandleFunc("/evolution", d.handlePage("evolution", "Evolution History"))
	d.mux.HandleFunc("/api/events", d.handleSSE)
	d.mux.HandleFunc("/api/emergency-stop", d.handleEmergencyStop)

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
	Title     string
	ActiveNav string
	Now       string
}

func (d *Dashboard) handlePage(name, title string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := pageData{
			Title:     title,
			ActiveNav: name,
			Now:       time.Now().Format(time.RFC3339),
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

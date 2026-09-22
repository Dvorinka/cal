package main

// Deferred backend: the Wails window opens immediately and shows setup.html
// while the database warms up — first run may download ~80MB of Postgres.
// Once NewHandler returns, all traffic delegates to the real mux. Errors
// surface on the page instead of a console the user cannot see.

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

//go:embed setup.html
var setupHTML []byte

//go:embed setup.js
var setupJS []byte

type setupStatus struct {
	Phase  string `json:"phase"`
	Detail string `json:"detail,omitempty"`
	Ready  bool   `json:"ready"`
	Error  string `json:"error,omitempty"`
	Log    string `json:"log,omitempty"`
}

type backend struct {
	mu      sync.Mutex
	status  setupStatus
	running bool
	dataDir string
	logPath string
	logFile *os.File

	ready   chan struct{} // closed when init finishes (success or failure)
	handler http.Handler
	closeFn func()
	closed  bool // set by Close; init finishing afterwards must clean up itself
}

func newBackend() *backend {
	dataDir := envOr("DATA_DIR", defaultDataDir())
	logDir := filepath.Join(dataDir, "logs")
	_ = os.MkdirAll(logDir, 0755)
	logPath := filepath.Join(logDir, "cal.log")
	logFile, _ := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)

	b := &backend{dataDir: dataDir, logPath: logPath, logFile: logFile}
	b.status.Log = logPath
	b.start()
	return b
}

// start kicks off NewHandler in the background. Safe to call again after a
// failure (the setup page's retry button).
func (b *backend) start() {
	b.mu.Lock()
	if b.running {
		b.mu.Unlock()
		return
	}
	b.running = true
	b.status = setupStatus{Phase: "start", Detail: "Starting…", Log: b.logPath}
	b.ready = make(chan struct{})
	b.handler, b.closeFn = nil, nil
	b.mu.Unlock()

	go func() {
		h, closeFn, err := NewHandler(b.dataDir, b.report, b.logFile)
		b.mu.Lock()
		b.running = false
		var lateClose func()
		if err != nil {
			b.status.Error = err.Error()
			b.status.Detail = ""
		} else if b.closed {
			// Window closed while init was in flight — stop Postgres now
			// instead of leaving it running past process exit.
			lateClose = closeFn
		} else {
			b.handler, b.closeFn = h, closeFn
			b.status.Ready = true
			b.status.Phase = "ready"
			b.status.Detail = ""
		}
		b.mu.Unlock()
		close(b.ready)
		if lateClose != nil {
			lateClose()
		}
	}()
}

func (b *backend) report(phase, detail string) {
	b.mu.Lock()
	b.status.Phase = phase
	b.status.Detail = detail
	b.mu.Unlock()
	if b.logFile != nil {
		fmt.Fprintf(b.logFile, "%s [%s] %s\n", time.Now().Format(time.RFC3339), phase, detail)
	}
}

// Close is called on shutdown; safe while init is still running.
func (b *backend) Close() {
	b.mu.Lock()
	b.closed = true
	closeFn := b.closeFn
	b.mu.Unlock()
	if closeFn != nil {
		closeFn()
	}
	if b.logFile != nil {
		_ = b.logFile.Close()
	}
}

func (b *backend) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/__setup/status":
		b.mu.Lock()
		st := b.status
		b.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache")
		_ = json.NewEncoder(w).Encode(st)
		return
	case "/__setup/setup.js":
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(setupJS)
		return
	case "/__setup/retry":
		if r.Method == http.MethodPost {
			b.start()
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/api" {
		select {
		case <-b.ready:
		case <-r.Context().Done():
			return
		}
		if b.handler == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": b.status.Error})
			return
		}
		b.handler.ServeHTTP(w, r)
		return
	}

	if h := b.handlerIfReady(); h != nil {
		h.ServeHTTP(w, r)
		return
	}
	securityHeaders(w, r.URL.Path)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(setupHTML)
}

func (b *backend) handlerIfReady() http.Handler {
	select {
	case <-b.ready:
		return b.handler
	default:
		return nil
	}
}

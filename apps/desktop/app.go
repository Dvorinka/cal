package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"cal/apps/api/app"
)

//go:embed all:frontend/dist
var distFS embed.FS

// NewHandler runs the full Cal API in-process and serves the built SPA for
// every non-/api path. No TCP listener — Wails talks to it through
// AssetServer.Handler.
//
// DATABASE_URL set → external Postgres (dev, power users). Unset → an
// embedded Postgres starts in <dataDir>/pg on a free port so the binary is
// turnkey; binaries come from the installer-bundled pg-runtime when present,
// else download once into <dataDir>/pg-bin.
func NewHandler(dataDir string, report progress, logFile *os.File) (http.Handler, func(), error) {
	var pg *embeddedPG
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		var err error
		pg, databaseURL, err = startEmbeddedPostgres(dataDir, report, logFile)
		if err != nil {
			return nil, nil, fmt.Errorf("embedded postgres: %w", err)
		}
	} else {
		report("connect", "Connecting to Postgres")
	}

	report("migrate", "Preparing application")
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	api, closeDB, err := app.New(ctx, databaseURL, dataDir)
	if err != nil {
		if pg != nil {
			pg.stop()
		}
		return nil, nil, err
	}

	web, err := fs.Sub(distFS, "frontend/dist")
	if err != nil {
		return nil, nil, err
	}
	files := http.FileServer(http.FS(web))

	mux := http.NewServeMux()
	mux.Handle("/api/", api)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// SPA fallback — missing files serve index.html so client routes work.
		// fs.FS paths never start with "/", so strip before probing.
		p := strings.TrimPrefix(filepath.Clean("/"+r.URL.Path), "/")
		if f, ferr := web.Open(p); ferr == nil {
			st, _ := f.Stat()
			_ = f.Close()
			if st.IsDir() {
				r.URL.Path = "/"
			}
		} else {
			r.URL.Path = "/"
		}
		securityHeaders(w, r.URL.Path)
		files.ServeHTTP(w, r)
	})
	return mux, func() {
		closeDB()
		if pg != nil {
			pg.stop()
		}
	}, nil
}

// securityHeaders mirrors the headers nginx used to set when the web build was
// served standalone. connect-src allows the keyless Open-Meteo endpoints;
// the service worker must never be served stale.
func securityHeaders(w http.ResponseWriter, path string) {
	h := w.Header()
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("X-Frame-Options", "SAMEORIGIN")
	h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
	h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
	h.Set("Content-Security-Policy",
		"default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; "+
			"img-src 'self' data: https:; font-src 'self' data:; "+
			"connect-src 'self' https://api.open-meteo.com https://geocoding-api.open-meteo.com; "+
			"frame-ancestors 'self'; base-uri 'self'; form-action 'self'")
	if path == "/sw.js" || path == "/sw-push.js" {
		h.Set("Cache-Control", "no-cache")
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func defaultDataDir() string {
	// LocalAppData on Windows — a Postgres data cluster has no business in a
	// roaming profile. UserConfigDir there maps to %APPDATA% (roaming).
	if runtime.GOOS == "windows" {
		if dir := os.Getenv("LOCALAPPDATA"); dir != "" {
			return filepath.Join(dir, "cal")
		}
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "./data"
	}
	return filepath.Join(dir, "cal")
}

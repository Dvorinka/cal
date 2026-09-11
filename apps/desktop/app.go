package main

import (
	"context"
	"embed"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"cal/apps/api/app"
)

//go:embed all:frontend/dist
var distFS embed.FS

// NewHandler runs the full Cal API in-process and serves the built SPA for
// every non-/api path. No TCP listener — Wails talks to it through
// AssetServer.Handler.
func NewHandler() (http.Handler, func(), error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	api, closeDB, err := app.New(ctx,
		envOr("DATABASE_URL", "postgres://cal:cal@localhost:5432/cal?sslmode=disable"),
		envOr("DATA_DIR", defaultDataDir()),
	)
	if err != nil {
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
		p := filepath.Clean("/" + r.URL.Path)
		f, ferr := web.Open(p)
		if ferr != nil {
			r.URL.Path = "/"
		} else {
			st, _ := f.Stat()
			_ = f.Close()
			if st.IsDir() {
				r.URL.Path = "/"
			}
		}
		files.ServeHTTP(w, r)
	})
	return mux, closeDB, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func defaultDataDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "./data"
	}
	return filepath.Join(dir, "cal")
}

package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"cal/apps/api/app"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
)

//go:embed all:frontend/dist
var distFS embed.FS

// NewHandler runs the full Cal API in-process and serves the built SPA for
// every non-/api path. No TCP listener — Wails talks to it through
// AssetServer.Handler.
//
// DATABASE_URL set → external Postgres (dev, power users). Unset → an
// embedded Postgres starts in <dataDir>/pg on a free port so the binary is
// turnkey; first run unpacks ~80MB of binaries into the cache.
func NewHandler() (http.Handler, func(), error) {
	dataDir := envOr("DATA_DIR", defaultDataDir())

	var pg *embeddedpostgres.EmbeddedPostgres
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		var err error
		pg, databaseURL, err = startEmbeddedPostgres(filepath.Join(dataDir, "pg"))
		if err != nil {
			return nil, nil, fmt.Errorf("embedded postgres: %w", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	api, closeDB, err := app.New(ctx, databaseURL, dataDir)
	if err != nil {
		if pg != nil {
			_ = pg.Stop()
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
	return mux, func() {
		closeDB()
		if pg != nil {
			_ = pg.Stop()
		}
	}, nil
}

// startEmbeddedPostgres boots a private Postgres in dir on a free port.
func startEmbeddedPostgres(dir string) (*embeddedpostgres.EmbeddedPostgres, string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, "", err
	}
	port := uint32(ln.Addr().(*net.TCPAddr).Port)
	_ = ln.Close()

	pg := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
		Port(port).
		DataPath(dir).
		Username("cal").
		Password("cal").
		Database("cal"))
	if err := pg.Start(); err != nil {
		return nil, "", err
	}
	return pg, fmt.Sprintf("postgres://cal:cal@127.0.0.1:%d/cal?sslmode=disable", port), nil
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

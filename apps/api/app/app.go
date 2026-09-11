// Package app composes the Cal API: store connection, background sync loops,
// and the HTTP handler. cmd/server serves it over TCP; apps/desktop mounts
// it inside a Wails window without a listener.
package app

import (
	"context"
	"net/http"
	"time"

	"cal/apps/api/internal/caldav"
	"cal/apps/api/internal/calendar"
	"cal/apps/api/internal/httpapi"
	"cal/apps/api/internal/store"
)

// New connects to Postgres, starts the background loops, and returns the API
// handler. Call the returned close func on shutdown.
func New(ctx context.Context, databaseURL, dataDir string) (http.Handler, func(), error) {
	db, err := store.Connect(ctx, databaseURL)
	if err != nil {
		return nil, nil, err
	}
	s := store.New(db)
	go httpapi.RefreshFeedsLoop(context.Background(), s, 30*time.Minute)
	go httpapi.PushLoop(context.Background(), s, time.Minute)
	go caldavLoop(context.Background(), s)
	go httpapi.GoogleSyncLoop(context.Background(), s, 15*time.Minute)
	go httpapi.GitHubSyncLoop(context.Background(), s, 15*time.Minute)
	go httpapi.BackupLoop(context.Background(), s, dataDir)

	router := httpapi.New(s, calendar.NewHolidayCache())
	return router, db.Close, nil
}

// caldavLoop syncs every connected account every 15 minutes.
func caldavLoop(ctx context.Context, s *store.Store) {
	syncer := caldav.NewSyncer(s)
	syncer.SyncAll(ctx)
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			syncer.SyncAll(ctx)
		}
	}
}

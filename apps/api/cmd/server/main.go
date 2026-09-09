package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"cal/apps/api/internal/calendar"
	"cal/apps/api/internal/httpapi"
	"cal/apps/api/internal/store"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := store.Connect(ctx, env("DATABASE_URL", "postgres://cal:cal@localhost:5432/cal?sslmode=disable"))
	if err != nil {
		slog.Error("connect database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	router := httpapi.New(store.New(db), calendar.NewHolidayCache())
	addr := ":" + env("PORT", "8080")
	slog.Info("api listening", "addr", addr)
	if err := router.Run(addr); err != nil {
		slog.Error("serve api", "error", err)
		os.Exit(1)
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

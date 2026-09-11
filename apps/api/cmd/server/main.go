package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"cal/apps/api/app"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	handler, closeDB, err := app.New(ctx,
		env("DATABASE_URL", "postgres://cal:cal@localhost:5432/cal?sslmode=disable"),
		env("DATA_DIR", "./data"),
	)
	if err != nil {
		slog.Error("init api", "error", err)
		os.Exit(1)
	}
	defer closeDB()

	addr := ":" + env("PORT", "8080")
	slog.Info("api listening", "addr", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
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

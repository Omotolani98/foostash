package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Omotolani98/foostash/internal/server"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))

	pgURL := envDefault("FOOSTASH_PG_URL", "postgres://foostash:foostash@localhost:5432/foostash?sslmode=disable")
	listenAddr := envDefault("FOOSTASH_LISTEN_ADDR", ":8400")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := server.Boot(ctx, pgURL, listenAddr, ""); err != nil {
		slog.Error("server exited", "err", err)
		os.Exit(1)
	}
	slog.Info("server stopped")
}

func envDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

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

	pgURL := os.Getenv("FOOSTASH_PG_URL")
	if pgURL == "" {
		pgURL = "postgres://foostash:foostash@localhost:5432/foostash?sslmode=disable"
	}
	listenAddr := os.Getenv("PORT")
	if listenAddr == "" {
		listenAddr = ":8400"
	}
	sshAddr := os.Getenv("FOOSTASH_SSH_ADDR")
	sshHostKey := os.Getenv("FOOSTASH_SSH_HOST_KEY")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := server.Boot(ctx, server.Config{
		HTTPAddr:       listenAddr,
		SSHAddr:        sshAddr,
		SSHHostKeyPath: sshHostKey,
		PGURL:          pgURL,
	}); err != nil {
		slog.Error("server exited", "err", err)
		os.Exit(1)
	}
	slog.Info("server stopped")
}

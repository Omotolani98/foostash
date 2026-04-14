package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Omotolani98/foostash/internal/migrations"
	"github.com/Omotolani98/foostash/internal/repo"
	"github.com/Omotolani98/foostash/internal/server"
	"github.com/Omotolani98/foostash/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))

	pgURL := envDefault("FOOSTASH_PG_URL", "postgres://foostash:foostash@localhost:5432/foostash?sslmode=disable")
	listenAddr := envDefault("FOOSTASH_LISTEN_ADDR", ":8400")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	connectCtx, connectCancel := context.WithTimeout(ctx, 10*time.Second)
	defer connectCancel()
	pool, err := pgxpool.New(connectCtx, pgURL)
	if err != nil {
		slog.Error("connect postgres", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := migrations.Apply(ctx, pool); err != nil {
		slog.Error("apply migrations", "err", err)
		os.Exit(1)
	}

	repos := repo.New(pool)
	deps := &server.Deps{
		Auth:     service.NewAuth(repos),
		Invites:  service.NewInvites(repos),
		Projects: service.NewProjects(repos),
	}

	srv := server.New(listenAddr, deps)
	if err := srv.Run(ctx); err != nil {
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

package server

import (
	"context"
	"fmt"
	"time"

	"github.com/Omotolani98/foostash/internal/migrations"
	"github.com/Omotolani98/foostash/internal/repo"
	"github.com/Omotolani98/foostash/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Boot connects to Postgres, applies migrations, wires services, and runs the
// HTTP server until ctx is canceled. version is surfaced on /v1/health.
func Boot(ctx context.Context, pgURL, listenAddr, version string) error {
	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(connectCtx, pgURL)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer pool.Close()

	if err := migrations.Apply(ctx, pool); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	repos := repo.New(pool)
	deps := &Deps{
		Auth:     service.NewAuth(repos),
		Invites:  service.NewInvites(repos),
		Projects: service.NewProjects(repos),
		Users:    service.NewUsers(repos),
		Audit:    service.NewAudit(repos),
		Pool:     pool,
		Version:  version,
	}

	return New(listenAddr, deps).Run(ctx)
}

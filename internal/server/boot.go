package server

import (
	"context"
	"fmt"
	"time"

	"github.com/Omotolani98/foostash/internal/billing"
	"github.com/Omotolani98/foostash/internal/migrations"
	"github.com/Omotolani98/foostash/internal/repo"
	"github.com/Omotolani98/foostash/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Config groups everything Boot needs to start the server.
type Config struct {
	HTTPAddr       string
	SSHAddr        string // empty disables SSH
	SSHHostKeyPath string
	PGURL          string
	Version        string
}

// Boot connects to Postgres, applies migrations, wires services, and runs the
// HTTP + SSH servers until ctx is canceled.
func Boot(ctx context.Context, cfg Config) error {
	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(connectCtx, cfg.PGURL)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer pool.Close()

	if err := migrations.Apply(ctx, pool); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	billingImpl, err := billing.FromEnv()
	if err != nil {
		return fmt.Errorf("init billing: %w", err)
	}

	repos := repo.New(pool)
	deps := &Deps{
		Auth:     service.NewAuth(repos),
		Invites:  service.NewInvites(repos),
		Projects: service.NewProjects(repos),
		Users:    service.NewUsers(repos),
		Audit:    service.NewAudit(repos),
		Secrets:  service.NewSecrets(repos),
		Vault:    service.NewVault(repos),
		Pool:     pool,
		Version:  cfg.Version,
		Billing:  billingImpl,
	}

	srv, err := New(Options{
		HTTPAddr:       cfg.HTTPAddr,
		SSHAddr:        cfg.SSHAddr,
		SSHHostKeyPath: cfg.SSHHostKeyPath,
	}, deps)
	if err != nil {
		return fmt.Errorf("build server: %w", err)
	}
	return srv.Run(ctx)
}

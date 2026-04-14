// Package repo contains pgx-backed repositories for server-side persistence.
package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when a lookup produces no row.
var ErrNotFound = errors.New("not found")

// Repos groups all repositories so services receive a single dependency.
type Repos struct {
	Orgs         *OrgRepo
	Users        *UserRepo
	SSHKeys      *SSHKeyRepo
	Invites      *InviteRepo
	Projects     *ProjectRepo
	Environments *EnvironmentRepo
	Pool         *pgxpool.Pool
}

// New constructs a Repos bundle from a pgx pool.
func New(pool *pgxpool.Pool) *Repos {
	return &Repos{
		Orgs:         &OrgRepo{pool: pool},
		Users:        &UserRepo{pool: pool},
		SSHKeys:      &SSHKeyRepo{pool: pool},
		Invites:      &InviteRepo{pool: pool},
		Projects:     &ProjectRepo{pool: pool},
		Environments: &EnvironmentRepo{pool: pool},
		Pool:         pool,
	}
}

// InTx runs fn inside a transaction, committing on success and rolling back on error.
func InTx(ctx context.Context, pool *pgxpool.Pool, fn func(pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

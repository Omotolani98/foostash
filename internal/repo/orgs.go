package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Org struct {
	ID        uuid.UUID
	Name      string
	Slug      string
	Plan      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type OrgRepo struct {
	pool *pgxpool.Pool
}

// Insert creates an org using the given executor (pool or tx).
func (r *OrgRepo) Insert(ctx context.Context, q Querier, name, slug string) (*Org, error) {
	const sql = `
		INSERT INTO organizations (name, slug)
		VALUES ($1, $2)
		RETURNING id, name, slug, plan, created_at, updated_at`
	var o Org
	err := q.QueryRow(ctx, sql, name, slug).Scan(&o.ID, &o.Name, &o.Slug, &o.Plan, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert org: %w", err)
	}
	return &o, nil
}

// GetBySlug looks up an org by its slug.
func (r *OrgRepo) GetBySlug(ctx context.Context, slug string) (*Org, error) {
	const sql = `SELECT id, name, slug, plan, created_at, updated_at FROM organizations WHERE slug = $1`
	var o Org
	err := r.pool.QueryRow(ctx, sql, slug).Scan(&o.ID, &o.Name, &o.Slug, &o.Plan, &o.CreatedAt, &o.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get org: %w", err)
	}
	return &o, nil
}

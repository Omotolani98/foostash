package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID        uuid.UUID
	OrgID     uuid.UUID
	Email     string
	Role      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type UserRepo struct {
	pool *pgxpool.Pool
}

// Querier matches *pgxpool.Pool and pgx.Tx so repos can be used inside
// transactions without wrapping types.
type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func (r *UserRepo) Insert(ctx context.Context, q Querier, orgID uuid.UUID, email, role string) (*User, error) {
	const sql = `
		INSERT INTO users (org_id, email, role)
		VALUES ($1, $2, $3)
		RETURNING id, org_id, email, role, created_at, updated_at`
	var u User
	err := q.QueryRow(ctx, sql, orgID, email, role).Scan(&u.ID, &u.OrgID, &u.Email, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}
	return &u, nil
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	const sql = `SELECT id, org_id, email, role, created_at, updated_at FROM users WHERE id = $1`
	var u User
	err := r.pool.QueryRow(ctx, sql, id).Scan(&u.ID, &u.OrgID, &u.Email, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return &u, nil
}

// Pool returns the underlying pool for repos that need direct access (helper for the pool field).
func (r *UserRepo) Pool() *pgxpool.Pool { return r.pool }

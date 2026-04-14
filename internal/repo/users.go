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
	RevokedAt *time.Time
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

const userCols = `id, org_id, email, role, revoked_at, created_at, updated_at`

func scanUser(row pgx.Row, u *User) error {
	return row.Scan(&u.ID, &u.OrgID, &u.Email, &u.Role, &u.RevokedAt, &u.CreatedAt, &u.UpdatedAt)
}

func (r *UserRepo) Insert(ctx context.Context, q Querier, orgID uuid.UUID, email, role string) (*User, error) {
	const sql = `
		INSERT INTO users (org_id, email, role)
		VALUES ($1, $2, $3)
		RETURNING ` + userCols
	var u User
	if err := scanUser(q.QueryRow(ctx, sql, orgID, email, role), &u); err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}
	return &u, nil
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	const sql = `SELECT ` + userCols + ` FROM users WHERE id = $1`
	var u User
	if err := scanUser(r.pool.QueryRow(ctx, sql, id), &u); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get user: %w", err)
	}
	return &u, nil
}

// ListByOrg returns all users in an org ordered by email. Includes revoked users.
func (r *UserRepo) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]User, error) {
	const sql = `SELECT ` + userCols + ` FROM users WHERE org_id = $1 ORDER BY email`
	rows, err := r.pool.Query(ctx, sql, orgID)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()
	var out []User
	for rows.Next() {
		var u User
		if err := scanUser(rows, &u); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// SetRole updates a user's role. Returns ErrNotFound if no user matched.
func (r *UserRepo) SetRole(ctx context.Context, id uuid.UUID, role string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE users SET role = $2, updated_at = now() WHERE id = $1`, id, role)
	if err != nil {
		return fmt.Errorf("set role: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Revoke marks a user as revoked. Idempotent (subsequent calls refresh the timestamp).
func (r *UserRepo) Revoke(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE users SET revoked_at = now(), updated_at = now() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("revoke user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Pool returns the underlying pool for repos that need direct access (helper for the pool field).
func (r *UserRepo) Pool() *pgxpool.Pool { return r.pool }

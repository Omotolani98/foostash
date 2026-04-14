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

type Invite struct {
	ID        uuid.UUID
	OrgID     uuid.UUID
	Email     string
	Role      string
	Token     string
	UsedAt    *time.Time
	ExpiresAt time.Time
	CreatedBy *uuid.UUID
	CreatedAt time.Time
}

type InviteRepo struct {
	pool *pgxpool.Pool
}

func (r *InviteRepo) Insert(ctx context.Context, q Querier, orgID uuid.UUID, email, role, token string, expiresAt time.Time, createdBy uuid.UUID) (*Invite, error) {
	const sql = `
		INSERT INTO invites (org_id, email, role, token, expires_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, org_id, email, role, token, used_at, expires_at, created_by, created_at`
	var i Invite
	err := q.QueryRow(ctx, sql, orgID, email, role, token, expiresAt, createdBy).Scan(
		&i.ID, &i.OrgID, &i.Email, &i.Role, &i.Token, &i.UsedAt, &i.ExpiresAt, &i.CreatedBy, &i.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert invite: %w", err)
	}
	return &i, nil
}

func (r *InviteRepo) GetByToken(ctx context.Context, q Querier, token string) (*Invite, error) {
	const sql = `
		SELECT id, org_id, email, role, token, used_at, expires_at, created_by, created_at
		FROM invites WHERE token = $1`
	var i Invite
	err := q.QueryRow(ctx, sql, token).Scan(
		&i.ID, &i.OrgID, &i.Email, &i.Role, &i.Token, &i.UsedAt, &i.ExpiresAt, &i.CreatedBy, &i.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get invite: %w", err)
	}
	return &i, nil
}

func (r *InviteRepo) MarkUsed(ctx context.Context, q Querier, id uuid.UUID) error {
	_, err := q.Exec(ctx, `UPDATE invites SET used_at = now() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("mark invite used: %w", err)
	}
	return nil
}

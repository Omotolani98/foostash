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

type SSHKey struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	PublicKey   string
	Fingerprint string
	Name        *string
	LastUsedAt  *time.Time
	CreatedAt   time.Time
}

// KeyOwner is the denormalized view of a key plus its user and org, used
// by the auth middleware to build an AuthContext in a single query.
type KeyOwner struct {
	Key   SSHKey
	User  User
	OrgID uuid.UUID
}

type SSHKeyRepo struct {
	pool *pgxpool.Pool
}

func (r *SSHKeyRepo) Insert(ctx context.Context, q Querier, userID uuid.UUID, publicKey, fingerprint string) (*SSHKey, error) {
	const sql = `
		INSERT INTO ssh_keys (user_id, public_key, fingerprint)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, public_key, fingerprint, name, last_used_at, created_at`
	var k SSHKey
	err := q.QueryRow(ctx, sql, userID, publicKey, fingerprint).Scan(
		&k.ID, &k.UserID, &k.PublicKey, &k.Fingerprint, &k.Name, &k.LastUsedAt, &k.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert ssh_key: %w", err)
	}
	return &k, nil
}

// GetByFingerprint returns the key and its owning user/org.
func (r *SSHKeyRepo) GetByFingerprint(ctx context.Context, fingerprint string) (*KeyOwner, error) {
	const sql = `
		SELECT k.id, k.user_id, k.public_key, k.fingerprint, k.name, k.last_used_at, k.created_at,
		       u.id, u.org_id, u.email, u.role, u.revoked_at, u.created_at, u.updated_at
		FROM ssh_keys k
		JOIN users u ON u.id = k.user_id
		WHERE k.fingerprint = $1`
	var out KeyOwner
	err := r.pool.QueryRow(ctx, sql, fingerprint).Scan(
		&out.Key.ID, &out.Key.UserID, &out.Key.PublicKey, &out.Key.Fingerprint, &out.Key.Name, &out.Key.LastUsedAt, &out.Key.CreatedAt,
		&out.User.ID, &out.User.OrgID, &out.User.Email, &out.User.Role, &out.User.RevokedAt, &out.User.CreatedAt, &out.User.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get ssh_key: %w", err)
	}
	out.OrgID = out.User.OrgID
	return &out, nil
}

func (r *SSHKeyRepo) TouchLastUsed(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE ssh_keys SET last_used_at = now() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("touch ssh_key: %w", err)
	}
	return nil
}

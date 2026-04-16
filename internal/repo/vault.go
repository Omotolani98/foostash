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

// VaultSecret is a currently-live org-scoped vault secret row.
type VaultSecret struct {
	ID         uuid.UUID
	OrgID      uuid.UUID
	Key        string
	Ciphertext []byte
	Nonce      []byte
	Version    int
	UpdatedBy  *uuid.UUID
	UpdatedAt  time.Time
}

// VaultSecretHistory is a single historical version of a vault secret.
type VaultSecretHistory struct {
	Version    int
	Ciphertext []byte
	Nonce      []byte
	SetBy      *uuid.UUID
	SetAt      time.Time
}

type VaultRepo struct {
	pool *pgxpool.Pool
}

// Upsert inserts a new vault secret or bumps the version of an existing one,
// atomically writing a matching vault_secret_history row. Returns the new version.
func (r *VaultRepo) Upsert(ctx context.Context, orgID uuid.UUID, key string, ciphertext, nonce []byte, updatedBy uuid.UUID) (int, error) {
	var version int
	err := InTx(ctx, r.pool, func(tx pgx.Tx) error {
		const upsertSQL = `
			INSERT INTO vault_secrets (org_id, key, ciphertext, nonce, version, updated_by, updated_at)
			VALUES ($1, $2, $3, $4, 1, $5, now())
			ON CONFLICT (org_id, key) WHERE deleted_at IS NULL
			DO UPDATE SET
				ciphertext = EXCLUDED.ciphertext,
				nonce      = EXCLUDED.nonce,
				version    = vault_secrets.version + 1,
				updated_by = EXCLUDED.updated_by,
				updated_at = now()
			RETURNING version`
		if err := tx.QueryRow(ctx, upsertSQL, orgID, key, ciphertext, nonce, updatedBy).Scan(&version); err != nil {
			return fmt.Errorf("upsert vault secret: %w", err)
		}
		const histSQL = `
			INSERT INTO vault_secret_history (org_id, key, version, ciphertext, nonce, set_by)
			VALUES ($1, $2, $3, $4, $5, $6)`
		if _, err := tx.Exec(ctx, histSQL, orgID, key, version, ciphertext, nonce, updatedBy); err != nil {
			return fmt.Errorf("insert vault history: %w", err)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return version, nil
}

// Get returns a single live vault secret for an org/key.
func (r *VaultRepo) Get(ctx context.Context, orgID uuid.UUID, key string) (*VaultSecret, error) {
	const sql = `
		SELECT id, org_id, key, ciphertext, nonce, version, updated_by, updated_at
		FROM vault_secrets
		WHERE org_id = $1 AND key = $2 AND deleted_at IS NULL`
	var v VaultSecret
	err := r.pool.QueryRow(ctx, sql, orgID, key).Scan(
		&v.ID, &v.OrgID, &v.Key, &v.Ciphertext, &v.Nonce, &v.Version, &v.UpdatedBy, &v.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get vault secret: %w", err)
	}
	return &v, nil
}

// List returns all live vault secrets for an org, ordered by key.
func (r *VaultRepo) List(ctx context.Context, orgID uuid.UUID) ([]VaultSecret, error) {
	const sql = `
		SELECT id, org_id, key, ciphertext, nonce, version, updated_by, updated_at
		FROM vault_secrets
		WHERE org_id = $1 AND deleted_at IS NULL
		ORDER BY key`
	rows, err := r.pool.Query(ctx, sql, orgID)
	if err != nil {
		return nil, fmt.Errorf("list vault secrets: %w", err)
	}
	defer rows.Close()
	var out []VaultSecret
	for rows.Next() {
		var v VaultSecret
		if err := rows.Scan(&v.ID, &v.OrgID, &v.Key, &v.Ciphertext, &v.Nonce, &v.Version, &v.UpdatedBy, &v.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan vault secret: %w", err)
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// Delete soft-deletes a live vault secret by stamping deleted_at.
func (r *VaultRepo) Delete(ctx context.Context, orgID uuid.UUID, key string) error {
	const sql = `
		UPDATE vault_secrets SET deleted_at = now()
		WHERE org_id = $1 AND key = $2 AND deleted_at IS NULL`
	tag, err := r.pool.Exec(ctx, sql, orgID, key)
	if err != nil {
		return fmt.Errorf("delete vault secret: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// History returns all historical versions of a vault key, newest first.
func (r *VaultRepo) History(ctx context.Context, orgID uuid.UUID, key string) ([]VaultSecretHistory, error) {
	const sql = `
		SELECT version, ciphertext, nonce, set_by, set_at
		FROM vault_secret_history
		WHERE org_id = $1 AND key = $2
		ORDER BY version DESC`
	rows, err := r.pool.Query(ctx, sql, orgID, key)
	if err != nil {
		return nil, fmt.Errorf("history vault secret: %w", err)
	}
	defer rows.Close()
	var out []VaultSecretHistory
	for rows.Next() {
		var h VaultSecretHistory
		if err := rows.Scan(&h.Version, &h.Ciphertext, &h.Nonce, &h.SetBy, &h.SetAt); err != nil {
			return nil, fmt.Errorf("scan vault history: %w", err)
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// GetHistoryVersion returns a specific historical version's ciphertext+nonce.
func (r *VaultRepo) GetHistoryVersion(ctx context.Context, orgID uuid.UUID, key string, version int) (*VaultSecretHistory, error) {
	const sql = `
		SELECT version, ciphertext, nonce, set_by, set_at
		FROM vault_secret_history
		WHERE org_id = $1 AND key = $2 AND version = $3`
	var h VaultSecretHistory
	err := r.pool.QueryRow(ctx, sql, orgID, key, version).Scan(
		&h.Version, &h.Ciphertext, &h.Nonce, &h.SetBy, &h.SetAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get vault history version: %w", err)
	}
	return &h, nil
}

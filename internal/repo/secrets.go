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

// Secret is a currently-live secret row.
type Secret struct {
	ID         uuid.UUID
	EnvID      uuid.UUID
	Key        string
	Ciphertext []byte
	Nonce      []byte
	Version    int
	UpdatedBy  *uuid.UUID
	UpdatedAt  time.Time
}

// SecretHistory is a single historical version of a secret.
type SecretHistory struct {
	Version    int
	Ciphertext []byte
	Nonce      []byte
	SetBy      *uuid.UUID
	SetAt      time.Time
}

type SecretsRepo struct {
	pool *pgxpool.Pool
}

// Upsert inserts a new secret or bumps the version of an existing one,
// atomically writing a matching secret_history row. Returns the new version.
func (r *SecretsRepo) Upsert(ctx context.Context, envID uuid.UUID, key string, ciphertext, nonce []byte, updatedBy uuid.UUID) (int, error) {
	var version int
	err := InTx(ctx, r.pool, func(tx pgx.Tx) error {
		const upsertSQL = `
			INSERT INTO secrets (env_id, key, ciphertext, nonce, version, updated_by, updated_at)
			VALUES ($1, $2, $3, $4, 1, $5, now())
			ON CONFLICT (env_id, key) WHERE deleted_at IS NULL
			DO UPDATE SET
				ciphertext = EXCLUDED.ciphertext,
				nonce      = EXCLUDED.nonce,
				version    = secrets.version + 1,
				updated_by = EXCLUDED.updated_by,
				updated_at = now()
			RETURNING version`
		if err := tx.QueryRow(ctx, upsertSQL, envID, key, ciphertext, nonce, updatedBy).Scan(&version); err != nil {
			return fmt.Errorf("upsert secret: %w", err)
		}
		const histSQL = `
			INSERT INTO secret_history (env_id, key, version, ciphertext, nonce, set_by)
			VALUES ($1, $2, $3, $4, $5, $6)`
		if _, err := tx.Exec(ctx, histSQL, envID, key, version, ciphertext, nonce, updatedBy); err != nil {
			return fmt.Errorf("insert history: %w", err)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return version, nil
}

// Get returns a single live secret for an env/key.
func (r *SecretsRepo) Get(ctx context.Context, envID uuid.UUID, key string) (*Secret, error) {
	const sql = `
		SELECT id, env_id, key, ciphertext, nonce, version, updated_by, updated_at
		FROM secrets
		WHERE env_id = $1 AND key = $2 AND deleted_at IS NULL`
	var s Secret
	err := r.pool.QueryRow(ctx, sql, envID, key).Scan(
		&s.ID, &s.EnvID, &s.Key, &s.Ciphertext, &s.Nonce, &s.Version, &s.UpdatedBy, &s.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get secret: %w", err)
	}
	return &s, nil
}

// List returns all live secrets for an env, ordered by key.
func (r *SecretsRepo) List(ctx context.Context, envID uuid.UUID) ([]Secret, error) {
	const sql = `
		SELECT id, env_id, key, ciphertext, nonce, version, updated_by, updated_at
		FROM secrets
		WHERE env_id = $1 AND deleted_at IS NULL
		ORDER BY key`
	rows, err := r.pool.Query(ctx, sql, envID)
	if err != nil {
		return nil, fmt.Errorf("list secrets: %w", err)
	}
	defer rows.Close()
	var out []Secret
	for rows.Next() {
		var s Secret
		if err := rows.Scan(&s.ID, &s.EnvID, &s.Key, &s.Ciphertext, &s.Nonce, &s.Version, &s.UpdatedBy, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan secret: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// Delete soft-deletes a live secret by stamping deleted_at.
func (r *SecretsRepo) Delete(ctx context.Context, envID uuid.UUID, key string) error {
	const sql = `
		UPDATE secrets SET deleted_at = now()
		WHERE env_id = $1 AND key = $2 AND deleted_at IS NULL`
	tag, err := r.pool.Exec(ctx, sql, envID, key)
	if err != nil {
		return fmt.Errorf("delete secret: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// History returns all historical versions of a key, newest first.
func (r *SecretsRepo) History(ctx context.Context, envID uuid.UUID, key string) ([]SecretHistory, error) {
	const sql = `
		SELECT version, ciphertext, nonce, set_by, set_at
		FROM secret_history
		WHERE env_id = $1 AND key = $2
		ORDER BY version DESC`
	rows, err := r.pool.Query(ctx, sql, envID, key)
	if err != nil {
		return nil, fmt.Errorf("history secret: %w", err)
	}
	defer rows.Close()
	var out []SecretHistory
	for rows.Next() {
		var h SecretHistory
		if err := rows.Scan(&h.Version, &h.Ciphertext, &h.Nonce, &h.SetBy, &h.SetAt); err != nil {
			return nil, fmt.Errorf("scan history: %w", err)
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// GetHistoryVersion returns a specific historical version's ciphertext+nonce.
func (r *SecretsRepo) GetHistoryVersion(ctx context.Context, envID uuid.UUID, key string, version int) (*SecretHistory, error) {
	const sql = `
		SELECT version, ciphertext, nonce, set_by, set_at
		FROM secret_history
		WHERE env_id = $1 AND key = $2 AND version = $3`
	var h SecretHistory
	err := r.pool.QueryRow(ctx, sql, envID, key, version).Scan(
		&h.Version, &h.Ciphertext, &h.Nonce, &h.SetBy, &h.SetAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get history version: %w", err)
	}
	return &h, nil
}

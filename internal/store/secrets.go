package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type SecretRow struct {
	ID             string
	EnvironmentID  string
	Key            string
	EncryptedValue []byte
	Nonce          []byte
	Version        int
	CreatedBy      sql.NullString
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type SecretsStore struct {
	db *sql.DB
}

func NewSecretsStore(db *sql.DB) *SecretsStore {
	return &SecretsStore{db: db}
}

func (s *SecretsStore) GetByEnvironment(ctx context.Context, envID string) ([]SecretRow, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, environment_id, key, encrypted_value, nonce, version, created_by, created_at, updated_at
		 FROM secrets WHERE environment_id = $1 ORDER BY key`, envID)
	if err != nil {
		return nil, fmt.Errorf("query secrets: %w", err)
	}
	defer rows.Close()

	var out []SecretRow
	for rows.Next() {
		var r SecretRow
		if err := rows.Scan(&r.ID, &r.EnvironmentID, &r.Key, &r.EncryptedValue,
			&r.Nonce, &r.Version, &r.CreatedBy, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan secret: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *SecretsStore) GetOne(ctx context.Context, envID, key string) (*SecretRow, error) {
	var r SecretRow
	err := s.db.QueryRowContext(ctx,
		`SELECT id, environment_id, key, encrypted_value, nonce, version, created_by, created_at, updated_at
		 FROM secrets WHERE environment_id = $1 AND key = $2`, envID, key).
		Scan(&r.ID, &r.EnvironmentID, &r.Key, &r.EncryptedValue, &r.Nonce, &r.Version,
			&r.CreatedBy, &r.CreatedAt, &r.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get secret: %w", err)
	}
	return &r, nil
}

func (s *SecretsStore) Upsert(ctx context.Context, envID, key string, encVal, nonce []byte, userID string) (int, error) {
	var createdBy any
	if userID != "" {
		createdBy = userID
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var existingID string
	var existingVersion int
	var existingValue, existingNonce []byte
	var existingCreatedBy sql.NullString
	err = tx.QueryRowContext(ctx,
		`SELECT id, version, encrypted_value, nonce, created_by FROM secrets
		 WHERE environment_id = $1 AND key = $2 FOR UPDATE`, envID, key).
		Scan(&existingID, &existingVersion, &existingValue, &existingNonce, &existingCreatedBy)
	exists := err == nil
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("lock existing secret: %w", err)
	}

	if exists {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO secret_versions (secret_id, version, encrypted_value, nonce, created_by)
			 VALUES ($1, $2, $3, $4, $5)`,
			existingID, existingVersion, existingValue, existingNonce, existingCreatedBy); err != nil {
			return 0, fmt.Errorf("snapshot version: %w", err)
		}
	}

	var version int
	err = tx.QueryRowContext(ctx,
		`INSERT INTO secrets (environment_id, key, encrypted_value, nonce, version, created_by)
		 VALUES ($1, $2, $3, $4, 1, $5)
		 ON CONFLICT (environment_id, key) DO UPDATE
		 SET encrypted_value = EXCLUDED.encrypted_value,
		     nonce = EXCLUDED.nonce,
		     version = secrets.version + 1,
		     created_by = EXCLUDED.created_by,
		     updated_at = now()
		 RETURNING version`,
		envID, key, encVal, nonce, createdBy).Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("upsert secret: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit upsert: %w", err)
	}
	return version, nil
}

type SecretVersionRow struct {
	Version        int
	EncryptedValue []byte
	Nonce          []byte
	CreatedBy      sql.NullString
	CreatedAt      time.Time
}

func (s *SecretsStore) ListVersions(ctx context.Context, envID, key string) ([]SecretVersionRow, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT sv.version, sv.encrypted_value, sv.nonce, sv.created_by, sv.created_at
		 FROM secret_versions sv
		 JOIN secrets s ON s.id = sv.secret_id
		 WHERE s.environment_id = $1 AND s.key = $2
		 UNION ALL
		 SELECT version, encrypted_value, nonce, created_by, updated_at
		 FROM secrets
		 WHERE environment_id = $1 AND key = $2
		 ORDER BY version DESC`, envID, key)
	if err != nil {
		return nil, fmt.Errorf("list versions: %w", err)
	}
	defer rows.Close()

	var out []SecretVersionRow
	for rows.Next() {
		var r SecretVersionRow
		if err := rows.Scan(&r.Version, &r.EncryptedValue, &r.Nonce, &r.CreatedBy, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan version: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *SecretsStore) Delete(ctx context.Context, envID, key string) error {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM secrets WHERE environment_id = $1 AND key = $2`, envID, key)
	if err != nil {
		return fmt.Errorf("delete secret: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *SecretsStore) MaxVersion(ctx context.Context, envID string) (int, error) {
	var v sql.NullInt64
	err := s.db.QueryRowContext(ctx,
		`SELECT MAX(version) FROM secrets WHERE environment_id = $1`, envID).Scan(&v)
	if err != nil {
		return 0, fmt.Errorf("max version: %w", err)
	}
	if !v.Valid {
		return 0, nil
	}
	return int(v.Int64), nil
}

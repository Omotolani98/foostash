package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
)

type APIKeyRow struct {
	ID         string
	UserID     string
	OrgID      string
	KeyHash    string
	KeyPrefix  string
	Name       sql.NullString
	Scopes     []string
	LastUsedAt sql.NullTime
	ExpiresAt  sql.NullTime
	CreatedAt  time.Time
}

type APIKeysStore struct {
	db *sql.DB
}

func NewAPIKeysStore(db *sql.DB) *APIKeysStore {
	return &APIKeysStore{db: db}
}

func (s *APIKeysStore) Create(ctx context.Context, userID, orgID, keyHash, keyPrefix, name string, scopes []string) (*APIKeyRow, error) {
	var row APIKeyRow
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO api_keys (user_id, org_id, key_hash, key_prefix, name, scopes)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, user_id, org_id, key_hash, key_prefix, name, scopes, last_used_at, expires_at, created_at`,
		userID, orgID, keyHash, keyPrefix, name, pq.Array(scopes)).
		Scan(&row.ID, &row.UserID, &row.OrgID, &row.KeyHash, &row.KeyPrefix, &row.Name,
			pq.Array(&row.Scopes), &row.LastUsedAt, &row.ExpiresAt, &row.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert api key: %w", err)
	}
	return &row, nil
}

func (s *APIKeysStore) GetByHash(ctx context.Context, keyHash string) (*APIKeyRow, error) {
	var row APIKeyRow
	err := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, org_id, key_hash, key_prefix, name, scopes, last_used_at, expires_at, created_at
		 FROM api_keys WHERE key_hash = $1`, keyHash).
		Scan(&row.ID, &row.UserID, &row.OrgID, &row.KeyHash, &row.KeyPrefix, &row.Name,
			pq.Array(&row.Scopes), &row.LastUsedAt, &row.ExpiresAt, &row.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get api key: %w", err)
	}
	return &row, nil
}

func (s *APIKeysStore) ListByUser(ctx context.Context, userID string) ([]APIKeyRow, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_id, org_id, key_hash, key_prefix, name, scopes, last_used_at, expires_at, created_at
		 FROM api_keys WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	defer rows.Close()

	var out []APIKeyRow
	for rows.Next() {
		var r APIKeyRow
		if err := rows.Scan(&r.ID, &r.UserID, &r.OrgID, &r.KeyHash, &r.KeyPrefix, &r.Name,
			pq.Array(&r.Scopes), &r.LastUsedAt, &r.ExpiresAt, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan api key: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *APIKeysStore) Revoke(ctx context.Context, userID, id string) error {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM api_keys WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("revoke api key: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *APIKeysStore) TouchLastUsed(ctx context.Context, id string) {
	_, _ = s.db.ExecContext(ctx,
		`UPDATE api_keys SET last_used_at = now() WHERE id = $1`, id)
}

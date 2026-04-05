package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type UserRow struct {
	ID           string
	OrgID        string
	Email        string
	PasswordHash sql.NullString
	Role         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type UsersStore struct {
	db *sql.DB
}

func NewUsersStore(db *sql.DB) *UsersStore {
	return &UsersStore{db: db}
}

func (s *UsersStore) Create(ctx context.Context, orgID, email, passwordHash, role string) (*UserRow, error) {
	var row UserRow
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO users (org_id, email, password, role) VALUES ($1, $2, $3, $4)
		 RETURNING id, org_id, email, password, role, created_at, updated_at`,
		orgID, strings.ToLower(email), passwordHash, role).
		Scan(&row.ID, &row.OrgID, &row.Email, &row.PasswordHash, &row.Role, &row.CreatedAt, &row.UpdatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return nil, ErrConflict
		}
		return nil, fmt.Errorf("insert user: %w", err)
	}
	return &row, nil
}

func (s *UsersStore) GetByEmail(ctx context.Context, email string) (*UserRow, error) {
	var row UserRow
	err := s.db.QueryRowContext(ctx,
		`SELECT id, org_id, email, password, role, created_at, updated_at
		 FROM users WHERE email = $1 LIMIT 1`, strings.ToLower(email)).
		Scan(&row.ID, &row.OrgID, &row.Email, &row.PasswordHash, &row.Role, &row.CreatedAt, &row.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return &row, nil
}

func (s *UsersStore) GetByID(ctx context.Context, id string) (*UserRow, error) {
	var row UserRow
	err := s.db.QueryRowContext(ctx,
		`SELECT id, org_id, email, password, role, created_at, updated_at
		 FROM users WHERE id = $1`, id).
		Scan(&row.ID, &row.OrgID, &row.Email, &row.PasswordHash, &row.Role, &row.CreatedAt, &row.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return &row, nil
}

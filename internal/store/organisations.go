package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type OrganizationRow struct {
	ID        string
	Name      string
	Plan      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type OrganizationsStore struct {
	db *sql.DB
}

func NewOrganizationsStore(db *sql.DB) *OrganizationsStore {
	return &OrganizationsStore{db: db}
}

func (s *OrganizationsStore) Create(ctx context.Context, name string) (*OrganizationRow, error) {
	var row OrganizationRow
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO organizations (name) VALUES ($1)
		 RETURNING id, name, plan, created_at, updated_at`, name).
		Scan(&row.ID, &row.Name, &row.Plan, &row.CreatedAt, &row.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert organization: %w", err)
	}
	return &row, nil
}

func (s *OrganizationsStore) GetByID(ctx context.Context, id string) (*OrganizationRow, error) {
	var row OrganizationRow
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, plan, created_at, updated_at FROM organizations WHERE id = $1`, id).
		Scan(&row.ID, &row.Name, &row.Plan, &row.CreatedAt, &row.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get organization: %w", err)
	}
	return &row, nil
}

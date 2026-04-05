package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type ProjectRow struct {
	ID        string
	OrgID     string
	Slug      string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ProjectsStore struct {
	db *sql.DB
}

func NewProjectsStore(db *sql.DB) *ProjectsStore {
	return &ProjectsStore{db: db}
}

func (s *ProjectsStore) Create(ctx context.Context, orgID, slug, name string) (*ProjectRow, error) {
	var row ProjectRow
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO projects (org_id, slug, name) VALUES ($1, $2, $3)
		 RETURNING id, org_id, slug, name, created_at, updated_at`,
		orgID, slug, name).
		Scan(&row.ID, &row.OrgID, &row.Slug, &row.Name, &row.CreatedAt, &row.UpdatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return nil, ErrConflict
		}
		return nil, fmt.Errorf("insert project: %w", err)
	}
	return &row, nil
}

func (s *ProjectsStore) GetBySlug(ctx context.Context, orgID, slug string) (*ProjectRow, error) {
	var row ProjectRow
	err := s.db.QueryRowContext(ctx,
		`SELECT id, org_id, slug, name, created_at, updated_at
		 FROM projects WHERE org_id = $1 AND slug = $2`, orgID, slug).
		Scan(&row.ID, &row.OrgID, &row.Slug, &row.Name, &row.CreatedAt, &row.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}
	return &row, nil
}

func (s *ProjectsStore) ListByOrg(ctx context.Context, orgID string) ([]ProjectRow, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, org_id, slug, name, created_at, updated_at
		 FROM projects WHERE org_id = $1 ORDER BY slug`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	var out []ProjectRow
	for rows.Next() {
		var r ProjectRow
		if err := rows.Scan(&r.ID, &r.OrgID, &r.Slug, &r.Name, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *ProjectsStore) CountByOrg(ctx context.Context, orgID string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM projects WHERE org_id = $1`, orgID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count projects: %w", err)
	}
	return n, nil
}

func (s *ProjectsStore) Delete(ctx context.Context, orgID, slug string) error {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM projects WHERE org_id = $1 AND slug = $2`, orgID, slug)
	if err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

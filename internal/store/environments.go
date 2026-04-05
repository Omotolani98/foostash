package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type EnvironmentRow struct {
	ID        string
	ProjectID string
	Slug      string
	CreatedAt time.Time
}

type EnvironmentsStore struct {
	db *sql.DB
}

func NewEnvironmentsStore(db *sql.DB) *EnvironmentsStore {
	return &EnvironmentsStore{db: db}
}

func (s *EnvironmentsStore) Create(ctx context.Context, projectID, slug string) (*EnvironmentRow, error) {
	var row EnvironmentRow
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO environments (project_id, slug) VALUES ($1, $2)
		 RETURNING id, project_id, slug, created_at`, projectID, slug).
		Scan(&row.ID, &row.ProjectID, &row.Slug, &row.CreatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return nil, ErrConflict
		}
		return nil, fmt.Errorf("insert environment: %w", err)
	}
	return &row, nil
}

func (s *EnvironmentsStore) GetByProjectAndSlug(ctx context.Context, orgID, projectSlug, envSlug string) (*EnvironmentRow, error) {
	var row EnvironmentRow
	err := s.db.QueryRowContext(ctx,
		`SELECT e.id, e.project_id, e.slug, e.created_at
		 FROM environments e
		 JOIN projects p ON p.id = e.project_id
		 WHERE p.org_id = $1 AND p.slug = $2 AND e.slug = $3`,
		orgID, projectSlug, envSlug).
		Scan(&row.ID, &row.ProjectID, &row.Slug, &row.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get environment: %w", err)
	}
	return &row, nil
}

func (s *EnvironmentsStore) ListByProject(ctx context.Context, projectID string) ([]EnvironmentRow, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, slug, created_at FROM environments WHERE project_id = $1 ORDER BY slug`, projectID)
	if err != nil {
		return nil, fmt.Errorf("list environments: %w", err)
	}
	defer rows.Close()

	var out []EnvironmentRow
	for rows.Next() {
		var r EnvironmentRow
		if err := rows.Scan(&r.ID, &r.ProjectID, &r.Slug, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan environment: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *EnvironmentsStore) Delete(ctx context.Context, projectID, slug string) error {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM environments WHERE project_id = $1 AND slug = $2`, projectID, slug)
	if err != nil {
		return fmt.Errorf("delete environment: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

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

type Project struct {
	ID        uuid.UUID
	OrgID     uuid.UUID
	Slug      string
	Name      string
	CreatedBy *uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ProjectListItem is the row shape returned by List, including a denormalized
// environment count computed in the same query.
type ProjectListItem struct {
	Project
	EnvCount int
}

type ProjectRepo struct {
	pool *pgxpool.Pool
}

func (r *ProjectRepo) Insert(ctx context.Context, q Querier, orgID uuid.UUID, slug, name string, createdBy uuid.UUID) (*Project, error) {
	const sql = `
		INSERT INTO projects (org_id, slug, name, created_by)
		VALUES ($1, $2, $3, $4)
		RETURNING id, org_id, slug, name, created_by, created_at, updated_at`
	var p Project
	err := q.QueryRow(ctx, sql, orgID, slug, name, createdBy).Scan(
		&p.ID, &p.OrgID, &p.Slug, &p.Name, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert project: %w", err)
	}
	return &p, nil
}

func (r *ProjectRepo) GetBySlug(ctx context.Context, orgID uuid.UUID, slug string) (*Project, error) {
	const sql = `
		SELECT id, org_id, slug, name, created_by, created_at, updated_at
		FROM projects WHERE org_id = $1 AND slug = $2`
	var p Project
	err := r.pool.QueryRow(ctx, sql, orgID, slug).Scan(
		&p.ID, &p.OrgID, &p.Slug, &p.Name, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}
	return &p, nil
}

func (r *ProjectRepo) List(ctx context.Context, orgID uuid.UUID) ([]ProjectListItem, error) {
	const sql = `
		SELECT p.id, p.org_id, p.slug, p.name, p.created_by, p.created_at, p.updated_at,
		       COALESCE(COUNT(e.id), 0) AS env_count
		FROM projects p
		LEFT JOIN environments e ON e.project_id = p.id
		WHERE p.org_id = $1
		GROUP BY p.id
		ORDER BY p.slug`
	rows, err := r.pool.Query(ctx, sql, orgID)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()
	var out []ProjectListItem
	for rows.Next() {
		var item ProjectListItem
		if err := rows.Scan(
			&item.ID, &item.OrgID, &item.Slug, &item.Name, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt,
			&item.EnvCount,
		); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *ProjectRepo) Delete(ctx context.Context, orgID uuid.UUID, slug string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM projects WHERE org_id = $1 AND slug = $2`, orgID, slug)
	if err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

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

type Environment struct {
	ID        uuid.UUID
	ProjectID uuid.UUID
	Slug      string
	CreatedAt time.Time
}

type EnvironmentRepo struct {
	pool *pgxpool.Pool
}

func (r *EnvironmentRepo) Insert(ctx context.Context, q Querier, projectID uuid.UUID, slug string) (*Environment, error) {
	const sql = `
		INSERT INTO environments (project_id, slug)
		VALUES ($1, $2)
		RETURNING id, project_id, slug, created_at`
	var e Environment
	err := q.QueryRow(ctx, sql, projectID, slug).Scan(&e.ID, &e.ProjectID, &e.Slug, &e.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert environment: %w", err)
	}
	return &e, nil
}

func (r *EnvironmentRepo) GetBySlug(ctx context.Context, q Querier, projectID uuid.UUID, slug string) (*Environment, error) {
	const sql = `
		SELECT id, project_id, slug, created_at
		FROM environments WHERE project_id = $1 AND slug = $2`
	var e Environment
	err := q.QueryRow(ctx, sql, projectID, slug).Scan(&e.ID, &e.ProjectID, &e.Slug, &e.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get environment: %w", err)
	}
	return &e, nil
}

func (r *EnvironmentRepo) ListByProject(ctx context.Context, projectID uuid.UUID) ([]Environment, error) {
	const sql = `
		SELECT id, project_id, slug, created_at
		FROM environments WHERE project_id = $1 ORDER BY slug`
	rows, err := r.pool.Query(ctx, sql, projectID)
	if err != nil {
		return nil, fmt.Errorf("list environments: %w", err)
	}
	defer rows.Close()
	var out []Environment
	for rows.Next() {
		var e Environment
		if err := rows.Scan(&e.ID, &e.ProjectID, &e.Slug, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan environment: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *EnvironmentRepo) Delete(ctx context.Context, projectID uuid.UUID, slug string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM environments WHERE project_id = $1 AND slug = $2`, projectID, slug)
	if err != nil {
		return fmt.Errorf("delete environment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Pool exposes the underlying pool for callers that need direct access.
func (r *EnvironmentRepo) Pool() *pgxpool.Pool { return r.pool }

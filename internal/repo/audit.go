package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditEntry struct {
	ID           uuid.UUID
	OrgID        uuid.UUID
	UserID       *uuid.UUID
	Action       string
	ResourceType *string
	ResourceID   *string
	Status       int
	RequestIP    *string
	Metadata     []byte // raw JSON
	CreatedAt    time.Time
}

// AuditInsert is the payload for inserting a new row.
type AuditInsert struct {
	OrgID        uuid.UUID
	UserID       *uuid.UUID
	Action       string
	ResourceType string
	ResourceID   string
	Status       int
	RequestIP    string
	Metadata     []byte
}

type AuditRepo struct {
	pool *pgxpool.Pool
}

func (r *AuditRepo) Insert(ctx context.Context, in AuditInsert) error {
	const sql = `
		INSERT INTO audit_log (org_id, user_id, action, resource_type, resource_id, status, request_ip, metadata)
		VALUES ($1, $2, $3, NULLIF($4,''), NULLIF($5,''), $6, NULLIF($7,'')::inet, $8)`
	_, err := r.pool.Exec(ctx, sql,
		in.OrgID, in.UserID, in.Action, in.ResourceType, in.ResourceID,
		in.Status, in.RequestIP, nullableBytes(in.Metadata),
	)
	if err != nil {
		return fmt.Errorf("insert audit: %w", err)
	}
	return nil
}

// AuditFilter constrains a query.
type AuditFilter struct {
	OrgID  uuid.UUID
	UserID *uuid.UUID
	Action string
	Since  *time.Time
	Limit  int
}

func (r *AuditRepo) Query(ctx context.Context, f AuditFilter) ([]AuditEntry, error) {
	sql := `
		SELECT id, org_id, user_id, action, resource_type, resource_id, status,
		       host(request_ip), metadata, created_at
		FROM audit_log
		WHERE org_id = $1`
	args := []any{f.OrgID}
	n := 2
	if f.UserID != nil {
		sql += fmt.Sprintf(" AND user_id = $%d", n)
		args = append(args, *f.UserID)
		n++
	}
	if f.Action != "" {
		sql += fmt.Sprintf(" AND action = $%d", n)
		args = append(args, f.Action)
		n++
	}
	if f.Since != nil {
		sql += fmt.Sprintf(" AND created_at >= $%d", n)
		args = append(args, *f.Since)
		n++
	}
	sql += " ORDER BY created_at DESC"
	limit := f.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	sql += fmt.Sprintf(" LIMIT $%d", n)
	args = append(args, limit)

	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query audit: %w", err)
	}
	defer rows.Close()
	var out []AuditEntry
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ID, &e.OrgID, &e.UserID, &e.Action, &e.ResourceType, &e.ResourceID,
			&e.Status, &e.RequestIP, &e.Metadata, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan audit: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func nullableBytes(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return b
}

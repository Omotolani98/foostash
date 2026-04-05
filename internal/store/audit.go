package store

import (
	"context"
	"database/sql"
	"fmt"
)

type AuditEntry struct {
	OrgID           string
	UserID          string
	Action          string
	ProjectSlug     string
	EnvironmentSlug string
	SecretKey       string
	IPAddress       string
	UserAgent       string
}

type AuditStore struct {
	db *sql.DB
}

func NewAuditStore(db *sql.DB) *AuditStore {
	return &AuditStore{db: db}
}

func (s *AuditStore) Insert(ctx context.Context, e AuditEntry) error {
	var userID any
	if e.UserID != "" {
		userID = e.UserID
	}
	var ipAddr any
	if e.IPAddress != "" {
		ipAddr = e.IPAddress
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO audit_logs (org_id, user_id, action, project_slug, environment_slug, secret_key, ip_address, user_agent)
		 VALUES ($1, $2, $3, NULLIF($4,''), NULLIF($5,''), NULLIF($6,''), $7, NULLIF($8,''))`,
		e.OrgID, userID, e.Action, e.ProjectSlug, e.EnvironmentSlug, e.SecretKey, ipAddr, e.UserAgent)
	if err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}

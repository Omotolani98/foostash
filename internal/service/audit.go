package service

import (
	"context"
	"time"

	"github.com/Omotolani98/foostash/internal/repo"
	"github.com/google/uuid"
)

type AuditEntryView struct {
	ID           uuid.UUID  `json:"id"`
	UserID       *uuid.UUID `json:"user_id,omitempty"`
	Action       string     `json:"action"`
	ResourceType *string    `json:"resource_type,omitempty"`
	ResourceID   *string    `json:"resource_id,omitempty"`
	Status       int        `json:"status"`
	RequestIP    *string    `json:"request_ip,omitempty"`
	Metadata     []byte     `json:"metadata,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

type Audit struct {
	repos *repo.Repos
}

func NewAudit(r *repo.Repos) *Audit { return &Audit{repos: r} }

// Record writes an audit entry. Non-fatal: errors are swallowed after logging
// at the caller. Keeps the hot request path unaffected by audit outages.
func (a *Audit) Record(ctx context.Context, in repo.AuditInsert) error {
	return a.repos.Audit.Insert(ctx, in)
}

type AuditQuery struct {
	UserID *uuid.UUID
	Action string
	Since  *time.Time
	Limit  int
}

func (a *Audit) Query(ctx context.Context, caller *AuthContext, q AuditQuery) ([]AuditEntryView, error) {
	if !isAdmin(caller) {
		return nil, ErrForbidden
	}
	rows, err := a.repos.Audit.Query(ctx, repo.AuditFilter{
		OrgID:  caller.OrgID,
		UserID: q.UserID,
		Action: q.Action,
		Since:  q.Since,
		Limit:  q.Limit,
	})
	if err != nil {
		return nil, err
	}
	out := make([]AuditEntryView, 0, len(rows))
	for _, r := range rows {
		out = append(out, AuditEntryView{
			ID:           r.ID,
			UserID:       r.UserID,
			Action:       r.Action,
			ResourceType: r.ResourceType,
			ResourceID:   r.ResourceID,
			Status:       r.Status,
			RequestIP:    r.RequestIP,
			Metadata:     r.Metadata,
			CreatedAt:    r.CreatedAt,
		})
	}
	return out, nil
}

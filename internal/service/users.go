package service

import (
	"context"
	"errors"
	"time"

	"github.com/Omotolani98/foostash/internal/repo"
	"github.com/google/uuid"
)

// UserView is the admin-facing projection of a user.
type UserView struct {
	ID        uuid.UUID  `json:"id"`
	Email     string     `json:"email"`
	Role      string     `json:"role"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type Users struct {
	repos *repo.Repos
}

func NewUsers(r *repo.Repos) *Users {
	return &Users{repos: r}
}

// List returns all users in the caller's org. Admin-only.
func (u *Users) List(ctx context.Context, caller *AuthContext) ([]UserView, error) {
	if !isAdmin(caller) {
		return nil, ErrForbidden
	}
	rows, err := u.repos.Users.ListByOrg(ctx, caller.OrgID)
	if err != nil {
		return nil, err
	}
	out := make([]UserView, 0, len(rows))
	for _, r := range rows {
		out = append(out, toUserView(r))
	}
	return out, nil
}

// ChangeRole updates the role of a user in the caller's org. Admin-only.
// Callers cannot demote themselves (protects against accidental org lockout).
func (u *Users) ChangeRole(ctx context.Context, caller *AuthContext, targetID uuid.UUID, role string) (*UserView, error) {
	if !isAdmin(caller) {
		return nil, ErrForbidden
	}
	if role != "admin" && role != "developer" {
		return nil, ErrInvalidArgument
	}
	target, err := u.loadInOrg(ctx, caller.OrgID, targetID)
	if err != nil {
		return nil, err
	}
	if target.ID == caller.UserID && role != "admin" {
		return nil, ErrForbidden
	}
	if err := u.repos.Users.SetRole(ctx, targetID, role); err != nil {
		return nil, err
	}
	refreshed, err := u.repos.Users.GetByID(ctx, targetID)
	if err != nil {
		return nil, err
	}
	v := toUserView(*refreshed)
	return &v, nil
}

// Revoke marks a user as revoked. Admin-only. Callers cannot revoke themselves.
func (u *Users) Revoke(ctx context.Context, caller *AuthContext, targetID uuid.UUID) error {
	if !isAdmin(caller) {
		return ErrForbidden
	}
	if targetID == caller.UserID {
		return ErrForbidden
	}
	if _, err := u.loadInOrg(ctx, caller.OrgID, targetID); err != nil {
		return err
	}
	return u.repos.Users.Revoke(ctx, targetID)
}

func (u *Users) loadInOrg(ctx context.Context, orgID, id uuid.UUID) (*repo.User, error) {
	user, err := u.repos.Users.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	if user.OrgID != orgID {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func toUserView(u repo.User) UserView {
	return UserView{
		ID:        u.ID,
		Email:     u.Email,
		Role:      u.Role,
		RevokedAt: u.RevokedAt,
		CreatedAt: u.CreatedAt,
	}
}

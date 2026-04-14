package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Omotolani98/foostash/internal/repo"
	"github.com/Omotolani98/foostash/internal/sshauth"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	InviteTTL      = 7 * 24 * time.Hour
	InviteTokenLen = 24 // bytes → 32 url-safe chars
	invitePrefix   = "fst_inv_"
)

type Invites struct {
	repos *repo.Repos
}

func NewInvites(r *repo.Repos) *Invites {
	return &Invites{repos: r}
}

type CreateInviteInput struct {
	Email string
	Role  string
}

type CreateInviteOutput struct {
	Token     string
	ExpiresAt time.Time
}

// Create generates a new invite. The caller must already be verified as admin.
func (s *Invites) Create(ctx context.Context, caller *AuthContext, in CreateInviteInput) (*CreateInviteOutput, error) {
	if caller == nil {
		return nil, ErrForbidden
	}
	if caller.Role != "admin" {
		return nil, ErrForbidden
	}
	email := strings.TrimSpace(strings.ToLower(in.Email))
	if email == "" {
		return nil, ErrInvalidArgument
	}
	role := in.Role
	if role == "" {
		role = "developer"
	}
	if role != "admin" && role != "developer" {
		return nil, ErrInvalidArgument
	}

	token, err := newInviteToken()
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().UTC().Add(InviteTTL)

	inv, err := s.repos.Invites.Insert(ctx, s.repos.Pool, caller.OrgID, email, role, token, expiresAt, caller.UserID)
	if err != nil {
		return nil, err
	}
	return &CreateInviteOutput{Token: inv.Token, ExpiresAt: inv.ExpiresAt}, nil
}

type ConsumeInviteInput struct {
	Token     string
	PublicKey string
}

type ConsumeInviteOutput struct {
	UserID uuid.UUID
	OrgID  uuid.UUID
	Email  string
	Role   string
}

// Consume redeems an invite and creates the developer user + ssh_key.
func (s *Invites) Consume(ctx context.Context, in ConsumeInviteInput) (*ConsumeInviteOutput, error) {
	if in.Token == "" || in.PublicKey == "" {
		return nil, ErrInvalidArgument
	}
	pub, err := sshauth.ParseAuthorizedKey(in.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidArgument, err)
	}
	fingerprint := sshauth.Fingerprint(pub)

	var out ConsumeInviteOutput
	err = repo.InTx(ctx, s.repos.Pool, func(tx pgx.Tx) error {
		inv, err := s.repos.Invites.GetByToken(ctx, tx, in.Token)
		if err != nil {
			if errors.Is(err, repo.ErrNotFound) {
				return ErrInviteNotFound
			}
			return err
		}
		if inv.UsedAt != nil {
			return ErrInviteAlreadyUsed
		}
		if time.Now().UTC().After(inv.ExpiresAt) {
			return ErrInviteExpired
		}
		user, err := s.repos.Users.Insert(ctx, tx, inv.OrgID, inv.Email, inv.Role)
		if err != nil {
			return err
		}
		if _, err := s.repos.SSHKeys.Insert(ctx, tx, user.ID, in.PublicKey, fingerprint); err != nil {
			return err
		}
		if err := s.repos.Invites.MarkUsed(ctx, tx, inv.ID); err != nil {
			return err
		}
		out = ConsumeInviteOutput{
			UserID: user.ID,
			OrgID:  inv.OrgID,
			Email:  inv.Email,
			Role:   inv.Role,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func newInviteToken() (string, error) {
	b := make([]byte, InviteTokenLen)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("random: %w", err)
	}
	return invitePrefix + base64.RawURLEncoding.EncodeToString(b), nil
}

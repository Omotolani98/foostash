package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/Omotolani98/foostash/internal/repo"
	"github.com/Omotolani98/foostash/internal/sshauth"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/ssh"
)

// AuthContext is the authenticated caller, resolved from the presented SSH key.
type AuthContext struct {
	UserID uuid.UUID
	OrgID  uuid.UUID
	Email  string
	Role   string
}

type Auth struct {
	repos *repo.Repos
}

func NewAuth(r *repo.Repos) *Auth {
	return &Auth{repos: r}
}

// RegisterInput is what the /v1/auth/register handler forwards here.
type RegisterInput struct {
	Email     string
	OrgName   string
	PublicKey string // OpenSSH authorized_keys format
}

type RegisterOutput struct {
	UserID uuid.UUID
	OrgID  uuid.UUID
	Role   string
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

func Slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = slugRe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// Register creates an org + admin user + ssh_key in a single tx.
func (a *Auth) Register(ctx context.Context, in RegisterInput) (*RegisterOutput, error) {
	email := strings.TrimSpace(strings.ToLower(in.Email))
	if email == "" || in.OrgName == "" || in.PublicKey == "" {
		return nil, ErrInvalidArgument
	}
	pub, err := sshauth.ParseAuthorizedKey(in.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidArgument, err)
	}
	fingerprint := sshauth.Fingerprint(pub)

	slug := Slugify(in.OrgName)
	if slug == "" {
		return nil, ErrInvalidArgument
	}

	if existing, err := a.repos.Orgs.GetBySlug(ctx, slug); err == nil && existing != nil {
		return nil, ErrOrgExists
	} else if err != nil && !errors.Is(err, repo.ErrNotFound) {
		return nil, err
	}

	var out RegisterOutput
	err = repo.InTx(ctx, a.repos.Pool, func(tx pgx.Tx) error {
		org, err := a.repos.Orgs.Insert(ctx, tx, in.OrgName, slug)
		if err != nil {
			return err
		}
		user, err := a.repos.Users.Insert(ctx, tx, org.ID, email, "admin")
		if err != nil {
			return err
		}
		if _, err := a.repos.SSHKeys.Insert(ctx, tx, user.ID, in.PublicKey, fingerprint); err != nil {
			return err
		}
		out = RegisterOutput{UserID: user.ID, OrgID: org.ID, Role: user.Role}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// PublicKeyFor returns the parsed SSH public key for a fingerprint.
// Used by middleware to verify the request signature.
func (a *Auth) PublicKeyFor(ctx context.Context, fingerprint string) (ssh.PublicKey, error) {
	owner, err := a.repos.SSHKeys.GetByFingerprint(ctx, fingerprint)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrUnknownKey
		}
		return nil, err
	}
	return sshauth.ParseAuthorizedKey(owner.Key.PublicKey)
}

// ResolveSSHKey is called by auth middleware to turn a fingerprint into an
// AuthContext. It also updates last_used_at best-effort.
func (a *Auth) ResolveSSHKey(ctx context.Context, fingerprint string) (*AuthContext, error) {
	owner, err := a.repos.SSHKeys.GetByFingerprint(ctx, fingerprint)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrUnknownKey
		}
		return nil, err
	}
	_ = a.repos.SSHKeys.TouchLastUsed(ctx, owner.Key.ID)
	return &AuthContext{
		UserID: owner.User.ID,
		OrgID:  owner.OrgID,
		Email:  owner.User.Email,
		Role:   owner.User.Role,
	}, nil
}

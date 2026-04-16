package service

import (
	"context"
	"errors"
	"time"

	"github.com/Omotolani98/foostash/internal/repo"
	"github.com/google/uuid"
)

// VaultView is the server-stored, client-decryptable payload for one vault key.
type VaultView struct {
	Key        string     `json:"key"`
	Ciphertext []byte     `json:"ciphertext"`
	Nonce      []byte     `json:"nonce"`
	Version    int        `json:"version"`
	UpdatedBy  *uuid.UUID `json:"updated_by,omitempty"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// VaultHistoryView mirrors VaultView for a historical row.
type VaultHistoryView struct {
	Version    int        `json:"version"`
	Ciphertext []byte     `json:"ciphertext"`
	Nonce      []byte     `json:"nonce"`
	SetBy      *uuid.UUID `json:"set_by,omitempty"`
	SetAt      time.Time  `json:"set_at"`
}

type Vault struct {
	repos *repo.Repos
}

func NewVault(r *repo.Repos) *Vault {
	return &Vault{repos: r}
}

// Set stores a new version of an org-wide vault secret. Admin-only.
// Callers submit client-side ciphertext; the server never sees plaintext.
func (s *Vault) Set(ctx context.Context, actx *AuthContext, key string, ciphertext, nonce []byte) (*VaultView, error) {
	if !isAdmin(actx) {
		return nil, ErrForbidden
	}
	if key == "" || len(ciphertext) == 0 || len(nonce) == 0 {
		return nil, ErrInvalidArgument
	}
	version, err := s.repos.Vault.Upsert(ctx, actx.OrgID, key, ciphertext, nonce, actx.UserID)
	if err != nil {
		return nil, err
	}
	return &VaultView{
		Key:        key,
		Ciphertext: ciphertext,
		Nonce:      nonce,
		Version:    version,
		UpdatedBy:  &actx.UserID,
		UpdatedAt:  time.Now().UTC(),
	}, nil
}

// Get returns the live vault secret for a key. Readable by any org member.
func (s *Vault) Get(ctx context.Context, actx *AuthContext, key string) (*VaultView, error) {
	row, err := s.repos.Vault.Get(ctx, actx.OrgID, key)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrVaultKeyNotFound
		}
		return nil, err
	}
	return toVaultView(*row), nil
}

// List returns all live vault secrets for the caller's org.
func (s *Vault) List(ctx context.Context, actx *AuthContext) ([]VaultView, error) {
	rows, err := s.repos.Vault.List(ctx, actx.OrgID)
	if err != nil {
		return nil, err
	}
	out := make([]VaultView, 0, len(rows))
	for _, r := range rows {
		out = append(out, *toVaultView(r))
	}
	return out, nil
}

// Delete soft-deletes a live vault secret. Admin-only.
func (s *Vault) Delete(ctx context.Context, actx *AuthContext, key string) error {
	if !isAdmin(actx) {
		return ErrForbidden
	}
	if err := s.repos.Vault.Delete(ctx, actx.OrgID, key); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return ErrVaultKeyNotFound
		}
		return err
	}
	return nil
}

// History returns the full version history for a vault key, newest first.
func (s *Vault) History(ctx context.Context, actx *AuthContext, key string) ([]VaultHistoryView, error) {
	rows, err := s.repos.Vault.History(ctx, actx.OrgID, key)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrVaultKeyNotFound
	}
	out := make([]VaultHistoryView, 0, len(rows))
	for _, r := range rows {
		out = append(out, VaultHistoryView{
			Version:    r.Version,
			Ciphertext: r.Ciphertext,
			Nonce:      r.Nonce,
			SetBy:      r.SetBy,
			SetAt:      r.SetAt,
		})
	}
	return out, nil
}

// Rollback restores a prior version by writing its ciphertext as a new live
// version. Admin-only. Returns the new (bumped) live version.
func (s *Vault) Rollback(ctx context.Context, actx *AuthContext, key string, targetVersion int) (*VaultView, error) {
	if !isAdmin(actx) {
		return nil, ErrForbidden
	}
	if targetVersion <= 0 {
		return nil, ErrInvalidArgument
	}
	h, err := s.repos.Vault.GetHistoryVersion(ctx, actx.OrgID, key, targetVersion)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrVaultKeyNotFound
		}
		return nil, err
	}
	version, err := s.repos.Vault.Upsert(ctx, actx.OrgID, key, h.Ciphertext, h.Nonce, actx.UserID)
	if err != nil {
		return nil, err
	}
	return &VaultView{
		Key:        key,
		Ciphertext: h.Ciphertext,
		Nonce:      h.Nonce,
		Version:    version,
		UpdatedBy:  &actx.UserID,
		UpdatedAt:  time.Now().UTC(),
	}, nil
}

func toVaultView(r repo.VaultSecret) *VaultView {
	return &VaultView{
		Key:        r.Key,
		Ciphertext: r.Ciphertext,
		Nonce:      r.Nonce,
		Version:    r.Version,
		UpdatedBy:  r.UpdatedBy,
		UpdatedAt:  r.UpdatedAt,
	}
}

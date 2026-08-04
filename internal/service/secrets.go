package service

import (
	"context"
	"errors"
	"time"

	"github.com/Omotolani98/foostash/internal/repo"
	"github.com/google/uuid"
)

// SecretView is the server-stored, client-decryptable payload for one key.
type SecretView struct {
	Key        string     `json:"key"`
	Ciphertext []byte     `json:"ciphertext"`
	Nonce      []byte     `json:"nonce"`
	Version    int        `json:"version"`
	UpdatedBy  *uuid.UUID `json:"updated_by,omitempty"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// SecretHistoryView mirrors SecretView for a historical row.
type SecretHistoryView struct {
	Version    int        `json:"version"`
	Ciphertext []byte     `json:"ciphertext"`
	Nonce      []byte     `json:"nonce"`
	SetBy      *uuid.UUID `json:"set_by,omitempty"`
	SetAt      time.Time  `json:"set_at"`
}

type Secrets struct {
	repos *repo.Repos
}

const (
	maxBulkSecrets         = 500
	maxSecretCiphertextLen = 64 * 1024
)

// SecretUpsert is one client-encrypted secret payload for a bulk write.
type SecretUpsert struct {
	Key        string
	Ciphertext []byte
	Nonce      []byte
}

func NewSecrets(r *repo.Repos) *Secrets {
	return &Secrets{repos: r}
}

// Set stores a new version of a secret. Callers submit client-side ciphertext;
// the server never sees plaintext. Any authenticated org member may write.
func (s *Secrets) Set(ctx context.Context, actx *AuthContext, projectSlug, envSlug, key string, ciphertext, nonce []byte) (*SecretView, error) {
	if key == "" || len(ciphertext) == 0 || len(nonce) == 0 {
		return nil, ErrInvalidArgument
	}
	env, err := s.resolveEnv(ctx, actx, projectSlug, envSlug)
	if err != nil {
		return nil, err
	}
	version, err := s.repos.Secrets.Upsert(ctx, env.ID, key, ciphertext, nonce, actx.UserID)
	if err != nil {
		return nil, err
	}
	return &SecretView{
		Key:        key,
		Ciphertext: ciphertext,
		Nonce:      nonce,
		Version:    version,
		UpdatedBy:  &actx.UserID,
		UpdatedAt:  time.Now().UTC(),
	}, nil
}

// BulkSet stores multiple client-encrypted secrets atomically. The server still
// only sees ciphertext + nonce; plaintext comparison, if any, happens client-side.
func (s *Secrets) BulkSet(ctx context.Context, actx *AuthContext, projectSlug, envSlug string, items []SecretUpsert) (map[string]int, error) {
	if len(items) == 0 || len(items) > maxBulkSecrets {
		return nil, ErrInvalidArgument
	}
	seen := make(map[string]struct{}, len(items))
	repoItems := make([]repo.SecretUpsertItem, 0, len(items))
	for _, it := range items {
		if it.Key == "" || len(it.Ciphertext) == 0 || len(it.Nonce) == 0 || len(it.Ciphertext) > maxSecretCiphertextLen {
			return nil, ErrInvalidArgument
		}
		if _, ok := seen[it.Key]; ok {
			return nil, ErrInvalidArgument
		}
		seen[it.Key] = struct{}{}
		repoItems = append(repoItems, repo.SecretUpsertItem{
			Key:        it.Key,
			Ciphertext: it.Ciphertext,
			Nonce:      it.Nonce,
		})
	}

	env, err := s.resolveEnv(ctx, actx, projectSlug, envSlug)
	if err != nil {
		return nil, err
	}
	return s.repos.Secrets.BulkUpsert(ctx, env.ID, repoItems, actx.UserID)
}

// Get returns the live secret for a key.
func (s *Secrets) Get(ctx context.Context, actx *AuthContext, projectSlug, envSlug, key string) (*SecretView, error) {
	env, err := s.resolveEnv(ctx, actx, projectSlug, envSlug)
	if err != nil {
		return nil, err
	}
	row, err := s.repos.Secrets.Get(ctx, env.ID, key)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrSecretNotFound
		}
		return nil, err
	}
	return toSecretView(*row), nil
}

// List returns all live secrets for the env.
func (s *Secrets) List(ctx context.Context, actx *AuthContext, projectSlug, envSlug string) ([]SecretView, error) {
	env, err := s.resolveEnv(ctx, actx, projectSlug, envSlug)
	if err != nil {
		return nil, err
	}
	rows, err := s.repos.Secrets.List(ctx, env.ID)
	if err != nil {
		return nil, err
	}
	out := make([]SecretView, 0, len(rows))
	for _, r := range rows {
		out = append(out, *toSecretView(r))
	}
	return out, nil
}

// Delete soft-deletes a live secret.
func (s *Secrets) Delete(ctx context.Context, actx *AuthContext, projectSlug, envSlug, key string) error {
	env, err := s.resolveEnv(ctx, actx, projectSlug, envSlug)
	if err != nil {
		return err
	}
	if err := s.repos.Secrets.Delete(ctx, env.ID, key); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return ErrSecretNotFound
		}
		return err
	}
	return nil
}

// History returns the full version history for a key, newest first.
func (s *Secrets) History(ctx context.Context, actx *AuthContext, projectSlug, envSlug, key string) ([]SecretHistoryView, error) {
	env, err := s.resolveEnv(ctx, actx, projectSlug, envSlug)
	if err != nil {
		return nil, err
	}
	rows, err := s.repos.Secrets.History(ctx, env.ID, key)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrSecretNotFound
	}
	out := make([]SecretHistoryView, 0, len(rows))
	for _, r := range rows {
		out = append(out, SecretHistoryView{
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
// version. Returns the new (bumped) live version.
func (s *Secrets) Rollback(ctx context.Context, actx *AuthContext, projectSlug, envSlug, key string, targetVersion int) (*SecretView, error) {
	if targetVersion <= 0 {
		return nil, ErrInvalidArgument
	}
	env, err := s.resolveEnv(ctx, actx, projectSlug, envSlug)
	if err != nil {
		return nil, err
	}
	h, err := s.repos.Secrets.GetHistoryVersion(ctx, env.ID, key, targetVersion)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrSecretNotFound
		}
		return nil, err
	}
	version, err := s.repos.Secrets.Upsert(ctx, env.ID, key, h.Ciphertext, h.Nonce, actx.UserID)
	if err != nil {
		return nil, err
	}
	return &SecretView{
		Key:        key,
		Ciphertext: h.Ciphertext,
		Nonce:      h.Nonce,
		Version:    version,
		UpdatedBy:  &actx.UserID,
		UpdatedAt:  time.Now().UTC(),
	}, nil
}

// resolveEnv scopes env lookup to the caller's org and returns ErrProjectNotFound
// or ErrEnvNotFound without leaking cross-org existence.
func (s *Secrets) resolveEnv(ctx context.Context, actx *AuthContext, projectSlug, envSlug string) (*repo.Environment, error) {
	proj, err := s.repos.Projects.GetBySlug(ctx, actx.OrgID, projectSlug)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}
	env, err := s.repos.Environments.GetBySlug(ctx, s.repos.Pool, proj.ID, envSlug)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrEnvNotFound
		}
		return nil, err
	}
	return env, nil
}

func toSecretView(r repo.Secret) *SecretView {
	return &SecretView{
		Key:        r.Key,
		Ciphertext: r.Ciphertext,
		Nonce:      r.Nonce,
		Version:    r.Version,
		UpdatedBy:  r.UpdatedBy,
		UpdatedAt:  r.UpdatedAt,
	}
}

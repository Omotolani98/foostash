package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/Omotolani98/foostash/internal/crypto"
	"github.com/Omotolani98/foostash/internal/store"
)

type SecretsService struct {
	secrets *store.SecretsStore
	envs    *store.EnvironmentsStore
	crypto  *crypto.Engine
	audit   *AuditService
}

func NewSecretsService(secrets *store.SecretsStore, envs *store.EnvironmentsStore, engine *crypto.Engine, audit *AuditService) *SecretsService {
	return &SecretsService{secrets: secrets, envs: envs, crypto: engine, audit: audit}
}

type PullResult struct {
	Secrets map[string]string
	Version int
}

func (s *SecretsService) Pull(ctx context.Context, orgID, projectSlug, envSlug string, userID, ip, ua string) (*PullResult, error) {
	env, err := s.resolveEnv(ctx, orgID, projectSlug, envSlug)
	if err != nil {
		return nil, err
	}

	rows, err := s.secrets.GetByEnvironment(ctx, env.ID)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string, len(rows))
	maxVersion := 0
	for _, row := range rows {
		plaintext, err := s.crypto.Decrypt(row.EncryptedValue, row.Nonce)
		if err != nil {
			return nil, fmt.Errorf("decrypt %s: %w", row.Key, err)
		}
		result[row.Key] = string(plaintext)
		if row.Version > maxVersion {
			maxVersion = row.Version
		}
	}

	s.audit.LogAsync(store.AuditEntry{
		OrgID:           orgID,
		UserID:          userID,
		Action:          "secrets.pull",
		ProjectSlug:     projectSlug,
		EnvironmentSlug: envSlug,
		IPAddress:       ip,
		UserAgent:       ua,
	})

	return &PullResult{Secrets: result, Version: maxVersion}, nil
}

func (s *SecretsService) Set(ctx context.Context, orgID, projectSlug, envSlug string, secrets map[string]string, userID, ip, ua string) ([]string, int, error) {
	if len(secrets) == 0 {
		return nil, 0, fmt.Errorf("%w: no secrets provided", ErrValidation)
	}

	env, err := s.resolveEnv(ctx, orgID, projectSlug, envSlug)
	if err != nil {
		return nil, 0, err
	}

	var keys []string
	maxVersion := 0
	for key, value := range secrets {
		if key == "" {
			return nil, 0, fmt.Errorf("%w: empty secret key", ErrValidation)
		}
		ciphertext, nonce, err := s.crypto.Encrypt([]byte(value))
		if err != nil {
			return nil, 0, fmt.Errorf("encrypt %s: %w", key, err)
		}
		v, err := s.secrets.Upsert(ctx, env.ID, key, ciphertext, nonce, userID)
		if err != nil {
			return nil, 0, err
		}
		if v > maxVersion {
			maxVersion = v
		}
		keys = append(keys, key)
	}

	s.audit.LogAsync(store.AuditEntry{
		OrgID:           orgID,
		UserID:          userID,
		Action:          "secrets.set",
		ProjectSlug:     projectSlug,
		EnvironmentSlug: envSlug,
		IPAddress:       ip,
		UserAgent:       ua,
	})

	return keys, maxVersion, nil
}

func (s *SecretsService) Delete(ctx context.Context, orgID, projectSlug, envSlug, key string, userID, ip, ua string) error {
	env, err := s.resolveEnv(ctx, orgID, projectSlug, envSlug)
	if err != nil {
		return err
	}
	if err := s.secrets.Delete(ctx, env.ID, key); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	s.audit.LogAsync(store.AuditEntry{
		OrgID:           orgID,
		UserID:          userID,
		Action:          "secrets.delete",
		ProjectSlug:     projectSlug,
		EnvironmentSlug: envSlug,
		SecretKey:       key,
		IPAddress:       ip,
		UserAgent:       ua,
	})
	return nil
}

func (s *SecretsService) resolveEnv(ctx context.Context, orgID, projectSlug, envSlug string) (*store.EnvironmentRow, error) {
	env, err := s.envs.GetByProjectAndSlug(ctx, orgID, projectSlug, envSlug)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return env, nil
}

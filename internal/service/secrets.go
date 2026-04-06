package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"time"

	"github.com/Omotolani98/foostash/internal/crypto"
	"github.com/Omotolani98/foostash/internal/store"
)

const (
	MaxSecretKeyLen   = 255
	MaxSecretValueLen = 64 * 1024
	MaxSecretsPerSet  = 500
)

var secretKeyRegex = regexp.MustCompile(`^[A-Z][A-Z0-9_]{0,254}$`)

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
	if len(secrets) > MaxSecretsPerSet {
		return nil, 0, fmt.Errorf("%w: too many secrets in one request (max %d)", ErrValidation, MaxSecretsPerSet)
	}
	for key, value := range secrets {
		if !secretKeyRegex.MatchString(key) {
			return nil, 0, fmt.Errorf("%w: invalid key %q (must match [A-Z][A-Z0-9_]*)", ErrValidation, key)
		}
		if len(value) > MaxSecretValueLen {
			return nil, 0, fmt.Errorf("%w: value for %q exceeds %d bytes", ErrValidation, key, MaxSecretValueLen)
		}
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

type VersionEntry struct {
	Version   int       `json:"version"`
	Value     string    `json:"value"`
	CreatedBy string    `json:"created_by,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *SecretsService) ListVersions(ctx context.Context, orgID, projectSlug, envSlug, key string) ([]VersionEntry, error) {
	env, err := s.resolveEnv(ctx, orgID, projectSlug, envSlug)
	if err != nil {
		return nil, err
	}
	rows, err := s.secrets.ListVersions(ctx, env.ID, key)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	out := make([]VersionEntry, 0, len(rows))
	for _, r := range rows {
		plaintext, err := s.crypto.Decrypt(r.EncryptedValue, r.Nonce)
		if err != nil {
			return nil, fmt.Errorf("decrypt version %d: %w", r.Version, err)
		}
		entry := VersionEntry{Version: r.Version, Value: string(plaintext), CreatedAt: r.CreatedAt}
		if r.CreatedBy.Valid {
			entry.CreatedBy = r.CreatedBy.String
		}
		out = append(out, entry)
	}
	return out, nil
}

type DiffResult struct {
	OnlyLeft        []string `json:"only_left"`
	OnlyRight       []string `json:"only_right"`
	DifferentValues []string `json:"different_values"`
	Identical       []string `json:"identical"`
}

func (s *SecretsService) Diff(ctx context.Context, orgID, projectSlug, leftEnv, rightEnv string) (*DiffResult, error) {
	left, err := s.resolveEnv(ctx, orgID, projectSlug, leftEnv)
	if err != nil {
		return nil, err
	}
	right, err := s.resolveEnv(ctx, orgID, projectSlug, rightEnv)
	if err != nil {
		return nil, err
	}

	leftRows, err := s.secrets.GetByEnvironment(ctx, left.ID)
	if err != nil {
		return nil, err
	}
	rightRows, err := s.secrets.GetByEnvironment(ctx, right.ID)
	if err != nil {
		return nil, err
	}

	leftMap := make(map[string]string, len(leftRows))
	for _, r := range leftRows {
		pt, err := s.crypto.Decrypt(r.EncryptedValue, r.Nonce)
		if err != nil {
			return nil, fmt.Errorf("decrypt left %s: %w", r.Key, err)
		}
		leftMap[r.Key] = string(pt)
	}
	rightMap := make(map[string]string, len(rightRows))
	for _, r := range rightRows {
		pt, err := s.crypto.Decrypt(r.EncryptedValue, r.Nonce)
		if err != nil {
			return nil, fmt.Errorf("decrypt right %s: %w", r.Key, err)
		}
		rightMap[r.Key] = string(pt)
	}

	result := &DiffResult{
		OnlyLeft:        []string{},
		OnlyRight:       []string{},
		DifferentValues: []string{},
		Identical:       []string{},
	}
	for k, lv := range leftMap {
		if rv, ok := rightMap[k]; ok {
			if rv == lv {
				result.Identical = append(result.Identical, k)
			} else {
				result.DifferentValues = append(result.DifferentValues, k)
			}
		} else {
			result.OnlyLeft = append(result.OnlyLeft, k)
		}
	}
	for k := range rightMap {
		if _, ok := leftMap[k]; !ok {
			result.OnlyRight = append(result.OnlyRight, k)
		}
	}
	sort.Strings(result.OnlyLeft)
	sort.Strings(result.OnlyRight)
	sort.Strings(result.DifferentValues)
	sort.Strings(result.Identical)
	return result, nil
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

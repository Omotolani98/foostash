package secrets

import (
	"fmt"
	"sort"
	"time"

	"github.com/Omotolani98/foostash/internal/store"
)

const maxHistoryPerKey = 50

type Service struct {
	store *store.Store
}

func NewService(s *store.Store) *Service {
	return &Service{store: s}
}

// Set updates or creates secrets for a project environment.
func (s *Service) Set(project, env string, pairs map[string]string) error {
	path, err := store.EnvPath(project, env)
	if err != nil {
		return fmt.Errorf("resolve env path: %w", err)
	}
	return s.setAtPath(path, pairs)
}

// SetGlobal updates or creates global secrets.
func (s *Service) SetGlobal(pairs map[string]string) error {
	path, err := store.GlobalsPath()
	if err != nil {
		return fmt.Errorf("resolve globals path: %w", err)
	}
	return s.setAtPath(path, pairs)
}

func (s *Service) setAtPath(path string, pairs map[string]string) error {
	sf, err := s.store.Load(path)
	if err != nil {
		return fmt.Errorf("load secrets: %w", err)
	}

	now := time.Now().UTC()
	for key, value := range pairs {
		entry, exists := sf.Secrets[key]
		newVersion := 1
		if exists {
			newVersion = entry.Version + 1
		}

		sf.Secrets[key] = store.SecretEntry{
			Value:     value,
			Version:   newVersion,
			UpdatedAt: now,
		}

		// append history
		sf.History[key] = append(sf.History[key], store.HistoryEntry{
			Version: newVersion,
			Value:   value,
			SetAt:   now,
		})

		// cap history
		if len(sf.History[key]) > maxHistoryPerKey {
			sf.History[key] = sf.History[key][len(sf.History[key])-maxHistoryPerKey:]
		}
	}

	sf.Version++
	sf.UpdatedAt = now

	if err := s.store.Save(path, sf); err != nil {
		return fmt.Errorf("save secrets: %w", err)
	}
	return nil
}

// Get returns a single secret value.
func (s *Service) Get(project, env, key string) (string, error) {
	path, err := store.EnvPath(project, env)
	if err != nil {
		return "", fmt.Errorf("resolve env path: %w", err)
	}

	sf, err := s.store.Load(path)
	if err != nil {
		return "", fmt.Errorf("load secrets: %w", err)
	}

	entry, exists := sf.Secrets[key]
	if !exists {
		return "", fmt.Errorf("secret %q not found in %s/%s", key, project, env)
	}
	return entry.Value, nil
}

// Pull returns all secrets for an environment, optionally merged with globals.
func (s *Service) Pull(project, env string, withGlobals bool) (map[string]string, error) {
	merged := make(map[string]string)

	if withGlobals {
		globalsPath, err := store.GlobalsPath()
		if err != nil {
			return nil, fmt.Errorf("resolve globals path: %w", err)
		}
		gf, err := s.store.Load(globalsPath)
		if err != nil {
			return nil, fmt.Errorf("load globals: %w", err)
		}
		for k, v := range gf.Secrets {
			merged[k] = v.Value
		}
	}

	path, err := store.EnvPath(project, env)
	if err != nil {
		return nil, fmt.Errorf("resolve env path: %w", err)
	}
	sf, err := s.store.Load(path)
	if err != nil {
		return nil, fmt.Errorf("load secrets: %w", err)
	}
	for k, v := range sf.Secrets {
		merged[k] = v.Value // project overrides globals
	}

	return merged, nil
}

// ListKeys returns sorted keys for an environment.
func (s *Service) ListKeys(project, env string) ([]string, error) {
	path, err := store.EnvPath(project, env)
	if err != nil {
		return nil, fmt.Errorf("resolve env path: %w", err)
	}

	sf, err := s.store.Load(path)
	if err != nil {
		return nil, fmt.Errorf("load secrets: %w", err)
	}

	keys := make([]string, 0, len(sf.Secrets))
	for k := range sf.Secrets {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys, nil
}

// GetHistory returns version history for a specific key.
func (s *Service) GetHistory(project, env, key string) ([]store.HistoryEntry, error) {
	path, err := store.EnvPath(project, env)
	if err != nil {
		return nil, fmt.Errorf("resolve env path: %w", err)
	}

	sf, err := s.store.Load(path)
	if err != nil {
		return nil, fmt.Errorf("load secrets: %w", err)
	}

	history, exists := sf.History[key]
	if !exists {
		return nil, fmt.Errorf("no history for %q in %s/%s", key, project, env)
	}
	return history, nil
}

// Rollback sets a key back to a specific version from its history.
func (s *Service) Rollback(project, env, key string, targetVersion int) error {
	path, err := store.EnvPath(project, env)
	if err != nil {
		return fmt.Errorf("resolve env path: %w", err)
	}

	sf, err := s.store.Load(path)
	if err != nil {
		return fmt.Errorf("load secrets: %w", err)
	}

	history, exists := sf.History[key]
	if !exists {
		return fmt.Errorf("no history for %q in %s/%s", key, project, env)
	}

	var targetValue string
	found := false
	for _, h := range history {
		if h.Version == targetVersion {
			targetValue = h.Value
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("version %d not found for %q", targetVersion, key)
	}

	entry := sf.Secrets[key]
	newVersion := entry.Version + 1
	now := time.Now().UTC()

	sf.Secrets[key] = store.SecretEntry{
		Value:     targetValue,
		Version:   newVersion,
		UpdatedAt: now,
	}
	sf.History[key] = append(sf.History[key], store.HistoryEntry{
		Version: newVersion,
		Value:   targetValue,
		SetAt:   now,
	})
	if len(sf.History[key]) > maxHistoryPerKey {
		sf.History[key] = sf.History[key][len(sf.History[key])-maxHistoryPerKey:]
	}

	sf.Version++
	sf.UpdatedAt = now

	return s.store.Save(path, sf)
}

// Delete removes a secret key from an environment.
func (s *Service) Delete(project, env, key string) error {
	path, err := store.EnvPath(project, env)
	if err != nil {
		return fmt.Errorf("resolve env path: %w", err)
	}

	sf, err := s.store.Load(path)
	if err != nil {
		return fmt.Errorf("load secrets: %w", err)
	}

	if _, exists := sf.Secrets[key]; !exists {
		return fmt.Errorf("secret %q not found in %s/%s", key, project, env)
	}

	delete(sf.Secrets, key)
	sf.Version++
	sf.UpdatedAt = time.Now().UTC()

	return s.store.Save(path, sf)
}

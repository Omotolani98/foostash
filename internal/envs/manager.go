package envs

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Omotolani98/foostash/internal/store"
)

type Manager struct {
	store *store.Store
}

func NewManager(s *store.Store) *Manager {
	return &Manager{store: s}
}

// List returns all environment names for a project by scanning .enc files.
func (m *Manager) List(project string) ([]string, error) {
	dir, err := store.ProjectDir(project)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read project dir: %w", err)
	}

	var envs []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".enc") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".enc")
		envs = append(envs, name)
	}
	sort.Strings(envs)
	return envs, nil
}

// Create creates a new empty environment.
func (m *Manager) Create(project, env string) error {
	path, err := store.EnvPath(project, env)
	if err != nil {
		return err
	}

	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("environment %q already exists", env)
	}

	sf := store.NewSecretFile()
	return m.store.Save(path, sf)
}

// Clone copies all secrets from one environment to another.
func (m *Manager) Clone(project, src, dest string) error {
	srcPath, err := store.EnvPath(project, src)
	if err != nil {
		return err
	}
	destPath, err := store.EnvPath(project, dest)
	if err != nil {
		return err
	}

	if _, err := os.Stat(destPath); err == nil {
		return fmt.Errorf("destination environment %q already exists", dest)
	}

	// load and re-save (re-encrypts with fresh nonce)
	sf, err := m.store.Load(srcPath)
	if err != nil {
		return fmt.Errorf("load source environment: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0700); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	return m.store.Save(destPath, sf)
}

// Delete removes an environment's .enc file.
func (m *Manager) Delete(project, env string) error {
	path, err := store.EnvPath(project, env)
	if err != nil {
		return err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("environment %q does not exist", env)
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("delete environment: %w", err)
	}
	return nil
}

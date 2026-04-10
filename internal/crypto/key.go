package crypto

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FoostashDir returns the path to ~/.foostash/, creating it with 0700 if needed.
func FoostashDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	dir := filepath.Join(home, ".foostash")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("create foostash dir: %w", err)
	}
	return dir, nil
}

// ProjectsDir returns the path to ~/.foostash/projects/, creating it if needed.
func ProjectsDir() (string, error) {
	base, err := FoostashDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "projects")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("create projects dir: %w", err)
	}
	return dir, nil
}

// LoadOrGenerateMasterKey returns the base64-encoded master key by checking:
// 1. FOOSTASH_MASTER_KEY env var
// 2. ~/.foostash/master.key file
// 3. Auto-generates and writes a new key
func LoadOrGenerateMasterKey() (string, error) {
	if key := os.Getenv("FOOSTASH_MASTER_KEY"); key != "" {
		return strings.TrimSpace(key), nil
	}

	dir, err := FoostashDir()
	if err != nil {
		return "", err
	}
	keyPath := filepath.Join(dir, "master.key")

	data, err := os.ReadFile(keyPath)
	if err == nil {
		return strings.TrimSpace(string(data)), nil
	}
	if !os.IsNotExist(err) {
		return "", fmt.Errorf("read master key: %w", err)
	}

	key, err := GenerateKey()
	if err != nil {
		return "", fmt.Errorf("generate master key: %w", err)
	}
	if err := os.WriteFile(keyPath, []byte(key+"\n"), 0600); err != nil {
		return "", fmt.Errorf("write master key: %w", err)
	}

	return key, nil
}

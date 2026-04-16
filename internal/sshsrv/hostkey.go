package sshsrv

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/keygen"
)

// EnsureHostKey makes sure an ed25519 host key exists at path, generating one
// if needed. It refuses to run if the existing file or its parent directory
// has permissions looser than 0600/0700 respectively.
func EnsureHostKey(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create host key dir: %w", err)
	}
	if di, err := os.Stat(dir); err == nil {
		if di.Mode().Perm()&^0o700 != 0 {
			return fmt.Errorf("host key dir %s has permissions %o, expected 0700 or stricter", dir, di.Mode().Perm())
		}
	}

	fi, err := os.Stat(path)
	if err == nil {
		if fi.Mode().Perm()&^0o600 != 0 {
			return fmt.Errorf("host key %s has permissions %o, expected 0600 or stricter", path, fi.Mode().Perm())
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("stat host key: %w", err)
	}

	if _, err := keygen.New(path, keygen.WithKeyType(keygen.Ed25519), keygen.WithWrite()); err != nil {
		return fmt.Errorf("generate host key: %w", err)
	}
	return nil
}

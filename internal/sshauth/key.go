// Package sshauth implements SSH-key-based request signing and verification
// shared by the CLI client and the server.
package sshauth

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

var ErrNoDefaultKey = errors.New("no SSH private key found in ~/.ssh")

// ResolveDefaultKeyPath returns the first existing key from a prioritized list
// of standard locations under ~/.ssh.
func ResolveDefaultKeyPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate home dir: %w", err)
	}
	candidates := []string{
		filepath.Join(home, ".ssh", "id_ed25519"),
		filepath.Join(home, ".ssh", "id_rsa"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", ErrNoDefaultKey
}

// LoadSigner reads a private key from disk and returns an ssh.Signer.
// If passphrase is non-nil it is used to decrypt the key; otherwise the key
// must be unencrypted or the caller is expected to retry with a passphrase.
func LoadSigner(path string, passphrase []byte) (ssh.Signer, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}
	if len(passphrase) > 0 {
		return ssh.ParsePrivateKeyWithPassphrase(raw, passphrase)
	}
	return ssh.ParsePrivateKey(raw)
}

// IsPassphraseError reports whether an error from LoadSigner indicates that
// a passphrase is required.
func IsPassphraseError(err error) bool {
	var missing *ssh.PassphraseMissingError
	return errors.As(err, &missing)
}

// LoadPublicKey reads a public key file (foo.pub) and parses it.
func LoadPublicKey(path string) (ssh.PublicKey, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read public key: %w", err)
	}
	pub, _, _, _, err := ssh.ParseAuthorizedKey(raw)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}
	return pub, nil
}

// PublicKeyForPrivate derives the public key path for a given private key
// path by appending ".pub".
func PublicKeyForPrivate(privPath string) string {
	return privPath + ".pub"
}

// Fingerprint returns the SHA256 fingerprint of a public key
// (e.g. "SHA256:abc...").
func Fingerprint(pub ssh.PublicKey) string {
	return ssh.FingerprintSHA256(pub)
}

// MarshalAuthorizedKey returns the wire representation of a public key
// suitable for storing in the ssh_keys.public_key column.
func MarshalAuthorizedKey(pub ssh.PublicKey) string {
	return string(ssh.MarshalAuthorizedKey(pub))
}

// ParseAuthorizedKey is the inverse of MarshalAuthorizedKey.
func ParseAuthorizedKey(s string) (ssh.PublicKey, error) {
	pub, _, _, _, err := ssh.ParseAuthorizedKey([]byte(s))
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}
	return pub, nil
}

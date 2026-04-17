package sshauth

import (
	"fmt"
	"os"

	"golang.org/x/crypto/ssh"
)

// LoadSigner reads an unencrypted private key from disk and returns an
// ssh.Signer. Passphrase-protected keys are out of scope for the SDK —
// service-account keys should not require interactive input.
func LoadSigner(path string) (ssh.Signer, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}
	signer, err := ssh.ParsePrivateKey(raw)
	if err != nil {
		return nil, fmt.Errorf("parse private key (note: SDK requires an unencrypted key): %w", err)
	}
	return signer, nil
}

// Fingerprint returns the SHA256 fingerprint of a public key
// (e.g. "SHA256:abc...").
func Fingerprint(pub ssh.PublicKey) string {
	return ssh.FingerprintSHA256(pub)
}

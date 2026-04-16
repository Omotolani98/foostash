package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/Omotolani98/foostash/internal/apiclient"
	"github.com/Omotolani98/foostash/internal/config"
	"github.com/Omotolani98/foostash/internal/sshauth"
	"golang.org/x/crypto/ssh"
)

// resolveKeyPath picks the SSH private key path with this precedence:
//
//	flag → FOOSTASH_SSH_KEY env → config → ~/.ssh default.
func resolveKeyPath(flag string) (string, error) {
	if flag != "" {
		return flag, nil
	}
	if v := os.Getenv("FOOSTASH_SSH_KEY"); v != "" {
		return v, nil
	}
	if cfg, err := config.LoadGlobal(); err == nil && cfg.Identity != nil && cfg.Identity.SSHKeyPath != "" {
		return cfg.Identity.SSHKeyPath, nil
	}
	return sshauth.ResolveDefaultKeyPath()
}

// loadSigner loads the signer at path, prompting for a passphrase if needed.
func loadSigner(path string) (ssh.Signer, error) {
	signer, err := sshauth.LoadSigner(path, nil)
	if err == nil {
		return signer, nil
	}
	if !sshauth.IsPassphraseError(err) {
		return nil, err
	}
	pass := promptPassword(fmt.Sprintf("Passphrase for %s: ", path))
	if pass == "" {
		return nil, errors.New("passphrase required")
	}
	return sshauth.LoadSigner(path, []byte(pass))
}

// loadPublicKeyAuthorizedFormat returns the public key in authorized_keys
// format for the given private key path. Tries .pub file first, falls back
// to deriving from the loaded signer.
func loadPublicKeyAuthorized(privPath string, signer ssh.Signer) (string, string, error) {
	pubPath := sshauth.PublicKeyForPrivate(privPath)
	if pub, err := sshauth.LoadPublicKey(pubPath); err == nil {
		return sshauth.MarshalAuthorizedKey(pub), sshauth.Fingerprint(pub), nil
	}
	pub := signer.PublicKey()
	return sshauth.MarshalAuthorizedKey(pub), sshauth.Fingerprint(pub), nil
}

// newAPIClient is a small constructor used by all auth commands.
func newAPIClient(serverURL string, signer ssh.Signer, fingerprint string) *apiclient.Client {
	return apiclient.New(serverURL, signer, fingerprint)
}

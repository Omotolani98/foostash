package cli

import (
	"fmt"

	"github.com/Omotolani98/foostash/internal/apiclient"
	"github.com/Omotolani98/foostash/internal/config"
)

// serverClient loads the global config and constructs a signed apiclient
// against the configured server. Returns a friendly error if no server is set.
//
// sshKeyFlag is the value of a command's --ssh-key flag (may be empty).
func serverClient(sshKeyFlag string) (*apiclient.Client, *config.GlobalConfig, error) {
	cfg, err := config.LoadGlobal()
	if err != nil {
		return nil, nil, err
	}
	if cfg.Server == "" {
		return nil, nil, fmt.Errorf("no server configured; run `foostash register` first")
	}
	keyPath, err := resolveKeyPath(sshKeyFlag)
	if err != nil {
		return nil, nil, err
	}
	signer, err := loadSigner(keyPath)
	if err != nil {
		return nil, nil, err
	}
	_, fingerprint, err := loadPublicKeyAuthorized(keyPath, signer)
	if err != nil {
		return nil, nil, err
	}
	return apiclient.New(cfg.Server, signer, fingerprint), cfg, nil
}

package cli

import (
	"context"
	"fmt"

	"github.com/Omotolani98/foostash/internal/config"
	"github.com/spf13/cobra"
)

func newJoinCmd() *cobra.Command {
	var serverURL, sshKey string
	cmd := &cobra.Command{
		Use:   "join <invite-token>",
		Short: "Join an organization using an invite token",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token := args[0]

			cfg, err := config.LoadGlobal()
			if err != nil {
				return err
			}
			if serverURL == "" {
				serverURL = cfg.Server
			}
			if serverURL == "" {
				return fmt.Errorf("--server is required (or run after registering)")
			}

			keyPath, err := resolveKeyPath(sshKey)
			if err != nil {
				return err
			}
			signer, err := loadSigner(keyPath)
			if err != nil {
				return err
			}
			pubKey, fingerprint, err := loadPublicKeyAuthorized(keyPath, signer)
			if err != nil {
				return err
			}

			client := newAPIClient(serverURL, signer, fingerprint)
			req := map[string]string{
				"token":      token,
				"public_key": pubKey,
			}
			var resp struct {
				UserID string `json:"user_id"`
				OrgID  string `json:"org_id"`
				Email  string `json:"email"`
				Role   string `json:"role"`
			}
			if err := client.Do(context.Background(), "POST", "/v1/admin/join", req, &resp); err != nil {
				return err
			}

			cfg.Server = serverURL
			cfg.Identity = &config.Identity{
				Email:          resp.Email,
				SSHKeyPath:     keyPath,
				KeyFingerprint: fingerprint,
			}
			if err := config.SaveGlobal(cfg); err != nil {
				return err
			}

			fmt.Printf("joined org=%s email=%s role=%s\n", resp.OrgID, resp.Email, resp.Role)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "", "Foostash server URL (default: from config)")
	cmd.Flags().StringVar(&sshKey, "ssh-key", "", "Path to SSH private key (default: ~/.ssh/id_ed25519)")
	return cmd
}

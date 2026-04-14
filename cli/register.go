package cli

import (
	"context"
	"fmt"

	"github.com/Omotolani98/foostash/internal/config"
	"github.com/spf13/cobra"
)

func newRegisterCmd() *cobra.Command {
	var serverURL, email, orgName, sshKey string
	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register a new organization on a foostash server",
		RunE: func(cmd *cobra.Command, args []string) error {
			if serverURL == "" {
				return fmt.Errorf("--server is required")
			}
			if email == "" {
				email = promptLine("Email: ")
			}
			if orgName == "" {
				orgName = promptLine("Organization name: ")
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
				"email":      email,
				"org_name":   orgName,
				"public_key": pubKey,
			}
			var resp struct {
				UserID string `json:"user_id"`
				OrgID  string `json:"org_id"`
				Role   string `json:"role"`
			}
			if err := client.Do(context.Background(), "POST", "/v1/auth/register", req, &resp); err != nil {
				return err
			}

			cfg, err := config.LoadGlobal()
			if err != nil {
				return err
			}
			cfg.Server = serverURL
			cfg.Identity = &config.Identity{
				Email:          email,
				SSHKeyPath:     keyPath,
				KeyFingerprint: fingerprint,
			}
			if err := config.SaveGlobal(cfg); err != nil {
				return err
			}

			fmt.Printf("registered org=%s email=%s role=%s\n", orgName, email, resp.Role)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "", "Foostash server URL (required)")
	cmd.Flags().StringVar(&email, "email", "", "Your email")
	cmd.Flags().StringVar(&orgName, "org", "", "Organization name")
	cmd.Flags().StringVar(&sshKey, "ssh-key", "", "Path to SSH private key (default: ~/.ssh/id_ed25519)")
	return cmd
}

package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func newAdminCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "admin",
		Short: "Org administration commands",
	}
	cmd.AddCommand(newAdminInviteCmd())
	return cmd
}

func newAdminInviteCmd() *cobra.Command {
	var email, role, sshKey string
	cmd := &cobra.Command{
		Use:   "invite",
		Short: "Generate an invite token for a new user",
		RunE: func(cmd *cobra.Command, args []string) error {
			if email == "" {
				return fmt.Errorf("--email is required")
			}
			client, _, err := serverClient(sshKey)
			if err != nil {
				return err
			}
			req := map[string]string{"email": email, "role": role}
			var resp struct {
				Token     string `json:"token"`
				ExpiresAt string `json:"expires_at"`
			}
			if err := client.Do(context.Background(), "POST", "/v1/admin/invites", req, &resp); err != nil {
				return err
			}

			fmt.Printf("invite created (expires %s)\n\n", resp.ExpiresAt)
			fmt.Printf("Share this with %s:\n  foostash join %s\n", email, resp.Token)
			return nil
		},
	}
	cmd.Flags().StringVar(&email, "email", "", "Invitee email (required)")
	cmd.Flags().StringVar(&role, "role", "developer", "Role: admin or developer")
	cmd.Flags().StringVar(&sshKey, "ssh-key", "", "Path to SSH private key (default: ~/.ssh/id_ed25519)")
	return cmd
}

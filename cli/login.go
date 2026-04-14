package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func newLoginCmd() *cobra.Command {
	var sshKey string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Verify your SSH key against the configured foostash server",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := serverClient(sshKey)
			if err != nil {
				return err
			}
			var resp struct {
				UserID string `json:"user_id"`
				OrgID  string `json:"org_id"`
				Email  string `json:"email"`
				Role   string `json:"role"`
			}
			if err := client.Do(context.Background(), "POST", "/v1/auth/login", nil, &resp); err != nil {
				return err
			}
			fmt.Printf("ok email=%s role=%s org=%s\n", resp.Email, resp.Role, resp.OrgID)
			return nil
		},
	}
	cmd.Flags().StringVar(&sshKey, "ssh-key", "", "Path to SSH private key (default: ~/.ssh/id_ed25519)")
	return cmd
}

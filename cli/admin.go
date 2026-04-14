package cli

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

func newAdminCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "admin",
		Short: "Org administration commands",
	}
	cmd.AddCommand(newAdminInviteCmd())
	cmd.AddCommand(newAdminUsersCmd())
	cmd.AddCommand(newAdminAuditCmd())
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

type adminUserView struct {
	ID        string     `json:"id"`
	Email     string     `json:"email"`
	Role      string     `json:"role"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

func newAdminUsersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "users",
		Short: "Manage users in the org",
	}
	cmd.AddCommand(newAdminUsersListCmd(), newAdminUsersRoleCmd(), newAdminUsersRevokeCmd())
	return cmd
}

func newAdminUsersListCmd() *cobra.Command {
	var sshKey string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List users in the org",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := serverClient(sshKey)
			if err != nil {
				return err
			}
			var resp struct {
				Users []adminUserView `json:"users"`
			}
			if err := client.Do(context.Background(), "GET", "/v1/admin/users", nil, &resp); err != nil {
				return err
			}
			tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
			fmt.Fprintln(tw, "ID\tEMAIL\tROLE\tSTATUS\tCREATED")
			for _, u := range resp.Users {
				status := "active"
				if u.RevokedAt != nil {
					status = "revoked"
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
					u.ID, u.Email, u.Role, status, u.CreatedAt.Format(time.RFC3339))
			}
			return tw.Flush()
		},
	}
	cmd.Flags().StringVar(&sshKey, "ssh-key", "", "Path to SSH private key (default: ~/.ssh/id_ed25519)")
	return cmd
}

func newAdminUsersRoleCmd() *cobra.Command {
	var sshKey string
	cmd := &cobra.Command{
		Use:   "role <user-id> <admin|developer>",
		Short: "Change a user's role",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := serverClient(sshKey)
			if err != nil {
				return err
			}
			req := map[string]string{"role": args[1]}
			var resp adminUserView
			path := fmt.Sprintf("/v1/admin/users/%s/role", args[0])
			if err := client.Do(context.Background(), "PATCH", path, req, &resp); err != nil {
				return err
			}
			fmt.Printf("%s is now %s\n", resp.Email, resp.Role)
			return nil
		},
	}
	cmd.Flags().StringVar(&sshKey, "ssh-key", "", "Path to SSH private key (default: ~/.ssh/id_ed25519)")
	return cmd
}

type auditEntryView struct {
	ID           string    `json:"id"`
	UserID       *string   `json:"user_id,omitempty"`
	Action       string    `json:"action"`
	ResourceType *string   `json:"resource_type,omitempty"`
	ResourceID   *string   `json:"resource_id,omitempty"`
	Status       int       `json:"status"`
	RequestIP    *string   `json:"request_ip,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

func newAdminAuditCmd() *cobra.Command {
	var sshKey, user, action, since string
	var limit int
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Query the audit log",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := serverClient(sshKey)
			if err != nil {
				return err
			}
			q := "?"
			if user != "" {
				q += "user=" + user + "&"
			}
			if action != "" {
				q += "action=" + action + "&"
			}
			if since != "" {
				d, err := time.ParseDuration(since)
				if err == nil {
					q += "since=" + time.Now().Add(-d).UTC().Format(time.RFC3339) + "&"
				} else {
					q += "since=" + since + "&"
				}
			}
			if limit > 0 {
				q += fmt.Sprintf("limit=%d&", limit)
			}
			if q == "?" {
				q = ""
			} else {
				q = q[:len(q)-1]
			}
			var resp struct {
				Entries []auditEntryView `json:"entries"`
			}
			if err := client.Do(context.Background(), "GET", "/v1/admin/audit"+q, nil, &resp); err != nil {
				return err
			}
			tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
			fmt.Fprintln(tw, "TIME\tACTION\tUSER\tRESOURCE\tSTATUS\tIP")
			for _, e := range resp.Entries {
				uid := "-"
				if e.UserID != nil {
					uid = *e.UserID
				}
				res := "-"
				if e.ResourceType != nil {
					res = *e.ResourceType
					if e.ResourceID != nil {
						res += "/" + *e.ResourceID
					}
				}
				ip := "-"
				if e.RequestIP != nil {
					ip = *e.RequestIP
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%d\t%s\n",
					e.CreatedAt.Format(time.RFC3339), e.Action, uid, res, e.Status, ip)
			}
			return tw.Flush()
		},
	}
	cmd.Flags().StringVar(&sshKey, "ssh-key", "", "Path to SSH private key")
	cmd.Flags().StringVar(&user, "user", "", "Filter by user ID")
	cmd.Flags().StringVar(&action, "action", "", "Filter by action (e.g. project.create)")
	cmd.Flags().StringVar(&since, "since", "", "Show entries since (duration like 1h, or RFC3339 time)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Max rows (default 100, max 500)")
	return cmd
}

func newAdminUsersRevokeCmd() *cobra.Command {
	var sshKey string
	cmd := &cobra.Command{
		Use:   "revoke <user-id>",
		Short: "Revoke a user's access",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := serverClient(sshKey)
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/v1/admin/users/%s", args[0])
			if err := client.Do(context.Background(), "DELETE", path, nil, nil); err != nil {
				return err
			}
			fmt.Printf("user %s revoked\n", args[0])
			return nil
		},
	}
	cmd.Flags().StringVar(&sshKey, "ssh-key", "", "Path to SSH private key (default: ~/.ssh/id_ed25519)")
	return cmd
}

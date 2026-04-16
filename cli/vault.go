package cli

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

func newVaultCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vault",
		Short: "Manage the org-wide shared vault on the configured server",
		Long:  "Vault secrets are org-wide shared secrets stored on the server. All members can read; admins can write.",
	}
	cmd.AddCommand(
		newVaultSetCmd(app),
		newVaultGetCmd(app),
		newVaultListCmd(app),
		newVaultDeleteCmd(app),
		newVaultHistoryCmd(app),
		newVaultRollbackCmd(app),
	)
	return cmd
}

func newVaultSetCmd(app *App) *cobra.Command {
	var sshKey string
	var secretPrompt bool
	cmd := &cobra.Command{
		Use:   "set KEY=VALUE [KEY=VALUE ...]",
		Short: "Set one or more vault secrets (admin-only)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pairs := make(map[string]string)
			for _, a := range args {
				parts := strings.SplitN(a, "=", 2)
				if len(parts) == 2 {
					pairs[parts[0]] = parts[1]
				} else if secretPrompt {
					val := promptPassword(fmt.Sprintf("Value for %s: ", parts[0]))
					pairs[parts[0]] = val
				} else {
					return fmt.Errorf("invalid pair: %s (expected KEY=VALUE or use --secret for hidden input)", a)
				}
			}
			client, _, err := serverClient(sshKey)
			if err != nil {
				return err
			}
			ctx := context.Background()
			for k, v := range pairs {
				if _, err := remoteVaultSet(ctx, client, app.Crypto, k, v); err != nil {
					return fmt.Errorf("set %s: %w", k, err)
				}
			}
			fmt.Printf("set %d vault secret(s)\n", len(pairs))
			return nil
		},
	}
	cmd.Flags().BoolVar(&secretPrompt, "secret", false, "Prompt for value with hidden input")
	cmd.Flags().StringVar(&sshKey, "ssh-key", "", "Path to SSH private key (default: ~/.ssh/id_ed25519)")
	return cmd
}

func newVaultGetCmd(app *App) *cobra.Command {
	var sshKey string
	cmd := &cobra.Command{
		Use:   "get KEY",
		Short: "Get a vault secret value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := serverClient(sshKey)
			if err != nil {
				return err
			}
			val, err := remoteVaultGet(context.Background(), client, app.Crypto, args[0])
			if err != nil {
				return err
			}
			fmt.Println(val)
			return nil
		},
	}
	cmd.Flags().StringVar(&sshKey, "ssh-key", "", "Path to SSH private key (default: ~/.ssh/id_ed25519)")
	return cmd
}

func newVaultListCmd(app *App) *cobra.Command {
	var sshKey string
	var showValues bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List vault secrets",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := serverClient(sshKey)
			if err != nil {
				return err
			}
			secrets, err := remoteVaultList(context.Background(), client, app.Crypto)
			if err != nil {
				return err
			}
			keys := make([]string, 0, len(secrets))
			for k := range secrets {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				if showValues {
					fmt.Printf("%s=%s\n", k, shellQuote(secrets[k]))
				} else {
					fmt.Println(k)
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&showValues, "values", false, "Print decrypted values alongside keys")
	cmd.Flags().StringVar(&sshKey, "ssh-key", "", "Path to SSH private key (default: ~/.ssh/id_ed25519)")
	return cmd
}

func newVaultDeleteCmd(app *App) *cobra.Command {
	var sshKey string
	cmd := &cobra.Command{
		Use:   "delete KEY",
		Short: "Delete a vault secret (admin-only)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := serverClient(sshKey)
			if err != nil {
				return err
			}
			if err := remoteVaultDelete(context.Background(), client, args[0]); err != nil {
				return err
			}
			fmt.Printf("deleted vault secret %s\n", args[0])
			return nil
		},
	}
	cmd.Flags().StringVar(&sshKey, "ssh-key", "", "Path to SSH private key (default: ~/.ssh/id_ed25519)")
	return cmd
}

func newVaultHistoryCmd(app *App) *cobra.Command {
	var sshKey string
	cmd := &cobra.Command{
		Use:   "history KEY",
		Short: "Show version history for a vault key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := serverClient(sshKey)
			if err != nil {
				return err
			}
			entries, err := remoteVaultHistory(context.Background(), client, app.Crypto, args[0])
			if err != nil {
				return err
			}
			fmt.Printf("  %-8s  %-30s  %s\n", "Version", "Value", "Set At")
			fmt.Printf("  %s\n", "──────────────────────────────────────────────────────────")
			for _, h := range entries {
				fmt.Printf("  %-8d  %-30s  %s\n", h.Version, h.Value, h.SetAt.Format("2006-01-02 15:04:05"))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&sshKey, "ssh-key", "", "Path to SSH private key (default: ~/.ssh/id_ed25519)")
	return cmd
}

func newVaultRollbackCmd(app *App) *cobra.Command {
	var sshKey string
	var version int
	cmd := &cobra.Command{
		Use:   "rollback KEY",
		Short: "Rollback a vault key to a previous version (admin-only)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if version == 0 {
				return fmt.Errorf("--version is required")
			}
			client, _, err := serverClient(sshKey)
			if err != nil {
				return err
			}
			if err := remoteVaultRollback(context.Background(), client, args[0], version); err != nil {
				return err
			}
			fmt.Printf("rolled back vault %s to version %d\n", args[0], version)
			return nil
		},
	}
	cmd.Flags().IntVar(&version, "version", 0, "Target version to rollback to")
	cmd.Flags().StringVar(&sshKey, "ssh-key", "", "Path to SSH private key (default: ~/.ssh/id_ed25519)")
	return cmd
}

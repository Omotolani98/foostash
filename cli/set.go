package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newSetCmd(app *App) *cobra.Command {
	var envFlag string
	var global bool
	var secret bool
	var remote bool
	var sshKey string

	cmd := &cobra.Command{
		Use:   "set KEY=VALUE [KEY=VALUE ...]",
		Short: "Set one or more secrets",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			pairs := make(map[string]string)

			for _, a := range args {
				parts := strings.SplitN(a, "=", 2)
				if len(parts) == 2 {
					pairs[parts[0]] = parts[1]
				} else if secret {
					// key without value + --secret flag → prompt for hidden input
					val := promptPassword(fmt.Sprintf("Value for %s: ", parts[0]))
					pairs[parts[0]] = val
				} else {
					return fmt.Errorf("invalid pair: %s (expected KEY=VALUE or use --secret for hidden input)", a)
				}
			}

			if len(pairs) == 0 {
				return fmt.Errorf("no secrets to set")
			}

			if global {
				if remote {
					return fmt.Errorf("--global and --remote cannot be combined")
				}
				if err := app.Secrets.SetGlobal(pairs); err != nil {
					return err
				}
				fmt.Printf("set %d global secret(s)\n", len(pairs))
				return nil
			}

			proj, err := loadProjectConfig()
			if err != nil {
				return fmt.Errorf("no .foostash.yaml in current directory (run `foostash init` first)")
			}
			env := envFlag
			if env == "" {
				env = proj.DefaultEnv
			}

			if remote {
				client, _, err := serverClient(sshKey)
				if err != nil {
					return err
				}
				ctx := context.Background()
				for k, v := range pairs {
					if _, err := remoteSet(ctx, client, app.Crypto, proj.Project, env, k, v); err != nil {
						return fmt.Errorf("set %s: %w", k, err)
					}
				}
				fmt.Printf("set %d remote secret(s) in %s/%s\n", len(pairs), proj.Project, env)
				return nil
			}

			if err := app.Secrets.Set(proj.Project, env, pairs); err != nil {
				return err
			}
			fmt.Printf("set %d secret(s) in %s/%s\n", len(pairs), proj.Project, env)
			return nil
		},
	}
	cmd.Flags().StringVarP(&envFlag, "env", "e", "", "Target environment")
	cmd.Flags().BoolVarP(&global, "global", "g", false, "Set as global secret")
	cmd.Flags().BoolVar(&secret, "secret", false, "Prompt for value with hidden input")
	cmd.Flags().BoolVar(&remote, "remote", false, "Store on the configured server (encrypted locally, synced across machines)")
	cmd.Flags().StringVar(&sshKey, "ssh-key", "", "Path to SSH private key (default: ~/.ssh/id_ed25519)")
	return cmd
}

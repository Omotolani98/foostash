package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func newGetCmd(app *App) *cobra.Command {
	var envFlag, sshKey string
	var remote bool

	cmd := &cobra.Command{
		Use:   "get KEY",
		Short: "Get a single secret value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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
				val, err := remoteGet(context.Background(), client, app.Crypto, proj.Project, env, args[0])
				if err != nil {
					return err
				}
				fmt.Println(val)
				return nil
			}

			val, err := app.Secrets.Get(proj.Project, env, args[0])
			if err != nil {
				return err
			}
			fmt.Println(val)
			return nil
		},
	}
	cmd.Flags().StringVarP(&envFlag, "env", "e", "", "Target environment")
	cmd.Flags().BoolVar(&remote, "remote", false, "Read from the configured server")
	cmd.Flags().StringVar(&sshKey, "ssh-key", "", "Path to SSH private key (default: ~/.ssh/id_ed25519)")
	return cmd
}

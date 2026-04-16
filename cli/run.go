package cli

import (
	"fmt"

	"github.com/Omotolani98/foostash/internal/runner"
	"github.com/spf13/cobra"
)

func newRunCmd(app *App) *cobra.Command {
	var envFlag, sshKey string
	var withGlobals, remote bool

	cmd := &cobra.Command{
		Use:   "run -- command [args...]",
		Short: "Run a command with secrets injected as env vars",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			proj, err := loadProjectConfig()
			if err != nil {
				return fmt.Errorf("no .foostash.yaml in current directory (run `foostash init` first)")
			}
			env := envFlag
			if env == "" {
				env = proj.DefaultEnv
			}

			var secrets map[string]string
			if remote {
				client, _, err := serverClient(sshKey)
				if err != nil {
					return err
				}
				secrets, err = remoteList(cmd.Context(), client, app.Crypto, proj.Project, env)
				if err != nil {
					return err
				}
			} else {
				secrets, err = app.Secrets.Pull(proj.Project, env, withGlobals)
				if err != nil {
					return err
				}
			}

			return runner.Exec(args, secrets)
		},
	}
	cmd.Flags().StringVarP(&envFlag, "env", "e", "", "Target environment")
	cmd.Flags().BoolVar(&withGlobals, "with-globals", false, "Include global secrets")
	cmd.Flags().BoolVar(&remote, "remote", false, "Fetch secrets from the configured server")
	cmd.Flags().StringVar(&sshKey, "ssh-key", "", "Path to SSH private key (default: ~/.ssh/id_ed25519)")
	return cmd
}

package cli

import (
	"fmt"

	"github.com/Omotolani98/foostash/internal/runner"
	"github.com/spf13/cobra"
)

func newRunCmd(app *App) *cobra.Command {
	var envFlag string
	var withGlobals bool

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

			secrets, err := app.Secrets.Pull(proj.Project, env, withGlobals)
			if err != nil {
				return err
			}

			return runner.Exec(args, secrets)
		},
	}
	cmd.Flags().StringVarP(&envFlag, "env", "e", "", "Target environment")
	cmd.Flags().BoolVar(&withGlobals, "with-globals", false, "Include global secrets")
	return cmd
}

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newGetCmd(app *App) *cobra.Command {
	var envFlag string

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

			val, err := app.Secrets.Get(proj.Project, env, args[0])
			if err != nil {
				return err
			}
			fmt.Println(val)
			return nil
		},
	}
	cmd.Flags().StringVarP(&envFlag, "env", "e", "", "Target environment")
	return cmd
}

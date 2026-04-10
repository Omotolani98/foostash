package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newKeysCmd(app *App) *cobra.Command {
	var envFlag string

	cmd := &cobra.Command{
		Use:   "keys",
		Short: "List all keys in an environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			proj, err := loadProjectConfig()
			if err != nil {
				return fmt.Errorf("no .foostash.yaml in current directory (run `foostash init` first)")
			}
			env := envFlag
			if env == "" {
				env = proj.DefaultEnv
			}

			keys, err := app.Secrets.ListKeys(proj.Project, env)
			if err != nil {
				return err
			}

			for _, k := range keys {
				fmt.Println(k)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&envFlag, "env", "e", "", "Target environment")
	return cmd
}

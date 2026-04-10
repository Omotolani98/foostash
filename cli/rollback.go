package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newRollbackCmd(app *App) *cobra.Command {
	var envFlag string
	var version int

	cmd := &cobra.Command{
		Use:   "rollback KEY",
		Short: "Rollback a key to a previous version",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if version == 0 {
				return fmt.Errorf("--version is required")
			}

			proj, err := loadProjectConfig()
			if err != nil {
				return fmt.Errorf("no .foostash.yaml in current directory (run `foostash init` first)")
			}
			env := envFlag
			if env == "" {
				env = proj.DefaultEnv
			}

			if err := app.Secrets.Rollback(proj.Project, env, args[0], version); err != nil {
				return err
			}

			fmt.Printf("rolled back %s to version %d in %s/%s\n", args[0], version, proj.Project, env)
			return nil
		},
	}
	cmd.Flags().StringVarP(&envFlag, "env", "e", "", "Target environment")
	cmd.Flags().IntVar(&version, "version", 0, "Target version to rollback to")
	return cmd
}

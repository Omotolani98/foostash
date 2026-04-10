package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newDeleteCmd(app *App) *cobra.Command {
	var envFlag string
	var force bool

	cmd := &cobra.Command{
		Use:   "delete KEY",
		Short: "Delete a secret from an environment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			proj, err := loadProjectConfig()
			if err != nil {
				return fmt.Errorf("no .foostash.yaml in current directory (run `foostash init` first)")
			}
			env := envFlag
			if env == "" {
				env = proj.DefaultEnv
			}

			if !force {
				confirm := promptLine(fmt.Sprintf("Delete secret %q from %s/%s? (y/N): ", key, proj.Project, env))
				if strings.ToLower(strings.TrimSpace(confirm)) != "y" {
					fmt.Println("aborted")
					return nil
				}
			}

			if err := app.Secrets.Delete(proj.Project, env, key); err != nil {
				return err
			}
			fmt.Printf("deleted %q from %s/%s\n", key, proj.Project, env)
			return nil
		},
	}
	cmd.Flags().StringVarP(&envFlag, "env", "e", "", "Target environment")
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}

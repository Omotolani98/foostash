package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newHistoryCmd(app *App) *cobra.Command {
	var envFlag string

	cmd := &cobra.Command{
		Use:   "history KEY",
		Short: "Show version history for a key",
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

			history, err := app.Secrets.GetHistory(proj.Project, env, args[0])
			if err != nil {
				return err
			}

			fmt.Printf("  %-8s  %-30s  %s\n", "Version", "Value", "Set At")
			fmt.Printf("  %s\n", "──────────────────────────────────────────────────────────")
			// show newest first
			for i := len(history) - 1; i >= 0; i-- {
				h := history[i]
				fmt.Printf("  %-8d  %-30s  %s\n", h.Version, h.Value, h.SetAt.Format("2006-01-02 15:04:05"))
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&envFlag, "env", "e", "", "Target environment")
	return cmd
}

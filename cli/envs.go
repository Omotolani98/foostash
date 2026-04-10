package cli

import (
	"fmt"

	"github.com/Omotolani98/foostash/internal/config"
	"github.com/spf13/cobra"
)

func newEnvsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "envs",
		Short: "List environments for the current project",
		RunE: func(cmd *cobra.Command, args []string) error {
			proj, err := loadProjectConfig()
			if err != nil {
				return fmt.Errorf("no .foostash.yaml in current directory (run `foostash init` first)")
			}

			envs, err := app.Envs.List(proj.Project)
			if err != nil {
				return err
			}

			if len(envs) == 0 {
				fmt.Println("no environments found")
				return nil
			}
			for _, e := range envs {
				fmt.Println(e)
			}
			return nil
		},
	}

	cmd.AddCommand(
		newEnvsCreateCmd(app),
		newEnvsCloneCmd(app),
		newEnvsDeleteCmd(app),
	)

	return cmd
}

func newEnvsCreateCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "create NAME",
		Short: "Create a new environment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			proj, err := loadProjectConfig()
			if err != nil {
				return fmt.Errorf("no .foostash.yaml in current directory (run `foostash init` first)")
			}

			if err := app.Envs.Create(proj.Project, args[0]); err != nil {
				return err
			}

			// update .foostash.yaml environments list
			proj.Environments = appendUnique(proj.Environments, args[0])
			if err := config.SaveProject(".", proj); err != nil {
				return fmt.Errorf("update .foostash.yaml: %w", err)
			}

			fmt.Printf("created environment %q\n", args[0])
			return nil
		},
	}
}

func newEnvsCloneCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "clone SOURCE DEST",
		Short: "Clone an environment",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			proj, err := loadProjectConfig()
			if err != nil {
				return fmt.Errorf("no .foostash.yaml in current directory (run `foostash init` first)")
			}

			if err := app.Envs.Clone(proj.Project, args[0], args[1]); err != nil {
				return err
			}

			proj.Environments = appendUnique(proj.Environments, args[1])
			if err := config.SaveProject(".", proj); err != nil {
				return fmt.Errorf("update .foostash.yaml: %w", err)
			}

			fmt.Printf("cloned %s → %s\n", args[0], args[1])
			return nil
		},
	}
}

func newEnvsDeleteCmd(app *App) *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "delete NAME",
		Short: "Delete an environment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			proj, err := loadProjectConfig()
			if err != nil {
				return fmt.Errorf("no .foostash.yaml in current directory (run `foostash init` first)")
			}

			if !force {
				confirm := promptLine(fmt.Sprintf("Delete environment %q? This cannot be undone. (y/N): ", args[0]))
				if confirm != "y" && confirm != "Y" {
					fmt.Println("cancelled")
					return nil
				}
			}

			if err := app.Envs.Delete(proj.Project, args[0]); err != nil {
				return err
			}

			// remove from .foostash.yaml
			proj.Environments = removeStr(proj.Environments, args[0])
			if err := config.SaveProject(".", proj); err != nil {
				return fmt.Errorf("update .foostash.yaml: %w", err)
			}

			fmt.Printf("deleted environment %q\n", args[0])
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}

func appendUnique(slice []string, val string) []string {
	for _, s := range slice {
		if s == val {
			return slice
		}
	}
	return append(slice, val)
}

func removeStr(slice []string, val string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != val {
			result = append(result, s)
		}
	}
	return result
}

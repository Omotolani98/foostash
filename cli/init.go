package cli

import (
	"fmt"
	"os"

	"github.com/Omotolani98/foostash/internal/config"
	"github.com/Omotolani98/foostash/internal/crypto"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var project, defaultEnv string
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a project in the current directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			if project == "" {
				project = promptLine("Project name: ")
			}
			if project == "" {
				return fmt.Errorf("project name is required")
			}
			if defaultEnv == "" {
				defaultEnv = "dev"
			}

			// create project directory under ~/.foostash/projects/
			dir, err := crypto.ProjectsDir()
			if err != nil {
				return err
			}
			projDir := dir + "/" + project
			if err := os.MkdirAll(projDir, 0700); err != nil {
				return fmt.Errorf("create project dir: %w", err)
			}

			// write .foostash.yaml
			cfg := &config.ProjectConfig{
				Project:      project,
				DefaultEnv:   defaultEnv,
				Environments: []string{defaultEnv},
			}
			if err := config.SaveProject(".", cfg); err != nil {
				return err
			}

			fmt.Printf("initialized .foostash.yaml for project=%s env=%s\n", project, defaultEnv)
			return nil
		},
	}
	cmd.Flags().StringVar(&project, "project", "", "Project name")
	cmd.Flags().StringVar(&defaultEnv, "env", "", "Default environment (default: dev)")
	return cmd
}

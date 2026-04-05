package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var project, defaultEnv string
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Link the current directory to a Foostash project",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := mustClient()
			if project == "" {
				project = promptLine("Project slug: ")
			}
			if defaultEnv == "" {
				defaultEnv = "dev"
			}

			if _, err := c.CreateProject(project, project); err != nil {
				if apiErr, ok := err.(interface{ Error() string }); ok {
					fmt.Printf("note: project may already exist (%s)\n", apiErr.Error())
				}
			}
			if _, err := c.CreateEnv(project, defaultEnv); err != nil {
				fmt.Printf("note: env may already exist (%v)\n", err)
			}

			if err := SaveProjectConfig(&ProjectConfig{Project: project, DefaultEnv: defaultEnv}); err != nil {
				return err
			}
			fmt.Printf("initialized .foostash.yaml for project=%s env=%s\n", project, defaultEnv)
			return nil
		},
	}
	cmd.Flags().StringVar(&project, "project", "", "Project slug")
	cmd.Flags().StringVar(&defaultEnv, "env", "", "Default environment slug")
	return cmd
}

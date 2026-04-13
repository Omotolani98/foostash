package cli

import (
	"context"
	"fmt"

	"github.com/Omotolani98/foostash/internal/config"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var project, sshKey string
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a project on the configured foostash server",
		RunE: func(cmd *cobra.Command, args []string) error {
			if project == "" {
				project = promptLine("Project name: ")
			}
			if project == "" {
				return fmt.Errorf("project name is required")
			}

			client, _, err := serverClient(sshKey)
			if err != nil {
				return err
			}

			var resp struct {
				ID        string `json:"id"`
				Slug      string `json:"slug"`
				Name      string `json:"name"`
				CreatedAt string `json:"created_at"`
			}
			req := map[string]string{"name": project}
			if err := client.Do(context.Background(), "POST", "/v1/projects", req, &resp); err != nil {
				return err
			}

			cfg := &config.ProjectConfig{
				Project:      resp.Slug,
				DefaultEnv:   "dev",
				Environments: []string{"dev"},
			}
			if err := config.SaveProject(".", cfg); err != nil {
				return err
			}

			fmt.Printf("created project=%s (name=%q)\n", resp.Slug, resp.Name)
			return nil
		},
	}
	cmd.Flags().StringVar(&project, "project", "", "Project name")
	cmd.Flags().StringVar(&sshKey, "ssh-key", "", "Path to SSH private key (default: ~/.ssh/id_ed25519)")
	return cmd
}

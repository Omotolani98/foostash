package cli

import (
	"context"
	"fmt"

	"github.com/Omotolani98/foostash/internal/config"
	"github.com/spf13/cobra"
)

func newEnvsCmd(_ *App) *cobra.Command {
	var sshKey string
	cmd := &cobra.Command{
		Use:   "envs",
		Short: "List environments for the current project",
		RunE: func(cmd *cobra.Command, args []string) error {
			proj, err := loadProjectConfig()
			if err != nil {
				return fmt.Errorf("no .foostash.yaml in current directory (run `foostash init` first)")
			}
			client, _, err := serverClient(sshKey)
			if err != nil {
				return err
			}
			var resp struct {
				Environments []struct {
					Slug      string `json:"slug"`
					CreatedAt string `json:"created_at"`
				} `json:"environments"`
			}
			path := fmt.Sprintf("/v1/projects/%s/envs", proj.Project)
			if err := client.Do(context.Background(), "GET", path, nil, &resp); err != nil {
				return err
			}
			if len(resp.Environments) == 0 {
				fmt.Println("no environments found")
				return nil
			}
			for _, e := range resp.Environments {
				fmt.Println(e.Slug)
			}
			return nil
		},
	}
	cmd.PersistentFlags().StringVar(&sshKey, "ssh-key", "", "Path to SSH private key (default: ~/.ssh/id_ed25519)")

	cmd.AddCommand(
		newEnvsCreateCmd(&sshKey),
		newEnvsCloneCmd(&sshKey),
		newEnvsDeleteCmd(&sshKey),
	)

	return cmd
}

func newEnvsCreateCmd(sshKey *string) *cobra.Command {
	return &cobra.Command{
		Use:   "create NAME",
		Short: "Create a new environment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			proj, err := loadProjectConfig()
			if err != nil {
				return fmt.Errorf("no .foostash.yaml in current directory (run `foostash init` first)")
			}
			client, _, err := serverClient(*sshKey)
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/v1/projects/%s/envs", proj.Project)
			req := map[string]string{"slug": args[0]}
			if err := client.Do(context.Background(), "POST", path, req, nil); err != nil {
				return err
			}

			proj.Environments = appendUnique(proj.Environments, args[0])
			if err := config.SaveProject(".", proj); err != nil {
				return fmt.Errorf("update .foostash.yaml: %w", err)
			}
			fmt.Printf("created environment %q\n", args[0])
			return nil
		},
	}
}

func newEnvsCloneCmd(sshKey *string) *cobra.Command {
	return &cobra.Command{
		Use:   "clone SOURCE DEST",
		Short: "Clone an environment",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			proj, err := loadProjectConfig()
			if err != nil {
				return fmt.Errorf("no .foostash.yaml in current directory (run `foostash init` first)")
			}
			client, _, err := serverClient(*sshKey)
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/v1/projects/%s/envs/%s/clone", proj.Project, args[0])
			req := map[string]string{"dest": args[1]}
			if err := client.Do(context.Background(), "POST", path, req, nil); err != nil {
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

func newEnvsDeleteCmd(sshKey *string) *cobra.Command {
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
			client, _, err := serverClient(*sshKey)
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/v1/projects/%s/envs/%s", proj.Project, args[0])
			if err := client.Do(context.Background(), "DELETE", path, nil, nil); err != nil {
				return err
			}

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

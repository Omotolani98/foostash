package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/Omotolani98/foostash/internal/crypto"
	"github.com/spf13/cobra"
)

func newExportCmd(app *App) *cobra.Command {
	var envFlag, outFile, password string

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export password-protected bundle for sharing",
		RunE: func(cmd *cobra.Command, args []string) error {
			proj, err := loadProjectConfig()
			if err != nil {
				return fmt.Errorf("no .foostash.yaml in current directory (run `foostash init` first)")
			}
			env := envFlag
			if env == "" {
				env = proj.DefaultEnv
			}

			secrets, err := app.Secrets.Pull(proj.Project, env, false)
			if err != nil {
				return err
			}

			bundle := exportBundle{
				Project: proj.Project,
				Env:     env,
				Secrets: secrets,
			}

			bundleJSON, err := json.Marshal(bundle)
			if err != nil {
				return fmt.Errorf("marshal bundle: %w", err)
			}

			if password == "" {
				password = promptPassword("Enter encryption password: ")
				if password == "" {
					return fmt.Errorf("password is required")
				}
				confirm := promptPassword("Confirm password: ")
				if confirm != password {
					return fmt.Errorf("passwords do not match")
				}
			}

			encrypted, err := crypto.EncryptWithPassword(bundleJSON, password)
			if err != nil {
				return fmt.Errorf("encrypt bundle: %w", err)
			}

			if outFile == "" {
				outFile = fmt.Sprintf("%s-%s.foostash", proj.Project, env)
			}

			if err := os.WriteFile(outFile, encrypted, 0600); err != nil {
				return fmt.Errorf("write export: %w", err)
			}

			fmt.Printf("exported %d secret(s) to %s\n", len(secrets), outFile)
			fmt.Println("Share this file securely — recipient needs the password to import.")
			return nil
		},
	}
	cmd.Flags().StringVarP(&envFlag, "env", "e", "", "Target environment")
	cmd.Flags().StringVarP(&outFile, "out", "o", "", "Output file (default: PROJECT-ENV.foostash)")
	cmd.Flags().StringVar(&password, "password", "", "Encryption password (prompts if not provided)")
	return cmd
}

type exportBundle struct {
	Project string            `json:"project"`
	Env     string            `json:"env"`
	Secrets map[string]string `json:"secrets"`
}

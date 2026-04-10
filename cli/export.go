package cli

import (
	"encoding/json"
	"fmt"

	"github.com/Omotolani98/foostash/internal/store"
	"github.com/spf13/cobra"
)

func newExportCmd(app *App) *cobra.Command {
	var envFlag, outFile string

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export encrypted bundle for sharing",
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

			b, err := json.Marshal(bundle)
			if err != nil {
				return fmt.Errorf("marshal bundle: %w", err)
			}

			// encrypt the bundle using the store
			path, err := store.EnvPath(proj.Project, env)
			if err != nil {
				return err
			}
			_ = path // we just need the store's engine

			if outFile == "" {
				outFile = fmt.Sprintf("%s-%s.foostash", proj.Project, env)
			}

			// use the store to create a temporary secret file, then export as encrypted
			sf := store.NewSecretFile()
			for k, v := range secrets {
				sf.Secrets[k] = store.SecretEntry{Value: v}
			}
			if err := app.Store.Save(outFile, sf); err != nil {
				return fmt.Errorf("write export: %w", err)
			}

			_ = b // bundle JSON used for future format; for now we export as .enc format
			fmt.Printf("exported %d secret(s) to %s\n", len(secrets), outFile)
			return nil
		},
	}
	cmd.Flags().StringVarP(&envFlag, "env", "e", "", "Target environment")
	cmd.Flags().StringVarP(&outFile, "out", "o", "", "Output file (default: PROJECT-ENV.foostash)")
	return cmd
}

type exportBundle struct {
	Project string            `json:"project"`
	Env     string            `json:"env"`
	Secrets map[string]string `json:"secrets"`
}

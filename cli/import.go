package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func newImportCmd(app *App) *cobra.Command {
	var envFlag string
	var global bool

	cmd := &cobra.Command{
		Use:   "import FILE",
		Short: "Import secrets from a .env file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]

			pairs, err := parseDotenvFile(filePath)
			if err != nil {
				return fmt.Errorf("parse file: %w", err)
			}

			if len(pairs) == 0 {
				fmt.Println("no secrets found in file")
				return nil
			}

			if global {
				if err := app.Secrets.SetGlobal(pairs); err != nil {
					return err
				}
				fmt.Printf("imported %d global secret(s)\n", len(pairs))
				return nil
			}

			proj, err := loadProjectConfig()
			if err != nil {
				return fmt.Errorf("no .foostash.yaml in current directory (run `foostash init` first)")
			}
			env := envFlag
			if env == "" {
				env = proj.DefaultEnv
			}

			if err := app.Secrets.Set(proj.Project, env, pairs); err != nil {
				return err
			}
			fmt.Printf("imported %d secret(s) into %s/%s\n", len(pairs), proj.Project, env)
			return nil
		},
	}
	cmd.Flags().StringVarP(&envFlag, "env", "e", "", "Target environment")
	cmd.Flags().BoolVarP(&global, "global", "g", false, "Import as global secrets")
	return cmd
}

func parseDotenvFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	pairs := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// strip "export " prefix
		line = strings.TrimPrefix(line, "export ")

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// strip surrounding quotes
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		pairs[key] = value
	}
	return pairs, scanner.Err()
}

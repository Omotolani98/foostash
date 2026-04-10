package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Omotolani98/foostash/internal/crypto"
	"github.com/spf13/cobra"
)

func newImportCmd(app *App) *cobra.Command {
	var envFlag string
	var global bool
	var password string

	cmd := &cobra.Command{
		Use:   "import FILE",
		Short: "Import secrets from a .env file or encrypted .foostash bundle",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]

			isBundle := strings.HasSuffix(filePath, ".foostash")
			var pairs map[string]string
			var bundle exportBundle

			if isBundle {
				data, err := os.ReadFile(filePath)
				if err != nil {
					return fmt.Errorf("read file: %w", err)
				}

				if password == "" {
					password = promptPassword("Enter decryption password: ")
					if password == "" {
						return fmt.Errorf("password is required for encrypted bundles")
					}
				}

				plaintext, err := crypto.DecryptWithPassword(data, password)
				if err != nil {
					return fmt.Errorf("decrypt bundle: %w (check password and try again)", err)
				}

				if err := json.Unmarshal(plaintext, &bundle); err != nil {
					return fmt.Errorf("parse bundle: %w", err)
				}
				pairs = bundle.Secrets

				fmt.Printf("importing bundle for %s/%s\n", bundle.Project, bundle.Env)
			} else {
				var err error
				pairs, err = parseDotenvFile(filePath)
				if err != nil {
					return fmt.Errorf("parse file: %w", err)
				}
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
				if isBundle {
					env = bundle.Env
				}
				if env == "" {
					env = proj.DefaultEnv
				}
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
	cmd.Flags().StringVar(&password, "password", "", "Decryption password (prompts if not provided)")
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
		line = strings.TrimPrefix(line, "export ")

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

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

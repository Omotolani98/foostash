package cli

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

func newPullCmd(app *App) *cobra.Command {
	var envFlag, format string
	var withGlobals bool

	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull all secrets for an environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			proj, err := loadProjectConfig()
			if err != nil {
				return fmt.Errorf("no .foostash.yaml in current directory (run `foostash init` first)")
			}
			env := envFlag
			if env == "" {
				env = proj.DefaultEnv
			}

			secrets, err := app.Secrets.Pull(proj.Project, env, withGlobals)
			if err != nil {
				return err
			}

			keys := make([]string, 0, len(secrets))
			for k := range secrets {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			switch format {
			case "dotenv", "":
				for _, k := range keys {
					fmt.Printf("%s=%s\n", k, shellQuote(secrets[k]))
				}
			case "json":
				b, _ := json.MarshalIndent(secrets, "", "  ")
				fmt.Println(string(b))
			case "export":
				for _, k := range keys {
					fmt.Printf("export %s=%s\n", k, shellQuote(secrets[k]))
				}
			default:
				return fmt.Errorf("unknown format: %s", format)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&envFlag, "env", "e", "", "Target environment")
	cmd.Flags().StringVarP(&format, "format", "f", "dotenv", "Output format: dotenv|json|export")
	cmd.Flags().BoolVar(&withGlobals, "with-globals", false, "Include global secrets")
	return cmd
}

func shellQuote(v string) string {
	if !strings.ContainsAny(v, " \t\"'$\\") && v != "" {
		return v
	}
	return `"` + strings.ReplaceAll(strings.ReplaceAll(v, `\`, `\\`), `"`, `\"`) + `"`
}

package cli

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

func newPullCmd() *cobra.Command {
	var envFlag, format string
	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull secrets for an environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			proj, err := LoadProjectConfig()
			if err != nil {
				return fmt.Errorf("no .foostash.yaml in current directory (run `foostash init` first)")
			}
			env := envFlag
			if env == "" {
				env = proj.DefaultEnv
			}

			c := mustClient()
			res, err := c.Pull(proj.Project, env)
			if err != nil {
				return err
			}

			keys := make([]string, 0, len(res.Secrets))
			for k := range res.Secrets {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			switch format {
			case "dotenv", "":
				for _, k := range keys {
					fmt.Printf("%s=%s\n", k, shellQuote(res.Secrets[k]))
				}
			case "json":
				b, _ := json.MarshalIndent(res.Secrets, "", "  ")
				fmt.Println(string(b))
			case "export":
				for _, k := range keys {
					fmt.Printf("export %s=%s\n", k, shellQuote(res.Secrets[k]))
				}
			default:
				return fmt.Errorf("unknown format: %s", format)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&envFlag, "env", "", "Environment slug")
	cmd.Flags().StringVar(&format, "format", "dotenv", "Output format: dotenv|json|export")
	return cmd
}

func shellQuote(v string) string {
	if !strings.ContainsAny(v, " \t\"'$\\") && v != "" {
		return v
	}
	return `"` + strings.ReplaceAll(strings.ReplaceAll(v, `\`, `\\`), `"`, `\"`) + `"`
}

package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newSetCmd() *cobra.Command {
	var envFlag string
	cmd := &cobra.Command{
		Use:   "set KEY=VALUE [KEY=VALUE ...]",
		Short: "Set one or more secrets for an environment",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			proj, err := LoadProjectConfig()
			if err != nil {
				return fmt.Errorf("no .foostash.yaml in current directory (run `foostash init` first)")
			}
			env := envFlag
			if env == "" {
				env = proj.DefaultEnv
			}

			secrets := map[string]string{}
			for _, a := range args {
				parts := strings.SplitN(a, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid pair: %s (expected KEY=VALUE)", a)
				}
				secrets[parts[0]] = parts[1]
			}

			c := mustClient()
			res, err := c.Set(proj.Project, env, secrets)
			if err != nil {
				return err
			}
			fmt.Printf("set %d secret(s) in %s/%s (version %d)\n", len(res.Set), proj.Project, env, res.Version)
			return nil
		},
	}
	cmd.Flags().StringVar(&envFlag, "env", "", "Environment slug (defaults to project default)")
	return cmd
}

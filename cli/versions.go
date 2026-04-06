package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionsCmd() *cobra.Command {
	var envFlag string
	cmd := &cobra.Command{
		Use:   "versions KEY",
		Short: "Show version history for a secret key",
		Args:  cobra.ExactArgs(1),
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
			res, err := c.Versions(proj.Project, env, args[0])
			if err != nil {
				return err
			}
			fmt.Printf("%s/%s/%s (%d version(s))\n", proj.Project, env, args[0], len(res.Versions))
			for _, v := range res.Versions {
				fmt.Printf("  v%-3d %s  %s\n", v.Version, v.CreatedAt.Format("2006-01-02 15:04:05"), v.Value)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&envFlag, "env", "", "Environment slug")
	return cmd
}

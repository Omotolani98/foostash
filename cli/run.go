package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

func newRunCmd() *cobra.Command {
	var envFlag string
	cmd := &cobra.Command{
		Use:   "run -- command [args...]",
		Short: "Run a command with secrets injected as env vars",
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

			c := mustClient()
			res, err := c.Pull(proj.Project, env)
			if err != nil {
				return err
			}

			child := exec.Command(args[0], args[1:]...)
			child.Stdout = os.Stdout
			child.Stderr = os.Stderr
			child.Stdin = os.Stdin
			child.Env = os.Environ()
			for k, v := range res.Secrets {
				child.Env = append(child.Env, k+"="+v)
			}
			if err := child.Run(); err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					os.Exit(exitErr.ExitCode())
				}
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&envFlag, "env", "", "Environment slug")
	return cmd
}

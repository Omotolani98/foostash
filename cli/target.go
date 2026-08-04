package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/Omotolani98/foostash/internal/targets"
	"github.com/spf13/cobra"
)

type targetFlags struct {
	env         string
	withGlobals bool
	remote      bool
	sshKey      string
}

func newTargetCmd(app *App) *cobra.Command {
	var flags targetFlags

	cmd := &cobra.Command{
		Use:   "target",
		Short: "Render or push secrets to deployment platforms (docker, kubernetes, github-actions)",
	}
	cmd.PersistentFlags().StringVarP(&flags.env, "env", "e", "", "Target environment")
	cmd.PersistentFlags().BoolVar(&flags.withGlobals, "with-globals", false, "Include global secrets")
	cmd.PersistentFlags().BoolVar(&flags.remote, "remote", false, "Fetch from the configured server (no globals merge)")
	cmd.PersistentFlags().StringVar(&flags.sshKey, "ssh-key", "", "Path to SSH private key (default: ~/.ssh/id_ed25519)")

	cmd.AddCommand(
		newTargetDockerCmd(app, &flags),
		newTargetKubernetesCmd(app, &flags),
		newTargetGithubActionsCmd(app, &flags),
	)
	return cmd
}

// resolveTargetEnv returns the environment name a target command operates on:
// the explicit -e/--env flag, falling back to the project's DefaultEnv.
func resolveTargetEnv(flags *targetFlags) (string, error) {
	proj, err := loadProjectConfig()
	if err != nil {
		return "", fmt.Errorf("no .foostash.yaml in current directory (run `foostash init` first)")
	}
	env := flags.env
	if env == "" {
		env = proj.DefaultEnv
	}
	return env, nil
}

func fetchTargetSecrets(app *App, flags *targetFlags) (map[string]string, error) {
	proj, err := loadProjectConfig()
	if err != nil {
		return nil, fmt.Errorf("no .foostash.yaml in current directory (run `foostash init` first)")
	}
	env := flags.env
	if env == "" {
		env = proj.DefaultEnv
	}
	if flags.remote {
		client, _, err := serverClient(flags.sshKey)
		if err != nil {
			return nil, err
		}
		return remoteList(context.Background(), client, app.Crypto, proj.Project, env)
	}
	return app.Secrets.Pull(proj.Project, env, flags.withGlobals)
}

func writeTargetOutput(out string, data []byte) error {
	if out == "" {
		_, err := os.Stdout.Write(data)
		return err
	}
	return os.WriteFile(out, data, 0o600)
}

// --- docker ---

func newTargetDockerCmd(app *App, flags *targetFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docker",
		Short: "Docker / docker-compose integration",
	}
	cmd.AddCommand(newTargetDockerRenderCmd(app, flags))
	return cmd
}

func newTargetDockerRenderCmd(app *App, flags *targetFlags) *cobra.Command {
	var out string
	cmd := &cobra.Command{
		Use:   "render",
		Short: "Render a Docker env file (docker run --env-file or compose env_file:)",
		RunE: func(cmd *cobra.Command, args []string) error {
			secrets, err := fetchTargetSecrets(app, flags)
			if err != nil {
				return err
			}
			data, err := targets.RenderEnvFile(secrets)
			if err != nil {
				return err
			}
			return writeTargetOutput(out, data)
		},
	}
	cmd.Flags().StringVarP(&out, "out", "o", "", "Write to file (default: stdout)")
	return cmd
}

// --- kubernetes ---

func newTargetKubernetesCmd(app *App, flags *targetFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "kubernetes",
		Aliases: []string{"k8s"},
		Short:   "Kubernetes Secret integration",
	}
	cmd.AddCommand(
		newTargetK8sRenderCmd(app, flags),
		newTargetK8sApplyCmd(app, flags),
	)
	return cmd
}

func newTargetK8sRenderCmd(app *App, flags *targetFlags) *cobra.Command {
	var name, namespace, out string
	cmd := &cobra.Command{
		Use:   "render",
		Short: "Render a Kubernetes v1 Secret manifest (Opaque, base64-encoded)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			secrets, err := fetchTargetSecrets(app, flags)
			if err != nil {
				return err
			}
			data, err := targets.RenderSecret(name, namespace, secrets)
			if err != nil {
				return err
			}
			return writeTargetOutput(out, data)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Secret name (required)")
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Kubernetes namespace")
	cmd.Flags().StringVarP(&out, "out", "o", "", "Write to file (default: stdout)")
	return cmd
}

func newTargetK8sApplyCmd(app *App, flags *targetFlags) *cobra.Command {
	var name, namespace string
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Render a Secret manifest and pipe to `kubectl apply -f -`",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if _, err := exec.LookPath("kubectl"); err != nil {
				return fmt.Errorf("kubectl not found in $PATH; install it or use `foostash target k8s render`")
			}
			secrets, err := fetchTargetSecrets(app, flags)
			if err != nil {
				return err
			}
			manifest, err := targets.RenderSecret(name, namespace, secrets)
			if err != nil {
				return err
			}
			kargs := []string{"apply", "-f", "-"}
			if namespace != "" {
				kargs = append(kargs, "--namespace", namespace)
			}
			if dryRun {
				kargs = append(kargs, "--dry-run=client", "-o", "yaml")
			}
			k := exec.Command("kubectl", kargs...)
			k.Stdin = bytes.NewReader(manifest)
			k.Stdout = os.Stdout
			k.Stderr = os.Stderr
			return k.Run()
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Secret name (required)")
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Kubernetes namespace")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Client-side dry-run (validates without applying)")
	return cmd
}

// --- github-actions ---

func newTargetGithubActionsCmd(app *App, flags *targetFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "github-actions",
		Aliases: []string{"gha"},
		Short:   "GitHub Actions integration",
	}
	cmd.AddCommand(
		newTargetGHARenderCmd(app, flags),
		newTargetGHAPushCmd(app, flags),
	)
	return cmd
}

func newTargetGHARenderCmd(app *App, flags *targetFlags) *cobra.Command {
	var out string
	cmd := &cobra.Command{
		Use:   "render",
		Short: "Render a workflow step snippet that masks and exports secrets to $GITHUB_ENV",
		RunE: func(cmd *cobra.Command, args []string) error {
			secrets, err := fetchTargetSecrets(app, flags)
			if err != nil {
				return err
			}
			data, err := targets.RenderWorkflowEnv(secrets)
			if err != nil {
				return err
			}
			return writeTargetOutput(out, data)
		},
	}
	cmd.Flags().StringVarP(&out, "out", "o", "", "Write to file (default: stdout)")
	return cmd
}

func newTargetGHAPushCmd(app *App, flags *targetFlags) *cobra.Command {
	var repo, varsPrefix string
	var vars []string
	var repoLevel bool
	cmd := &cobra.Command{
		Use:   "push",
		Short: "Push all secrets to a GitHub repo: keys go to Actions Secrets, --vars keys go to Variables",
		Long: `Push every secret in the environment to a GitHub repository via the gh CLI.

By default every key is written as an encrypted Actions Secret (gh secret set).
Keys named in --vars, or matching --vars-prefix, are instead written as plaintext
Actions Variables (gh variable set) — use this for non-sensitive config such as
REDIS_URL or NODE_ENV so it stays out of the masked secret store.

Secrets/variables are scoped to the GitHub Actions environment matching the
foostash env (-e/--env, or the project default). Pass --repo-level to write
repo-wide secrets/variables instead.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if repo == "" {
				return fmt.Errorf("--repo OWNER/REPO is required")
			}
			ghEnv := ""
			if !repoLevel {
				env, err := resolveTargetEnv(flags)
				if err != nil {
					return err
				}
				ghEnv = env
			}
			if _, err := exec.LookPath("gh"); err != nil {
				return fmt.Errorf("gh CLI not found in $PATH; install https://cli.github.com/")
			}
			auth := exec.Command("gh", "auth", "status")
			auth.Stdout = io.Discard
			auth.Stderr = io.Discard
			if err := auth.Run(); err != nil {
				return fmt.Errorf("gh is not authenticated; run `gh auth login`")
			}
			secrets, err := fetchTargetSecrets(app, flags)
			if err != nil {
				return err
			}

			varSet := make(map[string]bool, len(vars))
			for _, v := range vars {
				varSet[strings.TrimSpace(v)] = true
			}
			isVar := func(k string) bool {
				if varSet[k] {
					return true
				}
				return varsPrefix != "" && strings.HasPrefix(k, varsPrefix)
			}

			keys := make([]string, 0, len(secrets))
			for k := range secrets {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				var gargs []string
				kind := "secret"
				if isVar(k) {
					kind = "variable"
					gargs = []string{"variable", "set", k, "--repo", repo, "--body-file", "-"}
				} else {
					gargs = []string{"secret", "set", k, "--repo", repo, "--body-file", "-"}
				}
				if ghEnv != "" {
					gargs = append(gargs, "--env", ghEnv)
				}
				c := exec.Command("gh", gargs...)
				c.Stdin = strings.NewReader(secrets[k])
				c.Stderr = os.Stderr
				if err := c.Run(); err != nil {
					return fmt.Errorf("push %s %s: %w", kind, k, err)
				}
				fmt.Printf("set %s %s\n", kind, k)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&repo, "repo", "", "OWNER/REPO (required)")
	cmd.Flags().BoolVar(&repoLevel, "repo-level", false, "Write repo-wide secrets/variables instead of scoping to the -e/--env GitHub environment")
	cmd.Flags().StringSliceVar(&vars, "vars", nil, "Keys to push as plaintext Variables instead of Secrets (comma-separated or repeated)")
	cmd.Flags().StringVar(&varsPrefix, "vars-prefix", "", "Push keys with this prefix as plaintext Variables instead of Secrets")
	return cmd
}

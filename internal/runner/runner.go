package runner

import (
	"os"
	"os/exec"
)

// Exec runs a command with the given secrets injected as environment variables.
// Shell env vars take precedence — secrets only fill in what's not already set.
func Exec(args []string, secrets map[string]string) error {
	child := exec.Command(args[0], args[1:]...)
	child.Stdout = os.Stdout
	child.Stderr = os.Stderr
	child.Stdin = os.Stdin
	child.Env = os.Environ()

	// inject secrets (existing shell vars already in Environ take precedence
	// because we only add the secret if the key isn't already present)
	existing := make(map[string]bool)
	for _, e := range child.Env {
		for i := 0; i < len(e); i++ {
			if e[i] == '=' {
				existing[e[:i]] = true
				break
			}
		}
	}
	for k, v := range secrets {
		if !existing[k] {
			child.Env = append(child.Env, k+"="+v)
		}
	}

	if err := child.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return err
	}
	return nil
}

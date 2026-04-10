package cli

import (
	"fmt"
	"os"

	"github.com/Omotolani98/foostash/internal/crypto"
	"github.com/Omotolani98/foostash/internal/envs"
	"github.com/Omotolani98/foostash/internal/secrets"
	"github.com/Omotolani98/foostash/internal/store"
	"github.com/spf13/cobra"
)

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// App holds shared dependencies for all CLI commands.
type App struct {
	Store   *store.Store
	Secrets *secrets.Service
	Envs    *envs.Manager
}

func NewRootCmd() *cobra.Command {
	var app App

	root := &cobra.Command{
		Use:     "foostash",
		Short:   "Foostash: encrypted secrets and environment manager",
		Version: fmt.Sprintf("%s (commit %s, built %s)", Version, Commit, Date),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// skip init for commands that don't need encryption
			if cmd.Name() == "init" || cmd.Name() == "help" || cmd.Name() == "version" || cmd.Name() == "completion" {
				return nil
			}

			masterKey, err := crypto.LoadOrGenerateMasterKey()
			if err != nil {
				return fmt.Errorf("load master key: %w", err)
			}

			engine, err := crypto.NewEngine(masterKey)
			if err != nil {
				return fmt.Errorf("init encryption engine: %w", err)
			}

			app.Store = store.New(engine)
			app.Secrets = secrets.NewService(app.Store)
			app.Envs = envs.NewManager(app.Store)
			return nil
		},
		SilenceUsage: true,
	}

	root.AddCommand(
		newInitCmd(),
		newSetCmd(&app),
		newGetCmd(&app),
		newPullCmd(&app),
		newRunCmd(&app),
		newKeysCmd(&app),
		newDiffCmd(&app),
		newEnvsCmd(&app),
		newHistoryCmd(&app),
		newRollbackCmd(&app),
		newImportCmd(&app),
		newExportCmd(&app),
	)

	return root
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}

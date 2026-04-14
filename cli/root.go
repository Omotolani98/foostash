package cli

import (
	"fmt"
	"os"
	"regexp"
	"runtime/debug"

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

// buildInfo returns version/commit/date, preferring ldflag values when set
// and falling back to Go module build info (populated by `go install`).
func buildInfo() (version, commit, date string) {
	version, commit, date = Version, Commit, Date
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	if version == "dev" && info.Main.Version != "" && info.Main.Version != "(devel)" {
		version = info.Main.Version
	}
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			if commit == "none" && s.Value != "" {
				commit = s.Value
				if len(commit) > 7 {
					commit = commit[:7]
				}
			}
		case "vcs.time":
			if date == "unknown" && s.Value != "" {
				date = s.Value
			}
		}
	}
	// When installed via `go install pkg@rev`, the module proxy strips VCS
	// info, but the pseudo-version still encodes the timestamp and commit:
	//   vX.Y.Z-pre.0.YYYYMMDDHHMMSS-abcdef123456
	if commit == "none" || date == "unknown" {
		if m := pseudoVersionRE.FindStringSubmatch(info.Main.Version); m != nil {
			if date == "unknown" {
				date = m[1]
			}
			if commit == "none" {
				commit = m[2]
			}
		}
	}
	return
}

var pseudoVersionRE = regexp.MustCompile(`(\d{14})-([0-9a-f]{12})$`)

// App holds shared dependencies for all CLI commands.
type App struct {
	Store   *store.Store
	Secrets *secrets.Service
	Envs    *envs.Manager
}

func NewRootCmd() *cobra.Command {
	var app App
	version, commit, date := buildInfo()

	root := &cobra.Command{
		Use:     "foostash",
		Short:   "Foostash: encrypted secrets and environment manager",
		Version: fmt.Sprintf("%s (commit %s, built %s)", version, commit, date),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// skip init for commands that don't need encryption
			switch cmd.Name() {
			case "init", "help", "version", "completion",
				"register", "login", "join", "admin", "invite", "serve":
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
		newDeleteCmd(&app),
		newPullCmd(&app),
		newRunCmd(&app),
		newKeysCmd(&app),
		newDiffCmd(&app),
		newEnvsCmd(&app),
		newHistoryCmd(&app),
		newRollbackCmd(&app),
		newImportCmd(&app),
		newExportCmd(&app),
		newRegisterCmd(),
		newLoginCmd(),
		newJoinCmd(),
		newAdminCmd(),
		newServeCmd(),
	)

	return root
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}

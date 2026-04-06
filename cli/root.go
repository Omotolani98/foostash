package cli

import (
	"fmt"
	"os"

	"github.com/Omotolani98/foostash/cli/client"
	"github.com/spf13/cobra"
)

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:     "foostash",
		Short:   "Foostash: centralized secrets and environment manager",
		Version: fmt.Sprintf("%s (commit %s, built %s)", Version, Commit, Date),
	}
	root.AddCommand(
		newLoginCmd(),
		newRegisterCmd(),
		newInitCmd(),
		newSetCmd(),
		newPullCmd(),
		newRunCmd(),
		newDiffCmd(),
		newVersionsCmd(),
	)
	return root
}

func mustClient() *client.Client {
	cfg, err := LoadGlobalConfig()
	if err != nil {
		fail(err)
	}
	if cfg.Token == "" {
		fmt.Fprintln(os.Stderr, "not logged in. run `foostash login` or `foostash register`")
		os.Exit(1)
	}
	return client.New(cfg.Server, cfg.Token)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}

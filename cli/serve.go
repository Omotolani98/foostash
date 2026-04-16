package cli

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/Omotolani98/foostash/internal/server"
	"github.com/spf13/cobra"
)

func newServeCmd() *cobra.Command {
	var addr, pgURL, sshAddr, sshHostKey string
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the foostash HTTP API server (and optional SSH TUI)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if addr == "" {
				addr = envDefault("FOOSTASH_LISTEN_ADDR", ":8400")
			}
			if pgURL == "" {
				pgURL = envDefault("FOOSTASH_PG_URL", "postgres://foostash:foostash@localhost:5432/foostash?sslmode=disable")
			}
			if sshAddr == "" {
				sshAddr = envDefault("FOOSTASH_SSH_ADDR", ":2222")
			}
			if sshHostKey == "" {
				sshHostKey = envDefault("FOOSTASH_SSH_HOST_KEY", defaultSSHHostKeyPath())
			}

			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()

			version, _, _ := buildInfo()
			return server.Boot(ctx, server.Config{
				HTTPAddr:       addr,
				SSHAddr:        sshAddr,
				SSHHostKeyPath: sshHostKey,
				PGURL:          pgURL,
				Version:        version,
			})
		},
	}
	cmd.Flags().StringVar(&addr, "addr", "", "HTTP listen address (default $FOOSTASH_LISTEN_ADDR or :8400)")
	cmd.Flags().StringVar(&pgURL, "pg-url", "", "Postgres connection URL (default $FOOSTASH_PG_URL)")
	cmd.Flags().StringVar(&sshAddr, "ssh-addr", "", "SSH TUI listen address (default $FOOSTASH_SSH_ADDR or :2222; empty disables)")
	cmd.Flags().StringVar(&sshHostKey, "ssh-host-key", "", "SSH host key path (default $FOOSTASH_SSH_HOST_KEY or ~/.foostash/server/ssh_host_ed25519)")
	return cmd
}

func envDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func defaultSSHHostKeyPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "ssh_host_ed25519"
	}
	return filepath.Join(home, ".foostash", "server", "ssh_host_ed25519")
}

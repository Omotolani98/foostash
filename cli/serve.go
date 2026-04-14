package cli

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/Omotolani98/foostash/internal/server"
	"github.com/spf13/cobra"
)

func newServeCmd() *cobra.Command {
	var addr, pgURL string
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the foostash HTTP API server",
		RunE: func(cmd *cobra.Command, args []string) error {
			if addr == "" {
				addr = envDefault("FOOSTASH_LISTEN_ADDR", ":8400")
			}
			if pgURL == "" {
				pgURL = envDefault("FOOSTASH_PG_URL", "postgres://foostash:foostash@localhost:5432/foostash?sslmode=disable")
			}

			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()

			version, _, _ := buildInfo()
			return server.Boot(ctx, pgURL, addr, version)
		},
	}
	cmd.Flags().StringVar(&addr, "addr", "", "HTTP listen address (default $FOOSTASH_LISTEN_ADDR or :8400)")
	cmd.Flags().StringVar(&pgURL, "pg-url", "", "Postgres connection URL (default $FOOSTASH_PG_URL)")
	return cmd
}

func envDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

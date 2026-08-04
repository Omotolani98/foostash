package cli

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/Omotolani98/foostash/internal/mcpserver"
	"github.com/spf13/cobra"
)

func newMcpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Run the local Foostash MCP server over stdio",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()
			return mcpserver.RunStdio(ctx)
		},
	}
}

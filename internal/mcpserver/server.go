// Package mcpserver exposes safe Foostash setup operations to AI agents via MCP.
//
// It intentionally runs locally over stdio: SSH private keys, the master key,
// and plaintext secrets stay on the user's machine and are never returned in
// tool results.
package mcpserver

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RunStdio serves the Foostash MCP tools over stdin/stdout until the client
// disconnects or ctx is canceled.
func RunStdio(ctx context.Context) error {
	return NewServer().Run(ctx, &mcp.StdioTransport{})
}

// NewServer builds the MCP server. It is separate from RunStdio so tests can
// connect with in-memory transports.
func NewServer() *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "foostash", Version: "v1"}, nil)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "inspect",
		Description: "Inspect local Foostash configuration and optional server health. Does not return secrets.",
		Annotations: &mcp.ToolAnnotations{
			Title:         "Inspect Foostash setup",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, inspectTool)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "identity_setup",
		Description: "Register a new Foostash org or join with an invite using a local SSH key, then write ~/.foostash/config.yaml. Does not return invite tokens or private key material.",
		Annotations: &mcp.ToolAnnotations{
			Title:           "Set up Foostash identity",
			IdempotentHint:  false,
			OpenWorldHint:   boolPtr(true),
			DestructiveHint: boolPtr(false),
		},
	}, identitySetupTool)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "project_setup",
		Description: "Idempotently create or reconnect a Foostash project, ensure environments, and write .foostash.yaml in a project directory.",
		Annotations: &mcp.ToolAnnotations{
			Title:          "Set up Foostash project",
			IdempotentHint: true,
			OpenWorldHint:  boolPtr(true),
		},
	}, projectSetupTool)

	return s
}

func boolPtr(v bool) *bool { return &v }

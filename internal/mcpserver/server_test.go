package mcpserver

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestInspectToolOverMCP(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	ctx := context.Background()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := NewServer().Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	defer serverSession.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v1"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer clientSession.Close()

	res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "inspect", Arguments: map[string]any{}})
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if res.IsError {
		t.Fatalf("inspect returned tool error: %v", res.Content)
	}
	if res.StructuredContent == nil {
		t.Fatalf("expected structured content")
	}
}

func TestNormalizeEnvs(t *testing.T) {
	got := normalizeEnvs([]string{"prod", "dev", "Prod", "staging", ""})
	want := []string{"dev", "prod", "staging"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

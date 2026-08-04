package mcpserver

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type serverSetupPlanInput struct {
	InstallDir string `json:"install_dir,omitempty" jsonschema:"directory for managed docker-compose.yml; defaults to ~/.foostash/server"`
	ImageTag   string `json:"image_tag,omitempty" jsonschema:"Foostash image tag; defaults to latest"`
	BindHost   string `json:"bind_host,omitempty" jsonschema:"host interface for published ports; defaults to 127.0.0.1"`
	HTTPPort   int    `json:"http_port,omitempty" jsonschema:"host HTTP port; defaults to 8400"`
	EnableSSH  bool   `json:"enable_ssh,omitempty" jsonschema:"publish the SSH TUI listener too"`
	SSHPort    int    `json:"ssh_port,omitempty" jsonschema:"host SSH TUI port when enable_ssh is true; defaults to 2222"`
}

type serverSetupPlanOutput struct {
	PlanID      string   `json:"plan_id"`
	InstallDir  string   `json:"install_dir"`
	ComposePath string   `json:"compose_path"`
	HTTPURL     string   `json:"http_url"`
	Image       string   `json:"image"`
	Warnings    []string `json:"warnings,omitempty"`
	Actions     []string `json:"actions"`
}

type setupPlan struct {
	Output     serverSetupPlanOutput
	ComposeYML string
}

var plans = struct {
	sync.Mutex
	m map[string]setupPlan
}{m: make(map[string]setupPlan)}

var runCommand = func(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	return cmd.CombinedOutput()
}

func serverSetupPlanTool(_ context.Context, _ *mcp.CallToolRequest, in serverSetupPlanInput) (*mcp.CallToolResult, serverSetupPlanOutput, error) {
	installDir, err := defaultInstallDir(in.InstallDir)
	if err != nil {
		return nil, serverSetupPlanOutput{}, err
	}
	imageTag := in.ImageTag
	if imageTag == "" {
		imageTag = "latest"
	}
	bindHost := in.BindHost
	if bindHost == "" {
		bindHost = "127.0.0.1"
	}
	httpPort := in.HTTPPort
	if httpPort == 0 {
		httpPort = 8400
	}
	sshPort := in.SSHPort
	if sshPort == 0 {
		sshPort = 2222
	}
	dbPassword, err := randomToken(24)
	if err != nil {
		return nil, serverSetupPlanOutput{}, err
	}
	compose := composeYAML(composeOptions{
		ImageTag:   imageTag,
		BindHost:   bindHost,
		HTTPPort:   httpPort,
		EnableSSH:  in.EnableSSH,
		SSHPort:    sshPort,
		DBPassword: dbPassword,
	})
	planID := planID(compose, installDir)
	out := serverSetupPlanOutput{
		PlanID:      planID,
		InstallDir:  installDir,
		ComposePath: filepath.Join(installDir, "docker-compose.yml"),
		HTTPURL:     fmt.Sprintf("http://%s:%d", bindHost, httpPort),
		Image:       "ghcr.io/omotolani98/foostash:" + imageTag,
		Actions: []string{
			"create managed install directory if missing",
			"write docker-compose.yml with generated Postgres credentials",
			"run docker compose up -d",
			"poll /v1/health until healthy",
		},
	}
	if bindHost != "127.0.0.1" && bindHost != "localhost" {
		out.Warnings = append(out.Warnings, "public HTTP exposure requires TLS termination in front of Foostash")
	}
	if in.EnableSSH {
		out.Warnings = append(out.Warnings, "SSH TUI is enabled; restrict network access to trusted users")
	}

	plans.Lock()
	plans.m[planID] = setupPlan{Output: out, ComposeYML: compose}
	plans.Unlock()
	return nil, out, nil
}

type serverSetupApplyInput struct {
	PlanID string `json:"plan_id" jsonschema:"plan_id returned by server_setup_plan"`
}

type serverSetupApplyOutput struct {
	PlanID      string `json:"plan_id"`
	ComposePath string `json:"compose_path"`
	HTTPURL     string `json:"http_url"`
	Healthy     bool   `json:"healthy"`
}

func serverSetupApplyTool(ctx context.Context, _ *mcp.CallToolRequest, in serverSetupApplyInput) (*mcp.CallToolResult, serverSetupApplyOutput, error) {
	if in.PlanID == "" {
		return nil, serverSetupApplyOutput{}, errors.New("plan_id is required")
	}
	plans.Lock()
	plan, ok := plans.m[in.PlanID]
	plans.Unlock()
	if !ok {
		return nil, serverSetupApplyOutput{}, errors.New("unknown plan_id; run server_setup_plan again")
	}
	if err := os.MkdirAll(plan.Output.InstallDir, 0o700); err != nil {
		return nil, serverSetupApplyOutput{}, err
	}
	if existing, err := os.ReadFile(plan.Output.ComposePath); err == nil && string(existing) != plan.ComposeYML {
		return nil, serverSetupApplyOutput{}, fmt.Errorf("refusing to overwrite unmanaged compose file: %s", plan.Output.ComposePath)
	}
	if err := os.WriteFile(plan.Output.ComposePath, []byte(plan.ComposeYML), 0o600); err != nil {
		return nil, serverSetupApplyOutput{}, err
	}
	cmdCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	if out, err := runCommand(cmdCtx, plan.Output.InstallDir, "docker", "compose", "up", "-d"); err != nil {
		return nil, serverSetupApplyOutput{}, fmt.Errorf("docker compose up: %w: %s", err, strings.TrimSpace(string(out)))
	}
	if err := waitHealthy(ctx, plan.Output.HTTPURL); err != nil {
		return nil, serverSetupApplyOutput{}, err
	}
	return nil, serverSetupApplyOutput{PlanID: in.PlanID, ComposePath: plan.Output.ComposePath, HTTPURL: plan.Output.HTTPURL, Healthy: true}, nil
}

type composeOptions struct {
	ImageTag   string
	BindHost   string
	HTTPPort   int
	EnableSSH  bool
	SSHPort    int
	DBPassword string
}

func composeYAML(o composeOptions) string {
	sshAddr := ""
	sshPorts := ""
	if o.EnableSSH {
		sshAddr = ":2222"
		sshPorts = fmt.Sprintf("\n      - %q", fmt.Sprintf("%s:%d:2222", o.BindHost, o.SSHPort))
	}
	return fmt.Sprintf(`services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: foostash
      POSTGRES_PASSWORD: %q
      POSTGRES_DB: foostash
    volumes:
      - foostash-pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U foostash -d foostash"]
      interval: 5s
      timeout: 5s
      retries: 10
    restart: unless-stopped

  foostash:
    image: ghcr.io/omotolani98/foostash:%s
    depends_on:
      postgres:
        condition: service_healthy
    environment:
      FOOSTASH_PG_URL: postgres://foostash:%s@postgres:5432/foostash?sslmode=disable
      FOOSTASH_LISTEN_ADDR: ":8400"
      FOOSTASH_SSH_ADDR: %q
    ports:
      - %q%s
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:8400/v1/health"]
      interval: 10s
      timeout: 3s
      retries: 5
      start_period: 10s
    restart: unless-stopped

volumes:
  foostash-pgdata:
`, o.DBPassword, o.ImageTag, urlEscapePassword(o.DBPassword), sshAddr, fmt.Sprintf("%s:%d:8400", o.BindHost, o.HTTPPort), sshPorts)
}

func defaultInstallDir(dir string) (string, error) {
	if dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".foostash", "server"), nil
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func planID(compose, dir string) string {
	sum := sha256.Sum256([]byte(dir + "\n" + compose))
	return hex.EncodeToString(sum[:8])
}

func urlEscapePassword(s string) string {
	return strings.ReplaceAll(s, "@", "%40")
}

func waitHealthy(ctx context.Context, serverURL string) error {
	deadline := time.Now().Add(45 * time.Second)
	for {
		if _, err := health(ctx, serverURL); err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("server did not become healthy at %s", serverURL)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

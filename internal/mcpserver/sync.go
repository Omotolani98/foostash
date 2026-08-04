package mcpserver

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/Omotolani98/foostash/internal/apiclient"
	"github.com/Omotolani98/foostash/internal/config"
	"github.com/Omotolani98/foostash/internal/crypto"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type environmentSyncInput struct {
	ServerURL  string `json:"server_url,omitempty" jsonschema:"Foostash server URL; defaults to ~/.foostash/config.yaml"`
	ProjectDir string `json:"project_dir,omitempty" jsonschema:"directory containing .foostash.yaml; defaults to current directory"`
	Project    string `json:"project,omitempty" jsonschema:"project slug override; defaults to .foostash.yaml"`
	Env        string `json:"env,omitempty" jsonschema:"environment slug; defaults to .foostash.yaml default_env"`
	DotenvPath string `json:"dotenv_path" jsonschema:"local dotenv file to read; values are encrypted locally and never returned"`
	SSHKeyPath string `json:"ssh_key_path,omitempty" jsonschema:"local SSH private key path; defaults like the CLI"`
}

type environmentSyncOutput struct {
	Project     string   `json:"project"`
	Env         string   `json:"env"`
	SourcePath  string   `json:"source_path"`
	TotalKeys   int      `json:"total_keys"`
	ChangedKeys []string `json:"changed_keys,omitempty"`
	Changed     int      `json:"changed"`
}

type remoteSecret struct {
	Key        string `json:"key"`
	Ciphertext []byte `json:"ciphertext"`
	Nonce      []byte `json:"nonce"`
	Version    int    `json:"version"`
}

func environmentSyncTool(ctx context.Context, _ *mcp.CallToolRequest, in environmentSyncInput) (*mcp.CallToolResult, environmentSyncOutput, error) {
	if strings.TrimSpace(in.DotenvPath) == "" {
		return nil, environmentSyncOutput{}, fmt.Errorf("dotenv_path is required")
	}
	pairs, err := parseDotenvFile(in.DotenvPath)
	if err != nil {
		return nil, environmentSyncOutput{}, err
	}
	serverURL, err := resolveServerURL(in.ServerURL)
	if err != nil {
		return nil, environmentSyncOutput{}, err
	}
	project, env, err := resolveProjectEnv(in.ProjectDir, in.Project, in.Env)
	if err != nil {
		return nil, environmentSyncOutput{}, err
	}
	_, signer, _, fingerprint, err := loadIdentityKey(in.SSHKeyPath)
	if err != nil {
		return nil, environmentSyncOutput{}, err
	}
	masterKey, err := crypto.LoadOrGenerateMasterKey()
	if err != nil {
		return nil, environmentSyncOutput{}, err
	}
	engine, err := crypto.NewEngine(masterKey)
	if err != nil {
		return nil, environmentSyncOutput{}, err
	}
	client := apiclient.New(serverURL, signer, fingerprint)

	current, err := pullRemoteSecrets(ctx, client, engine, project, env)
	if err != nil {
		return nil, environmentSyncOutput{}, err
	}
	type bulkItem struct {
		Key        string `json:"key"`
		Ciphertext []byte `json:"ciphertext"`
		Nonce      []byte `json:"nonce"`
	}
	var changed []string
	var items []bulkItem
	for key, value := range pairs {
		if current[key] == value {
			continue
		}
		ct, nonce, err := engine.Encrypt([]byte(value))
		if err != nil {
			return nil, environmentSyncOutput{}, err
		}
		changed = append(changed, key)
		items = append(items, bulkItem{Key: key, Ciphertext: ct, Nonce: nonce})
	}
	sort.Strings(changed)
	out := environmentSyncOutput{Project: project, Env: env, SourcePath: in.DotenvPath, TotalKeys: len(pairs), ChangedKeys: changed, Changed: len(changed)}
	if len(items) == 0 {
		return nil, out, nil
	}
	path := fmt.Sprintf("/v1/projects/%s/envs/%s/secrets", url.PathEscape(project), url.PathEscape(env))
	if err := client.Do(ctx, http.MethodPut, path, map[string]any{"secrets": items}, nil); err != nil {
		return nil, out, err
	}
	return nil, out, nil
}

func resolveProjectEnv(projectDir, project, env string) (string, string, error) {
	if project != "" && env != "" {
		return project, env, nil
	}
	dir := projectDir
	if dir == "" {
		dir = "."
	}
	cfg, err := config.LoadProject(dir)
	if err != nil {
		return "", "", err
	}
	if project == "" {
		project = cfg.Project
	}
	if env == "" {
		env = cfg.DefaultEnv
	}
	if project == "" || env == "" {
		return "", "", fmt.Errorf("project and env are required")
	}
	return project, env, nil
}

func pullRemoteSecrets(ctx context.Context, client *apiclient.Client, engine *crypto.Engine, project, env string) (map[string]string, error) {
	var out struct {
		Secrets []remoteSecret `json:"secrets"`
	}
	path := fmt.Sprintf("/v1/projects/%s/envs/%s/secrets", url.PathEscape(project), url.PathEscape(env))
	if err := client.Do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	result := make(map[string]string, len(out.Secrets))
	for _, s := range out.Secrets {
		pt, err := engine.Decrypt(s.Ciphertext, s.Nonce)
		if err != nil {
			return nil, fmt.Errorf("decrypt %s: %w", s.Key, err)
		}
		result[s.Key] = string(pt)
	}
	return result, nil
}

func parseDotenvFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	pairs := make(map[string]string)
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}
		if key != "" {
			pairs[key] = value
		}
	}
	return pairs, scanner.Err()
}

package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Omotolani98/foostash/internal/apiclient"
	"github.com/Omotolani98/foostash/internal/config"
	"github.com/Omotolani98/foostash/internal/service"
	"github.com/Omotolani98/foostash/internal/sshauth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/crypto/ssh"
)

type inspectInput struct {
	ServerURL  string `json:"server_url,omitempty" jsonschema:"optional Foostash server URL; defaults to ~/.foostash/config.yaml"`
	ProjectDir string `json:"project_dir,omitempty" jsonschema:"directory containing .foostash.yaml; defaults to current directory"`
}

type inspectOutput struct {
	ServerURL          string   `json:"server_url,omitempty"`
	ServerReachable    bool     `json:"server_reachable"`
	ServerStatus       string   `json:"server_status,omitempty"`
	ServerError        string   `json:"server_error,omitempty"`
	IdentityConfigured bool     `json:"identity_configured"`
	Email              string   `json:"email,omitempty"`
	KeyFingerprint     string   `json:"key_fingerprint,omitempty"`
	MasterKeyPresent   bool     `json:"master_key_present"`
	ProjectConfigured  bool     `json:"project_configured"`
	Project            string   `json:"project,omitempty"`
	DefaultEnv         string   `json:"default_env,omitempty"`
	Environments       []string `json:"environments,omitempty"`
}

func inspectTool(ctx context.Context, _ *mcp.CallToolRequest, in inspectInput) (*mcp.CallToolResult, inspectOutput, error) {
	out := inspectOutput{MasterKeyPresent: masterKeyPresent()}

	cfg, err := config.LoadGlobal()
	if err == nil {
		out.ServerURL = cfg.Server
		if cfg.Identity != nil {
			out.IdentityConfigured = true
			out.Email = cfg.Identity.Email
			out.KeyFingerprint = cfg.Identity.KeyFingerprint
		}
	}
	if in.ServerURL != "" {
		out.ServerURL = in.ServerURL
	}
	if out.ServerURL != "" {
		status, err := health(ctx, out.ServerURL)
		if err != nil {
			out.ServerError = err.Error()
		} else {
			out.ServerReachable = true
			out.ServerStatus = status
		}
	}

	projectDir := in.ProjectDir
	if projectDir == "" {
		projectDir = "."
	}
	if pcfg, err := config.LoadProject(projectDir); err == nil {
		out.ProjectConfigured = true
		out.Project = pcfg.Project
		out.DefaultEnv = pcfg.DefaultEnv
		out.Environments = append([]string(nil), pcfg.Environments...)
	}

	return nil, out, nil
}

type identitySetupInput struct {
	Mode        string `json:"mode" jsonschema:"register or join"`
	ServerURL   string `json:"server_url" jsonschema:"Foostash server URL"`
	Email       string `json:"email,omitempty" jsonschema:"email for register mode"`
	Org         string `json:"org,omitempty" jsonschema:"organization name for register mode"`
	InviteToken string `json:"invite_token,omitempty" jsonschema:"invite token for join mode"`
	SSHKeyPath  string `json:"ssh_key_path,omitempty" jsonschema:"local SSH private key path; defaults like the CLI"`
}

type identitySetupOutput struct {
	Mode           string `json:"mode"`
	ServerURL      string `json:"server_url"`
	Email          string `json:"email,omitempty"`
	Org            string `json:"org,omitempty"`
	Role           string `json:"role"`
	UserID         string `json:"user_id"`
	OrgID          string `json:"org_id"`
	SSHKeyPath     string `json:"ssh_key_path"`
	KeyFingerprint string `json:"key_fingerprint"`
	ConfigWritten  bool   `json:"config_written"`
}

func identitySetupTool(ctx context.Context, _ *mcp.CallToolRequest, in identitySetupInput) (*mcp.CallToolResult, identitySetupOutput, error) {
	mode := strings.ToLower(strings.TrimSpace(in.Mode))
	if mode != "register" && mode != "join" {
		return nil, identitySetupOutput{}, errors.New("mode must be register or join")
	}
	if strings.TrimSpace(in.ServerURL) == "" {
		return nil, identitySetupOutput{}, errors.New("server_url is required")
	}

	keyPath, signer, pubKey, fingerprint, err := loadIdentityKey(in.SSHKeyPath)
	if err != nil {
		return nil, identitySetupOutput{}, err
	}
	client := apiclient.New(in.ServerURL, signer, fingerprint)

	out := identitySetupOutput{Mode: mode, ServerURL: in.ServerURL, SSHKeyPath: keyPath, KeyFingerprint: fingerprint}
	if mode == "register" {
		if strings.TrimSpace(in.Email) == "" || strings.TrimSpace(in.Org) == "" {
			return nil, out, errors.New("email and org are required for register mode")
		}
		var resp struct {
			UserID string `json:"user_id"`
			OrgID  string `json:"org_id"`
			Role   string `json:"role"`
		}
		body := map[string]string{"email": in.Email, "org_name": in.Org, "public_key": pubKey}
		if err := client.Do(ctx, http.MethodPost, "/v1/auth/register", body, &resp); err != nil {
			return nil, out, err
		}
		out.Email, out.Org, out.Role, out.UserID, out.OrgID = in.Email, in.Org, resp.Role, resp.UserID, resp.OrgID
	} else {
		if strings.TrimSpace(in.InviteToken) == "" {
			return nil, out, errors.New("invite_token is required for join mode")
		}
		var resp struct {
			UserID string `json:"user_id"`
			OrgID  string `json:"org_id"`
			Email  string `json:"email"`
			Role   string `json:"role"`
		}
		body := map[string]string{"token": in.InviteToken, "public_key": pubKey}
		if err := client.Do(ctx, http.MethodPost, "/v1/admin/join", body, &resp); err != nil {
			return nil, out, err
		}
		out.Email, out.Role, out.UserID, out.OrgID = resp.Email, resp.Role, resp.UserID, resp.OrgID
	}

	cfg, err := config.LoadGlobal()
	if err != nil {
		return nil, out, err
	}
	cfg.Server = in.ServerURL
	cfg.Identity = &config.Identity{Email: out.Email, SSHKeyPath: keyPath, KeyFingerprint: fingerprint}
	if err := config.SaveGlobal(cfg); err != nil {
		return nil, out, err
	}
	out.ConfigWritten = true
	return nil, out, nil
}

type projectSetupInput struct {
	ServerURL   string   `json:"server_url,omitempty" jsonschema:"Foostash server URL; defaults to ~/.foostash/config.yaml"`
	ProjectName string   `json:"project_name" jsonschema:"project display name to create or reconnect"`
	ProjectDir  string   `json:"project_dir,omitempty" jsonschema:"directory where .foostash.yaml should be written; defaults to current directory"`
	Envs        []string `json:"envs,omitempty" jsonschema:"environment slugs to ensure; dev is always included"`
	SSHKeyPath  string   `json:"ssh_key_path,omitempty" jsonschema:"local SSH private key path; defaults like the CLI"`
}

type projectSetupOutput struct {
	Project       string   `json:"project"`
	Created       bool     `json:"created"`
	ConfigWritten bool     `json:"config_written"`
	Environments  []string `json:"environments"`
}

func projectSetupTool(ctx context.Context, _ *mcp.CallToolRequest, in projectSetupInput) (*mcp.CallToolResult, projectSetupOutput, error) {
	if strings.TrimSpace(in.ProjectName) == "" {
		return nil, projectSetupOutput{}, errors.New("project_name is required")
	}
	serverURL, err := resolveServerURL(in.ServerURL)
	if err != nil {
		return nil, projectSetupOutput{}, err
	}
	_, signer, _, fingerprint, err := loadIdentityKey(in.SSHKeyPath)
	if err != nil {
		return nil, projectSetupOutput{}, err
	}
	client := apiclient.New(serverURL, signer, fingerprint)

	projectSlug := service.Slugify(in.ProjectName)
	if projectSlug == "" {
		return nil, projectSetupOutput{}, errors.New("project_name does not produce a valid slug")
	}
	out := projectSetupOutput{Project: projectSlug}

	var created struct {
		Slug string `json:"slug"`
	}
	err = client.Do(ctx, http.MethodPost, "/v1/projects", map[string]string{"name": in.ProjectName}, &created)
	if err == nil {
		out.Created = true
		projectSlug = created.Slug
		out.Project = created.Slug
	} else if !apiCode(err, http.StatusConflict, "project_exists") {
		return nil, out, err
	}

	envs := normalizeEnvs(in.Envs)
	for _, env := range envs {
		if env == "dev" {
			continue
		}
		path := fmt.Sprintf("/v1/projects/%s/envs", url.PathEscape(projectSlug))
		err := client.Do(ctx, http.MethodPost, path, map[string]string{"slug": env}, nil)
		if err != nil && !apiCode(err, http.StatusConflict, "env_exists") {
			return nil, out, err
		}
	}

	projectDir := in.ProjectDir
	if projectDir == "" {
		projectDir = "."
	}
	if err := config.SaveProject(projectDir, &config.ProjectConfig{Project: projectSlug, DefaultEnv: "dev", Environments: envs}); err != nil {
		return nil, out, err
	}
	out.ConfigWritten = true
	out.Environments = envs
	return nil, out, nil
}

func health(ctx context.Context, serverURL string) (string, error) {
	base := strings.TrimRight(serverURL, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/v1/health", nil)
	if err != nil {
		return "", err
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("health returned status %d", resp.StatusCode)
	}
	return "ok", nil
}

func masterKeyPresent() bool {
	if strings.TrimSpace(os.Getenv("FOOSTASH_MASTER_KEY")) != "" {
		return true
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(home, ".foostash", "master.key"))
	return err == nil
}

func resolveServerURL(flag string) (string, error) {
	if strings.TrimSpace(flag) != "" {
		return flag, nil
	}
	cfg, err := config.LoadGlobal()
	if err != nil {
		return "", err
	}
	if cfg.Server == "" {
		return "", errors.New("server_url is required because ~/.foostash/config.yaml has no server")
	}
	return cfg.Server, nil
}

func loadIdentityKey(path string) (string, ssh.Signer, string, string, error) {
	keyPath, err := resolveKeyPath(path)
	if err != nil {
		return "", nil, "", "", err
	}
	signer, err := sshauth.LoadSigner(keyPath, nil)
	if err != nil {
		if sshauth.IsPassphraseError(err) {
			return "", nil, "", "", errors.New("passphrase-protected SSH keys are not supported by foostash mcp; provide an unencrypted service key")
		}
		return "", nil, "", "", err
	}
	pubKey, fingerprint, err := publicKeyAuthorized(keyPath, signer)
	if err != nil {
		return "", nil, "", "", err
	}
	return keyPath, signer, pubKey, fingerprint, nil
}

func resolveKeyPath(flag string) (string, error) {
	if flag != "" {
		return flag, nil
	}
	if v := os.Getenv("FOOSTASH_SSH_KEY"); v != "" {
		return v, nil
	}
	if cfg, err := config.LoadGlobal(); err == nil && cfg.Identity != nil && cfg.Identity.SSHKeyPath != "" {
		return cfg.Identity.SSHKeyPath, nil
	}
	return sshauth.ResolveDefaultKeyPath()
}

func publicKeyAuthorized(privPath string, signer ssh.Signer) (string, string, error) {
	pubPath := sshauth.PublicKeyForPrivate(privPath)
	if pub, err := sshauth.LoadPublicKey(pubPath); err == nil {
		return strings.TrimSpace(sshauth.MarshalAuthorizedKey(pub)), sshauth.Fingerprint(pub), nil
	}
	pub := signer.PublicKey()
	return strings.TrimSpace(sshauth.MarshalAuthorizedKey(pub)), sshauth.Fingerprint(pub), nil
}

func apiCode(err error, status int, code string) bool {
	var apiErr *apiclient.APIError
	return errors.As(err, &apiErr) && apiErr.Status == status && apiErr.Code == code
}

func normalizeEnvs(input []string) []string {
	seen := map[string]struct{}{"dev": {}}
	out := []string{"dev"}
	for _, env := range input {
		slug := service.Slugify(env)
		if slug == "" {
			continue
		}
		if _, ok := seen[slug]; ok {
			continue
		}
		seen[slug] = struct{}{}
		out = append(out, slug)
	}
	return out
}

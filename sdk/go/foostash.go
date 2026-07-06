// Package foostash is the Go SDK for the foostash secrets manager.
//
// Typical usage from a service at boot time:
//
//	client, err := foostash.New(foostash.Config{
//	    ServerURL:  "https://foostash.example.com",
//	    Project:    "payments-api",
//	    Env:        "prod",
//	    SSHKeyPath: "/var/run/secrets/foostash-key",
//	    MasterKey:  os.Getenv("FOOSTASH_MASTER_KEY"),
//	})
//	if err != nil { log.Fatal(err) }
//	secrets, err := client.Pull(ctx)
//
// The master key is the AES-256-GCM key the org encrypts under. Provision
// the SSH key and master key via your own secret delivery (KMS, env, file);
// the SDK itself never reads the user's ~/.foostash directory.
package foostash

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/Omotolani98/foostash/sdk/go/internal/apiclient"
	"github.com/Omotolani98/foostash/sdk/go/internal/sshauth"
)

// Sentinel errors returned by the SDK. Callers should branch on these with
// errors.Is rather than inspecting string messages.
var (
	ErrNotFound     = errors.New("foostash: not found")
	ErrUnauthorized = errors.New("foostash: unauthorized")
	ErrForbidden    = errors.New("foostash: forbidden")
)

// Config is the set of inputs required to construct a Client.
type Config struct {
	// ServerURL is the foostash server base URL, e.g. https://foostash.example.com.
	ServerURL string
	// Project is the project slug (as created by `foostash init`).
	Project string
	// Env is the environment name (e.g. "prod", "staging").
	Env string
	// SSHKeyPath is the path to an unencrypted ed25519 or RSA private key whose
	// public key is registered with the server.
	SSHKeyPath string
	// MasterKey is the base64-encoded 32-byte AES-256 key for the org.
	MasterKey string
	// HTTPTimeout overrides the default 30s per-request timeout when non-zero.
	HTTPTimeout time.Duration
}

// Client pulls and decrypts secrets from a foostash server.
type Client struct {
	project string
	env     string
	api     *apiclient.Client
	crypto  *engine
}

// Snapshot is a point-in-time view of an environment's secrets returned by Watch.
type Snapshot struct {
	Secrets  map[string]string
	Versions map[string]int
	PulledAt time.Time
}

// New validates the config, loads the SSH key, and returns a ready Client.
func New(cfg Config) (*Client, error) {
	if cfg.ServerURL == "" {
		return nil, errors.New("foostash: ServerURL is required")
	}
	if cfg.Project == "" {
		return nil, errors.New("foostash: Project is required")
	}
	if cfg.Env == "" {
		return nil, errors.New("foostash: Env is required")
	}
	if cfg.SSHKeyPath == "" {
		return nil, errors.New("foostash: SSHKeyPath is required")
	}
	if cfg.MasterKey == "" {
		return nil, errors.New("foostash: MasterKey is required")
	}

	signer, err := sshauth.LoadSigner(cfg.SSHKeyPath)
	if err != nil {
		return nil, fmt.Errorf("load ssh key: %w", err)
	}
	fp := sshauth.Fingerprint(signer.PublicKey())

	eng, err := newEngine(cfg.MasterKey)
	if err != nil {
		return nil, fmt.Errorf("build crypto engine: %w", err)
	}

	api := apiclient.New(cfg.ServerURL, signer, fp, cfg.HTTPTimeout)
	return &Client{
		project: cfg.Project,
		env:     cfg.Env,
		api:     api,
		crypto:  eng,
	}, nil
}

// secretDTO mirrors the server's JSON representation of a secret.
type secretDTO struct {
	Key        string `json:"key"`
	Ciphertext []byte `json:"ciphertext"`
	Nonce      []byte `json:"nonce"`
	Version    int    `json:"version"`
	UpdatedAt  string `json:"updated_at"`
	UpdatedBy  string `json:"updated_by,omitempty"`
}

// Get returns a single decrypted secret by key.
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	if key == "" {
		return "", errors.New("foostash: key is required")
	}
	var out secretDTO
	path := fmt.Sprintf("/v1/projects/%s/envs/%s/secrets/%s",
		url.PathEscape(c.project), url.PathEscape(c.env), url.PathEscape(key))
	if err := c.api.Do(ctx, "GET", path, nil, &out); err != nil {
		return "", mapError(err)
	}
	pt, err := c.crypto.Decrypt(out.Ciphertext, out.Nonce)
	if err != nil {
		return "", fmt.Errorf("decrypt %s: %w", key, err)
	}
	return string(pt), nil
}

// Pull returns every secret in the configured project+env, decrypted.
func (c *Client) Pull(ctx context.Context) (map[string]string, error) {
	snap, err := c.pullSnapshot(ctx)
	if err != nil {
		return nil, err
	}
	return snap.Secrets, nil
}

// pullSnapshot is the shared implementation used by both Pull and Watch.
func (c *Client) pullSnapshot(ctx context.Context) (Snapshot, error) {
	var out struct {
		Secrets []secretDTO `json:"secrets"`
	}
	path := fmt.Sprintf("/v1/projects/%s/envs/%s/secrets",
		url.PathEscape(c.project), url.PathEscape(c.env))
	if err := c.api.Do(ctx, "GET", path, nil, &out); err != nil {
		return Snapshot{}, mapError(err)
	}
	secrets := make(map[string]string, len(out.Secrets))
	versions := make(map[string]int, len(out.Secrets))
	for _, s := range out.Secrets {
		pt, err := c.crypto.Decrypt(s.Ciphertext, s.Nonce)
		if err != nil {
			return Snapshot{}, fmt.Errorf("decrypt %s: %w", s.Key, err)
		}
		secrets[s.Key] = string(pt)
		versions[s.Key] = s.Version
	}
	return Snapshot{
		Secrets:  secrets,
		Versions: versions,
		PulledAt: time.Now().UTC(),
	}, nil
}

// mapError converts an apiclient.APIError into an SDK sentinel where possible.
// Anything else is returned unchanged so callers can inspect it with errors.As.
func mapError(err error) error {
	var apiErr *apiclient.APIError
	if !errors.As(err, &apiErr) {
		return err
	}
	switch apiErr.Status {
	case 401:
		return fmt.Errorf("%w: %s", ErrUnauthorized, apiErr.Message)
	case 403:
		return fmt.Errorf("%w: %s", ErrForbidden, apiErr.Message)
	case 404:
		return fmt.Errorf("%w: %s", ErrNotFound, apiErr.Message)
	default:
		return err
	}
}

package cli

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/Omotolani98/foostash/internal/apiclient"
	"github.com/Omotolani98/foostash/internal/crypto"
)

// remoteSecretDTO mirrors handlers.secretDTO. []byte fields transit as base64.
type remoteSecretDTO struct {
	Key        string `json:"key"`
	Ciphertext []byte `json:"ciphertext"`
	Nonce      []byte `json:"nonce"`
	Version    int    `json:"version"`
	UpdatedAt  string `json:"updated_at"`
	UpdatedBy  string `json:"updated_by,omitempty"`
}

type remoteHistoryDTO struct {
	Version    int    `json:"version"`
	Ciphertext []byte `json:"ciphertext"`
	Nonce      []byte `json:"nonce"`
	SetAt      string `json:"set_at"`
	SetBy      string `json:"set_by,omitempty"`
}

// remoteSet encrypts value locally and PUTs it to the server.
func remoteSet(ctx context.Context, c *apiclient.Client, eng *crypto.Engine, project, env, key, value string) (*remoteSecretDTO, error) {
	ct, nonce, err := eng.Encrypt([]byte(value))
	if err != nil {
		return nil, fmt.Errorf("encrypt: %w", err)
	}
	body := map[string][]byte{"ciphertext": ct, "nonce": nonce}
	var out remoteSecretDTO
	path := fmt.Sprintf("/v1/projects/%s/envs/%s/secrets/%s",
		url.PathEscape(project), url.PathEscape(env), url.PathEscape(key))
	if err := c.Do(ctx, "PUT", path, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// remoteGet fetches and decrypts a single secret.
func remoteGet(ctx context.Context, c *apiclient.Client, eng *crypto.Engine, project, env, key string) (string, error) {
	var out remoteSecretDTO
	path := fmt.Sprintf("/v1/projects/%s/envs/%s/secrets/%s",
		url.PathEscape(project), url.PathEscape(env), url.PathEscape(key))
	if err := c.Do(ctx, "GET", path, nil, &out); err != nil {
		return "", err
	}
	pt, err := eng.Decrypt(out.Ciphertext, out.Nonce)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}
	return string(pt), nil
}

// remoteList fetches all secrets for an env and decrypts them.
func remoteList(ctx context.Context, c *apiclient.Client, eng *crypto.Engine, project, env string) (map[string]string, error) {
	var out struct {
		Secrets []remoteSecretDTO `json:"secrets"`
	}
	path := fmt.Sprintf("/v1/projects/%s/envs/%s/secrets",
		url.PathEscape(project), url.PathEscape(env))
	if err := c.Do(ctx, "GET", path, nil, &out); err != nil {
		return nil, err
	}
	result := make(map[string]string, len(out.Secrets))
	for _, s := range out.Secrets {
		pt, err := eng.Decrypt(s.Ciphertext, s.Nonce)
		if err != nil {
			return nil, fmt.Errorf("decrypt %s: %w", s.Key, err)
		}
		result[s.Key] = string(pt)
	}
	return result, nil
}

// remoteDelete soft-deletes a secret on the server.
func remoteDelete(ctx context.Context, c *apiclient.Client, project, env, key string) error {
	path := fmt.Sprintf("/v1/projects/%s/envs/%s/secrets/%s",
		url.PathEscape(project), url.PathEscape(env), url.PathEscape(key))
	return c.Do(ctx, "DELETE", path, nil, nil)
}

// remoteHistoryEntry is the decrypted view used by the CLI.
type remoteHistoryEntry struct {
	Version int
	Value   string
	SetAt   time.Time
}

func remoteHistory(ctx context.Context, c *apiclient.Client, eng *crypto.Engine, project, env, key string) ([]remoteHistoryEntry, error) {
	var out struct {
		History []remoteHistoryDTO `json:"history"`
	}
	path := fmt.Sprintf("/v1/projects/%s/envs/%s/secrets/%s/history",
		url.PathEscape(project), url.PathEscape(env), url.PathEscape(key))
	if err := c.Do(ctx, "GET", path, nil, &out); err != nil {
		return nil, err
	}
	entries := make([]remoteHistoryEntry, 0, len(out.History))
	for _, h := range out.History {
		pt, err := eng.Decrypt(h.Ciphertext, h.Nonce)
		if err != nil {
			return nil, fmt.Errorf("decrypt v%d: %w", h.Version, err)
		}
		t, _ := time.Parse("2006-01-02T15:04:05Z", h.SetAt)
		entries = append(entries, remoteHistoryEntry{Version: h.Version, Value: string(pt), SetAt: t})
	}
	return entries, nil
}

func remoteRollback(ctx context.Context, c *apiclient.Client, project, env, key string, version int) error {
	path := fmt.Sprintf("/v1/projects/%s/envs/%s/secrets/%s/rollback",
		url.PathEscape(project), url.PathEscape(env), url.PathEscape(key))
	body := map[string]int{"version": version}
	return c.Do(ctx, "POST", path, body, nil)
}

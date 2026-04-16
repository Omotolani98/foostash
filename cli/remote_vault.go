package cli

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/Omotolani98/foostash/internal/apiclient"
	"github.com/Omotolani98/foostash/internal/crypto"
)

// remoteVaultDTO mirrors handlers.vaultDTO. []byte fields transit as base64.
type remoteVaultDTO struct {
	Key        string `json:"key"`
	Ciphertext []byte `json:"ciphertext"`
	Nonce      []byte `json:"nonce"`
	Version    int    `json:"version"`
	UpdatedAt  string `json:"updated_at"`
	UpdatedBy  string `json:"updated_by,omitempty"`
}

// remoteVaultSet encrypts value locally and PUTs it to /v1/vault/{key}.
func remoteVaultSet(ctx context.Context, c *apiclient.Client, eng *crypto.Engine, key, value string) (*remoteVaultDTO, error) {
	ct, nonce, err := eng.Encrypt([]byte(value))
	if err != nil {
		return nil, fmt.Errorf("encrypt: %w", err)
	}
	body := map[string][]byte{"ciphertext": ct, "nonce": nonce}
	var out remoteVaultDTO
	path := fmt.Sprintf("/v1/vault/%s", url.PathEscape(key))
	if err := c.Do(ctx, "PUT", path, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// remoteVaultGet fetches and decrypts a single vault secret.
func remoteVaultGet(ctx context.Context, c *apiclient.Client, eng *crypto.Engine, key string) (string, error) {
	var out remoteVaultDTO
	path := fmt.Sprintf("/v1/vault/%s", url.PathEscape(key))
	if err := c.Do(ctx, "GET", path, nil, &out); err != nil {
		return "", err
	}
	pt, err := eng.Decrypt(out.Ciphertext, out.Nonce)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}
	return string(pt), nil
}

// remoteVaultList fetches all vault secrets for the caller's org and decrypts them.
func remoteVaultList(ctx context.Context, c *apiclient.Client, eng *crypto.Engine) (map[string]string, error) {
	var out struct {
		Secrets []remoteVaultDTO `json:"secrets"`
	}
	if err := c.Do(ctx, "GET", "/v1/vault", nil, &out); err != nil {
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

// remoteVaultDelete soft-deletes a vault secret on the server.
func remoteVaultDelete(ctx context.Context, c *apiclient.Client, key string) error {
	path := fmt.Sprintf("/v1/vault/%s", url.PathEscape(key))
	return c.Do(ctx, "DELETE", path, nil, nil)
}

func remoteVaultHistory(ctx context.Context, c *apiclient.Client, eng *crypto.Engine, key string) ([]remoteHistoryEntry, error) {
	var out struct {
		History []remoteHistoryDTO `json:"history"`
	}
	path := fmt.Sprintf("/v1/vault/%s/history", url.PathEscape(key))
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

func remoteVaultRollback(ctx context.Context, c *apiclient.Client, key string, version int) error {
	path := fmt.Sprintf("/v1/vault/%s/rollback", url.PathEscape(key))
	body := map[string]int{"version": version}
	return c.Do(ctx, "POST", path, body, nil)
}

package foostash

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

var ErrKeyNotFound = errors.New("foostash: key not found")

type Client struct {
	cfg   Config
	http  *http.Client
	cache *ttlCache
}

func New(cfg Config) (*Client, error) {
	if err := cfg.applyDefaults(); err != nil {
		return nil, err
	}
	return &Client{
		cfg:   cfg,
		http:  &http.Client{Timeout: 15 * time.Second},
		cache: newTTLCache(cfg.CacheTTL),
	}, nil
}

type pullResponse struct {
	Project     string            `json:"project"`
	Environment string            `json:"environment"`
	Secrets     map[string]string `json:"secrets"`
	Version     int               `json:"version"`
}

type errorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (c *Client) GetAll(ctx context.Context) (map[string]string, error) {
	if cached, ok := c.cache.get(); ok {
		return copyMap(cached), nil
	}

	url := fmt.Sprintf("%s/v1/projects/%s/envs/%s/secrets", c.cfg.Server, c.cfg.ProjectID, c.cfg.Env)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.Token)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("foostash: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		var env errorEnvelope
		if json.Unmarshal(body, &env) == nil && env.Error.Message != "" {
			return nil, fmt.Errorf("foostash: %s: %s", env.Error.Code, env.Error.Message)
		}
		return nil, fmt.Errorf("foostash: http %d", resp.StatusCode)
	}

	var out pullResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("foostash: decode response: %w", err)
	}
	c.cache.set(out.Secrets)
	return copyMap(out.Secrets), nil
}

func (c *Client) Get(ctx context.Context, key string) (string, error) {
	secrets, err := c.GetAll(ctx)
	if err != nil {
		return "", err
	}
	v, ok := secrets[key]
	if !ok {
		return "", ErrKeyNotFound
	}
	return v, nil
}

func (c *Client) MustGet(ctx context.Context, key string) string {
	v, err := c.Get(ctx, key)
	if err != nil {
		panic(err)
	}
	return v
}

func (c *Client) Refresh() {
	c.cache.invalidate()
}

func copyMap(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

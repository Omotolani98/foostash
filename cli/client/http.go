package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	base  string
	token string
	http  *http.Client
}

func New(base, token string) *Client {
	return &Client{
		base:  base,
		token: token,
		http:  &http.Client{Timeout: 30 * time.Second},
	}
}

type APIError struct {
	Status  int    `json:"-"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

type errorEnvelope struct {
	Error APIError `json:"error"`
}

func (c *Client) do(method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.base+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		var env errorEnvelope
		if err := json.Unmarshal(respBody, &env); err == nil && env.Error.Code != "" {
			env.Error.Status = resp.StatusCode
			return &env.Error
		}
		return fmt.Errorf("http %d: %s", resp.StatusCode, string(respBody))
	}
	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

type AuthResponse struct {
	Token  string `json:"token"`
	UserID string `json:"user_id"`
	OrgID  string `json:"org_id"`
	Email  string `json:"email"`
}

func (c *Client) Register(orgName, email, password string) (*AuthResponse, error) {
	var out AuthResponse
	err := c.do("POST", "/v1/auth/register", map[string]string{
		"org_name": orgName, "email": email, "password": password,
	}, &out)
	return &out, err
}

func (c *Client) Login(email, password string) (*AuthResponse, error) {
	var out AuthResponse
	err := c.do("POST", "/v1/auth/login", map[string]string{
		"email": email, "password": password,
	}, &out)
	return &out, err
}

type APIKeyResponse struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Prefix string   `json:"prefix"`
	Scopes []string `json:"scopes"`
	Key    string   `json:"key"`
}

func (c *Client) CreateAPIKey(name string, scopes []string) (*APIKeyResponse, error) {
	var out APIKeyResponse
	err := c.do("POST", "/v1/auth/api-keys", map[string]any{
		"name": name, "scopes": scopes,
	}, &out)
	return &out, err
}

type Project struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

func (c *Client) CreateProject(slug, name string) (*Project, error) {
	var out Project
	err := c.do("POST", "/v1/projects", map[string]string{"slug": slug, "name": name}, &out)
	return &out, err
}

type Environment struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
}

func (c *Client) CreateEnv(projectSlug, envSlug string) (*Environment, error) {
	var out Environment
	err := c.do("POST", fmt.Sprintf("/v1/projects/%s/envs", projectSlug), map[string]string{"slug": envSlug}, &out)
	return &out, err
}

type PullResponse struct {
	Project     string            `json:"project"`
	Environment string            `json:"environment"`
	Secrets     map[string]string `json:"secrets"`
	Version     int               `json:"version"`
}

func (c *Client) Pull(projectSlug, env string) (*PullResponse, error) {
	var out PullResponse
	err := c.do("GET", fmt.Sprintf("/v1/projects/%s/envs/%s/secrets", projectSlug, env), nil, &out)
	return &out, err
}

type SetResponse struct {
	Project     string   `json:"project"`
	Environment string   `json:"environment"`
	Set         []string `json:"set"`
	Version     int      `json:"version"`
}

func (c *Client) Set(projectSlug, env string, secrets map[string]string) (*SetResponse, error) {
	var out SetResponse
	err := c.do("POST", fmt.Sprintf("/v1/projects/%s/envs/%s/secrets", projectSlug, env),
		map[string]any{"secrets": secrets}, &out)
	return &out, err
}

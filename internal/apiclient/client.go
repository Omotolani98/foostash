package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Omotolani98/foostash/internal/sshauth"
	"golang.org/x/crypto/ssh"
)

type Client struct {
	baseURL     string
	signer      ssh.Signer
	fingerprint string
	http        *http.Client
}

func New(baseURL string, signer ssh.Signer, fingerprint string) *Client {
	return &Client{
		baseURL:     strings.TrimRight(baseURL, "/"),
		signer:      signer,
		fingerprint: fingerprint,
		http:        &http.Client{Timeout: 30 * time.Second},
	}
}

// Do issues a signed request. reqBody is JSON-marshalled (nil → no body).
// If respOut is non-nil and the response is 2xx, the body is decoded into it.
func (c *Client) Do(ctx context.Context, method, path string, reqBody any, respOut any) error {
	var bodyBytes []byte
	if reqBody != nil {
		b, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		bodyBytes = b
	}

	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if bodyBytes != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	sig, ts, err := sshauth.Sign(c.signer, method, path, bodyBytes)
	if err != nil {
		return err
	}
	req.Header.Set(sshauth.HeaderTimestamp, ts)
	req.Header.Set(sshauth.HeaderFingerprint, c.fingerprint)
	req.Header.Set(sshauth.HeaderSignature, sig)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var env errorEnvelope
		_ = json.Unmarshal(respBytes, &env)
		return &APIError{
			Status:  resp.StatusCode,
			Code:    env.Error.Code,
			Message: env.Error.Message,
		}
	}

	if respOut != nil && len(respBytes) > 0 {
		if err := json.Unmarshal(respBytes, respOut); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

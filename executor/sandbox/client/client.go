package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/voocel/mas/executor/sandbox"
)

type HTTPClient struct {
	Endpoint  string
	Client    *http.Client
	AuthToken string
}

func NewHTTPClient(endpoint string) *HTTPClient {
	return &HTTPClient{
		Endpoint: strings.TrimRight(endpoint, "/"),
		Client:   &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *HTTPClient) CreateSandbox(ctx context.Context, req sandbox.CreateSandboxRequest) (*sandbox.CreateSandboxResponse, error) {
	var resp sandbox.CreateSandboxResponse
	if err := c.doJSON(ctx, http.MethodPost, "/v1/sandbox/create", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *HTTPClient) ExecuteTool(ctx context.Context, req sandbox.ExecuteToolRequest) (*sandbox.ExecuteToolResponse, error) {
	var resp sandbox.ExecuteToolResponse
	if err := c.doJSON(ctx, http.MethodPost, "/v1/sandbox/execute", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *HTTPClient) DestroySandbox(ctx context.Context, req sandbox.DestroySandboxRequest) (*sandbox.DestroySandboxResponse, error) {
	var resp sandbox.DestroySandboxResponse
	if err := c.doJSON(ctx, http.MethodPost, "/v1/sandbox/destroy", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *HTTPClient) Health(ctx context.Context) (*sandbox.HealthResponse, error) {
	var resp sandbox.HealthResponse
	if err := c.doJSON(ctx, http.MethodGet, "/v1/sandbox/health", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *HTTPClient) Close() error { return nil }

func (c *HTTPClient) doJSON(ctx context.Context, method, path string, in any, out any) error {
	if c == nil {
		return errors.New("sandbox client is nil")
	}
	if c.Client == nil {
		c.Client = &http.Client{Timeout: 30 * time.Second}
	}
	base := strings.TrimRight(c.Endpoint, "/")
	if base == "" {
		return errors.New("sandbox endpoint is empty")
	}
	if strings.HasPrefix(base, "unix://") {
		return errors.New("unix transport is not supported")
	}
	fullURL, err := url.JoinPath(base, path)
	if err != nil {
		return err
	}

	var body io.Reader
	if in != nil {
		data, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if strings.TrimSpace(c.AuthToken) != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		if len(data) > 0 {
			return fmt.Errorf("sandbox http error: %s: %s", resp.Status, strings.TrimSpace(string(data)))
		}
		return fmt.Errorf("sandbox http error: %s", resp.Status)
	}

	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

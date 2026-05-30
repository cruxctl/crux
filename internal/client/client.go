package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cruxctl/crux/pkg/cruxapi"
)

type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

type APIError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("api error %d %s: %s", e.StatusCode, e.Code, e.Message)
}

func New(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		http: &http.Client{
			Timeout: 20 * time.Minute,
		},
	}
}

func (c *Client) Health(ctx context.Context) error {
	var out map[string]any
	return c.do(ctx, http.MethodGet, "/healthz", nil, &out)
}

func (c *Client) Version(ctx context.Context) (string, error) {
	var out struct {
		Version string `json:"version"`
	}
	if err := c.do(ctx, http.MethodGet, "/v1/version", nil, &out); err != nil {
		return "", err
	}
	return out.Version, nil
}

func (c *Client) RuntimeConfig(ctx context.Context) (cruxapi.RuntimeConfig, error) {
	var out struct {
		Runtime cruxapi.RuntimeConfig `json:"runtime"`
	}
	if err := c.do(ctx, http.MethodGet, "/v1/config", nil, &out); err != nil {
		return cruxapi.RuntimeConfig{}, err
	}
	return out.Runtime, nil
}

func (c *Client) UpdateRuntimeConfig(ctx context.Context, patch cruxapi.RuntimeConfigPatch) (cruxapi.RuntimeConfig, error) {
	var out cruxapi.RuntimeConfig
	if err := c.do(ctx, http.MethodPatch, "/v1/config/runtime", patch, &out); err != nil {
		return cruxapi.RuntimeConfig{}, err
	}
	return out, nil
}

func (c *Client) ListExecutions(ctx context.Context) ([]cruxapi.Execution, error) {
	var out []cruxapi.Execution
	return out, c.do(ctx, http.MethodGet, "/v1/executions", nil, &out)
}

func (c *Client) Events(ctx context.Context, executionID string) ([]cruxapi.Event, error) {
	path := "/v1/events"
	if executionID != "" {
		path = "/v1/executions/" + url.PathEscape(executionID) + "/events"
	}
	var out []cruxapi.Event
	return out, c.do(ctx, http.MethodGet, path, nil, &out)
}

func (c *Client) do(ctx context.Context, method, path string, in, out any) error {
	if strings.TrimSpace(c.baseURL) == "" {
		return fmt.Errorf("server URL is required")
	}
	var body io.Reader
	if in != nil {
		data, err := json.Marshal(in)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, c.baseURL+path, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		var parsed struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.Unmarshal(data, &parsed)
		if parsed.Error.Code == "" {
			parsed.Error.Code = resp.Status
		}
		if parsed.Error.Message == "" {
			parsed.Error.Message = strings.TrimSpace(string(data))
		}
		return &APIError{StatusCode: resp.StatusCode, Code: parsed.Error.Code, Message: parsed.Error.Message}
	}
	if out == nil || len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

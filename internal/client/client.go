package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cruxctl/crux/internal/discovery"
	"github.com/cruxctl/crux/internal/domain"
	"github.com/cruxctl/crux/internal/service"
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
			Timeout: 30 * time.Second,
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

func (c *Client) RuntimeConfig(ctx context.Context) (domain.RuntimeConfig, error) {
	var out struct {
		Runtime domain.RuntimeConfig `json:"runtime"`
	}
	if err := c.do(ctx, http.MethodGet, "/v1/config", nil, &out); err != nil {
		return domain.RuntimeConfig{}, err
	}
	return out.Runtime, nil
}

func (c *Client) UpdateRuntimeConfig(ctx context.Context, patch domain.RuntimeConfigPatch) (domain.RuntimeConfig, error) {
	var out domain.RuntimeConfig
	if err := c.do(ctx, http.MethodPatch, "/v1/config/runtime", patch, &out); err != nil {
		return domain.RuntimeConfig{}, err
	}
	return out, nil
}

func (c *Client) ListAgents(ctx context.Context) ([]domain.Agent, error) {
	var out []domain.Agent
	return out, c.do(ctx, http.MethodGet, "/v1/agents", nil, &out)
}

func (c *Client) UpsertAgent(ctx context.Context, agent domain.Agent) (domain.Agent, error) {
	var out domain.Agent
	return out, c.do(ctx, http.MethodPost, "/v1/agents", agent, &out)
}

func (c *Client) DeleteAgent(ctx context.Context, name string) error {
	return c.do(ctx, http.MethodDelete, "/v1/agents/"+name, nil, nil)
}

func (c *Client) Discover(ctx context.Context) ([]discovery.Result, error) {
	var out []discovery.Result
	return out, c.do(ctx, http.MethodPost, "/v1/discover", map[string]any{}, &out)
}

func (c *Client) Run(ctx context.Context, req service.SubmitRequest) (domain.Execution, error) {
	var out domain.Execution
	return out, c.do(ctx, http.MethodPost, "/v1/executions", req, &out)
}

func (c *Client) ListExecutions(ctx context.Context) ([]domain.Execution, error) {
	var out []domain.Execution
	return out, c.do(ctx, http.MethodGet, "/v1/executions", nil, &out)
}

func (c *Client) GetExecution(ctx context.Context, id string) (domain.Execution, error) {
	var out domain.Execution
	return out, c.do(ctx, http.MethodGet, "/v1/executions/"+id, nil, &out)
}

func (c *Client) Events(ctx context.Context, executionID string) ([]domain.Event, error) {
	path := "/v1/events"
	if executionID != "" {
		path = "/v1/executions/" + executionID + "/events"
	}
	var out []domain.Event
	return out, c.do(ctx, http.MethodGet, path, nil, &out)
}

func (c *Client) do(ctx context.Context, method, path string, in, out any) error {
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
		return err
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

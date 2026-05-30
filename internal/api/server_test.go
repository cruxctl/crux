package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cruxctl/crux/internal/config"
)

func TestRoutes_RegisteredPathsExist(t *testing.T) {
	cfg := config.Defaults()
	srv := NewServer(cfg, nil, nil)
	h := srv.Handler()

	cases := []string{
		"/healthz", "/v1/version", "/v1/openapi.json",
		"/v1/config", "/v1/agents",
		"/v1/sessions", "/v1/gateway/status", "/v1/mcp/servers",
		"/v1/policies", "/v1/aos/events", "/v1/agbom/x",
		"/v1/costs", "/v1/audit", "/v1/machines", "/v1/metrics",
		"/v1/agents/refresh",
	}
	for _, path := range cases {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code == http.StatusNotFound {
			t.Errorf("route %s missing (404)", path)
		}
	}
	_ = context.Background()
}

func TestAgentsRefresh_CallsReload(t *testing.T) {
	cfg := config.Defaults()
	srv := NewServer(cfg, nil, nil)
	called := false
	srv.WithAgentReload(func() error {
		called = true
		return nil
	})
	h := srv.Handler()

	req := httptest.NewRequest(http.MethodPost, "/v1/agents/refresh", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !called {
		t.Error("expected reload callback to be called")
	}
}

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cruxctl/crux/internal/config"
)

func TestOpenAPI_BasicShape(t *testing.T) {
	srv := NewServer(config.Defaults(), nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/v1/openapi.json", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("code = %d", w.Code)
	}
	var doc map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if doc["openapi"] != "3.0.3" {
		t.Errorf("openapi field = %v", doc["openapi"])
	}
	info, ok := doc["info"].(map[string]any)
	if !ok || info["title"] != "Crux Control cruxd API" {
		t.Errorf("info.title = %v", doc["info"])
	}
	paths, ok := doc["paths"].(map[string]any)
	if !ok || len(paths) == 0 {
		t.Errorf("paths missing or empty")
	}
	if _, has := paths["/v1/sessions"]; !has {
		t.Errorf("expected /v1/sessions in paths")
	}
}

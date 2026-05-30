package api

import "net/http"

// handleOpenAPI returns a hand-curated OpenAPI 3.0.3 document. As subsystems
// land, expand the `paths` map with full request/response schemas. This stub
// publishes the *surface* so crux-console can codegen against it.
func (s *Server) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	doc := map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":   "Crux Control cruxd API",
			"version": "0.2.0",
		},
		"servers": []any{
			map[string]any{"url": "http://127.0.0.1:4357"},
		},
		"paths": map[string]any{
			"/healthz":                    stubGet("healthz"),
			"/v1/version":                 stubGet("version"),
			"/v1/config":                  stubGet("config"),
			"/v1/agents":                  stubGet("agents.list"),
			"/v1/discover":                stubPost("discover"),
			"/v1/discover/inject-gateway": stubPost("inject-gateway"),
			"/v1/sessions":                stubBoth("sessions"),
			"/v1/gateway/status":          stubGet("gateway.status"),
			"/v1/gateway/routes":          stubBoth("gateway.routes"),
			"/v1/mcp/servers":             stubBoth("mcp.servers"),
			"/v1/mcp/tools":               stubGet("mcp.tools"),
			"/v1/mcp/calls":               stubGet("mcp.calls"),
			"/v1/policies":                stubBoth("policies"),
			"/v1/approvals":               stubGet("approvals"),
			"/v1/aos/events":              stubGet("aos.events"),
			"/v1/aos/export":              stubPost("aos.export"),
			"/v1/aos/traces":              stubGet("aos.traces"),
			"/v1/agbom/generate":          stubPost("agbom.generate"),
			"/v1/costs":                   stubGet("costs"),
			"/v1/usage":                   stubGet("usage"),
			"/v1/usage/limits":            stubBoth("usage.limits"),
			"/v1/audit":                   stubGet("audit"),
			"/v1/machines":                stubGet("machines"),
			"/v1/machines/pair":           stubPost("machines.pair"),
			"/v1/metrics":                 stubGet("metrics"),
		},
	}
	writeJSON(w, http.StatusOK, doc)
}

func stubGet(opID string) map[string]any {
	return map[string]any{"get": map[string]any{
		"operationId": opID,
		"responses":   map[string]any{"200": map[string]any{"description": "ok"}},
	}}
}

func stubPost(opID string) map[string]any {
	return map[string]any{"post": map[string]any{
		"operationId": opID,
		"responses":   map[string]any{"200": map[string]any{"description": "ok"}},
	}}
}

func stubBoth(opID string) map[string]any {
	return map[string]any{
		"get":  map[string]any{"operationId": opID + ".list", "responses": map[string]any{"200": map[string]any{"description": "ok"}}},
		"post": map[string]any{"operationId": opID + ".create", "responses": map[string]any{"201": map[string]any{"description": "created"}}},
	}
}

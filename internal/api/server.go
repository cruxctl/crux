// Package api hosts the cruxd HTTP API. Routes are versioned at /v1/*.
// Auth (api_key, mtls, oidc) is layered as middleware; stub handlers in
// this plan return 501 Not Implemented for not-yet-built subsystems.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/cruxctl/crux/internal/config"
	"github.com/cruxctl/crux/internal/service"
	"github.com/cruxctl/crux/internal/state"
	"github.com/cruxctl/crux/pkg/cruxapi"
)

type Server struct {
	cfg          config.DaemonConfig
	service      *service.Service
	logger       *slog.Logger
	mux          *http.ServeMux
	reloadAgents func() error
}

func NewServer(cfg config.DaemonConfig, svc *service.Service, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	s := &Server{cfg: cfg, service: svc, logger: logger, mux: http.NewServeMux()}
	s.register()
	return s
}

// WithAgentReload registers a callback used by POST /v1/agents/refresh.
func (s *Server) WithAgentReload(fn func() error) *Server {
	s.reloadAgents = fn
	return s
}

func (s *Server) Handler() http.Handler { return s.withMiddleware(s.mux) }

func (s *Server) register() {
	// Health + meta
	s.mux.HandleFunc("/healthz", s.handleHealth)
	s.mux.HandleFunc("/v1/version", s.handleVersion)
	s.mux.HandleFunc("/v1/openapi.json", s.handleOpenAPI)

	// Config
	s.mux.HandleFunc("/v1/config", s.handleConfig)
	s.mux.HandleFunc("/v1/config/runtime", s.handleConfigRuntime)

	// Agents (existing surface — real impl carried over from current cruxd)
	s.mux.HandleFunc("/v1/agents", s.handleAgents)
	s.mux.HandleFunc("/v1/agents/", s.handleAgentsTree)

	// Agent refresh
	s.mux.HandleFunc("/v1/agents/refresh", s.handleAgentsRefresh)

	// Discovery
	s.mux.HandleFunc("/v1/discover", s.notImplemented("discover"))
	s.mux.HandleFunc("/v1/discover/inject-gateway", s.notImplemented("discover.inject-gateway"))
	s.mux.HandleFunc("/v1/discover/inject-gateway/undo", s.notImplemented("discover.inject-gateway.undo"))

	// Sessions
	s.mux.HandleFunc("/v1/sessions", s.handleSessions)
	s.mux.HandleFunc("/v1/sessions/", s.handleSessionsTree)

	// Projects
	s.mux.HandleFunc("/v1/projects", s.notImplemented("projects"))
	s.mux.HandleFunc("/v1/projects/", s.notImplemented("projects.tree"))

	// Gateway
	s.mux.HandleFunc("/v1/gateway/status", s.handleGatewayStatus)
	s.mux.HandleFunc("/v1/gateway/routes", s.handleGatewayRoutes)

	// MCP
	s.mux.HandleFunc("/v1/mcp/servers", s.handleMCPServers)
	s.mux.HandleFunc("/v1/mcp/servers/", s.handleMCPServersTree)
	s.mux.HandleFunc("/v1/mcp/tools", s.handleMCPTools)
	s.mux.HandleFunc("/v1/mcp/calls", s.handleMCPCalls)
	s.mux.HandleFunc("/v1/mcp/agent/", s.handleMCPAgentProxy)

	// Policy + Approvals
	s.mux.HandleFunc("/v1/policies", s.handlePolicies)
	s.mux.HandleFunc("/v1/policies/", s.handlePoliciesTree)
	s.mux.HandleFunc("/v1/approvals", s.handleApprovals)
	s.mux.HandleFunc("/v1/approvals/", s.handleApprovalsTree)

	// AOS
	s.mux.HandleFunc("/v1/aos/events", s.handleAOSEvents)
	s.mux.HandleFunc("/v1/aos/events/", s.handleAOSEventsTree)
	s.mux.HandleFunc("/v1/aos/export", s.handleAOSExport)
	s.mux.HandleFunc("/v1/aos/traces", s.handleAOSTraces)
	s.mux.HandleFunc("/v1/aos/traces/", s.handleAOSTracesTree)

	// AgBOM
	s.mux.HandleFunc("/v1/agbom/generate", s.handleAgBOMGenerate)
	s.mux.HandleFunc("/v1/agbom/", s.handleAgBOMTree)

	// Costs, Usage
	s.mux.HandleFunc("/v1/costs", s.handleCosts)
	s.mux.HandleFunc("/v1/usage", s.handleUsage)
	s.mux.HandleFunc("/v1/usage/limits", s.handleUsageLimits)

	// Audit
	s.mux.HandleFunc("/v1/audit", s.handleAudit)
	s.mux.HandleFunc("/v1/audit/export", s.handleAuditExport)

	// Machines (enterprise)
	s.mux.HandleFunc("/v1/machines", s.handleMachines)
	s.mux.HandleFunc("/v1/machines/pair", s.handleMachinesPair)

	// Metrics + daemon events (existing)
	s.mux.HandleFunc("/v1/metrics", s.handleMetrics)
	s.mux.HandleFunc("/v1/events", s.handleDaemonEvents)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	v, sha := "0.2.0", "dev"
	if s.service != nil {
		v, sha = s.service.Version()
	}
	writeJSON(w, http.StatusOK, map[string]string{"version": v, "commit": sha})
}

func (s *Server) notImplemented(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotImplemented, map[string]any{
			"error": map[string]string{
				"code":    "not_implemented",
				"message": name + " is not implemented yet; see corresponding plan",
			},
		})
	}
}

func (s *Server) withMiddleware(h http.Handler) http.Handler {
	return s.corsMiddleware(s.authMiddleware(h))
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			for _, allowed := range s.cfg.API.CORSOrigins {
				if origin == allowed {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
					break
				}
			}
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch s.cfg.API.AuthMode {
		case "none":
			next.ServeHTTP(w, r)
		case "api_key":
			key := r.Header.Get("Authorization")
			ok := false
			for _, k := range s.cfg.API.APIKeys {
				if key == "Bearer "+k {
					ok = true
					break
				}
			}
			if !ok {
				writeJSON(w, http.StatusUnauthorized, map[string]any{
					"error": map[string]string{"code": "unauthorized", "message": "missing or invalid API key"},
				})
				return
			}
			next.ServeHTTP(w, r)
		default:
			next.ServeHTTP(w, r) // mtls / oidc layered upstream
		}
	})
}

func (s *Server) requireService(w http.ResponseWriter) bool {
	if s.service == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "service unavailable"})
		return false
	}
	return true
}

// --- Existing handlers carried over from previous server.go ---

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return
	}
	if !s.requireService(w) {
		return
	}
	runtime, err := s.service.RuntimeConfig(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"runtime": runtime,
	})
}

func (s *Server) handleConfigRuntime(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch && r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return
	}
	if !s.requireService(w) {
		return
	}
	var patch cruxapi.RuntimeConfigPatch
	if err := decodeJSON(r, &patch); err != nil {
		writeError(w, http.StatusBadRequest, "BadRequest", err.Error())
		return
	}
	runtime, err := s.service.UpdateRuntimeConfig(r.Context(), patch)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, runtime)
}

func (s *Server) handleAgents(w http.ResponseWriter, r *http.Request) {
	if !s.requireService(w) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		agents, err := s.service.ListAgents(r.Context())
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, agents)
	case http.MethodPost:
		var agent cruxapi.Agent
		if err := decodeJSON(r, &agent); err != nil {
			writeError(w, http.StatusBadRequest, "BadRequest", err.Error())
			return
		}
		saved, err := s.service.UpsertAgent(r.Context(), agent)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, saved)
	default:
		writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
	}
}

func (s *Server) handleAgentsTree(w http.ResponseWriter, r *http.Request) {
	if !s.requireService(w) {
		return
	}
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/v1/agents/"), "/")
	if rest == "" {
		writeError(w, http.StatusNotFound, "NotFound", "agent not found")
		return
	}
	parts := strings.Split(rest, "/")
	name := parts[0]
	if len(parts) == 3 && parts[1] == "exec" {
		switch parts[2] {
		case "plan":
			if r.Method != http.MethodPost {
				writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
				return
			}
			var req cruxapi.AgentExecPlanRequest
			if err := decodeJSON(r, &req); err != nil {
				writeError(w, http.StatusBadRequest, "BadRequest", err.Error())
				return
			}
			plan, err := s.service.AgentExecPlan(r.Context(), name, req)
			if err != nil {
				writeServiceError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, plan)
			return
		case "record":
			if r.Method != http.MethodPost {
				writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
				return
			}
			var req cruxapi.AgentExecRecordRequest
			if err := decodeJSON(r, &req); err != nil {
				writeError(w, http.StatusBadRequest, "BadRequest", err.Error())
				return
			}
			record, err := s.service.RecordAgentExec(r.Context(), name, req)
			if err != nil {
				writeServiceError(w, err)
				return
			}
			writeJSON(w, http.StatusCreated, record)
			return
		}
	}
	if len(parts) == 2 {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
			return
		}
		switch parts[1] {
		case "usage":
			usage, err := s.service.AgentUsage(r.Context(), name)
			if err != nil {
				writeServiceError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, usage)
			return
		case "capabilities":
			capabilities, err := s.service.AgentCapabilities(r.Context(), name)
			if err != nil {
				writeServiceError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, capabilities)
			return
		case "cost":
			cost, err := s.service.AgentCost(r.Context(), name)
			if err != nil {
				writeServiceError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, cost)
			return
		case "sessions":
			sessions, err := s.service.AgentSessions(r.Context(), name)
			if err != nil {
				writeServiceError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, sessions)
			return
		case "history":
			history, err := s.service.AgentHistory(r.Context(), name)
			if err != nil {
				writeServiceError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, history)
			return
		}
	}
	if len(parts) != 1 {
		writeError(w, http.StatusNotFound, "NotFound", "agent endpoint not found")
		return
	}
	switch r.Method {
	case http.MethodGet:
		agent, err := s.service.GetAgent(r.Context(), name)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, agent)
	case http.MethodDelete:
		if err := s.service.DeleteAgent(r.Context(), name); err != nil {
			writeServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
	}
}

func (s *Server) handleAgentsRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return
	}
	if s.reloadAgents == nil {
		writeError(w, http.StatusNotImplemented, "NotImplemented", "agent refresh not configured")
		return
	}
	if err := s.reloadAgents(); err != nil {
		writeError(w, http.StatusInternalServerError, "InternalError", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleDaemonEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return
	}
	if !s.requireService(w) {
		return
	}
	events, err := s.service.ListEvents(r.Context(), r.URL.Query().Get("executionId"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, events)
}

// --- Helpers ---

func decodeJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode json: %w", err)
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, context.Canceled):
		writeError(w, 499, "Canceled", "request canceled")
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "NotFound", err.Error())
	default:
		writeError(w, http.StatusBadRequest, "BadRequest", err.Error())
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{"code": code, "message": message},
	})
}

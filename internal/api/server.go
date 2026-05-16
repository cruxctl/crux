package api

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/cruxctl/crux/internal/config"
	"github.com/cruxctl/crux/internal/domain"
	"github.com/cruxctl/crux/internal/service"
	"github.com/cruxctl/crux/internal/store"
)

type Server struct {
	cfg     config.DaemonConfig
	service *service.Service
	logger  *slog.Logger
}

type errorResponse struct {
	Error apiError `json:"error"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func NewServer(cfg config.DaemonConfig, svc *service.Service, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{cfg: cfg, service: svc, logger: logger}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.health)
	mux.HandleFunc("/v1/version", s.version)
	mux.HandleFunc("/v1/config", s.config)
	mux.HandleFunc("/v1/config/runtime", s.runtimeConfig)
	mux.HandleFunc("/v1/agents", s.agents)
	mux.HandleFunc("/v1/agents/", s.agent)
	mux.HandleFunc("/v1/discover", s.discover)
	mux.HandleFunc("/v1/executions", s.executions)
	mux.HandleFunc("/v1/executions/", s.execution)
	mux.HandleFunc("/v1/events", s.events)
	return s.auth(mux)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) version(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"version": domain.Version})
}

func (s *Server) config(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
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

func (s *Server) runtimeConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch && r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return
	}
	var patch domain.RuntimeConfigPatch
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

func (s *Server) agents(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		agents, err := s.service.ListAgents(r.Context())
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, agents)
	case http.MethodPost:
		var agent domain.Agent
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

func (s *Server) agent(w http.ResponseWriter, r *http.Request) {
	name := strings.Trim(strings.TrimPrefix(r.URL.Path, "/v1/agents/"), "/")
	if name == "" {
		writeError(w, http.StatusNotFound, "NotFound", "agent not found")
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

func (s *Server) discover(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return
	}
	results, err := s.service.Discover(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, results)
}

func (s *Server) executions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		executions, err := s.service.ListExecutions(r.Context())
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, executions)
	case http.MethodPost:
		var req service.SubmitRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "BadRequest", err.Error())
			return
		}
		execution, err := s.service.SubmitExecution(r.Context(), req)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		status := http.StatusAccepted
		if req.Wait {
			status = http.StatusOK
		}
		writeJSON(w, status, execution)
	default:
		writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
	}
}

func (s *Server) execution(w http.ResponseWriter, r *http.Request) {
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/v1/executions/"), "/")
	if rest == "" {
		writeError(w, http.StatusNotFound, "NotFound", "execution not found")
		return
	}
	parts := strings.Split(rest, "/")
	id := parts[0]
	if len(parts) == 2 && parts[1] == "events" {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
			return
		}
		events, err := s.service.ListEvents(r.Context(), id)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, events)
		return
	}
	if len(parts) != 1 || r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return
	}
	execution, err := s.service.GetExecution(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, execution)
}

func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return
	}
	events, err := s.service.ListEvents(r.Context(), r.URL.Query().Get("executionId"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, events)
}

func (s *Server) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.cfg.Security.APIKey == "" || r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if token == "" {
			token = r.Header.Get("X-Crux-API-Key")
		}
		if subtle.ConstantTimeCompare([]byte(token), []byte(s.cfg.Security.APIKey)) != 1 {
			writeError(w, http.StatusUnauthorized, "Unauthorized", "invalid API key")
			return
		}
		next.ServeHTTP(w, r)
	})
}

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
	writeJSON(w, status, errorResponse{Error: apiError{Code: code, Message: message}})
}

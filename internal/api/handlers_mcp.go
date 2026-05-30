package api

import (
	"net/http"
	"strings"

	"github.com/cruxctl/crux/pkg/cruxapi"
)

func (s *Server) handleMCPServers(w http.ResponseWriter, r *http.Request) {
	if !s.requireService(w) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		servers, err := s.service.ListMCPServers(r.Context())
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, servers)
	case http.MethodPost:
		var req cruxapi.MCPServer
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "BadRequest", err.Error())
			return
		}
		// For now, discover returns all; in future this may register a server
		servers, err := s.service.DiscoverMCPServers(r.Context())
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, servers)
	default:
		writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
	}
}

func (s *Server) handleMCPServersTree(w http.ResponseWriter, r *http.Request) {
	if !s.requireService(w) {
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/v1/mcp/servers/"), "/")
	if id == "" {
		writeError(w, http.StatusNotFound, "NotFound", "server not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return
	}
	server, err := s.service.GetMCPServer(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, server)
}

func (s *Server) handleMCPTools(w http.ResponseWriter, r *http.Request) {
	if !s.requireService(w) {
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return
	}
	tools, err := s.service.ListMCPTools(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tools)
}

func (s *Server) handleMCPCalls(w http.ResponseWriter, r *http.Request) {
	if !s.requireService(w) {
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return
	}
	var req cruxapi.MCPCallRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "BadRequest", err.Error())
		return
	}
	result, err := s.service.CallMCPTool(r.Context(), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleMCPAgentProxy(w http.ResponseWriter, r *http.Request) {
	if !s.requireService(w) {
		return
	}
	// Placeholder: proxy agent-specific MCP calls
	writeError(w, http.StatusNotImplemented, "NotImplemented", "agent MCP proxy not implemented")
}

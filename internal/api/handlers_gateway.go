package api

import "net/http"

func (s *Server) handleGatewayStatus(w http.ResponseWriter, r *http.Request) {
	if !s.requireService(w) {
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return
	}
	status, err := s.service.GetGatewayStatus(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleGatewayRoutes(w http.ResponseWriter, r *http.Request) {
	if !s.requireService(w) {
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return
	}
	routes, err := s.service.GetGatewayRoutes(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, routes)
}

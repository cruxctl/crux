package api

import "net/http"

func (s *Server) handleCosts(w http.ResponseWriter, r *http.Request) {
	if !s.requireService(w) {
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return
	}
	agent := r.URL.Query().Get("agent")
	project := r.URL.Query().Get("project")
	since := r.URL.Query().Get("since")
	costs, err := s.service.GetCosts(r.Context(), agent, project, since)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, costs)
}

package api

import "net/http"

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if !s.requireService(w) {
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return
	}
	metrics, err := s.service.GetMetrics(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, metrics)
}

package api

import (
	"net/http"

	"github.com/cruxctl/crux/pkg/cruxapi"
)

func (s *Server) handleUsage(w http.ResponseWriter, r *http.Request) {
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
	usage, err := s.service.GetUsage(r.Context(), agent, project, since)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, usage)
}

func (s *Server) handleUsageLimits(w http.ResponseWriter, r *http.Request) {
	if !s.requireService(w) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		limits, err := s.service.GetUsageLimits(r.Context())
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, limits)
	case http.MethodPut, http.MethodPatch:
		var limits cruxapi.UsageLimits
		if err := decodeJSON(r, &limits); err != nil {
			writeError(w, http.StatusBadRequest, "BadRequest", err.Error())
			return
		}
		updated, err := s.service.SetUsageLimits(r.Context(), limits)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, updated)
	default:
		writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
	}
}

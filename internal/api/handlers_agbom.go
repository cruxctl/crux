package api

import (
	"net/http"
	"strings"

	"github.com/cruxctl/crux/pkg/cruxapi"
)

func (s *Server) handleAgBOMGenerate(w http.ResponseWriter, r *http.Request) {
	if !s.requireService(w) {
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return
	}
	agentID := r.URL.Query().Get("agentId")
	projectID := r.URL.Query().Get("projectId")
	sessionID := r.URL.Query().Get("sessionId")
	bom, err := s.service.GenerateAgBOM(r.Context(), agentID, projectID, sessionID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, bom)
}

func (s *Server) handleAgBOMTree(w http.ResponseWriter, r *http.Request) {
	if !s.requireService(w) {
		return
	}
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/v1/agbom/"), "/")
	parts := strings.SplitN(rest, "/", 2)
	id := parts[0]
	if id == "" {
		writeError(w, http.StatusNotFound, "NotFound", "agbom not found")
		return
	}
	if len(parts) == 2 {
		switch parts[1] {
		case "export":
			if r.Method != http.MethodPost {
				writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
				return
			}
			var req cruxapi.AgBOMExportRequest
			if err := decodeJSON(r, &req); err != nil {
				writeError(w, http.StatusBadRequest, "BadRequest", err.Error())
				return
			}
			if req.Format == "" {
				req.Format = "crux-json"
			}
			data, err := s.service.ExportAgBOM(r.Context(), id, req)
			if err != nil {
				writeServiceError(w, err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Disposition", "attachment; filename=\"agbom."+req.Format+"\"")
			w.Write(data)
			return
		case "diff":
			if r.Method != http.MethodGet {
				writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
				return
			}
			since := r.URL.Query().Get("since")
			diff, err := s.service.DiffAgBOM(r.Context(), id, since)
			if err != nil {
				writeServiceError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, diff)
			return
		}
	}
	switch r.Method {
	case http.MethodGet:
		bom, err := s.service.GetAgBOM(r.Context(), id)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, bom)
	default:
		writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
	}
}

package api

import (
	"net/http"
	"strings"

	"github.com/cruxctl/crux/pkg/cruxapi"
)

func (s *Server) handleAOSEvents(w http.ResponseWriter, r *http.Request) {
	if !s.requireService(w) {
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return
	}
	var filter cruxapi.EventFilter
	if since := r.URL.Query().Get("since"); since != "" {
		// parse if needed; skipped for stub
	}
	filter.EventType = r.URL.Query().Get("eventType")
	filter.Agent = r.URL.Query().Get("agent")
	filter.SessionID = r.URL.Query().Get("sessionId")
	events, err := s.service.ListAOSEvents(r.Context(), filter)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, events)
}

func (s *Server) handleAOSEventsTree(w http.ResponseWriter, r *http.Request) {
	if !s.requireService(w) {
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/v1/aos/events/"), "/")
	if id == "" {
		writeError(w, http.StatusNotFound, "NotFound", "event not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return
	}
	event, err := s.service.GetAOSEvent(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, event)
}

func (s *Server) handleAOSExport(w http.ResponseWriter, r *http.Request) {
	if !s.requireService(w) {
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return
	}
	var req cruxapi.AOSExportRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "BadRequest", err.Error())
		return
	}
	if req.Format == "" {
		req.Format = "jsonl"
	}
	data, err := s.service.ExportAOSEvents(r.Context(), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=\"aos-events."+req.Format+"\"")
	w.Write(data)
}

func (s *Server) handleAOSTraces(w http.ResponseWriter, r *http.Request) {
	if !s.requireService(w) {
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return
	}
	since := r.URL.Query().Get("since")
	traces, err := s.service.ListTraces(r.Context(), since)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, traces)
}

func (s *Server) handleAOSTracesTree(w http.ResponseWriter, r *http.Request) {
	if !s.requireService(w) {
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/v1/aos/traces/"), "/")
	if id == "" {
		writeError(w, http.StatusNotFound, "NotFound", "trace not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return
	}
	trace, err := s.service.GetTrace(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, trace)
}

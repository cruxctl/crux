package api

import (
	"net/http"
	"strings"

	"github.com/cruxctl/crux/pkg/cruxapi"
)

func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	if !s.requireService(w) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		sessions, err := s.service.ListSessions(r.Context())
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, sessions)
	case http.MethodPost:
		var sess cruxapi.Session
		if err := decodeJSON(r, &sess); err != nil {
			writeError(w, http.StatusBadRequest, "BadRequest", err.Error())
			return
		}
		saved, err := s.service.CreateSession(r.Context(), sess)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, saved)
	default:
		writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
	}
}

func (s *Server) handleSessionsTree(w http.ResponseWriter, r *http.Request) {
	if !s.requireService(w) {
		return
	}
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/v1/sessions/"), "/")
	parts := strings.SplitN(rest, "/", 2)
	id := parts[0]
	if id == "" {
		writeError(w, http.StatusNotFound, "NotFound", "session not found")
		return
	}
	if len(parts) == 2 {
		switch parts[1] {
		case "events":
			if r.Method != http.MethodGet {
				writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
				return
			}
			events, err := s.service.GetSessionEvents(r.Context(), id)
			if err != nil {
				writeServiceError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, events)
			return
		case "transcript":
			if r.Method != http.MethodGet {
				writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
				return
			}
			format := r.URL.Query().Get("format")
			if format == "" {
				format = "text"
			}
			data, err := s.service.GetSessionTranscript(r.Context(), id, format)
			if err != nil {
				writeServiceError(w, err)
				return
			}
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Write(data)
			return
		case "continue":
			if r.Method != http.MethodPost {
				writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
				return
			}
			var req cruxapi.ContinueSessionRequest
			if err := decodeJSON(r, &req); err != nil {
				writeError(w, http.StatusBadRequest, "BadRequest", err.Error())
				return
			}
			sess, err := s.service.ContinueSession(r.Context(), id, req)
			if err != nil {
				writeServiceError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, sess)
			return
		case "stop":
			if r.Method != http.MethodPost {
				writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
				return
			}
			if err := s.service.StopSession(r.Context(), id); err != nil {
				writeServiceError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "stopped"})
			return
		case "replay":
			if r.Method != http.MethodPost {
				writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
				return
			}
			sess, err := s.service.ReplaySession(r.Context(), id)
			if err != nil {
				writeServiceError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, sess)
			return
		}
	}
	switch r.Method {
	case http.MethodGet:
		sess, err := s.service.GetSession(r.Context(), id)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, sess)
	default:
		writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
	}
}

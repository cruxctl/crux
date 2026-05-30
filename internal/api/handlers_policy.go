package api

import (
	"net/http"
	"strings"

	"github.com/cruxctl/crux/pkg/cruxapi"
)

func (s *Server) handlePolicies(w http.ResponseWriter, r *http.Request) {
	if !s.requireService(w) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		policies, err := s.service.ListPolicies(r.Context())
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, policies)
	case http.MethodPost:
		var policy cruxapi.PolicyProfile
		if err := decodeJSON(r, &policy); err != nil {
			writeError(w, http.StatusBadRequest, "BadRequest", err.Error())
			return
		}
		saved, err := s.service.CreatePolicy(r.Context(), policy)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, saved)
	default:
		writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
	}
}

func (s *Server) handlePoliciesTree(w http.ResponseWriter, r *http.Request) {
	if !s.requireService(w) {
		return
	}
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/v1/policies/"), "/")
	parts := strings.SplitN(rest, "/", 2)
	id := parts[0]
	if id == "" {
		writeError(w, http.StatusNotFound, "NotFound", "policy not found")
		return
	}
	if len(parts) == 2 {
		switch parts[1] {
		case "evaluate":
			if r.Method != http.MethodPost {
				writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
				return
			}
			var input cruxapi.EvaluationInput
			if err := decodeJSON(r, &input); err != nil {
				writeError(w, http.StatusBadRequest, "BadRequest", err.Error())
				return
			}
			decision, err := s.service.EvaluatePolicy(r.Context(), id, input)
			if err != nil {
				writeServiceError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, decision)
			return
		case "simulate":
			if r.Method != http.MethodPost {
				writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
				return
			}
			var input cruxapi.EvaluationInput
			if err := decodeJSON(r, &input); err != nil {
				writeError(w, http.StatusBadRequest, "BadRequest", err.Error())
				return
			}
			decision, err := s.service.SimulatePolicy(r.Context(), id, input)
			if err != nil {
				writeServiceError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, decision)
			return
		}
	}
	switch r.Method {
	case http.MethodGet:
		policy, err := s.service.GetPolicy(r.Context(), id)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, policy)
	case http.MethodDelete:
		if err := s.service.DeletePolicy(r.Context(), id); err != nil {
			writeServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
	}
}

func (s *Server) handleApprovals(w http.ResponseWriter, r *http.Request) {
	if !s.requireService(w) {
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return
	}
	approvals, err := s.service.ListApprovals(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, approvals)
}

func (s *Server) handleApprovalsTree(w http.ResponseWriter, r *http.Request) {
	if !s.requireService(w) {
		return
	}
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/v1/approvals/"), "/")
	parts := strings.SplitN(rest, "/", 2)
	id := parts[0]
	if id == "" {
		writeError(w, http.StatusNotFound, "NotFound", "approval not found")
		return
	}
	if len(parts) == 2 {
		switch parts[1] {
		case "grant":
			if r.Method != http.MethodPost {
				writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
				return
			}
			rec, err := s.service.GrantApproval(r.Context(), id)
			if err != nil {
				writeServiceError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, rec)
			return
		case "deny":
			if r.Method != http.MethodPost {
				writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
				return
			}
			rec, err := s.service.DenyApproval(r.Context(), id)
			if err != nil {
				writeServiceError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, rec)
			return
		}
	}
	switch r.Method {
	case http.MethodGet:
		rec, err := s.service.GetApproval(r.Context(), id)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, rec)
	default:
		writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
	}
}

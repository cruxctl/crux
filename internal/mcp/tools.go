package mcp

import (
	"encoding/json"
	"fmt"
	"time"
)

func (s *Server) listTools() map[string]any {
	return map[string]any{
		"tools": []map[string]any{
			{
				"name":        "list_sessions",
				"description": "List agent sessions",
				"inputSchema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"agent":  map[string]any{"type": "string", "description": "Filter by agent ID"},
						"since":  map[string]any{"type": "string", "description": "ISO timestamp"},
						"limit":  map[string]any{"type": "integer", "description": "Max results", "default": 50},
					},
				},
			},
			{
				"name":        "get_session",
				"description": "Get a full session by ID",
				"inputSchema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"session_id": map[string]any{"type": "string"},
					},
					"required": []string{"session_id"},
				},
			},
			{
				"name":        "search_conversations",
				"description": "Search messages across sessions",
				"inputSchema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"query":  map[string]any{"type": "string"},
						"agent":  map[string]any{"type": "string"},
						"limit":  map[string]any{"type": "integer", "default": 20},
					},
					"required": []string{"query"},
				},
			},
			{
				"name":        "get_token_usage",
				"description": "Aggregated token usage",
				"inputSchema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"agent":     map[string]any{"type": "string"},
						"since":     map[string]any{"type": "string"},
						"group_by":  map[string]any{"type": "string", "enum": []string{"agent", "session", "day"}},
					},
				},
			},
		},
	}
}

func (s *Server) callTool(params json.RawMessage) (map[string]any, error) {
	var call struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(params, &call); err != nil {
		return nil, err
	}

	s.maybeRescan()

	switch call.Name {
	case "list_sessions":
		var args struct {
			Agent string `json:"agent"`
			Since string `json:"since"`
			Limit int    `json:"limit"`
		}
		_ = json.Unmarshal(call.Arguments, &args)
		var since int64
		if args.Since != "" {
			if t, err := time.Parse(time.RFC3339, args.Since); err == nil {
				since = t.UnixMilli()
			}
		}
		sessions := s.store.ListSessions(args.Agent, since, args.Limit)
		data, _ := json.Marshal(sessions)
		return textResult(string(data)), nil

	case "get_session":
		var args struct {
			SessionID string `json:"session_id"`
		}
		_ = json.Unmarshal(call.Arguments, &args)
		sess, err := s.store.GetSession(args.SessionID)
		if err != nil {
			return nil, err
		}
		data, _ := json.Marshal(sess)
		return textResult(string(data)), nil

	case "search_conversations":
		var args struct {
			Query string `json:"query"`
			Agent string `json:"agent"`
			Limit int    `json:"limit"`
		}
		_ = json.Unmarshal(call.Arguments, &args)
		if args.Limit <= 0 {
			args.Limit = 20
		}
		ids := s.search.Search(args.Query)
		var results []SearchDoc
		for _, doc := range s.store.AllMessages() {
			for _, id := range ids {
				if doc.ID == id {
					if args.Agent == "" || doc.Agent == args.Agent {
						results = append(results, doc)
					}
					break
				}
			}
			if len(results) >= args.Limit {
				break
			}
		}
		data, _ := json.Marshal(results)
		return textResult(string(data)), nil

	case "get_token_usage":
		var args struct {
			Agent   string `json:"agent"`
			Since   string `json:"since"`
			GroupBy string `json:"group_by"`
		}
		_ = json.Unmarshal(call.Arguments, &args)
		usage := s.aggregateUsage(args.Agent, args.Since, args.GroupBy)
		data, _ := json.Marshal(usage)
		return textResult(string(data)), nil

	default:
		return nil, fmt.Errorf("unknown tool: %s", call.Name)
	}
}

func textResult(text string) map[string]any {
	return map[string]any{
		"content": []map[string]any{{"type": "text", "text": text}},
		"isError": false,
	}
}

func (s *Server) aggregateUsage(agent, since, groupBy string) any {
	sessions := s.store.ListSessions(agent, 0, 10000)
	var sinceMs int64
	if since != "" {
		if t, err := time.Parse(time.RFC3339, since); err == nil {
			sinceMs = t.UnixMilli()
		}
	}

	type agg struct {
		Key          string `json:"key"`
		InputTokens  int    `json:"input_tokens"`
		OutputTokens int    `json:"output_tokens"`
		TotalTokens  int    `json:"total_tokens"`
	}
	groups := map[string]*agg{}

	for _, sum := range sessions {
		if sinceMs > 0 && sum.StartedAt < sinceMs {
			continue
		}
		sess, err := s.store.GetSession(sum.ID)
		if err != nil {
			continue
		}
		key := "all"
		switch groupBy {
		case "agent":
			key = sum.Agent
		case "session":
			key = sum.ID
		case "day":
			key = time.UnixMilli(sum.StartedAt).UTC().Format("2006-01-02")
		}
		if groups[key] == nil {
			groups[key] = &agg{Key: key}
		}
		groups[key].InputTokens += sess.TokenUsage.InputTokens
		groups[key].OutputTokens += sess.TokenUsage.OutputTokens
		groups[key].TotalTokens += sess.TokenUsage.TotalTokens
	}

	var out []agg
	for _, g := range groups {
		out = append(out, *g)
	}
	return out
}

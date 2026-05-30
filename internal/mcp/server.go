package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"
)

// Server handles JSON-RPC over stdio.
type Server struct {
	store   *JsonStore
	search  *InvertedIndex
	watcher *Watcher
	logger  *slog.Logger
}

// NewServer creates an MCP server.
func NewServer(store *JsonStore, watcher *Watcher, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
	return &Server{
		store:   store,
		search:  NewInvertedIndex(),
		watcher: watcher,
		logger:  logger,
	}
}

// Run starts the JSON-RPC loop over stdio.
func (s *Server) Run() error {
	if err := s.watcher.Rescan(); err != nil {
		s.logger.Warn("initial rescan failed", "error", err)
	}
	reader := bufio.NewReader(os.Stdin)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		var req jsonRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			s.writeError(nil, -32700, "Parse error")
			continue
		}
		if req.Method == "initialize" {
			s.writeResult(req.ID, map[string]any{
				"protocolVersion": "2024-11-05",
				"capabilities":    map[string]any{"tools": map[string]any{}},
				"serverInfo":      map[string]any{"name": "crux-mcp", "version": "0.1.0"},
			})
			continue
		}
		if req.Method == "tools/list" {
			s.writeResult(req.ID, s.listTools())
			continue
		}
		if req.Method == "tools/call" {
			result, callErr := s.callTool(req.Params)
			if callErr != nil {
				s.writeResult(req.ID, map[string]any{"content": []map[string]any{{"type": "text", "text": callErr.Error()}}, "isError": true})
			} else {
				s.writeResult(req.ID, result)
			}
			continue
		}
		s.writeResult(req.ID, map[string]any{})
	}
}

func (s *Server) writeResult(id any, result any) {
	resp := jsonRPCResponse{JSONRPC: "2.0", ID: id, Result: result}
	data, _ := json.Marshal(resp)
	fmt.Println(string(data))
}

func (s *Server) writeError(id any, code int, message string) {
	resp := jsonRPCResponse{JSONRPC: "2.0", ID: id, Error: &jsonRPCError{Code: code, Message: message}}
	data, _ := json.Marshal(resp)
	fmt.Println(string(data))
}

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (s *Server) ensureIndex() {
	// Rebuild search index on demand.
	s.search.Build(s.store.AllMessages())
}

const rescanTTL = 30 * time.Second

func (s *Server) maybeRescan() {
	// For simplicity, rebuild index on every search/tool call.
	s.ensureIndex()
}

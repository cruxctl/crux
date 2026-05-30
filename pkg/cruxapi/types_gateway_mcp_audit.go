package cruxapi

import "time"

// GatewayStatus represents the current state of the MCP Gateway.
type GatewayStatus struct {
	Enabled   bool     `json:"enabled"`
	Version   string   `json:"version,omitempty"`
	Uptime    string   `json:"uptime,omitempty"`
	Routes    []string `json:"routes,omitempty"`
	Injected  []string `json:"injected_agents,omitempty"`
	Ready     bool     `json:"ready"`
}

// GatewayRoute is a single route registered in the gateway.
type GatewayRoute struct {
	ID        string            `json:"id"`
	Path      string            `json:"path"`
	Target    string            `json:"target"`
	AgentID   string            `json:"agent_id,omitempty"`
	Methods   []string          `json:"methods,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
	Enabled   bool              `json:"enabled"`
	CreatedAt time.Time         `json:"created_at"`
}

// MCPServer represents a discovered or configured MCP server.
type MCPServer struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	URL       string     `json:"url,omitempty"`
	Version   string     `json:"version,omitempty"`
	Tools     []string   `json:"tools,omitempty"`
	Status    string     `json:"status"`
	LastSeen  *time.Time `json:"last_seen,omitempty"`
}

// MCPTool represents a tool exposed by an MCP server.
type MCPTool struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ServerID    string `json:"server_id"`
	Description string `json:"description,omitempty"`
	InputSchema string `json:"input_schema,omitempty"`
}

// MCPCallRequest is used to invoke an MCP tool.
type MCPCallRequest struct {
	ToolID  string         `json:"tool_id"`
	Params  map[string]any `json:"params,omitempty"`
	Session string         `json:"session,omitempty"`
}

// MCPCallResult is the result of an MCP tool invocation.
type MCPCallResult struct {
	Success bool           `json:"success"`
	Data    map[string]any `json:"data,omitempty"`
	Error   string         `json:"error,omitempty"`
}

// AuditLogEntry represents a single audit log record.
type AuditLogEntry struct {
	ID        string    `json:"id"`
	Action    string    `json:"action"`
	Actor     string    `json:"actor"`
	Resource  string    `json:"resource"`
	ProjectID string    `json:"project_id,omitempty"`
	Details   string    `json:"details,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// AuditExportRequest controls audit log export format.
type AuditExportRequest struct {
	Format string `json:"format"` // jsonl | csv
}

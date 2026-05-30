package mcp


// NormalizedMessage is a generic chat message.
type NormalizedMessage struct {
	ID      string `json:"id"`
	Role    string `json:"role"`
	Content string `json:"content"`
	TS      int64  `json:"ts"`
}

// NormalizedSession is a parsed agent session.
type NormalizedSession struct {
	ID          string                `json:"id"`
	Agent       string                `json:"agent"`
	Title       string                `json:"title,omitempty"`
	StartedAt   int64                 `json:"started_at"`
	EndedAt     int64                 `json:"ended_at,omitempty"`
	Messages    []NormalizedMessage   `json:"messages"`
	TokenUsage  TokenUsage            `json:"token_usage,omitempty"`
}

// TokenUsage holds aggregated token stats.
type TokenUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// SessionSummary is a lightweight index entry.
type SessionSummary struct {
	ID        string `json:"id"`
	Agent     string `json:"agent"`
	Title     string `json:"title,omitempty"`
	StartedAt int64  `json:"started_at"`
	MessageCount int `json:"message_count"`
}

// SearchDoc is a document in the search index.
type SearchDoc struct {
	ID        string `json:"id"`
	SessionID string `json:"session_id"`
	Agent     string `json:"agent"`
	TS        int64  `json:"ts"`
	Role      string `json:"role"`
	Content   string `json:"content"`
}

// AgentAdapter parses agent-specific log files.
type AgentAdapter interface {
	ID() string
	DataPaths() []string
	Matches(path string) bool
	ParseFile(path string) ([]NormalizedSession, error)
}

// Checkpoint tracks parsed file state.
type Checkpoint struct {
	Size     int64 `json:"size"`
	MtimeMs  int64 `json:"mtime_ms"`
	Sessions []string `json:"session_ids"`
}

// StoreIndex is the root index file.
type StoreIndex struct {
	Sessions   []SessionSummary      `json:"sessions"`
	Checkpoints map[string]Checkpoint `json:"checkpoints"`
	UpdatedAt  int64                 `json:"updated_at"`
}

package adapters

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/cruxctl/crux/internal/mcp"
)

// DefaultRegistry returns all built-in adapters.
func DefaultRegistry() []mcp.AgentAdapter {
	return []mcp.AgentAdapter{
		&ClaudeCodeAdapter{},
		&CodexAdapter{},
	}
}

func stringHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])[:16]
}

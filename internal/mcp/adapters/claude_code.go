package adapters

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/cruxctl/crux/internal/mcp"
)

// ClaudeCodeAdapter parses Claude Code JSONL files.
type ClaudeCodeAdapter struct{}

func (a *ClaudeCodeAdapter) ID() string { return "claude-code" }

func (a *ClaudeCodeAdapter) DataPaths() []string {
	home, _ := os.UserHomeDir()
	return []string{filepath.Join(home, ".claude", "projects")}
}

func (a *ClaudeCodeAdapter) Matches(path string) bool {
	return strings.HasSuffix(path, ".jsonl")
}

func (a *ClaudeCodeAdapter) ParseFile(path string) ([]mcp.NormalizedSession, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var messages []mcp.NormalizedMessage
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var line map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}
		msg := mcp.NormalizedMessage{ID: stringHash(scanner.Text())}
		if role, ok := line["role"].(string); ok {
			msg.Role = role
		}
		if content, ok := line["content"].(string); ok {
			msg.Content = content
		}
		if ts, ok := line["timestamp"].(float64); ok {
			msg.TS = int64(ts * 1000)
		}
		messages = append(messages, msg)
	}

	if len(messages) == 0 {
		return nil, nil
	}

	sess := mcp.NormalizedSession{
		ID:        filepath.Base(path),
		Agent:     "claude-code",
		StartedAt: messages[0].TS,
		Messages:  messages,
	}
	if len(messages) > 0 {
		sess.EndedAt = messages[len(messages)-1].TS
	}
	return []mcp.NormalizedSession{sess}, scanner.Err()
}

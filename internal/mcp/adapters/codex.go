package adapters

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/cruxctl/crux/internal/mcp"
)

// CodexAdapter parses Codex CLI JSONL files.
type CodexAdapter struct{}

func (a *CodexAdapter) ID() string { return "codex" }

func (a *CodexAdapter) DataPaths() []string {
	home, _ := os.UserHomeDir()
	return []string{filepath.Join(home, ".codex", "sessions")}
}

func (a *CodexAdapter) Matches(path string) bool {
	return strings.HasSuffix(path, ".jsonl")
}

func (a *CodexAdapter) ParseFile(path string) ([]mcp.NormalizedSession, error) {
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
		Agent:     "codex",
		StartedAt: messages[0].TS,
		Messages:  messages,
	}
	if len(messages) > 0 {
		sess.EndedAt = messages[len(messages)-1].TS
	}
	return []mcp.NormalizedSession{sess}, scanner.Err()
}

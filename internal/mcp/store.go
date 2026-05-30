package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// JsonStore persists parsed sessions to disk.
type JsonStore struct {
	root  string
	mu    sync.RWMutex
	index StoreIndex
}

// NewJsonStore creates a store rooted at path.
func NewJsonStore(path string) *JsonStore {
	return &JsonStore{root: path, index: StoreIndex{Checkpoints: map[string]Checkpoint{}}}
}

// Init ensures directories exist and loads the index.
func (s *JsonStore) Init() error {
	for _, dir := range []string{s.root, filepath.Join(s.root, "sessions")} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	data, err := os.ReadFile(filepath.Join(s.root, "index.json"))
	if err == nil {
		_ = json.Unmarshal(data, &s.index)
	}
	return nil
}

// SaveSession writes a session atomically.
func (s *JsonStore) SaveSession(agent string, sess NormalizedSession) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir := filepath.Join(s.root, "sessions", agent)
	_ = os.MkdirAll(dir, 0o700)
	path := filepath.Join(dir, sess.ID+".json")
	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return err
	}
	if err := writeAtomic(path, data); err != nil {
		return err
	}

	// Update index.
	found := false
	for i, sum := range s.index.Sessions {
		if sum.ID == sess.ID {
			s.index.Sessions[i] = SessionSummary{
				ID:           sess.ID,
				Agent:        agent,
				Title:        sess.Title,
				StartedAt:    sess.StartedAt,
				MessageCount: len(sess.Messages),
			}
			found = true
			break
		}
	}
	if !found {
		s.index.Sessions = append(s.index.Sessions, SessionSummary{
			ID:           sess.ID,
			Agent:        agent,
			Title:        sess.Title,
			StartedAt:    sess.StartedAt,
			MessageCount: len(sess.Messages),
		})
	}
	s.index.UpdatedAt = time.Now().UnixMilli()
	return s.saveIndex()
}

// ListSessions returns summaries optionally filtered by agent.
func (s *JsonStore) ListSessions(agent string, since int64, limit int) []SessionSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 50
	}
	var out []SessionSummary
	for _, sum := range s.index.Sessions {
		if agent != "" && sum.Agent != agent {
			continue
		}
		if since > 0 && sum.StartedAt < since {
			continue
		}
		out = append(out, sum)
		if len(out) >= limit {
			break
		}
	}
	return out
}

// GetSession loads a full session by ID.
func (s *JsonStore) GetSession(sessionID string) (*NormalizedSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var agent string
	for _, sum := range s.index.Sessions {
		if sum.ID == sessionID {
			agent = sum.Agent
			break
		}
	}
	if agent == "" {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	data, err := os.ReadFile(filepath.Join(s.root, "sessions", agent, sessionID+".json"))
	if err != nil {
		return nil, err
	}
	var sess NormalizedSession
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

// AllMessages returns every message for search indexing.
func (s *JsonStore) AllMessages() []SearchDoc {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var docs []SearchDoc
	for _, sum := range s.index.Sessions {
		sess, err := s.getSessionUnlocked(sum.Agent, sum.ID)
		if err != nil {
			continue
		}
		for _, m := range sess.Messages {
			docs = append(docs, SearchDoc{
				ID:        m.ID,
				SessionID: sess.ID,
				Agent:     sum.Agent,
				TS:        m.TS,
				Role:      m.Role,
				Content:   m.Content,
			})
		}
	}
	return docs
}

func (s *JsonStore) getSessionUnlocked(agent, sessionID string) (*NormalizedSession, error) {
	data, err := os.ReadFile(filepath.Join(s.root, "sessions", agent, sessionID+".json"))
	if err != nil {
		return nil, err
	}
	var sess NormalizedSession
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

func (s *JsonStore) saveIndex() error {
	data, err := json.MarshalIndent(s.index, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(filepath.Join(s.root, "index.json"), data)
}

func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	_ = os.MkdirAll(dir, 0o700)
	tmp := fmt.Sprintf("%s.tmp-%d", path, time.Now().UnixNano())
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

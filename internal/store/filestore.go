package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/cruxctl/crux/internal/domain"
)

const schemaVersion = 1

type FileStore struct {
	path string
	mu   sync.Mutex
	data state
}

type state struct {
	SchemaVersion int                         `json:"schemaVersion"`
	RuntimeConfig domain.RuntimeConfig        `json:"runtimeConfig"`
	Agents        map[string]domain.Agent     `json:"agents"`
	Executions    map[string]domain.Execution `json:"executions"`
	Events        []domain.Event              `json:"events"`
}

func NewFileStore(path string, runtime domain.RuntimeConfig) (*FileStore, error) {
	fs := &FileStore{path: path}
	if err := fs.load(runtime); err != nil {
		return nil, err
	}
	return fs, nil
}

func (s *FileStore) RuntimeConfig(ctx context.Context) (domain.RuntimeConfig, error) {
	if err := ctx.Err(); err != nil {
		return domain.RuntimeConfig{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data.RuntimeConfig, nil
}

func (s *FileStore) UpdateRuntimeConfig(ctx context.Context, cfg domain.RuntimeConfig) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.RuntimeConfig = cfg
	return s.saveLocked()
}

func (s *FileStore) UpsertAgent(ctx context.Context, agent domain.Agent) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	name := domain.CleanAgentName(agent.Name)
	if name == "" {
		return fmt.Errorf("agent name is required")
	}
	agent.Name = name
	if agent.ID == "" {
		agent.ID = domain.NewID("agent")
	}
	if agent.Status == "" {
		agent.Status = domain.AgentReady
	}
	now := domain.Now()
	if agent.CreatedAt.IsZero() {
		agent.CreatedAt = now
	}
	agent.UpdatedAt = now

	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Agents[name] = agent
	return s.saveLocked()
}

func (s *FileStore) DeleteAgent(ctx context.Context, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	key := domain.CleanAgentName(name)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data.Agents[key]; !ok {
		return ErrNotFound
	}
	delete(s.data.Agents, key)
	return s.saveLocked()
}

func (s *FileStore) GetAgent(ctx context.Context, name string) (domain.Agent, error) {
	if err := ctx.Err(); err != nil {
		return domain.Agent{}, err
	}
	key := domain.CleanAgentName(name)
	s.mu.Lock()
	defer s.mu.Unlock()
	agent, ok := s.data.Agents[key]
	if !ok {
		return domain.Agent{}, ErrNotFound
	}
	return agent, nil
}

func (s *FileStore) ListAgents(ctx context.Context) ([]domain.Agent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	agents := make([]domain.Agent, 0, len(s.data.Agents))
	for _, agent := range s.data.Agents {
		agents = append(agents, agent)
	}
	sort.Slice(agents, func(i, j int) bool {
		return agents[i].Name < agents[j].Name
	})
	return agents, nil
}

func (s *FileStore) CreateExecution(ctx context.Context, execution domain.Execution) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if execution.ID == "" {
		return fmt.Errorf("execution id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data.Executions[execution.ID]; ok {
		return fmt.Errorf("execution %s already exists", execution.ID)
	}
	s.data.Executions[execution.ID] = execution
	return s.saveLocked()
}

func (s *FileStore) UpdateExecution(ctx context.Context, execution domain.Execution) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data.Executions[execution.ID]; !ok {
		return ErrNotFound
	}
	s.data.Executions[execution.ID] = execution
	return s.saveLocked()
}

func (s *FileStore) GetExecution(ctx context.Context, id string) (domain.Execution, error) {
	if err := ctx.Err(); err != nil {
		return domain.Execution{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	execution, ok := s.data.Executions[id]
	if !ok {
		return domain.Execution{}, ErrNotFound
	}
	return execution, nil
}

func (s *FileStore) ListExecutions(ctx context.Context) ([]domain.Execution, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	executions := make([]domain.Execution, 0, len(s.data.Executions))
	for _, execution := range s.data.Executions {
		executions = append(executions, execution)
	}
	sort.Slice(executions, func(i, j int) bool {
		return executions[i].QueuedAt.After(executions[j].QueuedAt)
	})
	return executions, nil
}

func (s *FileStore) AppendEvent(ctx context.Context, event domain.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if event.ID == "" {
		event.ID = domain.NewID("evt")
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = domain.Now()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Events = append(s.data.Events, event)
	retention := s.data.RuntimeConfig.TraceRetentionEntries
	if retention > 0 && len(s.data.Events) > retention {
		s.data.Events = s.data.Events[len(s.data.Events)-retention:]
	}
	return s.saveLocked()
}

func (s *FileStore) ListEvents(ctx context.Context, executionID string) ([]domain.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	events := make([]domain.Event, 0, len(s.data.Events))
	for _, event := range s.data.Events {
		if executionID == "" || event.ExecutionID == executionID {
			events = append(events, event)
		}
	}
	sort.Slice(events, func(i, j int) bool {
		return events[i].CreatedAt.Before(events[j].CreatedAt)
	})
	return events, nil
}

func (s *FileStore) load(runtime domain.RuntimeConfig) error {
	if err := runtime.Validate(); err != nil {
		return err
	}
	s.data = state{
		SchemaVersion: schemaVersion,
		RuntimeConfig: runtime,
		Agents:        map[string]domain.Agent{},
		Executions:    map[string]domain.Execution{},
		Events:        []domain.Event{},
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return s.saveLocked()
		}
		return fmt.Errorf("read state %s: %w", s.path, err)
	}
	if err := json.Unmarshal(data, &s.data); err != nil {
		return fmt.Errorf("parse state %s: %w", s.path, err)
	}
	if s.data.SchemaVersion != schemaVersion {
		return fmt.Errorf("unsupported state schema version %d", s.data.SchemaVersion)
	}
	if s.data.Agents == nil {
		s.data.Agents = map[string]domain.Agent{}
	}
	if s.data.Executions == nil {
		s.data.Executions = map[string]domain.Execution{}
	}
	if s.data.Events == nil {
		s.data.Events = []domain.Event{}
	}
	if err := s.data.RuntimeConfig.Validate(); err != nil {
		return fmt.Errorf("stored runtime config: %w", err)
	}
	return nil
}

func (s *FileStore) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}
	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write state temp: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("replace state: %w", err)
	}
	return nil
}

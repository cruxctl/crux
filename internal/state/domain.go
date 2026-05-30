// Package store domain layer wraps the generic Store with typed CRUD for
// agents, executions, events, and runtime config.
package store

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/cruxctl/crux/pkg/cruxapi"
)

// DomainStore provides typed operations on top of the generic tree Store.
type DomainStore interface {
	RuntimeConfig(ctx context.Context) (cruxapi.RuntimeConfig, error)
	UpdateRuntimeConfig(ctx context.Context, cfg cruxapi.RuntimeConfig) error

	UpsertAgent(ctx context.Context, agent cruxapi.Agent) error
	DeleteAgent(ctx context.Context, name string) error
	GetAgent(ctx context.Context, name string) (cruxapi.Agent, error)
	ListAgents(ctx context.Context) ([]cruxapi.Agent, error)

	CreateExecution(ctx context.Context, execution cruxapi.Execution) error
	UpdateExecution(ctx context.Context, execution cruxapi.Execution) error
	GetExecution(ctx context.Context, id string) (cruxapi.Execution, error)
	ListExecutions(ctx context.Context) ([]cruxapi.Execution, error)

	AppendEvent(ctx context.Context, event cruxapi.Event) error
	ListEvents(ctx context.Context, executionID string) ([]cruxapi.Event, error)

	// Sessions
	CreateSession(ctx context.Context, sess cruxapi.Session) error
	GetSession(ctx context.Context, id string) (cruxapi.Session, error)
	ListSessions(ctx context.Context) ([]cruxapi.Session, error)
	UpdateSession(ctx context.Context, sess cruxapi.Session) error

	// AOS Events
	AppendAOSEvent(ctx context.Context, event cruxapi.AOSEvent) error
	ListAOSEvents(ctx context.Context) ([]cruxapi.AOSEvent, error)
	GetAOSEvent(ctx context.Context, id string) (cruxapi.AOSEvent, error)

	// Policies
	UpsertPolicy(ctx context.Context, policy cruxapi.PolicyProfile) error
	GetPolicy(ctx context.Context, id string) (cruxapi.PolicyProfile, error)
	ListPolicies(ctx context.Context) ([]cruxapi.PolicyProfile, error)
	DeletePolicy(ctx context.Context, id string) error

	// Approvals
	CreateApproval(ctx context.Context, rec cruxapi.ApprovalRecord) error
	GetApproval(ctx context.Context, id string) (cruxapi.ApprovalRecord, error)
	ListApprovals(ctx context.Context) ([]cruxapi.ApprovalRecord, error)
	UpdateApproval(ctx context.Context, rec cruxapi.ApprovalRecord) error

	// Jobs
	CreateJob(ctx context.Context, job cruxapi.Job) error
	GetJob(ctx context.Context, id string) (cruxapi.Job, error)
	ListJobs(ctx context.Context) ([]cruxapi.Job, error)
	DeleteJob(ctx context.Context, id string) error

	// Machines
	CreateMachine(ctx context.Context, m cruxapi.Machine) error
	GetMachine(ctx context.Context, id string) (cruxapi.Machine, error)
	ListMachines(ctx context.Context) ([]cruxapi.Machine, error)

	// Usage
	GetUsageLimits(ctx context.Context) (cruxapi.UsageLimits, error)
	SetUsageLimits(ctx context.Context, limits cruxapi.UsageLimits) error
}

// domainStore is the private implementation of DomainStore.
type domainStore struct {
	store Store
	mu    sync.Mutex // coarse-grained mutex for event retention and runtime config updates
}

// NewDomainStore wraps a generic Store with domain methods.
func NewDomainStore(st Store) DomainStore {
	return &domainStore{store: st}
}

func (d *domainStore) RuntimeConfig(ctx context.Context) (cruxapi.RuntimeConfig, error) {
	data, err := d.store.Get(ctx, "config/runtime.json")
	if err != nil {
		if os.IsNotExist(err) {
			return cruxapi.DefaultRuntimeConfig(), nil
		}
		return cruxapi.RuntimeConfig{}, err
	}
	var cfg cruxapi.RuntimeConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cruxapi.RuntimeConfig{}, err
	}
	return cfg, nil
}

func (d *domainStore) UpdateRuntimeConfig(ctx context.Context, cfg cruxapi.RuntimeConfig) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return d.store.Put(ctx, "config/runtime.json", data)
}

func (d *domainStore) UpsertAgent(ctx context.Context, agent cruxapi.Agent) error {
	name := cruxapi.CleanAgentName(agent.Name)
	if name == "" {
		return fmt.Errorf("agent name is required")
	}
	agent.Name = name
	if agent.ID == "" {
		agent.ID = cruxapi.NewID("agent")
	}
	if agent.Status == "" {
		agent.Status = cruxapi.AgentReady
	}
	now := cruxapi.Now()
	if agent.CreatedAt.IsZero() {
		agent.CreatedAt = now
	}
	agent.UpdatedAt = now

	data, err := json.Marshal(agent)
	if err != nil {
		return err
	}
	return d.store.Put(ctx, filepath.Join("agents", name+".json"), data)
}

func (d *domainStore) DeleteAgent(ctx context.Context, name string) error {
	key := cruxapi.CleanAgentName(name)
	_, err := d.GetAgent(ctx, key)
	if err != nil {
		return err
	}
	return d.store.Delete(ctx, filepath.Join("agents", key+".json"))
}

func (d *domainStore) GetAgent(ctx context.Context, name string) (cruxapi.Agent, error) {
	key := cruxapi.CleanAgentName(name)
	data, err := d.store.Get(ctx, filepath.Join("agents", key+".json"))
	if err != nil {
		if os.IsNotExist(err) {
			return cruxapi.Agent{}, ErrNotFound
		}
		return cruxapi.Agent{}, err
	}
	var agent cruxapi.Agent
	if err := json.Unmarshal(data, &agent); err != nil {
		return cruxapi.Agent{}, err
	}
	return agent, nil
}

func (d *domainStore) ListAgents(ctx context.Context) ([]cruxapi.Agent, error) {
	objs, err := d.store.List(ctx, "agents")
	if err != nil {
		return nil, err
	}
	var agents []cruxapi.Agent
	for _, obj := range objs {
		if !strings.HasSuffix(obj.Path, ".json") {
			continue
		}
		data, err := d.store.Get(ctx, obj.Path)
		if err != nil {
			continue
		}
		var agent cruxapi.Agent
		if err := json.Unmarshal(data, &agent); err != nil {
			continue
		}
		agents = append(agents, agent)
	}
	sort.Slice(agents, func(i, j int) bool {
		return agents[i].Name < agents[j].Name
	})
	return agents, nil
}

func (d *domainStore) CreateExecution(ctx context.Context, execution cruxapi.Execution) error {
	if execution.ID == "" {
		return fmt.Errorf("execution id is required")
	}
	data, err := json.Marshal(execution)
	if err != nil {
		return err
	}
	return d.store.Put(ctx, filepath.Join("executions", execution.ID, "execution.json"), data)
}

func (d *domainStore) UpdateExecution(ctx context.Context, execution cruxapi.Execution) error {
	_, err := d.GetExecution(ctx, execution.ID)
	if err != nil {
		return err
	}
	data, err := json.Marshal(execution)
	if err != nil {
		return err
	}
	return d.store.Put(ctx, filepath.Join("executions", execution.ID, "execution.json"), data)
}

func (d *domainStore) GetExecution(ctx context.Context, id string) (cruxapi.Execution, error) {
	data, err := d.store.Get(ctx, filepath.Join("executions", id, "execution.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return cruxapi.Execution{}, ErrNotFound
		}
		return cruxapi.Execution{}, err
	}
	var execution cruxapi.Execution
	if err := json.Unmarshal(data, &execution); err != nil {
		return cruxapi.Execution{}, err
	}
	return execution, nil
}

func (d *domainStore) ListExecutions(ctx context.Context) ([]cruxapi.Execution, error) {
	objs, err := d.store.List(ctx, "executions")
	if err != nil {
		return nil, err
	}
	var executions []cruxapi.Execution
	for _, obj := range objs {
		if filepath.Base(obj.Path) != "execution.json" {
			continue
		}
		data, err := d.store.Get(ctx, obj.Path)
		if err != nil {
			continue
		}
		var execution cruxapi.Execution
		if err := json.Unmarshal(data, &execution); err != nil {
			continue
		}
		executions = append(executions, execution)
	}
	sort.Slice(executions, func(i, j int) bool {
		return executions[i].QueuedAt.After(executions[j].QueuedAt)
	})
	return executions, nil
}

func (d *domainStore) AppendEvent(ctx context.Context, event cruxapi.Event) error {
	if event.ID == "" {
		event.ID = cruxapi.NewID("evt")
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = cruxapi.Now()
	}
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	data = append(data, '\n')

	d.mu.Lock()
	defer d.mu.Unlock()

	if err := d.store.Append(ctx, "events/events.jsonl", data); err != nil {
		return err
	}

	// Apply retention
	cfg, err := d.RuntimeConfig(ctx)
	if err != nil {
		return nil
	}
	if cfg.TraceRetentionEntries <= 0 {
		return nil
	}
	all, err := d.ListEvents(ctx, "")
	if err != nil {
		return nil
	}
	if len(all) <= cfg.TraceRetentionEntries {
		return nil
	}
	keep := all[len(all)-cfg.TraceRetentionEntries:]
	var buf bytes.Buffer
	for _, e := range keep {
		b, _ := json.Marshal(e)
		buf.Write(b)
		buf.WriteByte('\n')
	}
	return d.store.Put(ctx, "events/events.jsonl", buf.Bytes())
}

func (d *domainStore) ListEvents(ctx context.Context, executionID string) ([]cruxapi.Event, error) {
	data, err := d.store.Get(ctx, "events/events.jsonl")
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var events []cruxapi.Event
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		var e cruxapi.Event
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			continue
		}
		if executionID == "" || e.ExecutionID == executionID {
			events = append(events, e)
		}
	}
	sort.Slice(events, func(i, j int) bool {
		return events[i].CreatedAt.Before(events[j].CreatedAt)
	})
	return events, nil
}

package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/cruxctl/crux/internal/discovery"
	"github.com/cruxctl/crux/internal/domain"
	"github.com/cruxctl/crux/internal/runner"
	"github.com/cruxctl/crux/internal/store"
	"github.com/cruxctl/crux/internal/worker"
)

type Service struct {
	store      store.Store
	runner     runner.Runner
	discoverer discovery.Discoverer
	limiter    *worker.Limiter
	logger     *slog.Logger
}

type SubmitRequest struct {
	AgentName string `json:"agentName"`
	Prompt    string `json:"prompt"`
	Wait      bool   `json:"wait"`
}

func New(st store.Store, run runner.Runner, disc discovery.Discoverer, runtime domain.RuntimeConfig, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		store:      st,
		runner:     run,
		discoverer: disc,
		limiter:    worker.NewLimiter(runtime.WorkerConcurrency),
		logger:     logger,
	}
}

func (s *Service) RuntimeConfig(ctx context.Context) (domain.RuntimeConfig, error) {
	return s.store.RuntimeConfig(ctx)
}

func (s *Service) UpdateRuntimeConfig(ctx context.Context, patch domain.RuntimeConfigPatch) (domain.RuntimeConfig, error) {
	current, err := s.store.RuntimeConfig(ctx)
	if err != nil {
		return domain.RuntimeConfig{}, err
	}
	next := current.ApplyPatch(patch)
	if err := next.Validate(); err != nil {
		return domain.RuntimeConfig{}, err
	}
	if err := s.store.UpdateRuntimeConfig(ctx, next); err != nil {
		return domain.RuntimeConfig{}, err
	}
	s.limiter.SetLimit(next.WorkerConcurrency)
	_ = s.store.AppendEvent(ctx, domain.Event{
		Type:      domain.EventConfigUpdated,
		Message:   "runtime config updated",
		CreatedAt: domain.Now(),
		Data: map[string]any{
			"runtime": next,
		},
	})
	return next, nil
}

func (s *Service) UpsertAgent(ctx context.Context, agent domain.Agent) (domain.Agent, error) {
	if agent.Command.Path == "" {
		return domain.Agent{}, fmt.Errorf("agent command.path is required")
	}
	if err := s.store.UpsertAgent(ctx, agent); err != nil {
		return domain.Agent{}, err
	}
	saved, err := s.store.GetAgent(ctx, agent.Name)
	if err != nil {
		return domain.Agent{}, err
	}
	_ = s.store.AppendEvent(ctx, domain.Event{
		Type:      domain.EventAgentRegistered,
		AgentName: saved.Name,
		Message:   "agent registered",
		CreatedAt: domain.Now(),
	})
	return saved, nil
}

func (s *Service) DeleteAgent(ctx context.Context, name string) error {
	if err := s.store.DeleteAgent(ctx, name); err != nil {
		return err
	}
	return s.store.AppendEvent(ctx, domain.Event{
		Type:      domain.EventAgentDeleted,
		AgentName: domain.CleanAgentName(name),
		Message:   "agent deleted",
		CreatedAt: domain.Now(),
	})
}

func (s *Service) GetAgent(ctx context.Context, name string) (domain.Agent, error) {
	return s.store.GetAgent(ctx, name)
}

func (s *Service) ListAgents(ctx context.Context) ([]domain.Agent, error) {
	return s.store.ListAgents(ctx)
}

func (s *Service) Discover(ctx context.Context) ([]discovery.Result, error) {
	runtime, err := s.store.RuntimeConfig(ctx)
	if err != nil {
		return nil, err
	}
	results, err := s.discoverer.Discover(ctx, runtime.DiscoveryTimeoutSecs)
	if err != nil {
		return nil, err
	}
	for _, result := range results {
		if err := s.store.UpsertAgent(ctx, result.Agent); err != nil {
			return nil, err
		}
	}
	_ = s.store.AppendEvent(ctx, domain.Event{
		Type:      domain.EventDiscoveryRun,
		Message:   "managed CLI discovery completed",
		CreatedAt: domain.Now(),
		Data: map[string]any{
			"count": len(results),
		},
	})
	return results, nil
}

func (s *Service) SubmitExecution(ctx context.Context, req SubmitRequest) (domain.Execution, error) {
	agent, err := s.store.GetAgent(ctx, req.AgentName)
	if err != nil {
		return domain.Execution{}, err
	}
	if agent.Status != domain.AgentReady {
		return domain.Execution{}, fmt.Errorf("agent %s is %s", agent.Name, agent.Status)
	}
	runtime, err := s.store.RuntimeConfig(ctx)
	if err != nil {
		return domain.Execution{}, err
	}
	now := domain.Now()
	execution := domain.Execution{
		ID:            domain.NewID("exec"),
		AgentName:     agent.Name,
		Prompt:        req.Prompt,
		Status:        domain.ExecutionQueued,
		QueuedAt:      now,
		UpdatedAt:     now,
		RuntimeConfig: runtime,
	}
	if err := s.store.CreateExecution(ctx, execution); err != nil {
		return domain.Execution{}, err
	}
	_ = s.store.AppendEvent(ctx, domain.Event{
		Type:        domain.EventExecutionQueued,
		ExecutionID: execution.ID,
		AgentName:   agent.Name,
		Message:     "execution queued",
		CreatedAt:   domain.Now(),
	})
	if req.Wait {
		return s.runExecution(ctx, execution.ID)
	}
	go func() {
		if _, err := s.runExecution(context.Background(), execution.ID); err != nil {
			s.logger.Error("run execution", "execution", execution.ID, "error", err)
		}
	}()
	return execution, nil
}

func (s *Service) GetExecution(ctx context.Context, id string) (domain.Execution, error) {
	return s.store.GetExecution(ctx, id)
}

func (s *Service) ListExecutions(ctx context.Context) ([]domain.Execution, error) {
	return s.store.ListExecutions(ctx)
}

func (s *Service) ListEvents(ctx context.Context, executionID string) ([]domain.Event, error) {
	return s.store.ListEvents(ctx, executionID)
}

func (s *Service) runExecution(ctx context.Context, id string) (domain.Execution, error) {
	if err := s.limiter.Acquire(ctx); err != nil {
		return domain.Execution{}, err
	}
	defer s.limiter.Release()

	execution, err := s.store.GetExecution(ctx, id)
	if err != nil {
		return domain.Execution{}, err
	}
	agent, err := s.store.GetAgent(ctx, execution.AgentName)
	if err != nil {
		return domain.Execution{}, err
	}
	runtime, err := s.store.RuntimeConfig(ctx)
	if err != nil {
		return domain.Execution{}, err
	}

	started := domain.Now()
	execution.Status = domain.ExecutionRunning
	execution.StartedAt = &started
	execution.UpdatedAt = started
	if err := s.store.UpdateExecution(ctx, execution); err != nil {
		return domain.Execution{}, err
	}
	_ = s.store.AppendEvent(ctx, domain.Event{
		Type:        domain.EventExecutionStart,
		ExecutionID: execution.ID,
		AgentName:   execution.AgentName,
		Message:     "execution started",
		CreatedAt:   started,
	})

	result := s.runner.Run(ctx, agent, execution, runtime)
	completed := domain.Now()
	execution.Stdout = result.Stdout
	execution.Stderr = result.Stderr
	execution.ExitCode = result.ExitCode
	execution.Error = result.Error
	execution.CompletedAt = &completed
	execution.UpdatedAt = completed
	if result.Error != "" || result.ExitCode != 0 {
		execution.Status = domain.ExecutionFailed
	} else {
		execution.Status = domain.ExecutionSucceeded
	}
	if err := s.store.UpdateExecution(ctx, execution); err != nil {
		return domain.Execution{}, err
	}
	eventType := domain.EventExecutionFinish
	message := "execution finished"
	if execution.Status == domain.ExecutionFailed {
		eventType = domain.EventExecutionFail
		message = "execution failed"
	}
	_ = s.store.AppendEvent(ctx, domain.Event{
		Type:        eventType,
		ExecutionID: execution.ID,
		AgentName:   execution.AgentName,
		Message:     message,
		CreatedAt:   completed,
		Data: map[string]any{
			"exitCode": execution.ExitCode,
			"error":    execution.Error,
		},
	})
	return execution, nil
}

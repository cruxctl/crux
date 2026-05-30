package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cruxctl/crux/internal/discovery"
	"github.com/cruxctl/crux/internal/managedops"
	"github.com/cruxctl/crux/internal/runner"
	"github.com/cruxctl/crux/internal/state"
	"github.com/cruxctl/crux/internal/worker"
	"github.com/cruxctl/crux/pkg/cruxapi"
)

type Service struct {
	store      store.DomainStore
	runner     runner.Runner
	discoverer *discovery.Discoverer
	limiter    *worker.Limiter
	logger     *slog.Logger
}

func New(st store.DomainStore, run runner.Runner, disc *discovery.Discoverer, runtime cruxapi.RuntimeConfig, logger *slog.Logger) *Service {
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

func (s *Service) Version() (string, string) {
	return cruxapi.Version, "dev"
}

func (s *Service) RuntimeConfig(ctx context.Context) (cruxapi.RuntimeConfig, error) {
	return s.store.RuntimeConfig(ctx)
}

func (s *Service) RecoverInterruptedExecutions(ctx context.Context) error {
	executions, err := s.store.ListExecutions(ctx)
	if err != nil {
		return err
	}
	now := cruxapi.Now()
	for _, execution := range executions {
		if execution.Status != cruxapi.ExecutionQueued && execution.Status != cruxapi.ExecutionRunning {
			continue
		}
		execution.Status = cruxapi.ExecutionFailed
		execution.ExitCode = 124
		execution.Error = "execution interrupted by daemon restart"
		if execution.StartedAt == nil {
			started := execution.QueuedAt
			execution.StartedAt = &started
		}
		execution.CompletedAt = &now
		execution.UpdatedAt = now
		if err := s.store.UpdateExecution(ctx, execution); err != nil {
			return err
		}
		_ = s.store.AppendEvent(ctx, cruxapi.Event{
			Type:        cruxapi.EventExecutionFail,
			ExecutionID: execution.ID,
			AgentName:   execution.AgentName,
			Message:     "execution interrupted by daemon restart",
			CreatedAt:   now,
			Data: map[string]any{
				"exitCode": execution.ExitCode,
				"error":    execution.Error,
			},
		})
	}
	return nil
}

func (s *Service) UpdateRuntimeConfig(ctx context.Context, patch cruxapi.RuntimeConfigPatch) (cruxapi.RuntimeConfig, error) {
	current, err := s.store.RuntimeConfig(ctx)
	if err != nil {
		return cruxapi.RuntimeConfig{}, err
	}
	next := current.ApplyPatch(patch)
	if err := next.Validate(); err != nil {
		return cruxapi.RuntimeConfig{}, err
	}
	if err := s.store.UpdateRuntimeConfig(ctx, next); err != nil {
		return cruxapi.RuntimeConfig{}, err
	}
	s.limiter.SetLimit(next.WorkerConcurrency)
	_ = s.store.AppendEvent(ctx, cruxapi.Event{
		Type:      cruxapi.EventConfigUpdated,
		Message:   "runtime config updated",
		CreatedAt: cruxapi.Now(),
		Data: map[string]any{
			"runtime": next,
		},
	})
	return next, nil
}

func (s *Service) UpsertAgent(ctx context.Context, agent cruxapi.Agent) (cruxapi.Agent, error) {
	if agent.Command.Path == "" {
		return cruxapi.Agent{}, fmt.Errorf("agent command.path is required")
	}
	if err := s.store.UpsertAgent(ctx, agent); err != nil {
		return cruxapi.Agent{}, err
	}
	saved, err := s.store.GetAgent(ctx, agent.Name)
	if err != nil {
		return cruxapi.Agent{}, err
	}
	_ = s.store.AppendEvent(ctx, cruxapi.Event{
		Type:      cruxapi.EventAgentRegistered,
		AgentName: saved.Name,
		Message:   "agent registered",
		CreatedAt: cruxapi.Now(),
	})
	return saved, nil
}

func (s *Service) DeleteAgent(ctx context.Context, name string) error {
	if err := s.store.DeleteAgent(ctx, name); err != nil {
		return err
	}
	return s.store.AppendEvent(ctx, cruxapi.Event{
		Type:      cruxapi.EventAgentDeleted,
		AgentName: cruxapi.CleanAgentName(name),
		Message:   "agent deleted",
		CreatedAt: cruxapi.Now(),
	})
}

func (s *Service) GetAgent(ctx context.Context, name string) (cruxapi.Agent, error) {
	return s.store.GetAgent(ctx, name)
}

func (s *Service) AgentUsage(ctx context.Context, name string) (cruxapi.AgentUsage, error) {
	agent, err := s.store.GetAgent(ctx, name)
	if err != nil {
		return cruxapi.AgentUsage{}, err
	}
	executions, err := s.store.ListExecutions(ctx)
	if err != nil {
		return cruxapi.AgentUsage{}, err
	}
	events, err := s.store.ListEvents(ctx, "")
	if err != nil {
		return cruxapi.AgentUsage{}, err
	}
	usage := cruxapi.AgentUsage{
		AgentID:     agent.ID,
		AgentName:   agent.Name,
		Description: agent.Description,
		Status:      agent.Status,
		CommandPath: agent.Command.Path,
		CommandArgs: agent.Command.Args,
		Version:     agent.Labels["cruxctl.io/version"],
		Labels:      agent.Labels,
		ExitCodes:   map[string]int{},
		EventCounts: map[cruxapi.EventType]int{},
	}
	for _, event := range events {
		if event.AgentName == agent.Name {
			usage.EventCounts[event.Type]++
		}
	}
	for _, execution := range executions {
		if execution.AgentName != agent.Name {
			continue
		}
		usage.ExecutionsTotal++
		switch execution.Status {
		case cruxapi.ExecutionQueued:
			usage.Queued++
		case cruxapi.ExecutionRunning:
			usage.Running++
		case cruxapi.ExecutionSucceeded:
			usage.Succeeded++
		case cruxapi.ExecutionFailed:
			usage.Failed++
		case cruxapi.ExecutionCanceled:
			usage.Canceled++
		}
		if execution.Error != "" {
			usage.ErrorCount++
		}
		usage.ExitCodes[strconv.Itoa(execution.ExitCode)]++
		usage.StdoutBytes += len(execution.Stdout)
		usage.StderrBytes += len(execution.Stderr)
		if usage.FirstQueuedAt == nil || execution.QueuedAt.Before(*usage.FirstQueuedAt) {
			t := execution.QueuedAt
			usage.FirstQueuedAt = &t
		}
		if usage.LastQueuedAt == nil || execution.QueuedAt.After(*usage.LastQueuedAt) {
			t := execution.QueuedAt
			usage.LastQueuedAt = &t
			usage.LastExecutionID = execution.ID
			usage.LastStatus = execution.Status
			usage.LastExitCode = execution.ExitCode
			usage.LastError = execution.Error
			usage.LastStdout = execution.Stdout
			usage.LastStderr = execution.Stderr
		}
		if execution.StartedAt != nil && execution.CompletedAt != nil {
			duration := execution.CompletedAt.Sub(*execution.StartedAt).Seconds()
			if duration > 0 {
				usage.TotalDurationSeconds += duration
				if duration > usage.MaxDurationSeconds {
					usage.MaxDurationSeconds = duration
				}
			}
		}
	}
	completed := usage.Succeeded + usage.Failed + usage.Canceled
	if completed > 0 {
		usage.SuccessRate = float64(usage.Succeeded) / float64(completed)
		usage.AverageDurationSeconds = usage.TotalDurationSeconds / float64(completed)
	}
	if usage.ExecutionsTotal == 0 {
		usage.Notes = append(usage.Notes, "No executions have been recorded for this agent.")
	}
	usage.Notes = append(usage.Notes, "Crux reports transcript-backed TTY execution metrics from daemon state. Refresh provider usage/cost/session evidence by running the agent through `crux agent <name> exec` or `crux run`.")
	return usage, nil
}

func (s *Service) AgentCapabilities(ctx context.Context, name string) (cruxapi.AgentCapabilities, error) {
	agent, err := s.store.GetAgent(ctx, name)
	if err != nil {
		return cruxapi.AgentCapabilities{}, err
	}
	return managedops.Capabilities(agent), nil
}

func (s *Service) AgentCost(ctx context.Context, name string) (cruxapi.AgentCostSnapshot, error) {
	usage, err := s.AgentUsage(ctx, name)
	if err != nil {
		return cruxapi.AgentCostSnapshot{}, err
	}
	snapshot := managedops.CostSnapshot(usage)
	executions, err := s.store.ListExecutions(ctx)
	if err != nil {
		return cruxapi.AgentCostSnapshot{}, err
	}
	now := cruxapi.Now()
	for _, execution := range executions {
		if execution.AgentName != usage.AgentName || execution.Status != cruxapi.ExecutionRunning || execution.StartedAt == nil {
			continue
		}
		snapshot.RealtimeRunningSeconds += now.Sub(*execution.StartedAt).Seconds()
	}
	return snapshot, nil
}

func (s *Service) AgentSessions(ctx context.Context, name string) ([]cruxapi.AgentSession, error) {
	agent, err := s.store.GetAgent(ctx, name)
	if err != nil {
		return nil, err
	}
	executions, err := s.store.ListExecutions(ctx)
	if err != nil {
		return nil, err
	}
	return managedops.Sessions(ctx, agent, executions, 10*time.Second), nil
}

func (s *Service) AgentHistory(ctx context.Context, name string) ([]cruxapi.AgentHistoryItem, error) {
	agent, err := s.store.GetAgent(ctx, name)
	if err != nil {
		return nil, err
	}
	executions, err := s.store.ListExecutions(ctx)
	if err != nil {
		return nil, err
	}
	return managedops.History(executions, agent.Name), nil
}

func (s *Service) AgentExecPlan(ctx context.Context, name string, req cruxapi.AgentExecPlanRequest) (cruxapi.AgentExecPlan, error) {
	agent, err := s.store.GetAgent(ctx, name)
	if err != nil {
		return cruxapi.AgentExecPlan{}, err
	}
	if agent.Status != cruxapi.AgentReady {
		return cruxapi.AgentExecPlan{}, fmt.Errorf("agent %s is %s", agent.Name, agent.Status)
	}
	if workingDir := strings.TrimSpace(req.WorkingDir); workingDir != "" && !filepath.IsAbs(workingDir) {
		return cruxapi.AgentExecPlan{}, fmt.Errorf("workingDir must be an absolute path")
	}
	planned, err := managedops.ExecAgent(agent, req)
	if err != nil {
		return cruxapi.AgentExecPlan{}, err
	}
	capability := managedops.Capabilities(agent)
	return cruxapi.AgentExecPlan{
		AgentName:     agent.Name,
		Provider:      capability.Provider,
		Command:       planned.Command,
		ResumeSession: strings.TrimSpace(req.ResumeSession),
		Prompt:        strings.TrimSpace(req.Prompt),
		Input:         managedops.ExecInput(agent, req, planned),
		Operation:     strings.TrimSpace(req.Operation),
		Notes: []string{
			"Run this command in a PTY to open the provider TUI. Import the transcript through exec/record when it exits so Crux-owned usage and history stay current.",
		},
	}, nil
}

func (s *Service) RecordAgentExec(ctx context.Context, name string, req cruxapi.AgentExecRecordRequest) (cruxapi.AgentExecRecordResponse, error) {
	agent, err := s.store.GetAgent(ctx, name)
	if err != nil {
		return cruxapi.AgentExecRecordResponse{}, err
	}
	runtime, err := s.store.RuntimeConfig(ctx)
	if err != nil {
		return cruxapi.AgentExecRecordResponse{}, err
	}
	workingDir := strings.TrimSpace(req.WorkingDir)
	if workingDir != "" && !filepath.IsAbs(workingDir) {
		return cruxapi.AgentExecRecordResponse{}, fmt.Errorf("workingDir must be an absolute path")
	}
	started := req.StartedAt
	if started.IsZero() {
		started = cruxapi.Now()
	}
	completed := req.CompletedAt
	if completed.IsZero() || completed.Before(started) {
		completed = cruxapi.Now()
	}
	exitCode := req.ExitCode
	errText := strings.TrimSpace(req.Error)
	if errText != "" && exitCode == 0 {
		exitCode = 1
	}
	status := cruxapi.ExecutionSucceeded
	if errText != "" || exitCode != 0 {
		status = cruxapi.ExecutionFailed
	}
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		prompt = "interactive TTY exec"
	}
	if strings.TrimSpace(req.TranscriptPath) != "" {
		prompt += " transcript=" + strings.TrimSpace(req.TranscriptPath)
	}
	execution := cruxapi.Execution{
		ID:             cruxapi.NewID("exec"),
		AgentName:      agent.Name,
		Prompt:         prompt,
		WorkingDir:     workingDir,
		ResumeSession:  strings.TrimSpace(req.ResumeSession),
		SourceExecID:   strings.TrimSpace(req.SourceExecID),
		FallbackAgents: cleanAgentList(req.FallbackAgents),
		Status:         status,
		ExitCode:       exitCode,
		Stdout:         truncateString(req.Transcript, runtime.MaxOutputBytes),
		Stderr:         truncateString(req.Stderr, runtime.MaxOutputBytes),
		Error:          errText,
		QueuedAt:       started,
		StartedAt:      &started,
		CompletedAt:    &completed,
		UpdatedAt:      completed,
		RuntimeConfig:  runtime,
	}
	if err := s.store.CreateExecution(ctx, execution); err != nil {
		return cruxapi.AgentExecRecordResponse{}, err
	}
	for _, event := range []cruxapi.Event{
		{Type: cruxapi.EventExecutionQueued, ExecutionID: execution.ID, AgentName: execution.AgentName, Message: "interactive TTY execution recorded", CreatedAt: started},
		{Type: cruxapi.EventExecutionStart, ExecutionID: execution.ID, AgentName: execution.AgentName, Message: "interactive TTY execution started", CreatedAt: started},
		{Type: finishEventType(status), ExecutionID: execution.ID, AgentName: execution.AgentName, Message: finishEventMessage(status), CreatedAt: completed, Data: map[string]any{
			"driver":         strings.TrimSpace(req.Driver),
			"transcriptPath": strings.TrimSpace(req.TranscriptPath),
			"exitCode":       execution.ExitCode,
			"error":          execution.Error,
			"args":           req.Args,
			"operation":      strings.TrimSpace(req.Operation),
		}},
	} {
		if err := s.store.AppendEvent(ctx, event); err != nil {
			return cruxapi.AgentExecRecordResponse{}, err
		}
	}
	usage, err := s.AgentUsage(ctx, agent.Name)
	if err != nil {
		return cruxapi.AgentExecRecordResponse{}, err
	}
	cost, err := s.AgentCost(ctx, agent.Name)
	if err != nil {
		return cruxapi.AgentExecRecordResponse{}, err
	}
	sessions, err := s.AgentSessions(ctx, agent.Name)
	if err != nil {
		return cruxapi.AgentExecRecordResponse{}, err
	}
	return cruxapi.AgentExecRecordResponse{
		Execution: execution,
		Usage:     usage,
		Cost:      cost,
		Sessions:  sessions,
	}, nil
}

func (s *Service) ListAgents(ctx context.Context) ([]cruxapi.Agent, error) {
	return s.store.ListAgents(ctx)
}

func (s *Service) Discover(ctx context.Context) ([]cruxapi.DiscoveryResult, error) {
	res, err := s.discoverer.Discover(ctx, "")
	if err != nil {
		return nil, err
	}
	results := make([]cruxapi.DiscoveryResult, 0, len(res.Instances))
	for _, inst := range res.Instances {
		agent := cruxapi.Agent{
			ID:          inst.SpecID,
			Name:        inst.Name,
			Description: inst.Provider + " agent",
			Labels:      inst.Labels,
			Status:      cruxapi.AgentReady,
			Command: cruxapi.CommandSpec{
				Path: inst.BinaryPath,
			},
		}
		if err := s.store.UpsertAgent(ctx, agent); err != nil {
			return nil, err
		}
		results = append(results, cruxapi.DiscoveryResult{Agent: agent, Version: inst.Version})
	}
	_ = s.store.AppendEvent(ctx, cruxapi.Event{
		Type:      cruxapi.EventDiscoveryRun,
		Message:   "managed CLI discovery completed",
		CreatedAt: cruxapi.Now(),
		Data: map[string]any{
			"count": len(results),
		},
	})
	return results, nil
}

func (s *Service) SubmitExecution(ctx context.Context, req cruxapi.SubmitExecutionRequest) (cruxapi.Execution, error) {
	agent, err := s.store.GetAgent(ctx, req.AgentName)
	if err != nil {
		return cruxapi.Execution{}, err
	}
	if agent.Status != cruxapi.AgentReady {
		return cruxapi.Execution{}, fmt.Errorf("agent %s is %s", agent.Name, agent.Status)
	}
	runtime, err := s.store.RuntimeConfig(ctx)
	if err != nil {
		return cruxapi.Execution{}, err
	}
	workingDir := strings.TrimSpace(req.WorkingDir)
	if workingDir != "" && !filepath.IsAbs(workingDir) {
		return cruxapi.Execution{}, fmt.Errorf("workingDir must be an absolute path")
	}
	now := cruxapi.Now()
	execution := cruxapi.Execution{
		ID:             cruxapi.NewID("exec"),
		AgentName:      agent.Name,
		Prompt:         req.Prompt,
		WorkingDir:     workingDir,
		ResumeSession:  strings.TrimSpace(req.ResumeSession),
		SourceExecID:   strings.TrimSpace(req.SourceExecID),
		FallbackAgents: cleanAgentList(req.FallbackAgents),
		Status:         cruxapi.ExecutionQueued,
		QueuedAt:       now,
		UpdatedAt:      now,
		RuntimeConfig:  runtime,
	}
	if err := s.store.CreateExecution(ctx, execution); err != nil {
		return cruxapi.Execution{}, err
	}
	_ = s.store.AppendEvent(ctx, cruxapi.Event{
		Type:        cruxapi.EventExecutionQueued,
		ExecutionID: execution.ID,
		AgentName:   agent.Name,
		Message:     "execution queued",
		CreatedAt:   cruxapi.Now(),
	})
	if req.Wait {
		return s.runExecution(context.Background(), execution.ID)
	}
	go func() {
		if _, err := s.runExecution(context.Background(), execution.ID); err != nil {
			s.logger.Error("run execution", "execution", execution.ID, "error", err)
		}
	}()
	return execution, nil
}

func (s *Service) GetExecution(ctx context.Context, id string) (cruxapi.Execution, error) {
	return s.store.GetExecution(ctx, id)
}

func (s *Service) ListExecutions(ctx context.Context) ([]cruxapi.Execution, error) {
	return s.store.ListExecutions(ctx)
}

func (s *Service) ListEvents(ctx context.Context, executionID string) ([]cruxapi.Event, error) {
	return s.store.ListEvents(ctx, executionID)
}

func (s *Service) runExecution(ctx context.Context, id string) (cruxapi.Execution, error) {
	if err := s.limiter.Acquire(ctx); err != nil {
		execution, finishErr := s.finishExecution(context.Background(), id, runner.Result{ExitCode: 124, Error: err.Error()})
		if finishErr != nil {
			return cruxapi.Execution{}, finishErr
		}
		return execution, err
	}
	defer s.limiter.Release()

	execution, err := s.store.GetExecution(ctx, id)
	if err != nil {
		return cruxapi.Execution{}, err
	}
	agent, err := s.store.GetAgent(ctx, execution.AgentName)
	if err != nil {
		return cruxapi.Execution{}, err
	}
	if execution.ResumeSession != "" {
		agent, err = managedops.ResumeAgent(agent, execution.ResumeSession)
		if err != nil {
			return s.finishExecution(context.Background(), id, runner.Result{ExitCode: 1, Error: err.Error()})
		}
		if strings.TrimSpace(execution.WorkingDir) == "" && strings.TrimSpace(agent.Command.WorkingDir) != "" {
			execution.WorkingDir = strings.TrimSpace(agent.Command.WorkingDir)
		}
	}
	runtime, err := s.store.RuntimeConfig(ctx)
	if err != nil {
		return cruxapi.Execution{}, err
	}

	started := cruxapi.Now()
	execution.Status = cruxapi.ExecutionRunning
	execution.StartedAt = &started
	execution.UpdatedAt = started
	if err := s.store.UpdateExecution(ctx, execution); err != nil {
		return cruxapi.Execution{}, err
	}
	_ = s.store.AppendEvent(ctx, cruxapi.Event{
		Type:        cruxapi.EventExecutionStart,
		ExecutionID: execution.ID,
		AgentName:   execution.AgentName,
		Message:     "execution started",
		CreatedAt:   started,
	})

	result := s.runner.Run(ctx, agent, execution, runtime)
	return s.finishExecution(context.Background(), id, result)
}

func (s *Service) finishExecution(ctx context.Context, id string, result runner.Result) (cruxapi.Execution, error) {
	execution, err := s.store.GetExecution(ctx, id)
	if err != nil {
		return cruxapi.Execution{}, err
	}
	completed := cruxapi.Now()
	execution.Stdout = result.Stdout
	execution.Stderr = result.Stderr
	execution.ExitCode = result.ExitCode
	execution.Error = result.Error
	execution.CompletedAt = &completed
	execution.UpdatedAt = completed
	if result.Error != "" || result.ExitCode != 0 {
		execution.Status = cruxapi.ExecutionFailed
	} else {
		execution.Status = cruxapi.ExecutionSucceeded
	}
	if err := s.store.UpdateExecution(ctx, execution); err != nil {
		return cruxapi.Execution{}, err
	}
	eventType := cruxapi.EventExecutionFinish
	message := "execution finished"
	if execution.Status == cruxapi.ExecutionFailed {
		eventType = cruxapi.EventExecutionFail
		message = "execution failed"
	}
	_ = s.store.AppendEvent(ctx, cruxapi.Event{
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

func finishEventType(status cruxapi.ExecutionStatus) cruxapi.EventType {
	if status == cruxapi.ExecutionFailed {
		return cruxapi.EventExecutionFail
	}
	return cruxapi.EventExecutionFinish
}

func finishEventMessage(status cruxapi.ExecutionStatus) string {
	if status == cruxapi.ExecutionFailed {
		return "interactive TTY execution failed"
	}
	return "interactive TTY execution finished"
}

func truncateString(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}

func cleanAgentList(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		name := cruxapi.CleanAgentName(value)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}

// --- Plan 04: Discovery + Inject Gateway ---

func (s *Service) DiscoverAgents(ctx context.Context, req cruxapi.DiscoveryRequest) ([]cruxapi.DiscoveryResult, error) {
	res, err := s.discoverer.Discover(ctx, req.Agent)
	if err != nil {
		return nil, err
	}
	results := make([]cruxapi.DiscoveryResult, 0, len(res.Instances))
	for _, inst := range res.Instances {
		agent := cruxapi.Agent{
			ID:          inst.SpecID,
			Name:        inst.Name,
			Description: inst.Provider + " agent",
			Labels:      inst.Labels,
			Status:      cruxapi.AgentReady,
			Command: cruxapi.CommandSpec{
				Path: inst.BinaryPath,
			},
		}
		if err := s.store.UpsertAgent(ctx, agent); err != nil {
			return nil, err
		}
		results = append(results, cruxapi.DiscoveryResult{Agent: agent, Version: inst.Version})
	}
	return results, nil
}

func (s *Service) InjectGateway(ctx context.Context, req cruxapi.GatewayInjectRequest) ([]cruxapi.GatewayInjectResult, error) {
	var results []cruxapi.GatewayInjectResult
	for _, agentID := range req.Agents {
		results = append(results, cruxapi.GatewayInjectResult{
			AgentID: agentID,
			Status:  "injected",
		})
	}
	return results, nil
}

func (s *Service) UndoGatewayInject(ctx context.Context, req cruxapi.GatewayInjectRequest) ([]cruxapi.GatewayInjectResult, error) {
	var results []cruxapi.GatewayInjectResult
	for _, agentID := range req.Agents {
		results = append(results, cruxapi.GatewayInjectResult{
			AgentID: agentID,
			Status:  "reverted",
		})
	}
	return results, nil
}

// --- Plan 05: Session Manager ---

func (s *Service) ListSessions(ctx context.Context) ([]cruxapi.Session, error) {
	return s.store.ListSessions(ctx)
}

func (s *Service) GetSession(ctx context.Context, id string) (cruxapi.Session, error) {
	return s.store.GetSession(ctx, id)
}

func (s *Service) CreateSession(ctx context.Context, sess cruxapi.Session) (cruxapi.Session, error) {
	if sess.ID == "" {
		sess.ID = cruxapi.NewID("sess")
	}
	if sess.Status == "" {
		sess.Status = cruxapi.SessionStatusCreated
	}
	if sess.StartedAt.IsZero() {
		sess.StartedAt = time.Now().UTC()
	}
	if err := s.store.CreateSession(ctx, sess); err != nil {
		return cruxapi.Session{}, err
	}
	return sess, nil
}

func (s *Service) GetSessionEvents(ctx context.Context, id string) ([]cruxapi.AOSEvent, error) {
	events, err := s.store.ListAOSEvents(ctx)
	if err != nil {
		return nil, err
	}
	var filtered []cruxapi.AOSEvent
	for _, e := range events {
		if e.Actor.SessionID == id {
			filtered = append(filtered, e)
		}
	}
	return filtered, nil
}

func (s *Service) GetSessionTranscript(ctx context.Context, id string, format string) ([]byte, error) {
	sess, err := s.store.GetSession(ctx, id)
	if err != nil {
		return nil, err
	}
	// TODO: read actual transcript files from sess.TranscriptPaths
	_ = sess
	return []byte("transcript placeholder"), nil
}

func (s *Service) ContinueSession(ctx context.Context, id string, req cruxapi.ContinueSessionRequest) (cruxapi.Session, error) {
	sess, err := s.store.GetSession(ctx, id)
	if err != nil {
		return cruxapi.Session{}, err
	}
	sess.Status = cruxapi.SessionStatusContinued
	if err := s.store.UpdateSession(ctx, sess); err != nil {
		return cruxapi.Session{}, err
	}
	return sess, nil
}

func (s *Service) StopSession(ctx context.Context, id string) error {
	sess, err := s.store.GetSession(ctx, id)
	if err != nil {
		return err
	}
	sess.Status = cruxapi.SessionStatusCanceled
	now := time.Now().UTC()
	sess.EndedAt = &now
	return s.store.UpdateSession(ctx, sess)
}

func (s *Service) ReplaySession(ctx context.Context, id string) (cruxapi.Session, error) {
	sess, err := s.store.GetSession(ctx, id)
	if err != nil {
		return cruxapi.Session{}, err
	}
	newSess := sess
	newSess.ID = cruxapi.NewID("sess")
	newSess.ParentSessionID = id
	newSess.Status = cruxapi.SessionStatusRunning
	newSess.StartedAt = time.Now().UTC()
	newSess.EndedAt = nil
	if err := s.store.CreateSession(ctx, newSess); err != nil {
		return cruxapi.Session{}, err
	}
	return newSess, nil
}

func (s *Service) GetContextIR(ctx context.Context, sessionID string) (cruxapi.ContextIR, error) {
	return cruxapi.ContextIR{Schema: "crux.context-ir.v1", SessionID: sessionID}, nil
}

// --- Plan 06: Usage + Cost ---

func (s *Service) GetUsage(ctx context.Context, agent, project, since string) ([]cruxapi.UsageReport, error) {
	return []cruxapi.UsageReport{}, nil
}

func (s *Service) GetCosts(ctx context.Context, agent, project, since string) ([]cruxapi.CostReport, error) {
	return []cruxapi.CostReport{}, nil
}

func (s *Service) GetUsageLimits(ctx context.Context) (cruxapi.UsageLimits, error) {
	return s.store.GetUsageLimits(ctx)
}

func (s *Service) SetUsageLimits(ctx context.Context, limits cruxapi.UsageLimits) (cruxapi.UsageLimits, error) {
	if err := s.store.SetUsageLimits(ctx, limits); err != nil {
		return cruxapi.UsageLimits{}, err
	}
	return limits, nil
}

// --- Plan 08: Policy + Approvals ---

func (s *Service) ListPolicies(ctx context.Context) ([]cruxapi.PolicyProfile, error) {
	return s.store.ListPolicies(ctx)
}

func (s *Service) CreatePolicy(ctx context.Context, policy cruxapi.PolicyProfile) (cruxapi.PolicyProfile, error) {
	if policy.Metadata.ID == "" {
		policy.Metadata.ID = cruxapi.NewID("pol")
	}
	if err := s.store.UpsertPolicy(ctx, policy); err != nil {
		return cruxapi.PolicyProfile{}, err
	}
	return policy, nil
}

func (s *Service) GetPolicy(ctx context.Context, id string) (cruxapi.PolicyProfile, error) {
	return s.store.GetPolicy(ctx, id)
}

func (s *Service) DeletePolicy(ctx context.Context, id string) error {
	return s.store.DeletePolicy(ctx, id)
}

func (s *Service) EvaluatePolicy(ctx context.Context, id string, input cruxapi.EvaluationInput) (cruxapi.Decision, error) {
	policy, err := s.store.GetPolicy(ctx, id)
	if err != nil {
		return cruxapi.Decision{}, err
	}
	// Simple rule matching: check if any rule matches the input tool
	for _, rule := range policy.Rules {
		if matchRule(rule, input) {
			return cruxapi.Decision{
				Action: cruxapi.PolicyAction(rule.Action),
				RuleID: rule.ID,
				Reason: "matched rule " + rule.ID,
			}, nil
		}
	}
	return cruxapi.Decision{Action: cruxapi.PolicyAllow, Reason: "default allow (no rules matched)"}, nil
}

func (s *Service) SimulatePolicy(ctx context.Context, id string, input cruxapi.EvaluationInput) (cruxapi.Decision, error) {
	// Simulation is the same as evaluation but without side effects
	return s.EvaluatePolicy(ctx, id, input)
}

func matchRule(rule cruxapi.PolicyRule, input cruxapi.EvaluationInput) bool {
	if toolMatch, ok := rule.Match["tool"].(string); ok && toolMatch != "" {
		if input.Tool != toolMatch {
			return false
		}
	}
	if agentMatch, ok := rule.Match["agent"].(string); ok && agentMatch != "" {
		if input.Actor.AgentID != agentMatch {
			return false
		}
	}
	return true
}

func (s *Service) ListApprovals(ctx context.Context) ([]cruxapi.ApprovalRecord, error) {
	return s.store.ListApprovals(ctx)
}

func (s *Service) GetApproval(ctx context.Context, id string) (cruxapi.ApprovalRecord, error) {
	return s.store.GetApproval(ctx, id)
}

func (s *Service) GrantApproval(ctx context.Context, id string) (cruxapi.ApprovalRecord, error) {
	rec, err := s.store.GetApproval(ctx, id)
	if err != nil {
		return cruxapi.ApprovalRecord{}, err
	}
	rec.Status = "granted"
	if err := s.store.UpdateApproval(ctx, rec); err != nil {
		return cruxapi.ApprovalRecord{}, err
	}
	return rec, nil
}

func (s *Service) DenyApproval(ctx context.Context, id string) (cruxapi.ApprovalRecord, error) {
	rec, err := s.store.GetApproval(ctx, id)
	if err != nil {
		return cruxapi.ApprovalRecord{}, err
	}
	rec.Status = "denied"
	if err := s.store.UpdateApproval(ctx, rec); err != nil {
		return cruxapi.ApprovalRecord{}, err
	}
	return rec, nil
}

// --- Plan 09: AOS Event Stream ---

func (s *Service) ListAOSEvents(ctx context.Context, filter cruxapi.EventFilter) ([]cruxapi.AOSEvent, error) {
	return s.store.ListAOSEvents(ctx)
}

func (s *Service) GetAOSEvent(ctx context.Context, id string) (cruxapi.AOSEvent, error) {
	return s.store.GetAOSEvent(ctx, id)
}

func (s *Service) ExportAOSEvents(ctx context.Context, req cruxapi.AOSExportRequest) ([]byte, error) {
	events, err := s.store.ListAOSEvents(ctx)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	for _, e := range events {
		b, _ := json.Marshal(e)
		buf.Write(b)
		buf.WriteByte('\n')
	}
	return buf.Bytes(), nil
}

func (s *Service) ListTraces(ctx context.Context, since string) ([]cruxapi.AOSEvent, error) {
	return s.store.ListAOSEvents(ctx)
}

func (s *Service) GetTrace(ctx context.Context, id string) (cruxapi.AOSEvent, error) {
	return s.store.GetAOSEvent(ctx, id)
}

// --- Plan 10: AgBOM ---

func (s *Service) GenerateAgBOM(ctx context.Context, agentID, projectID, sessionID string) (cruxapi.AgBOM, error) {
	agents, err := s.store.ListAgents(ctx)
	if err != nil {
		return cruxapi.AgBOM{}, err
	}
	var target cruxapi.Agent
	for _, a := range agents {
		if a.ID == agentID || a.Name == agentID {
			target = a
			break
		}
	}
	sessions, _ := s.store.ListSessions(ctx)
	provider := target.Labels["cruxctl.io/provider"]
	return cruxapi.AgBOM{
		Schema: "crux.agbom.v1",
		Agent:  cruxapi.AgentRef{Name: target.Name, Provider: provider},
		Sessions: cruxapi.AgBOMSessions{
			Count:    len(sessions),
			LastSeen: time.Now().UTC(),
		},
	}, nil
}

func (s *Service) GetAgBOM(ctx context.Context, id string) (cruxapi.AgBOM, error) {
	return s.GenerateAgBOM(ctx, id, "", "")
}

func (s *Service) ExportAgBOM(ctx context.Context, id string, req cruxapi.AgBOMExportRequest) ([]byte, error) {
	bom, err := s.GetAgBOM(ctx, id)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(bom, "", "  ")
}

func (s *Service) DiffAgBOM(ctx context.Context, id string, since string) (cruxapi.AgBOMDiff, error) {
	return cruxapi.AgBOMDiff{}, nil
}

// --- Plan 11: Scheduler ---

func (s *Service) ListJobs(ctx context.Context) ([]cruxapi.Job, error) {
	return s.store.ListJobs(ctx)
}

func (s *Service) CreateJob(ctx context.Context, job cruxapi.Job) (cruxapi.Job, error) {
	if job.ID == "" {
		job.ID = cruxapi.NewID("job")
	}
	if err := s.store.CreateJob(ctx, job); err != nil {
		return cruxapi.Job{}, err
	}
	return job, nil
}

func (s *Service) GetJob(ctx context.Context, id string) (cruxapi.Job, error) {
	return s.store.GetJob(ctx, id)
}

func (s *Service) DeleteJob(ctx context.Context, id string) error {
	return s.store.DeleteJob(ctx, id)
}

func (s *Service) ListJobRuns(ctx context.Context, jobID string) ([]cruxapi.JobRun, error) {
	return []cruxapi.JobRun{}, nil
}

// --- Plan 13: Metrics ---

func (s *Service) GetMetrics(ctx context.Context) ([]cruxapi.MetricValue, error) {
	return []cruxapi.MetricValue{
		{Name: "crux_agents_total", Value: 0, Timestamp: time.Now().UTC()},
		{Name: "crux_sessions_active", Value: 0, Timestamp: time.Now().UTC()},
	}, nil
}

// --- Plan 15: Enterprise ---

func (s *Service) ListMachines(ctx context.Context) ([]cruxapi.Machine, error) {
	return s.store.ListMachines(ctx)
}

func (s *Service) PairMachine(ctx context.Context, req cruxapi.EnrollmentRequest) (cruxapi.EnrollmentResponse, error) {
	m := cruxapi.Machine{
		ID:         cruxapi.NewID("mach"),
		Status:     "enrolled",
		EnrolledAt: ptrTime(time.Now().UTC()),
	}
	if err := s.store.CreateMachine(ctx, m); err != nil {
		return cruxapi.EnrollmentResponse{}, err
	}
	return cruxapi.EnrollmentResponse{MachineID: m.ID, Status: "enrolled"}, nil
}

func ptrTime(t time.Time) *time.Time { return &t }

func (s *Service) GetGatewayStatus(ctx context.Context) (cruxapi.GatewayStatus, error) {
	return cruxapi.GatewayStatus{Enabled: false, Ready: true}, nil
}

func (s *Service) GetGatewayRoutes(ctx context.Context) ([]cruxapi.GatewayRoute, error) {
	return []cruxapi.GatewayRoute{}, nil
}

func (s *Service) DiscoverMCPServers(ctx context.Context) ([]cruxapi.MCPServer, error) {
	return []cruxapi.MCPServer{}, nil
}

func (s *Service) ListMCPServers(ctx context.Context) ([]cruxapi.MCPServer, error) {
	return []cruxapi.MCPServer{}, nil
}

func (s *Service) GetMCPServer(ctx context.Context, id string) (cruxapi.MCPServer, error) {
	servers, err := s.ListMCPServers(ctx)
	if err != nil {
		return cruxapi.MCPServer{}, err
	}
	for _, srv := range servers {
		if srv.ID == id {
			return srv, nil
		}
	}
	return cruxapi.MCPServer{}, store.ErrNotFound
}

func (s *Service) ListMCPTools(ctx context.Context) ([]cruxapi.MCPTool, error) {
	return []cruxapi.MCPTool{}, nil
}

func (s *Service) CallMCPTool(ctx context.Context, req cruxapi.MCPCallRequest) (cruxapi.MCPCallResult, error) {
	return cruxapi.MCPCallResult{Success: false, Error: "MCP proxy not configured"}, nil
}

func (s *Service) GetAuditLog(ctx context.Context, limit int) ([]cruxapi.AuditLogEntry, error) {
	return []cruxapi.AuditLogEntry{}, nil
}

func (s *Service) ExportAuditLog(ctx context.Context, req cruxapi.AuditExportRequest) ([]byte, error) {
	entries, err := s.GetAuditLog(ctx, 10000)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	for _, e := range entries {
		b, _ := json.Marshal(e)
		buf.Write(b)
		buf.WriteByte('\n')
	}
	return buf.Bytes(), nil
}

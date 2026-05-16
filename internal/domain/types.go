package domain

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

const (
	Version = "0.1.0"
)

type AgentStatus string

const (
	AgentReady    AgentStatus = "ready"
	AgentDisabled AgentStatus = "disabled"
)

type ExecutionStatus string

const (
	ExecutionQueued    ExecutionStatus = "queued"
	ExecutionRunning   ExecutionStatus = "running"
	ExecutionSucceeded ExecutionStatus = "succeeded"
	ExecutionFailed    ExecutionStatus = "failed"
	ExecutionCanceled  ExecutionStatus = "canceled"
)

type EventType string

const (
	EventAgentRegistered EventType = "agent.registered"
	EventAgentDeleted    EventType = "agent.deleted"
	EventConfigUpdated   EventType = "config.updated"
	EventExecutionQueued EventType = "execution.queued"
	EventExecutionStart  EventType = "execution.started"
	EventExecutionOutput EventType = "execution.output"
	EventExecutionFinish EventType = "execution.finished"
	EventExecutionFail   EventType = "execution.failed"
	EventDiscoveryRun    EventType = "discovery.run"
)

type Agent struct {
	ID          string            `json:"id" yaml:"id"`
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Command     CommandSpec       `json:"command" yaml:"command"`
	Status      AgentStatus       `json:"status" yaml:"status"`
	CreatedAt   time.Time         `json:"createdAt" yaml:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt" yaml:"updatedAt"`
}

type CommandSpec struct {
	Path           string            `json:"path" yaml:"path"`
	Args           []string          `json:"args,omitempty" yaml:"args,omitempty"`
	Env            map[string]string `json:"env,omitempty" yaml:"env,omitempty"`
	WorkingDir     string            `json:"workingDir,omitempty" yaml:"workingDir,omitempty"`
	TimeoutSeconds int               `json:"timeoutSeconds,omitempty" yaml:"timeoutSeconds,omitempty"`
}

type Execution struct {
	ID            string          `json:"id" yaml:"id"`
	AgentName     string          `json:"agentName" yaml:"agentName"`
	Prompt        string          `json:"prompt" yaml:"prompt"`
	Status        ExecutionStatus `json:"status" yaml:"status"`
	ExitCode      int             `json:"exitCode" yaml:"exitCode"`
	Stdout        string          `json:"stdout,omitempty" yaml:"stdout,omitempty"`
	Stderr        string          `json:"stderr,omitempty" yaml:"stderr,omitempty"`
	Error         string          `json:"error,omitempty" yaml:"error,omitempty"`
	QueuedAt      time.Time       `json:"queuedAt" yaml:"queuedAt"`
	StartedAt     *time.Time      `json:"startedAt,omitempty" yaml:"startedAt,omitempty"`
	CompletedAt   *time.Time      `json:"completedAt,omitempty" yaml:"completedAt,omitempty"`
	UpdatedAt     time.Time       `json:"updatedAt" yaml:"updatedAt"`
	RuntimeConfig RuntimeConfig   `json:"runtimeConfig" yaml:"runtimeConfig"`
}

type Event struct {
	ID          string         `json:"id" yaml:"id"`
	Type        EventType      `json:"type" yaml:"type"`
	ExecutionID string         `json:"executionId,omitempty" yaml:"executionId,omitempty"`
	AgentName   string         `json:"agentName,omitempty" yaml:"agentName,omitempty"`
	Message     string         `json:"message" yaml:"message"`
	Data        map[string]any `json:"data,omitempty" yaml:"data,omitempty"`
	CreatedAt   time.Time      `json:"createdAt" yaml:"createdAt"`
}

type RuntimeConfig struct {
	WorkerConcurrency     int    `json:"workerConcurrency" yaml:"workerConcurrency"`
	JobTimeoutSeconds     int    `json:"jobTimeoutSeconds" yaml:"jobTimeoutSeconds"`
	MaxOutputBytes        int    `json:"maxOutputBytes" yaml:"maxOutputBytes"`
	DiscoveryTimeoutSecs  int    `json:"discoveryTimeoutSeconds" yaml:"discoveryTimeoutSeconds"`
	LogLevel              string `json:"logLevel" yaml:"logLevel"`
	DefaultNamespace      string `json:"defaultNamespace" yaml:"defaultNamespace"`
	AllowShellCommands    bool   `json:"allowShellCommands" yaml:"allowShellCommands"`
	TraceRetentionEntries int    `json:"traceRetentionEntries" yaml:"traceRetentionEntries"`
}

type RuntimeConfigPatch struct {
	WorkerConcurrency     *int    `json:"workerConcurrency"`
	JobTimeoutSeconds     *int    `json:"jobTimeoutSeconds"`
	MaxOutputBytes        *int    `json:"maxOutputBytes"`
	DiscoveryTimeoutSecs  *int    `json:"discoveryTimeoutSeconds"`
	LogLevel              *string `json:"logLevel"`
	DefaultNamespace      *string `json:"defaultNamespace"`
	AllowShellCommands    *bool   `json:"allowShellCommands"`
	TraceRetentionEntries *int    `json:"traceRetentionEntries"`
}

func DefaultRuntimeConfig() RuntimeConfig {
	return RuntimeConfig{
		WorkerConcurrency:     2,
		JobTimeoutSeconds:     900,
		MaxOutputBytes:        1048576,
		DiscoveryTimeoutSecs:  3,
		LogLevel:              "info",
		DefaultNamespace:      "default",
		AllowShellCommands:    false,
		TraceRetentionEntries: 10000,
	}
}

func (c RuntimeConfig) Validate() error {
	if c.WorkerConcurrency < 1 {
		return fmt.Errorf("workerConcurrency must be >= 1")
	}
	if c.JobTimeoutSeconds < 1 {
		return fmt.Errorf("jobTimeoutSeconds must be >= 1")
	}
	if c.MaxOutputBytes < 1024 {
		return fmt.Errorf("maxOutputBytes must be >= 1024")
	}
	if c.DiscoveryTimeoutSecs < 1 {
		return fmt.Errorf("discoveryTimeoutSeconds must be >= 1")
	}
	if strings.TrimSpace(c.LogLevel) == "" {
		return fmt.Errorf("logLevel is required")
	}
	if strings.TrimSpace(c.DefaultNamespace) == "" {
		return fmt.Errorf("defaultNamespace is required")
	}
	if c.TraceRetentionEntries < 100 {
		return fmt.Errorf("traceRetentionEntries must be >= 100")
	}
	return nil
}

func (c RuntimeConfig) ApplyPatch(p RuntimeConfigPatch) RuntimeConfig {
	next := c
	if p.WorkerConcurrency != nil {
		next.WorkerConcurrency = *p.WorkerConcurrency
	}
	if p.JobTimeoutSeconds != nil {
		next.JobTimeoutSeconds = *p.JobTimeoutSeconds
	}
	if p.MaxOutputBytes != nil {
		next.MaxOutputBytes = *p.MaxOutputBytes
	}
	if p.DiscoveryTimeoutSecs != nil {
		next.DiscoveryTimeoutSecs = *p.DiscoveryTimeoutSecs
	}
	if p.LogLevel != nil {
		next.LogLevel = *p.LogLevel
	}
	if p.DefaultNamespace != nil {
		next.DefaultNamespace = *p.DefaultNamespace
	}
	if p.AllowShellCommands != nil {
		next.AllowShellCommands = *p.AllowShellCommands
	}
	if p.TraceRetentionEntries != nil {
		next.TraceRetentionEntries = *p.TraceRetentionEntries
	}
	return next
}

func NewID(prefix string) string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(fmt.Sprintf("generate id: %v", err))
	}
	return prefix + "_" + hex.EncodeToString(b[:])
}

func CleanAgentName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func Now() time.Time {
	return time.Now().UTC()
}

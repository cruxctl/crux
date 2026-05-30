package cruxapi

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
	ID             string          `json:"id" yaml:"id"`
	AgentName      string          `json:"agentName" yaml:"agentName"`
	Prompt         string          `json:"prompt" yaml:"prompt"`
	WorkingDir     string          `json:"workingDir,omitempty" yaml:"workingDir,omitempty"`
	ResumeSession  string          `json:"resumeSession,omitempty" yaml:"resumeSession,omitempty"`
	SourceExecID   string          `json:"sourceExecutionId,omitempty" yaml:"sourceExecutionId,omitempty"`
	FallbackAgents []string        `json:"fallbackAgents,omitempty" yaml:"fallbackAgents,omitempty"`
	Status         ExecutionStatus `json:"status" yaml:"status"`
	ExitCode       int             `json:"exitCode" yaml:"exitCode"`
	Stdout         string          `json:"stdout,omitempty" yaml:"stdout,omitempty"`
	Stderr         string          `json:"stderr,omitempty" yaml:"stderr,omitempty"`
	Error          string          `json:"error,omitempty" yaml:"error,omitempty"`
	QueuedAt       time.Time       `json:"queuedAt" yaml:"queuedAt"`
	StartedAt      *time.Time      `json:"startedAt,omitempty" yaml:"startedAt,omitempty"`
	CompletedAt    *time.Time      `json:"completedAt,omitempty" yaml:"completedAt,omitempty"`
	UpdatedAt      time.Time       `json:"updatedAt" yaml:"updatedAt"`
	RuntimeConfig  RuntimeConfig   `json:"runtimeConfig" yaml:"runtimeConfig"`
}

type SubmitExecutionRequest struct {
	AgentName      string   `json:"agentName" yaml:"agentName"`
	Prompt         string   `json:"prompt" yaml:"prompt"`
	WorkingDir     string   `json:"workingDir,omitempty" yaml:"workingDir,omitempty"`
	ResumeSession  string   `json:"resumeSession,omitempty" yaml:"resumeSession,omitempty"`
	SourceExecID   string   `json:"sourceExecutionId,omitempty" yaml:"sourceExecutionId,omitempty"`
	FallbackAgents []string `json:"fallbackAgents,omitempty" yaml:"fallbackAgents,omitempty"`
	Wait           bool     `json:"wait" yaml:"wait"`
}

type DiscoveryResult struct {
	Agent   Agent  `json:"agent" yaml:"agent"`
	Version string `json:"version" yaml:"version"`
}

type AgentUsage struct {
	AgentID                string            `json:"agentId,omitempty" yaml:"agentId,omitempty"`
	AgentName              string            `json:"agentName" yaml:"agentName"`
	Description            string            `json:"description,omitempty" yaml:"description,omitempty"`
	Status                 AgentStatus       `json:"status,omitempty" yaml:"status,omitempty"`
	CommandPath            string            `json:"commandPath,omitempty" yaml:"commandPath,omitempty"`
	CommandArgs            []string          `json:"commandArgs,omitempty" yaml:"commandArgs,omitempty"`
	Version                string            `json:"version,omitempty" yaml:"version,omitempty"`
	Labels                 map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	ExecutionsTotal        int               `json:"executionsTotal" yaml:"executionsTotal"`
	Queued                 int               `json:"queued" yaml:"queued"`
	Running                int               `json:"running" yaml:"running"`
	Succeeded              int               `json:"succeeded" yaml:"succeeded"`
	Failed                 int               `json:"failed" yaml:"failed"`
	Canceled               int               `json:"canceled" yaml:"canceled"`
	SuccessRate            float64           `json:"successRate" yaml:"successRate"`
	ExitCodes              map[string]int    `json:"exitCodes,omitempty" yaml:"exitCodes,omitempty"`
	StdoutBytes            int               `json:"stdoutBytes" yaml:"stdoutBytes"`
	StderrBytes            int               `json:"stderrBytes" yaml:"stderrBytes"`
	ErrorCount             int               `json:"errorCount" yaml:"errorCount"`
	TotalDurationSeconds   float64           `json:"totalDurationSeconds" yaml:"totalDurationSeconds"`
	AverageDurationSeconds float64           `json:"averageDurationSeconds" yaml:"averageDurationSeconds"`
	MaxDurationSeconds     float64           `json:"maxDurationSeconds" yaml:"maxDurationSeconds"`
	FirstQueuedAt          *time.Time        `json:"firstQueuedAt,omitempty" yaml:"firstQueuedAt,omitempty"`
	LastQueuedAt           *time.Time        `json:"lastQueuedAt,omitempty" yaml:"lastQueuedAt,omitempty"`
	LastExecutionID        string            `json:"lastExecutionId,omitempty" yaml:"lastExecutionId,omitempty"`
	LastStatus             ExecutionStatus   `json:"lastStatus,omitempty" yaml:"lastStatus,omitempty"`
	LastExitCode           int               `json:"lastExitCode" yaml:"lastExitCode"`
	LastError              string            `json:"lastError,omitempty" yaml:"lastError,omitempty"`
	LastStdout             string            `json:"lastStdout,omitempty" yaml:"lastStdout,omitempty"`
	LastStderr             string            `json:"lastStderr,omitempty" yaml:"lastStderr,omitempty"`
	EventCounts            map[EventType]int `json:"eventCounts,omitempty" yaml:"eventCounts,omitempty"`
	ExternalMetrics        []UsageMetric     `json:"externalMetrics,omitempty" yaml:"externalMetrics,omitempty"`
	Notes                  []string          `json:"notes,omitempty" yaml:"notes,omitempty"`
}

type UsageMetric struct {
	Name        string `json:"name" yaml:"name"`
	Available   bool   `json:"available" yaml:"available"`
	Value       string `json:"value,omitempty" yaml:"value,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

type AgentCapabilities struct {
	AgentName             string   `json:"agentName" yaml:"agentName"`
	Provider              string   `json:"provider" yaml:"provider"`
	UsageProbe            bool     `json:"usageProbe" yaml:"usageProbe"`
	CostSignals           bool     `json:"costSignals" yaml:"costSignals"`
	ProviderSessions      bool     `json:"providerSessions" yaml:"providerSessions"`
	Resume                bool     `json:"resume" yaml:"resume"`
	TTYExec               bool     `json:"ttyExec" yaml:"ttyExec"`
	CruxHistory           bool     `json:"cruxHistory" yaml:"cruxHistory"`
	Fallback              bool     `json:"fallback" yaml:"fallback"`
	SessionListCommand    []string `json:"sessionListCommand,omitempty" yaml:"sessionListCommand,omitempty"`
	ResumeCommandTemplate []string `json:"resumeCommandTemplate,omitempty" yaml:"resumeCommandTemplate,omitempty"`
	TTYExecCommand        []string `json:"ttyExecCommand,omitempty" yaml:"ttyExecCommand,omitempty"`
	Notes                 []string `json:"notes,omitempty" yaml:"notes,omitempty"`
}

type AgentSession struct {
	AgentName       string     `json:"agentName" yaml:"agentName"`
	Provider        string     `json:"provider" yaml:"provider"`
	ID              string     `json:"id" yaml:"id"`
	Title           string     `json:"title,omitempty" yaml:"title,omitempty"`
	Age             string     `json:"age,omitempty" yaml:"age,omitempty"`
	WorkingDir      string     `json:"workingDir,omitempty" yaml:"workingDir,omitempty"`
	Source          string     `json:"source" yaml:"source"`
	ResumeSupported bool       `json:"resumeSupported" yaml:"resumeSupported"`
	UpdatedAt       *time.Time `json:"updatedAt,omitempty" yaml:"updatedAt,omitempty"`
	Raw             string     `json:"raw,omitempty" yaml:"raw,omitempty"`
}

type AgentCostSnapshot struct {
	AgentName               string          `json:"agentName" yaml:"agentName"`
	Provider                string          `json:"provider" yaml:"provider"`
	Status                  AgentStatus     `json:"status" yaml:"status"`
	ProviderCostAvailable   bool            `json:"providerCostAvailable" yaml:"providerCostAvailable"`
	ProviderCostValue       string          `json:"providerCostValue,omitempty" yaml:"providerCostValue,omitempty"`
	ProviderCostDescription string          `json:"providerCostDescription,omitempty" yaml:"providerCostDescription,omitempty"`
	ExecutionsTotal         int             `json:"executionsTotal" yaml:"executionsTotal"`
	Queued                  int             `json:"queued" yaml:"queued"`
	Running                 int             `json:"running" yaml:"running"`
	Succeeded               int             `json:"succeeded" yaml:"succeeded"`
	Failed                  int             `json:"failed" yaml:"failed"`
	SuccessRate             float64         `json:"successRate" yaml:"successRate"`
	StdoutBytes             int             `json:"stdoutBytes" yaml:"stdoutBytes"`
	StderrBytes             int             `json:"stderrBytes" yaml:"stderrBytes"`
	TotalDurationSeconds    float64         `json:"totalDurationSeconds" yaml:"totalDurationSeconds"`
	AverageDurationSeconds  float64         `json:"averageDurationSeconds" yaml:"averageDurationSeconds"`
	RealtimeRunningSeconds  float64         `json:"realtimeRunningSeconds" yaml:"realtimeRunningSeconds"`
	LastExecutionID         string          `json:"lastExecutionId,omitempty" yaml:"lastExecutionId,omitempty"`
	LastStatus              ExecutionStatus `json:"lastStatus,omitempty" yaml:"lastStatus,omitempty"`
	LastError               string          `json:"lastError,omitempty" yaml:"lastError,omitempty"`
	ExternalMetrics         []UsageMetric   `json:"externalMetrics,omitempty" yaml:"externalMetrics,omitempty"`
	Notes                   []string        `json:"notes,omitempty" yaml:"notes,omitempty"`
}

type AgentExecPlanRequest struct {
	WorkingDir    string   `json:"workingDir,omitempty" yaml:"workingDir,omitempty"`
	ResumeSession string   `json:"resumeSession,omitempty" yaml:"resumeSession,omitempty"`
	Prompt        string   `json:"prompt,omitempty" yaml:"prompt,omitempty"`
	Input         []string `json:"input,omitempty" yaml:"input,omitempty"`
	Args          []string `json:"args,omitempty" yaml:"args,omitempty"`
	Operation     string   `json:"operation,omitempty" yaml:"operation,omitempty"`
}

type AgentExecPlan struct {
	AgentName     string      `json:"agentName" yaml:"agentName"`
	Provider      string      `json:"provider" yaml:"provider"`
	Command       CommandSpec `json:"command" yaml:"command"`
	ResumeSession string      `json:"resumeSession,omitempty" yaml:"resumeSession,omitempty"`
	Prompt        string      `json:"prompt,omitempty" yaml:"prompt,omitempty"`
	Input         []string    `json:"input,omitempty" yaml:"input,omitempty"`
	Operation     string      `json:"operation,omitempty" yaml:"operation,omitempty"`
	Notes         []string    `json:"notes,omitempty" yaml:"notes,omitempty"`
}

type AgentExecRecordRequest struct {
	AgentName      string    `json:"agentName,omitempty" yaml:"agentName,omitempty"`
	WorkingDir     string    `json:"workingDir,omitempty" yaml:"workingDir,omitempty"`
	ResumeSession  string    `json:"resumeSession,omitempty" yaml:"resumeSession,omitempty"`
	Prompt         string    `json:"prompt,omitempty" yaml:"prompt,omitempty"`
	Operation      string    `json:"operation,omitempty" yaml:"operation,omitempty"`
	SourceExecID   string    `json:"sourceExecutionId,omitempty" yaml:"sourceExecutionId,omitempty"`
	FallbackAgents []string  `json:"fallbackAgents,omitempty" yaml:"fallbackAgents,omitempty"`
	Args           []string  `json:"args,omitempty" yaml:"args,omitempty"`
	Driver         string    `json:"driver,omitempty" yaml:"driver,omitempty"`
	TranscriptPath string    `json:"transcriptPath,omitempty" yaml:"transcriptPath,omitempty"`
	Transcript     string    `json:"transcript,omitempty" yaml:"transcript,omitempty"`
	Stderr         string    `json:"stderr,omitempty" yaml:"stderr,omitempty"`
	Error          string    `json:"error,omitempty" yaml:"error,omitempty"`
	ExitCode       int       `json:"exitCode" yaml:"exitCode"`
	StartedAt      time.Time `json:"startedAt,omitempty" yaml:"startedAt,omitempty"`
	CompletedAt    time.Time `json:"completedAt,omitempty" yaml:"completedAt,omitempty"`
}

type AgentExecRecordResponse struct {
	Execution Execution         `json:"execution" yaml:"execution"`
	Usage     AgentUsage        `json:"usage" yaml:"usage"`
	Cost      AgentCostSnapshot `json:"cost" yaml:"cost"`
	Sessions  []AgentSession    `json:"sessions" yaml:"sessions"`
}

type AgentHistoryItem struct {
	ID             string          `json:"id" yaml:"id"`
	AgentName      string          `json:"agentName" yaml:"agentName"`
	Status         ExecutionStatus `json:"status" yaml:"status"`
	ExitCode       int             `json:"exitCode" yaml:"exitCode"`
	Prompt         string          `json:"prompt,omitempty" yaml:"prompt,omitempty"`
	PromptPreview  string          `json:"promptPreview,omitempty" yaml:"promptPreview,omitempty"`
	StdoutPreview  string          `json:"stdoutPreview,omitempty" yaml:"stdoutPreview,omitempty"`
	StderrPreview  string          `json:"stderrPreview,omitempty" yaml:"stderrPreview,omitempty"`
	Error          string          `json:"error,omitempty" yaml:"error,omitempty"`
	WorkingDir     string          `json:"workingDir,omitempty" yaml:"workingDir,omitempty"`
	ResumeSession  string          `json:"resumeSession,omitempty" yaml:"resumeSession,omitempty"`
	SourceExecID   string          `json:"sourceExecutionId,omitempty" yaml:"sourceExecutionId,omitempty"`
	FallbackAgents []string        `json:"fallbackAgents,omitempty" yaml:"fallbackAgents,omitempty"`
	QueuedAt       time.Time       `json:"queuedAt" yaml:"queuedAt"`
	StartedAt      *time.Time      `json:"startedAt,omitempty" yaml:"startedAt,omitempty"`
	CompletedAt    *time.Time      `json:"completedAt,omitempty" yaml:"completedAt,omitempty"`
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

// --- Session / Execution unification (blueprint §15) ---

type SessionStatus string

const (
	SessionStatusCreated   SessionStatus = "created"
	SessionStatusRunning   SessionStatus = "running"
	SessionStatusCompleted SessionStatus = "completed"
	SessionStatusFailed    SessionStatus = "failed"
	SessionStatusCanceled  SessionStatus = "canceled"
	SessionStatusContinued SessionStatus = "continued"
)

type Session struct {
	ID                    string          `json:"id"`
	AgentID               string          `json:"agent_id"`
	ProjectID             string          `json:"project_id,omitempty"`
	UserID                string          `json:"user_id,omitempty"`
	MachineID             string          `json:"machine_id,omitempty"`
	ParentSessionID       string          `json:"parent_session_id,omitempty"`
	FallbackFromSessionID string          `json:"fallback_from_session_id,omitempty"`
	Status                SessionStatus   `json:"status"`
	StartedAt             time.Time       `json:"started_at"`
	EndedAt               *time.Time      `json:"ended_at,omitempty"`
	TranscriptPaths       TranscriptPaths `json:"transcript_paths,omitempty"`
	Cost                  Cost            `json:"cost"`
}

type TranscriptPaths struct {
	Raw  string `json:"raw,omitempty"`
	ANSI string `json:"ansi,omitempty"`
	Text string `json:"txt,omitempty"`
}

type Cost struct {
	TokensIn  int     `json:"tokens_in"`
	TokensOut int     `json:"tokens_out"`
	USD       float64 `json:"usd"`
}

// --- AOS Event (blueprint §17) ---

type AOSEvent struct {
	Schema    string         `json:"schema"`
	EventID   string         `json:"event_id"`
	Timestamp time.Time      `json:"timestamp"`
	EventType string         `json:"event_type"`
	Actor     Actor          `json:"actor"`
	Project   *ProjectRef    `json:"project,omitempty"`
	Agent     *AgentRef      `json:"agent,omitempty"`
	Tool      *ToolRef       `json:"tool,omitempty"`
	Policy    *PolicyRef     `json:"policy,omitempty"`
	Trace     *TraceRef      `json:"trace,omitempty"`
	Payload   map[string]any `json:"payload,omitempty"`
}

type Actor struct {
	UserID    string `json:"user_id,omitempty"`
	MachineID string `json:"machine_id,omitempty"`
	AgentID   string `json:"agent_id,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

type ProjectRef struct {
	ID       string `json:"id"`
	PathHash string `json:"path_hash,omitempty"`
}

type AgentRef struct {
	Name     string `json:"name"`
	Provider string `json:"provider,omitempty"`
	Version  string `json:"version,omitempty"`
}

type ToolRef struct {
	Name      string `json:"name"`
	Server    string `json:"server,omitempty"`
	Transport string `json:"transport,omitempty"`
}

type PolicyRef struct {
	Decision string `json:"decision"`
	RuleID   string `json:"rule_id,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

type TraceRef struct {
	TraceID string `json:"trace_id"`
	SpanID  string `json:"span_id"`
}

// --- AgBOM (blueprint §18) ---

type AgBOM struct {
	Schema      string        `json:"schema"`
	Agent       AgentRef      `json:"agent"`
	Runtime     AgBOMRuntime  `json:"runtime"`
	Models      []AgBOMModel  `json:"models,omitempty"`
	Tools       []AgBOMTool   `json:"tools,omitempty"`
	MCPServers  []AgBOMMCP    `json:"mcp_servers,omitempty"`
	Memory      []AgBOMMemory `json:"memory,omitempty"`
	Skills      []AgBOMSkill  `json:"skills,omitempty"`
	Permissions []AgBOMPerm   `json:"permissions,omitempty"`
	Sessions    AgBOMSessions `json:"sessions"`
}

type AgBOMRuntime struct {
	Kind        string `json:"kind"`
	Binary      string `json:"binary,omitempty"`
	RequiresPTY bool   `json:"requires_pty,omitempty"`
}

type AgBOMModel struct {
	Provider string `json:"provider"`
	Name     string `json:"name"`
	Source   string `json:"source,omitempty"`
}

type AgBOMTool struct {
	Name      string `json:"name"`
	Transport string `json:"transport,omitempty"`
	Risk      string `json:"risk,omitempty"`
}

type AgBOMMCP struct {
	Name      string `json:"name"`
	Transport string `json:"transport"`
	Command   string `json:"command,omitempty"`
	ArgsHash  string `json:"args_hash,omitempty"`
	Trust     string `json:"trust,omitempty"`
}

type AgBOMMemory struct {
	Type string `json:"type"`
	Path string `json:"path"`
}

type AgBOMSkill struct {
	Name   string `json:"name"`
	Source string `json:"source,omitempty"`
}

type AgBOMPerm struct {
	Name string `json:"name"`
	Mode string `json:"mode,omitempty"`
}

type AgBOMSessions struct {
	Count    int       `json:"count"`
	LastSeen time.Time `json:"last_seen,omitempty"`
}

// --- Policy (blueprint §16) ---

type PolicyProfile struct {
	APIVersion string         `yaml:"apiVersion" json:"apiVersion"`
	Kind       string         `yaml:"kind"       json:"kind"`
	Metadata   PolicyMetadata `yaml:"metadata"   json:"metadata"`
	Rules      []PolicyRule   `yaml:"rules"      json:"rules"`
}

type PolicyMetadata struct {
	ID string `yaml:"id" json:"id"`
}

type PolicyRule struct {
	ID     string         `yaml:"id"     json:"id"`
	Match  map[string]any `yaml:"match"  json:"match"`
	Action string         `yaml:"action" json:"action"`
	Rate   *PolicyRate    `yaml:"rate,omitempty" json:"rate,omitempty"`
}

type PolicyRate struct {
	PerMinute int `yaml:"per_minute" json:"per_minute"`
}

type PolicyDecision struct {
	Action string `json:"action"`
	RuleID string `json:"rule_id,omitempty"`
	Reason string `json:"reason,omitempty"`
}

type PolicyInput struct {
	Actor   Actor          `json:"actor"`
	Project ProjectRef     `json:"project"`
	Tool    ToolRef        `json:"tool"`
	Payload map[string]any `json:"payload,omitempty"`
}

// --- PTY (blueprint §5) ---

type PTYPurpose string

const (
	PTYPurposeInteractiveSession  PTYPurpose = "interactive_session"
	PTYPurposeProbe               PTYPurpose = "probe"
	PTYPurposeScheduledCollector  PTYPurpose = "scheduled_collector"
	PTYPurposeMCPDiscovery        PTYPurpose = "mcp_discovery"
	PTYPurposeSkillDiscovery      PTYPurpose = "skill_discovery"
	PTYPurposeHistoryDiscovery    PTYPurpose = "history_discovery"
	PTYPurposeUsageDiscovery      PTYPurpose = "usage_discovery"
	PTYPurposePermissionDiscovery PTYPurpose = "permission_discovery"
	PTYPurposeMemoryDiscovery     PTYPurpose = "memory_discovery"
	PTYPurposeFallback            PTYPurpose = "fallback"
)

type CaptureMode int

const (
	CaptureRawAndANSI CaptureMode = iota
	CaptureRawOnly
	CaptureANSIOnly
	CaptureDisabled
)

// PTYSpec describes one PTY allocation. Used for both interactive sessions
// and non-interactive scheduled probes (Script controls the latter).
type PTYSpec struct {
	ID      string            `json:"id"`
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	WorkDir string            `json:"work_dir,omitempty"`
	Rows    int               `json:"rows,omitempty"`
	Cols    int               `json:"cols,omitempty"`
	Timeout time.Duration     `json:"timeout_ns,omitempty"`
	Purpose PTYPurpose        `json:"purpose"`
	Capture CaptureMode       `json:"capture"`
	Script  []PTYStep         `json:"script,omitempty"`
}

// PTYStep is one item in the Script DSL. Exactly one of Send / Expect /
// ExpectAny / Sleep / Snapshot fields is non-zero per step.
type PTYStep struct {
	Send      string        `json:"send,omitempty"`
	Expect    string        `json:"expect,omitempty"`
	ExpectAny []string      `json:"expect_any,omitempty"`
	Sleep     time.Duration `json:"sleep_ns,omitempty"`
	Snapshot  bool          `json:"snapshot,omitempty"`
}

// PTYResult is what a Factory.Create + Wait returns.
type PTYResult struct {
	ExitCode   int       `json:"exit_code"`
	Started    time.Time `json:"started"`
	Ended      time.Time `json:"ended"`
	RawPath    string    `json:"raw_path,omitempty"`
	ANSIPath   string    `json:"ansi_path,omitempty"`
	TextPath   string    `json:"text_path,omitempty"`
	SnapsPath  string    `json:"snaps_path,omitempty"`
	EventsPath string    `json:"events_path,omitempty"`
}

// --- CodingAgentSpec YAML (blueprint §6.1) ---

type CodingAgentSpec struct {
	APIVersion      string                 `yaml:"apiVersion" json:"apiVersion"`
	Kind            string                 `yaml:"kind"       json:"kind"`
	Metadata        AgentSpecMetadata      `yaml:"metadata"   json:"metadata"`
	Detection       AgentSpecDetection     `yaml:"detection"  json:"detection"`
	Launch          AgentSpecLaunch        `yaml:"launch"     json:"launch"`
	PTY             AgentSpecPTY           `yaml:"pty"        json:"pty"`
	SpecialPrefixes map[string]string      `yaml:"special_prefixes,omitempty" json:"special_prefixes,omitempty"`
	TUICommands     map[string]AgentTUICmd `yaml:"tui_commands,omitempty"     json:"tui_commands,omitempty"`
	CLICommands     map[string]AgentCLICmd `yaml:"cli_commands,omitempty"     json:"cli_commands,omitempty"`
	FileCollectors  []AgentFileCollector   `yaml:"file_collectors,omitempty"  json:"file_collectors,omitempty"`
	Probes          []AgentProbe           `yaml:"probes,omitempty"           json:"probes,omitempty"`
	MCPInject       AgentMCPInject         `yaml:"mcp_inject,omitempty"       json:"mcp_inject,omitempty"`
	Normalization   AgentNormalization     `yaml:"normalization,omitempty"    json:"normalization,omitempty"`
}

type AgentSpecMetadata struct {
	ID       string   `yaml:"id"       json:"id"`
	Name     string   `yaml:"name"     json:"name"`
	Provider string   `yaml:"provider" json:"provider"`
	Kind     string   `yaml:"kind,omitempty" json:"kind,omitempty"` // e.g. vscode_extension
	Homepage string   `yaml:"homepage,omitempty" json:"homepage,omitempty"`
	Docs     []string `yaml:"docs,omitempty"     json:"docs,omitempty"`
}

type AgentSpecDetection struct {
	Binaries         []string              `yaml:"binaries,omitempty"          json:"binaries,omitempty"`
	Version          AgentSpecVersionProbe `yaml:"version,omitempty"           json:"version,omitempty"`
	ConfigPaths      []string              `yaml:"config_paths,omitempty"      json:"config_paths,omitempty"`
	VSCodeExtensions []string              `yaml:"vscode_extensions,omitempty" json:"vscode_extensions,omitempty"`
}

type AgentSpecVersionProbe struct {
	Command string         `yaml:"command,omitempty" json:"command,omitempty"`
	Args    []string       `yaml:"args,omitempty"    json:"args,omitempty"`
	Parse   map[string]any `yaml:"parse,omitempty"   json:"parse,omitempty"`
}

type AgentSpecLaunch struct {
	Interactive AgentLaunchMode `yaml:"interactive,omitempty" json:"interactive,omitempty"`
	OneShot     AgentLaunchMode `yaml:"one_shot,omitempty"    json:"one_shot,omitempty"`
	Print       AgentLaunchMode `yaml:"print,omitempty"       json:"print,omitempty"`
}

type AgentLaunchMode struct {
	Command     string   `yaml:"command,omitempty" json:"command,omitempty"`
	Args        []string `yaml:"args,omitempty"    json:"args,omitempty"`
	RequiresPTY bool     `yaml:"requires_pty,omitempty" json:"requires_pty,omitempty"`
	Supported   *bool    `yaml:"supported,omitempty" json:"supported,omitempty"`
	Reason      string   `yaml:"reason,omitempty"    json:"reason,omitempty"`
}

type AgentSpecPTY struct {
	StartupTimeoutSeconds int              `yaml:"startup_timeout_seconds,omitempty" json:"startup_timeout_seconds,omitempty"`
	CommandTimeoutSeconds int              `yaml:"command_timeout_seconds,omitempty" json:"command_timeout_seconds,omitempty"`
	PromptReady           AgentPromptReady `yaml:"prompt_ready,omitempty" json:"prompt_ready,omitempty"`
}

type AgentPromptReady struct {
	Strategy string   `yaml:"strategy,omitempty" json:"strategy,omitempty"`
	Patterns []string `yaml:"patterns,omitempty" json:"patterns,omitempty"`
}

type AgentTUICmd struct {
	Input   string `yaml:"input"   json:"input"`
	Purpose string `yaml:"purpose,omitempty" json:"purpose,omitempty"`
}

type AgentCLICmd struct {
	Command string   `yaml:"command" json:"command"`
	Args    []string `yaml:"args,omitempty" json:"args,omitempty"`
	Purpose string   `yaml:"purpose,omitempty" json:"purpose,omitempty"`
}

type AgentFileCollector struct {
	ID    string         `yaml:"id"    json:"id"`
	Paths []string       `yaml:"paths" json:"paths"`
	Parse map[string]any `yaml:"parse" json:"parse"`
}

type AgentProbe struct {
	ID            string         `yaml:"id" json:"id"`
	CommandRef    string         `yaml:"command_ref,omitempty"     json:"command_ref,omitempty"`
	CLICommandRef string         `yaml:"cli_command_ref,omitempty" json:"cli_command_ref,omitempty"`
	Parse         map[string]any `yaml:"parse,omitempty"           json:"parse,omitempty"`
}

type AgentMCPInject struct {
	Strategy string         `yaml:"strategy" json:"strategy"` // write_config_file | env_var | cli_command | tui_command | unsupported
	Reason   string         `yaml:"reason,omitempty" json:"reason,omitempty"`
	Details  map[string]any `yaml:"details,omitempty" json:"details,omitempty"`
}

type AgentNormalization struct {
	Events map[string]AgentEventPattern `yaml:"events,omitempty" json:"events,omitempty"`
}

type AgentEventPattern struct {
	Patterns []string `yaml:"patterns" json:"patterns"`
}

// --- Plan 04: Discovery + Inject Gateway (supplemental types) ---
// --- Plan 08: Policy + Approvals (supplemental types) ---

type PolicyAction string

const (
	PolicyAllow            PolicyAction = "allow"
	PolicyDeny             PolicyAction = "deny"
	PolicyRequireApproval  PolicyAction = "require_approval"
	PolicyRedact           PolicyAction = "redact"
	PolicyRateLimit        PolicyAction = "rate_limit"
	PolicyQuarantine       PolicyAction = "quarantine"
	PolicyWarn             PolicyAction = "warn"
	PolicyMetadataOnly     PolicyAction = "metadata_only"
)

type DiscoveryRequest struct {
	Agent   string `json:"agent,omitempty"`
	All     bool   `json:"all,omitempty"`
	Project string `json:"project,omitempty"`
}

type ProbeResult struct {
	ID       string         `json:"id"`
	Success  bool           `json:"success"`
	Output   string         `json:"output,omitempty"`
	Parsed   map[string]any `json:"parsed,omitempty"`
	Duration time.Duration  `json:"duration_ns,omitempty"`
}

type GatewayInjectRequest struct {
	Agents []string `json:"agents"`
}

type GatewayInjectResult struct {
	AgentID string `json:"agent_id"`
	Status  string `json:"status"`
	Backup  string `json:"backup_path,omitempty"`
}

// --- Plan 05: Session Manager + Context IR (supplemental types) ---

type ContextIR struct {
	Schema            string   `json:"schema"`
	SessionID         string   `json:"session_id"`
	SourceAgent       string   `json:"source_agent"`
	Goal              string   `json:"goal,omitempty"`
	CurrentState      string   `json:"current_state,omitempty"`
	ChangedFiles      []string `json:"changed_files,omitempty"`
	CommandsRun       []string `json:"commands_run,omitempty"`
	Decisions         []string `json:"decisions,omitempty"`
	OpenTasks         []string `json:"open_tasks,omitempty"`
	Warnings          []string `json:"warnings,omitempty"`
	TranscriptSummary string   `json:"transcript_summary,omitempty"`
}

type ContinueSessionRequest struct {
	With   string `json:"with"`
	Prompt string `json:"prompt,omitempty"`
}

// --- Plan 06: Usage + Cost (supplemental types) ---

type UsageReport struct {
	AgentID   string  `json:"agent_id"`
	ProjectID string  `json:"project_id,omitempty"`
	Period    string  `json:"period"`
	TokensIn  int     `json:"tokens_in"`
	TokensOut int     `json:"tokens_out"`
	USD       float64 `json:"usd"`
	Sessions  int     `json:"sessions"`
}

type CostReport struct {
	AgentID   string  `json:"agent_id"`
	SessionID string  `json:"session_id,omitempty"`
	TokensIn  int     `json:"tokens_in"`
	TokensOut int     `json:"tokens_out"`
	USD       float64 `json:"usd"`
	Model     string  `json:"model,omitempty"`
}

type UsageLimits struct {
	DailyUSDWarn  float64 `json:"daily_usd_warn"`
	DailyUSDBlock float64 `json:"daily_usd_block"`
	AgentID       string  `json:"agent_id,omitempty"`
	ProjectID     string  `json:"project_id,omitempty"`
}

// --- Plan 08: Policy + Approvals (supplemental types) ---

type EvaluationInput struct {
	Actor   EvaluationActor `json:"actor"`
	Project string          `json:"project,omitempty"`
	Tool    string          `json:"tool"`
	Payload map[string]any  `json:"payload,omitempty"`
}

type EvaluationActor struct {
	UserID    string `json:"user_id,omitempty"`
	MachineID string `json:"machine_id,omitempty"`
	AgentID   string `json:"agent_id,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

type Decision struct {
	Action PolicyAction `json:"action"`
	RuleID string       `json:"rule_id,omitempty"`
	Reason string       `json:"reason,omitempty"`
}

type ApprovalRecord struct {
	ID        string    `json:"id"`
	ToolCall  string    `json:"tool_call"`
	PolicyID  string    `json:"policy_id"`
	AgentID   string    `json:"agent_id"`
	SessionID string    `json:"session_id,omitempty"`
	Status    string    `json:"status"`
	Approvers []string  `json:"approvers,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// --- Plan 09: AOS Event Stream (supplemental types) ---

type AOSActor struct {
	Type      string `json:"type"`
	ID        string `json:"id"`
	SessionID string `json:"session_id,omitempty"`
}

type EventFilter struct {
	Since     *time.Time `json:"since,omitempty"`
	EventType string     `json:"event_type,omitempty"`
	Agent     string     `json:"agent,omitempty"`
	SessionID string     `json:"session_id,omitempty"`
}

type AOSExportRequest struct {
	Format string `json:"format"` // jsonl | otel | ocsf
}

// --- Plan 10: AgBOM (supplemental types) ---

type AgBOMAgent struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Version  string `json:"version"`
}

type AgBOMDiff struct {
	Added   []AgBOMChange `json:"added,omitempty"`
	Removed []AgBOMChange `json:"removed,omitempty"`
	Changed []AgBOMChange `json:"changed,omitempty"`
}

type AgBOMChange struct {
	Category string `json:"category"` // tools | mcp_servers | models | permissions
	Item     string `json:"item"`
	From     string `json:"from,omitempty"`
	To       string `json:"to,omitempty"`
}

type AgBOMExportRequest struct {
	Format string `json:"format"` // crux-json | cyclonedx | spdx | swid
}

// --- Plan 11: Scheduler + Collectors ---

type Job struct {
	ID        string     `json:"id"`
	Schedule  string     `json:"schedule"` // cron expression or interval
	Collector string     `json:"collector"`
	Agents    []string   `json:"agents,omitempty"`
	Probes    []string   `json:"probes,omitempty"`
	Mode      string     `json:"mode"` // pty | file | cli | gateway | vendor | enterprise | vscode
	Builtin   string     `json:"builtin,omitempty"`
	Enabled   bool       `json:"enabled"`
	LastRun   *time.Time `json:"last_run,omitempty"`
	NextRun   *time.Time `json:"next_run,omitempty"`
}

type JobRun struct {
	JobID     string    `json:"job_id"`
	RunID     string    `json:"run_id"`
	StartedAt time.Time `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
	Status    string    `json:"status"`
	Output    string    `json:"output,omitempty"`
}

// --- Plan 13: Telemetry ---

type MetricValue struct {
	Name      string            `json:"name"`
	Value     float64           `json:"value"`
	Labels    map[string]string `json:"labels,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// --- Plan 14: Security ---

type ReleaseSignature struct {
	Version  string `json:"version"`
	Commit   string `json:"commit"`
	Cosign   string `json:"cosign,omitempty"`
	Minisign string `json:"minisign,omitempty"`
	Checksum string `json:"checksum"`
}

// --- Plan 15: Enterprise ---

type Machine struct {
	ID         string            `json:"id"`
	Name       string            `json:"name,omitempty"`
	Status     string            `json:"status"`
	OS         string            `json:"os,omitempty"`
	Arch       string            `json:"arch,omitempty"`
	EnrolledAt *time.Time        `json:"enrolled_at,omitempty"`
	LastSeenAt *time.Time        `json:"last_seen_at,omitempty"`
	Tags       map[string]string `json:"tags,omitempty"`
}

type EnrollmentRequest struct {
	Token string `json:"token"`
}

type EnrollmentResponse struct {
	MachineID string `json:"machine_id"`
	Status    string `json:"status"`
}

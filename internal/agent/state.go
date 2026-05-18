package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/cruxctl/crux/internal/pty"
)

type StateStore interface {
	SaveAgent(agent AgentState) error
	SaveProbe(agentName string, probeName string, result ProbeResult) error
	SaveSession(session SessionState) error
}

type Store struct {
	Root string
}

type AgentState struct {
	ID               string    `json:"id" yaml:"id"`
	Name             string    `json:"name" yaml:"name"`
	Provider         string    `json:"provider,omitempty" yaml:"provider,omitempty"`
	Binary           string    `json:"binary" yaml:"binary"`
	BinaryPath       string    `json:"binaryPath,omitempty" yaml:"binaryPath,omitempty"`
	Version          string    `json:"version,omitempty" yaml:"version,omitempty"`
	Available        bool      `json:"available" yaml:"available"`
	Status           string    `json:"status" yaml:"status"`
	ConfigPaths      []string  `json:"configPaths,omitempty" yaml:"configPaths,omitempty"`
	KnownCommands    []string  `json:"knownCommands,omitempty" yaml:"knownCommands,omitempty"`
	LastDiscoveredAt time.Time `json:"lastDiscoveredAt" yaml:"lastDiscoveredAt"`
	LastProbeAt      time.Time `json:"lastProbeAt,omitempty" yaml:"lastProbeAt,omitempty"`
}

type ProbeResult struct {
	AgentName   string            `json:"agent" yaml:"agent"`
	ProbeName   string            `json:"probe" yaml:"probe"`
	Status      string            `json:"status" yaml:"status"`
	ParseSource string            `json:"parseSource" yaml:"parseSource"`
	Confidence  string            `json:"confidence" yaml:"confidence"`
	RawSaved    bool              `json:"rawSaved" yaml:"rawSaved"`
	CleanSaved  bool              `json:"cleanSaved" yaml:"cleanSaved"`
	RawPath     string            `json:"rawPath,omitempty" yaml:"rawPath,omitempty"`
	ANSIPath    string            `json:"ansiPath,omitempty" yaml:"ansiPath,omitempty"`
	CleanPath   string            `json:"cleanTextPath,omitempty" yaml:"cleanTextPath,omitempty"`
	ScreenPath  string            `json:"screenPath,omitempty" yaml:"screenPath,omitempty"`
	StartedAt   time.Time         `json:"startedAt" yaml:"startedAt"`
	EndedAt     time.Time         `json:"endedAt" yaml:"endedAt"`
	CleanText   string            `json:"cleanText,omitempty" yaml:"cleanText,omitempty"`
	FinalScreen string            `json:"finalScreen,omitempty" yaml:"finalScreen,omitempty"`
	Parsed      map[string]string `json:"parsed,omitempty" yaml:"parsed,omitempty"`
	Error       string            `json:"error,omitempty" yaml:"error,omitempty"`
}

type SessionState struct {
	ID             string    `json:"id" yaml:"id"`
	Agent          string    `json:"agent" yaml:"agent"`
	Status         string    `json:"status" yaml:"status"`
	StartedAt      time.Time `json:"startedAt" yaml:"startedAt"`
	EndedAt        time.Time `json:"endedAt,omitempty" yaml:"endedAt,omitempty"`
	TranscriptRaw  string    `json:"transcriptRaw,omitempty" yaml:"transcriptRaw,omitempty"`
	TranscriptText string    `json:"transcriptText,omitempty" yaml:"transcriptText,omitempty"`
	Error          string    `json:"error,omitempty" yaml:"error,omitempty"`
}

func DefaultStore() (Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Store{}, err
	}
	return Store{Root: filepath.Join(home, ".crux", "state")}, nil
}

func (s Store) SaveAgent(agent AgentState) error {
	if agent.ID == "" {
		return fmt.Errorf("agent id is required")
	}
	dir := s.agentDir(agent.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return writeJSON(filepath.Join(dir, "agent.json"), agent)
}

func (s Store) LoadAgent(name string) (AgentState, error) {
	path := filepath.Join(s.agentDir(CleanName(name)), "agent.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return AgentState{}, err
	}
	var state AgentState
	if err := json.Unmarshal(data, &state); err != nil {
		return AgentState{}, err
	}
	return state, nil
}

func (s Store) ListAgents() ([]AgentState, error) {
	root := filepath.Join(s.Root, "agents")
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	states := make([]AgentState, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		state, err := s.LoadAgent(entry.Name())
		if err == nil {
			states = append(states, state)
		}
	}
	sort.Slice(states, func(i, j int) bool { return states[i].ID < states[j].ID })
	return states, nil
}

func (s Store) SaveProbe(agentName string, probeName string, result ProbeResult) error {
	agentName = CleanName(agentName)
	probeName = CleanName(probeName)
	dir := s.agentDir(agentName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if result.RawPath != "" && result.CleanPath != "" {
		return writeJSON(filepath.Join(dir, "last_probe.json"), result)
	}
	rawPath := filepath.Join(dir, probeName+".raw")
	ansiPath := filepath.Join(dir, probeName+".ansi")
	cleanPath := filepath.Join(dir, probeName+".clean.txt")
	screenPath := filepath.Join(dir, probeName+".screen.latest.txt")
	if err := os.WriteFile(rawPath, nil, 0o600); err != nil {
		return err
	}
	if err := os.WriteFile(ansiPath, []byte(result.CleanText), 0o600); err != nil {
		return err
	}
	if err := os.WriteFile(cleanPath, []byte(result.CleanText), 0o600); err != nil {
		return err
	}
	if err := os.WriteFile(screenPath, []byte(result.FinalScreen), 0o600); err != nil {
		return err
	}
	result.RawPath = rawPath
	result.ANSIPath = ansiPath
	result.CleanPath = cleanPath
	result.ScreenPath = screenPath
	result.RawSaved = true
	result.CleanSaved = true
	return writeJSON(filepath.Join(dir, "last_probe.json"), result)
}

func (s Store) SavePTYProbe(agentName string, probeName string, ptyResult *pty.PTYResult, parseSource string, parsed map[string]string) (ProbeResult, error) {
	if ptyResult == nil {
		return ProbeResult{}, fmt.Errorf("pty result is required")
	}
	normalized := ptyResult.Normalized
	probe := ProbeResult{
		AgentName:   CleanName(agentName),
		ProbeName:   CleanName(probeName),
		Status:      ptyResult.Status,
		ParseSource: firstNonEmpty(parseSource, "clean_text"),
		Confidence:  "low",
		StartedAt:   ptyResult.StartedAt,
		EndedAt:     ptyResult.EndedAt,
		Parsed:      parsed,
		Error:       ptyResult.Error,
	}
	if normalized != nil {
		probe.Confidence = normalized.Confidence
		probe.CleanText = normalized.CleanText
		probe.FinalScreen = normalized.FinalScreen
	}
	dir := s.agentDir(CleanName(agentName))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return ProbeResult{}, err
	}
	rawPath := filepath.Join(dir, CleanName(probeName)+".raw")
	ansiPath := filepath.Join(dir, CleanName(probeName)+".ansi")
	cleanPath := filepath.Join(dir, CleanName(probeName)+".clean.txt")
	screenPath := filepath.Join(dir, CleanName(probeName)+".screen.latest.txt")
	if err := os.WriteFile(rawPath, ptyResult.Raw, 0o600); err != nil {
		return ProbeResult{}, err
	}
	ansi := ptyResult.Text
	if normalized != nil {
		ansi = normalized.ANSIText
	}
	if err := os.WriteFile(ansiPath, []byte(ansi), 0o600); err != nil {
		return ProbeResult{}, err
	}
	if err := os.WriteFile(cleanPath, []byte(probe.CleanText), 0o600); err != nil {
		return ProbeResult{}, err
	}
	if err := os.WriteFile(screenPath, []byte(probe.FinalScreen), 0o600); err != nil {
		return ProbeResult{}, err
	}
	probe.RawPath = rawPath
	probe.ANSIPath = ansiPath
	probe.CleanPath = cleanPath
	probe.ScreenPath = screenPath
	probe.RawSaved = true
	probe.CleanSaved = true
	if err := writeJSON(filepath.Join(dir, "last_probe.json"), probe); err != nil {
		return ProbeResult{}, err
	}
	return probe, nil
}

func (s Store) SaveSession(session SessionState) error {
	if session.ID == "" {
		return fmt.Errorf("session id is required")
	}
	dir := filepath.Join(s.Root, "sessions", session.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return writeJSON(filepath.Join(dir, "session.json"), session)
}

func (s Store) SaveSessionOutput(sessionID string, raw []byte, normalized *pty.PTYNormalizedOutput) (SessionState, error) {
	dir := filepath.Join(s.Root, "sessions", sessionID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return SessionState{}, err
	}
	rawPath := filepath.Join(dir, "transcript.raw")
	textPath := filepath.Join(dir, "transcript.clean.txt")
	if err := os.WriteFile(rawPath, raw, 0o600); err != nil {
		return SessionState{}, err
	}
	clean := string(raw)
	if normalized != nil {
		clean = normalized.CleanText
	}
	if err := os.WriteFile(textPath, []byte(clean), 0o600); err != nil {
		return SessionState{}, err
	}
	return SessionState{ID: sessionID, TranscriptRaw: rawPath, TranscriptText: textPath}, nil
}

func (s Store) loadLastProbe(agentName string) (ProbeResult, error) {
	data, err := os.ReadFile(filepath.Join(s.agentDir(CleanName(agentName)), "last_probe.json"))
	if err != nil {
		return ProbeResult{}, err
	}
	var result ProbeResult
	if err := json.Unmarshal(data, &result); err != nil {
		return ProbeResult{}, err
	}
	return result, nil
}

func (s Store) agentDir(agentName string) string {
	return filepath.Join(s.Root, "agents", CleanName(agentName))
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o600)
}

func firstNonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

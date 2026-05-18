package agent

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	agentconfigs "github.com/cruxctl/crux/configs/agents"
	"github.com/cruxctl/crux/internal/pty"
	"gopkg.in/yaml.v3"
)

type CommandSpec struct {
	Command string   `json:"command" yaml:"command"`
	Args    []string `json:"args,omitempty" yaml:"args,omitempty"`
}

type CommandProbe struct {
	Display      string          `json:"display,omitempty" yaml:"display,omitempty"`
	Command      string          `json:"command,omitempty" yaml:"command,omitempty"`
	Args         []string        `json:"args,omitempty" yaml:"args,omitempty"`
	WorkDir      string          `json:"workDir,omitempty" yaml:"work_dir,omitempty"`
	Input        string          `json:"input,omitempty" yaml:"input,omitempty"`
	Ready        pty.MatcherSpec `json:"ready,omitempty" yaml:"ready,omitempty"`
	Timeout      string          `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	ParseFrom    string          `json:"parseFrom,omitempty" yaml:"parse_from,omitempty"`
	CompleteWhen pty.MatcherSpec `json:"completeWhen,omitempty" yaml:"complete_when,omitempty"`
	Parser       ParserSpec      `json:"parser,omitempty" yaml:"parser,omitempty"`
}

type ParserSpec struct {
	Type     string            `json:"type,omitempty" yaml:"type,omitempty"`
	Patterns map[string]string `json:"patterns,omitempty" yaml:"patterns,omitempty"`
}

type Spec struct {
	ID          string                  `json:"id" yaml:"id"`
	Name        string                  `json:"name" yaml:"name"`
	Provider    string                  `json:"provider,omitempty" yaml:"provider,omitempty"`
	Binary      string                  `json:"binary" yaml:"binary"`
	Aliases     []string                `json:"aliases,omitempty" yaml:"aliases,omitempty"`
	ConfigPaths []string                `json:"configPaths,omitempty" yaml:"config_paths,omitempty"`
	Detect      CommandSpec             `json:"detect" yaml:"detect"`
	Launch      CommandSpec             `json:"launch" yaml:"launch"`
	Ready       pty.MatcherSpec         `json:"ready" yaml:"ready"`
	PTYEnv      map[string]string       `json:"ptyEnv,omitempty" yaml:"pty_env,omitempty"`
	Normalize   pty.NormalizeSpec       `json:"normalize" yaml:"normalize"`
	Commands    map[string]CommandProbe `json:"commands,omitempty" yaml:"commands,omitempty"`
}

func LoadBuiltinSpecs() ([]Spec, error) {
	return LoadSpecs(agentconfigs.FS, ".")
}

func LoadSpecs(fsys fs.FS, root string) ([]Spec, error) {
	pattern := filepath.Join(root, "*.yaml")
	matches, err := fs.Glob(fsys, pattern)
	if err != nil {
		return nil, err
	}
	specs := make([]Spec, 0, len(matches))
	for _, match := range matches {
		data, err := fs.ReadFile(fsys, match)
		if err != nil {
			return nil, fmt.Errorf("read agent spec %s: %w", match, err)
		}
		var spec Spec
		if err := yaml.Unmarshal(data, &spec); err != nil {
			return nil, fmt.Errorf("parse agent spec %s: %w", match, err)
		}
		spec.ApplyDefaults()
		if err := spec.Validate(); err != nil {
			return nil, fmt.Errorf("invalid agent spec %s: %w", match, err)
		}
		specs = append(specs, spec)
	}
	sort.Slice(specs, func(i, j int) bool { return specs[i].ID < specs[j].ID })
	return specs, nil
}

func (s Spec) Validate() error {
	if strings.TrimSpace(s.ID) == "" {
		return fmt.Errorf("id is required")
	}
	if strings.TrimSpace(s.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(s.Binary) == "" {
		return fmt.Errorf("binary is required")
	}
	return nil
}

func (s *Spec) ApplyDefaults() {
	if strings.TrimSpace(s.Detect.Command) == "" {
		s.Detect.Command = s.Binary
	}
	if strings.TrimSpace(s.Launch.Command) == "" {
		s.Launch.Command = s.Binary
	}
}

func (s Spec) KnownCommands() []string {
	names := make([]string, 0, len(s.Commands))
	for name := range s.Commands {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (s Spec) Matches(name string) bool {
	clean := CleanName(name)
	if clean == CleanName(s.ID) || clean == CleanName(s.Binary) {
		return true
	}
	for _, alias := range s.Aliases {
		if clean == CleanName(alias) {
			return true
		}
	}
	return false
}

func ResolveSpec(specs []Spec, name string) (Spec, bool) {
	for _, spec := range specs {
		if spec.Matches(name) {
			return spec, true
		}
	}
	return Spec{}, false
}

func ProbeTimeout(probe CommandProbe, fallback time.Duration) time.Duration {
	if strings.TrimSpace(probe.Timeout) == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(probe.Timeout)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func ResolveCommand(command string, specBinary string, binaryPath string, workDir string) string {
	command = strings.TrimSpace(ExpandValue(command, binaryPath, workDir))
	if command == "" || command == specBinary {
		return binaryPath
	}
	return command
}

func ExpandArgs(args []string, binaryPath string, workDir string) []string {
	out := make([]string, 0, len(args))
	for _, arg := range args {
		out = append(out, ExpandValue(arg, binaryPath, workDir))
	}
	return out
}

func ExpandValue(value string, binaryPath string, workDir string) string {
	home, _ := os.UserHomeDir()
	replacer := strings.NewReplacer(
		"{binary}", binaryPath,
		"{workdir}", workDir,
		"{cwd}", workDir,
		"{home}", home,
	)
	return replacer.Replace(value)
}

func CleanName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	return value
}

package pty

import (
	"context"
	"io"
	"time"
)

type PTYFactory interface {
	Create(ctx context.Context, spec PTYSpec) (*PTYTerminal, error)
}

type PTYRunner interface {
	Run(ctx context.Context, task PTYTask) (*PTYResult, error)
}

type PTYSpec struct {
	AgentName string
	Command   string
	Args      []string
	WorkDir   string
	Env       map[string]string
	Rows      int
	Cols      int
	Timeout   time.Duration
}

type PTYTask struct {
	AgentName     string
	Purpose       string
	Command       string
	Args          []string
	WorkDir       string
	Env           map[string]string
	Input         string
	ReadyMatcher  MatcherSpec
	DoneMatcher   MatcherSpec
	Normalize     NormalizeSpec
	ParseFrom     string
	Timeout       time.Duration
	CaptureOutput bool
	Interactive   bool
	Stdin         io.Reader
	Stdout        io.Writer
	Stderr        io.Writer
}

type PTYResult struct {
	AgentName  string
	Purpose    string
	Raw        []byte
	Text       string
	StartedAt  time.Time
	EndedAt    time.Time
	Status     string
	Error      string
	ExitCode   int
	Normalized *PTYNormalizedOutput
}

type MatcherSpec struct {
	Strategy    string   `json:"strategy" yaml:"strategy"`
	Patterns    []string `json:"patterns,omitempty" yaml:"patterns,omitempty"`
	Pattern     string   `json:"pattern,omitempty" yaml:"pattern,omitempty"`
	StableForMS int      `json:"stableForMs,omitempty" yaml:"stable_for_ms,omitempty"`
}

type NormalizeSpec struct {
	StripANSI           bool `json:"stripAnsi" yaml:"strip_ansi"`
	StripControlChars   bool `json:"stripControlChars" yaml:"strip_control_chars"`
	NormalizeBoxes      bool `json:"normalizeBoxes" yaml:"normalize_boxes"`
	NormalizeWhitespace bool `json:"normalizeWhitespace" yaml:"normalize_whitespace"`
	RemoveSpinners      bool `json:"removeSpinners" yaml:"remove_spinners"`
	RemoveStatusBars    bool `json:"removeStatusBars" yaml:"remove_status_bars"`
	CollapseRedraws     bool `json:"collapseRedraws" yaml:"collapse_redraws"`
	KeepFinalScreenOnly bool `json:"keepFinalScreenOnly" yaml:"keep_final_screen_only"`
}

type PTYRawOutput struct {
	AgentName string
	Purpose   string
	RawBytes  []byte
}

type PTYNormalizedOutput struct {
	RawPath       string `json:"rawPath,omitempty" yaml:"rawPath,omitempty"`
	ANSIPath      string `json:"ansiPath,omitempty" yaml:"ansiPath,omitempty"`
	CleanTextPath string `json:"cleanTextPath,omitempty" yaml:"cleanTextPath,omitempty"`

	RawBytes    []byte `json:"-" yaml:"-"`
	ANSIText    string `json:"ansiText,omitempty" yaml:"ansiText,omitempty"`
	CleanText   string `json:"cleanText,omitempty" yaml:"cleanText,omitempty"`
	FinalScreen string `json:"finalScreen,omitempty" yaml:"finalScreen,omitempty"`

	HasANSI    bool   `json:"hasAnsi" yaml:"hasAnsi"`
	HadRedraws bool   `json:"hadRedraws" yaml:"hadRedraws"`
	Confidence string `json:"confidence" yaml:"confidence"`
}

type OutputNormalizer interface {
	Normalize(ctx context.Context, input PTYRawOutput, spec NormalizeSpec) (*PTYNormalizedOutput, error)
}

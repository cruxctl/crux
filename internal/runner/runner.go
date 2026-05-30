package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/cruxctl/crux/internal/envpath"
	"github.com/cruxctl/crux/pkg/cruxapi"
)

var ErrOutputLimitExceeded = errors.New("command output exceeded maxOutputBytes")

type Runner interface {
	Run(ctx context.Context, agent cruxapi.Agent, execution cruxapi.Execution, runtime cruxapi.RuntimeConfig) Result
}

type CommandRunner struct{}

type Result struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Error    string
}

func NewCommandRunner() *CommandRunner {
	return &CommandRunner{}
}

func (r *CommandRunner) Run(ctx context.Context, agent cruxapi.Agent, execution cruxapi.Execution, runtime cruxapi.RuntimeConfig) Result {
	if isShell(agent.Command.Path) && !runtime.AllowShellCommands {
		return Result{
			ExitCode: 1,
			Error:    "shell-backed agents are disabled; set runtime.allowShellCommands=true to allow them",
		}
	}
	timeout := runtime.JobTimeoutSeconds
	if agent.Command.TimeoutSeconds > 0 {
		timeout = agent.Command.TimeoutSeconds
	}
	runCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	args, promptAsArg := renderArgs(agent.Command.Args, execution.Prompt)
	cmd := exec.CommandContext(runCtx, agent.Command.Path, args...)
	if !promptAsArg {
		cmd.Stdin = strings.NewReader(execution.Prompt)
	}
	workDir := strings.TrimSpace(agent.Command.WorkingDir)
	if workDir == "" {
		workDir = strings.TrimSpace(execution.WorkingDir)
	}
	if workDir != "" {
		cmd.Dir = workDir
	}
	cmd.Env = envpath.CommandEnv(os.Environ(), agent.Command.Path, agent.Command.Env)

	stdout := newLimitedBuffer(runtime.MaxOutputBytes)
	stderr := newLimitedBuffer(runtime.MaxOutputBytes)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err := cmd.Run()
	result := Result{
		ExitCode: exitCode(err),
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}
	if stdout.Exceeded() || stderr.Exceeded() {
		result.Error = ErrOutputLimitExceeded.Error()
		if result.ExitCode == 0 {
			result.ExitCode = 1
		}
		return result
	}
	if runCtx.Err() == context.DeadlineExceeded {
		result.Error = fmt.Sprintf("execution exceeded timeout of %d seconds", timeout)
		if result.ExitCode == 0 {
			result.ExitCode = 124
		}
		return result
	}
	if err != nil {
		result.Error = err.Error()
	}
	return result
}

func isShell(path string) bool {
	name := filepath.Base(path)
	switch name {
	case "sh", "bash", "zsh", "fish", "dash", "ksh":
		return true
	default:
		return false
	}
}

func renderArgs(args []string, prompt string) ([]string, bool) {
	rendered := make([]string, len(args))
	used := false
	for i, arg := range args {
		if strings.Contains(arg, "{prompt}") {
			used = true
			rendered[i] = strings.ReplaceAll(arg, "{prompt}", prompt)
			continue
		}
		rendered[i] = arg
	}
	return rendered, used
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus()
		}
	}
	return 1
}

type limitedBuffer struct {
	limit    int
	exceeded bool
	buf      bytes.Buffer
}

func newLimitedBuffer(limit int) *limitedBuffer {
	return &limitedBuffer{limit: limit}
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if b.limit < 1 {
		b.exceeded = true
		return 0, ErrOutputLimitExceeded
	}
	remaining := b.limit - b.buf.Len()
	if remaining <= 0 {
		b.exceeded = true
		return 0, ErrOutputLimitExceeded
	}
	if len(p) > remaining {
		_, _ = b.buf.Write(p[:remaining])
		b.exceeded = true
		return remaining, ErrOutputLimitExceeded
	}
	return b.buf.Write(p)
}

func (b *limitedBuffer) String() string {
	return b.buf.String()
}

func (b *limitedBuffer) Exceeded() bool {
	return b.exceeded
}

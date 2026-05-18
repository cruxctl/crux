package pty

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

type Runner struct {
	factory    PTYFactory
	normalizer OutputNormalizer
}

func NewRunner(factory PTYFactory, normalizer OutputNormalizer) *Runner {
	if factory == nil {
		factory = NewFactory()
	}
	if normalizer == nil {
		normalizer = NewNormalizer()
	}
	return &Runner{factory: factory, normalizer: normalizer}
}

func (r *Runner) Run(ctx context.Context, task PTYTask) (*PTYResult, error) {
	if strings.TrimSpace(task.Command) == "" {
		return nil, fmt.Errorf("pty command is required")
	}
	if task.Timeout <= 0 {
		task.Timeout = 20 * time.Second
	}
	runCtx, cancel := context.WithTimeout(ctx, task.Timeout)
	defer cancel()
	terminal, err := r.factory.Create(runCtx, PTYSpec{
		AgentName: task.AgentName,
		Command:   task.Command,
		Args:      task.Args,
		WorkDir:   task.WorkDir,
		Env:       task.Env,
		Timeout:   task.Timeout,
	})
	if err != nil {
		return nil, err
	}
	defer terminal.Close()
	if task.Interactive {
		return r.runInteractive(runCtx, terminal, task)
	}
	return r.runProbe(runCtx, terminal, task)
}

func (r *Runner) runInteractive(ctx context.Context, terminal *PTYTerminal, task PTYTask) (*PTYResult, error) {
	started := terminal.StartedAt
	if task.Stdin == nil {
		task.Stdin = io.Reader(nil)
	}
	if task.Stdout == nil {
		task.Stdout = io.Discard
	}
	var raw bytes.Buffer
	readDone := make(chan error, 1)
	go func() {
		readDone <- copyPTYOutput(io.MultiWriter(task.Stdout, &raw), terminal)
	}()
	if task.Stdin != nil {
		go func() {
			_, _ = io.Copy(terminal, task.Stdin)
		}()
	}
	waitErr := terminal.Wait()
	<-readDone
	ended := time.Now().UTC()
	result := &PTYResult{
		AgentName: task.AgentName,
		Purpose:   task.Purpose,
		Raw:       raw.Bytes(),
		Text:      raw.String(),
		StartedAt: started,
		EndedAt:   ended,
		Status:    "completed",
		ExitCode:  exitCode(waitErr),
	}
	if waitErr != nil {
		result.Status = "failed"
		result.Error = waitErr.Error()
	}
	normalized, err := r.normalizer.Normalize(ctx, PTYRawOutput{AgentName: task.AgentName, Purpose: task.Purpose, RawBytes: result.Raw}, task.Normalize)
	if err == nil {
		result.Normalized = normalized
	}
	return result, nil
}

func (r *Runner) runProbe(ctx context.Context, terminal *PTYTerminal, task PTYTask) (*PTYResult, error) {
	started := terminal.StartedAt
	chunks := make(chan []byte, 32)
	readDone := make(chan error, 1)
	waitDone := make(chan error, 1)
	go readPTYChunks(terminal, chunks, readDone)
	go func() { waitDone <- terminal.Wait() }()

	var raw bytes.Buffer
	if err := waitReady(ctx, task.ReadyMatcher, chunks, &raw); err != nil {
		_ = terminal.Kill()
		return probeResult(task, started, raw.Bytes(), "failed", err, 1, r.normalizer)
	}
	if task.Input != "" {
		if _, err := io.WriteString(terminal, normalizeProbeInput(task.Input)); err != nil {
			_ = terminal.Kill()
			return probeResult(task, started, raw.Bytes(), "failed", err, 1, r.normalizer)
		}
	}

	waitErr := waitProbeDone(ctx, task.DoneMatcher, chunks, waitDone, &raw)
	if waitErr != nil && !errors.Is(waitErr, context.DeadlineExceeded) {
		_ = terminal.Kill()
		return probeResult(task, started, raw.Bytes(), "failed", waitErr, exitCode(waitErr), r.normalizer)
	}
	_ = terminal.Kill()
	drainPTY(chunks, readDone, &raw)
	status := "completed"
	if errors.Is(waitErr, context.DeadlineExceeded) {
		status = "timeout"
	}
	return probeResult(task, started, raw.Bytes(), status, waitErr, exitCode(waitErr), r.normalizer)
}

func waitReady(ctx context.Context, spec MatcherSpec, chunks <-chan []byte, raw *bytes.Buffer) error {
	if strings.TrimSpace(spec.Strategy) == "" {
		return nil
	}
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	for {
		if Match(spec, raw.String()) {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return fmt.Errorf("pty ready matcher %q did not match before input", spec.Strategy)
		case chunk, ok := <-chunks:
			if !ok {
				return fmt.Errorf("pty output closed before ready matcher %q matched", spec.Strategy)
			}
			raw.Write(chunk)
		}
	}
}

func waitProbeDone(ctx context.Context, spec MatcherSpec, chunks <-chan []byte, waitDone <-chan error, raw *bytes.Buffer) error {
	strategy := strings.TrimSpace(spec.Strategy)
	stableFor := StableDuration(spec, 800*time.Millisecond)
	if strategy == "" {
		stableFor = 500 * time.Millisecond
	}
	waitForProcessExit := strategy == "process_exit"
	stabilityCompletes := strategy == "" || strategy == "screen_stable" || strategy == "silence_for_duration"
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	lastLen := raw.Len()
	sawOutput := false
	lastChange := time.Now()
	for {
		if doneMatcherMatched(spec, raw.String()) {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-waitDone:
			return err
		case chunk, ok := <-chunks:
			if ok {
				raw.Write(chunk)
				if raw.Len() != lastLen {
					lastLen = raw.Len()
					sawOutput = true
					lastChange = time.Now()
				}
			}
		case <-ticker.C:
			if sawOutput && stabilityCompletes && !waitForProcessExit && time.Since(lastChange) >= stableFor {
				return nil
			}
		}
	}
}

func normalizeProbeInput(input string) string {
	input = strings.ReplaceAll(input, "\r\n", "\n")
	input = strings.ReplaceAll(input, "\r", "\n")
	return strings.ReplaceAll(input, "\n", "\r")
}

func doneMatcherMatched(spec MatcherSpec, screen string) bool {
	switch strings.TrimSpace(spec.Strategy) {
	case "screen_contains_any", "screen_contains", "regex":
		return Match(spec, screen)
	default:
		return false
	}
}

func probeResult(task PTYTask, started time.Time, raw []byte, status string, err error, code int, normalizer OutputNormalizer) (*PTYResult, error) {
	ended := time.Now().UTC()
	result := &PTYResult{
		AgentName: task.AgentName,
		Purpose:   task.Purpose,
		Raw:       append([]byte{}, raw...),
		Text:      string(raw),
		StartedAt: started,
		EndedAt:   ended,
		Status:    status,
		ExitCode:  code,
	}
	if err != nil {
		result.Error = err.Error()
	}
	if normalizer != nil {
		normalized, normalizeErr := normalizer.Normalize(context.Background(), PTYRawOutput{AgentName: task.AgentName, Purpose: task.Purpose, RawBytes: raw}, task.Normalize)
		if normalizeErr == nil {
			result.Normalized = normalized
		}
	}
	return result, nil
}

func readPTYChunks(reader io.Reader, chunks chan<- []byte, done chan<- error) {
	defer close(chunks)
	buf := make([]byte, 4096)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			chunks <- chunk
		}
		if err != nil {
			done <- err
			return
		}
	}
}

func drainPTY(chunks <-chan []byte, readDone <-chan error, raw *bytes.Buffer) {
	timer := time.NewTimer(100 * time.Millisecond)
	defer timer.Stop()
	for {
		select {
		case chunk, ok := <-chunks:
			if ok {
				raw.Write(chunk)
				continue
			}
			return
		case <-readDone:
			return
		case <-timer.C:
			return
		}
	}
}

func copyPTYOutput(dst io.Writer, src io.Reader) error {
	_, err := io.Copy(dst, src)
	return err
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return 1
}

package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/cruxctl/crux/internal/client"
)

const (
	defaultCruxInstallScriptURL      = "https://raw.githubusercontent.com/cruxctl/crux/main/scripts/install-crux.sh"
	defaultCruxInstallPowerShellURL  = "https://raw.githubusercontent.com/cruxctl/crux/main/scripts/install-crux.ps1"
	defaultCruxdInstallScriptURL     = "https://raw.githubusercontent.com/cruxctl/cruxd/main/scripts/install-cruxd.sh"
	defaultCruxdInstallPowerShellURL = "https://raw.githubusercontent.com/cruxctl/cruxd/main/scripts/install-cruxd.ps1"
)

var (
	healthCheckTimeout       = 2 * time.Second
	postInstallHealthTimeout = 15 * time.Second
)

func findCruxd() (string, bool) {
	if path, err := exec.LookPath("cruxd"); err == nil {
		return path, true
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false
	}
	path := filepath.Join(home, ".local", "bin", "cruxd")
	info, err := os.Stat(path)
	if err != nil || info.IsDir() || info.Mode()&0o111 == 0 {
		return "", false
	}
	return path, true
}

func (c *CLI) confirm(prompt string) (bool, error) {
	fmt.Fprintf(c.out, "%s [y/N] ", prompt)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return false, err
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes", nil
}

func (c *CLI) runInstaller(ctx context.Context, shellURL, powershellURL string, args []string, env map[string]string) error {
	if runtime.GOOS == "windows" {
		return c.runPowerShellInstaller(ctx, powershellURL, args, env)
	}
	return c.runShellInstaller(ctx, shellURL, args, env)
}

func (c *CLI) runShellInstaller(ctx context.Context, scriptURL string, args []string, env map[string]string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, scriptURL, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download install script: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download install script: HTTP %d", resp.StatusCode)
	}
	tmp, err := os.CreateTemp("", "crux-install-*.sh")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := io.Copy(tmp, io.LimitReader(resp.Body, 1024*1024)); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write install script: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpPath, 0o700); err != nil {
		return err
	}
	cmdArgs := append([]string{tmpPath}, args...)
	cmd := exec.CommandContext(ctx, "/bin/sh", cmdArgs...)
	cmd.Env = mergeProcessEnv(os.Environ(), env)
	cmd.Stdout = c.out
	cmd.Stderr = c.err
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run install script: %w", err)
	}
	return nil
}

func (c *CLI) runPowerShellInstaller(ctx context.Context, scriptURL string, args []string, env map[string]string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, scriptURL, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download install script: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download install script: HTTP %d", resp.StatusCode)
	}
	tmp, err := os.CreateTemp("", "crux-install-*.ps1")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := io.Copy(tmp, io.LimitReader(resp.Body, 1024*1024)); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write install script: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	ps, err := powershellPath()
	if err != nil {
		return err
	}
	cmdArgs := append([]string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-File", tmpPath}, args...)
	cmd := exec.CommandContext(ctx, ps, cmdArgs...)
	cmd.Env = mergeProcessEnv(os.Environ(), env)
	cmd.Stdout = c.out
	cmd.Stderr = c.err
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func waitForHealth(ctx context.Context, cl *client.Client, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		healthCtx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
		lastErr = cl.Health(healthCtx)
		cancel()
		if lastErr == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
	return lastErr
}

func powershellPath() (string, error) {
	for _, name := range []string{"pwsh", "powershell"} {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("PowerShell was not found on PATH")
}

func mergeProcessEnv(base []string, extra map[string]string) []string {
	if len(extra) == 0 {
		return base
	}
	out := append([]string{}, base...)
	for key, value := range extra {
		out = append(out, key+"="+value)
	}
	return out
}

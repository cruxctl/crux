package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode"

	"github.com/cruxctl/crux/internal/client"
)

const (
	defaultCruxInstallScriptURL      = "https://raw.githubusercontent.com/cruxctl/crux/main/scripts/install-crux.sh"
	defaultCruxInstallPowerShellURL  = "https://raw.githubusercontent.com/cruxctl/crux/main/scripts/install-crux.ps1"
	defaultCruxdInstallScriptURL     = "https://raw.githubusercontent.com/cruxctl/cruxd/main/scripts/install-cruxd.sh"
	defaultCruxdInstallPowerShellURL = "https://raw.githubusercontent.com/cruxctl/cruxd/main/scripts/install-cruxd.ps1"
	githubAPIBaseURL                 = "https://api.github.com"
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

func (c *CLI) runInstaller(ctx context.Context, shellURL, powershellURL string, args []string, env map[string]string, stdout io.Writer) error {
	if runtime.GOOS == "windows" {
		return c.runPowerShellInstaller(ctx, powershellURL, args, env, stdout)
	}
	return c.runShellInstaller(ctx, shellURL, args, env, stdout)
}

func (c *CLI) runShellInstaller(ctx context.Context, scriptURL string, args []string, env map[string]string, stdout io.Writer) error {
	resolvedURL, err := resolveInstallScriptURL(ctx, scriptURL)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, resolvedURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("User-Agent", "crux")
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
	cmd.Stdout = stdout
	cmd.Stderr = c.err
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run install script: %w", err)
	}
	return nil
}

func (c *CLI) runPowerShellInstaller(ctx context.Context, scriptURL string, args []string, env map[string]string, stdout io.Writer) error {
	resolvedURL, err := resolveInstallScriptURL(ctx, scriptURL)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, resolvedURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("User-Agent", "crux")
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
	cmd.Stdout = stdout
	cmd.Stderr = c.err
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func resolveInstallScriptURL(ctx context.Context, scriptURL string) (string, error) {
	switch scriptURL {
	case defaultCruxInstallScriptURL:
		return commitPinnedRawURL(ctx, "cruxctl", "crux", "main", "scripts/install-crux.sh")
	case defaultCruxInstallPowerShellURL:
		return commitPinnedRawURL(ctx, "cruxctl", "crux", "main", "scripts/install-crux.ps1")
	case defaultCruxdInstallScriptURL:
		return commitPinnedRawURL(ctx, "cruxctl", "cruxd", "main", "scripts/install-cruxd.sh")
	case defaultCruxdInstallPowerShellURL:
		return commitPinnedRawURL(ctx, "cruxctl", "cruxd", "main", "scripts/install-cruxd.ps1")
	default:
		return scriptURL, nil
	}
}

func commitPinnedRawURL(ctx context.Context, owner, repo, ref, path string) (string, error) {
	sha, err := resolveGitHubSHA(ctx, owner, repo, ref)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", owner, repo, sha, path), nil
}

func resolveGitHubSHA(ctx context.Context, owner, repo, ref string) (string, error) {
	if isSHA(ref) {
		return strings.ToLower(ref), nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/repos/%s/%s/commits/%s", githubAPIBaseURL, owner, repo, ref), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "crux")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("resolve %s/%s@%s: %w", owner, repo, ref, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("resolve %s/%s@%s: HTTP %d", owner, repo, ref, resp.StatusCode)
	}
	var commit struct {
		SHA string `json:"sha"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64*1024)).Decode(&commit); err != nil {
		return "", fmt.Errorf("resolve %s/%s@%s: %w", owner, repo, ref, err)
	}
	if !isSHA(commit.SHA) {
		return "", fmt.Errorf("resolve %s/%s@%s: invalid sha %q", owner, repo, ref, commit.SHA)
	}
	return strings.ToLower(commit.SHA), nil
}

func isSHA(value string) bool {
	if len(value) != 40 {
		return false
	}
	for _, r := range value {
		if !unicode.Is(unicode.ASCII_Hex_Digit, r) {
			return false
		}
	}
	return true
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

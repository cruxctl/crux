// Package console provides crux console lifecycle commands.
package console

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	defaultPort = 4358
	pidFile     = "console.pid"
)

// Options configures console commands.
type Options struct {
	Out    io.Writer
	Err    io.Writer
	Port   int
	Dev    bool
	APIURL string
}

func pidPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".crux", pidFile)
}

func consoleDir() string {
	// Look for web/console relative to the binary location.
	ex, err := os.Executable()
	if err == nil {
		dir := filepath.Join(filepath.Dir(ex), "..", "web", "console")
		if _, serr := os.Stat(filepath.Join(dir, "package.json")); serr == nil {
			return dir
		}
	}
	// Fallback: relative to working directory (development).
	if _, err := os.Stat("web/console/package.json"); err == nil {
		return "web/console"
	}
	return ""
}

// Start launches the console Next.js app.
func Start(ctx context.Context, opts Options) error {
	if opts.Port == 0 {
		opts.Port = defaultPort
	}

	// Check if already running.
	if pid, _ := readPID(); pid > 0 {
		if isRunning(pid) {
			fmt.Fprintf(opts.Out, "Console already running at http://127.0.0.1:%d (pid %d)\n", opts.Port, pid)
			return nil
		}
	}

	dir := consoleDir()
	if dir == "" {
		return fmt.Errorf("console not found: web/console/package.json missing")
	}

	var cmd *exec.Cmd
	if opts.Dev {
		cmd = exec.CommandContext(ctx, "pnpm", "dev", "-p", strconv.Itoa(opts.Port))
	} else {
		// Ensure the app is built.
		if _, err := os.Stat(filepath.Join(dir, ".next", "standalone")); os.IsNotExist(err) {
			fmt.Fprintln(opts.Out, "Building console...")
			build := exec.CommandContext(ctx, "pnpm", "build")
			build.Dir = dir
			build.Stdout = opts.Out
			build.Stderr = opts.Err
			if err := build.Run(); err != nil {
				return fmt.Errorf("console build failed: %w", err)
			}
		}
		cmd = exec.CommandContext(ctx, "node", "server.js")
		cmd.Dir = filepath.Join(dir, ".next", "standalone")
	}
	cmd.Dir = dir
	cmd.Stdout = opts.Out
	cmd.Stderr = opts.Err
	cmd.Env = os.Environ()
	if opts.APIURL != "" {
		cmd.Env = append(cmd.Env, "CRUX_CONSOLE_API_URL="+opts.APIURL)
	}
	cmd.Env = append(cmd.Env, "PORT="+strconv.Itoa(opts.Port))

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start console: %w", err)
	}

	if err := writePID(cmd.Process.Pid); err != nil {
		_ = cmd.Process.Kill()
		return fmt.Errorf("failed to record console pid: %w", err)
	}

	fmt.Fprintf(opts.Out, "Console starting at http://127.0.0.1:%d (pid %d)\n", opts.Port, cmd.Process.Pid)
	return nil
}

// Stop terminates the running console.
func Stop(ctx context.Context, opts Options) error {
	pid, err := readPID()
	if err != nil || pid <= 0 {
		fmt.Fprintln(opts.Out, "Console not running")
		return nil
	}
	if !isRunning(pid) {
		_ = os.Remove(pidPath())
		fmt.Fprintln(opts.Out, "Console not running")
		return nil
	}

	p, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find console process: %w", err)
	}

	// Graceful shutdown with SIGTERM, then SIGKILL.
	if err := p.Signal(syscall.SIGTERM); err != nil {
		_ = p.Kill()
	}

	done := make(chan error, 1)
	go func() {
		_, err := p.Wait()
		done <- err
	}()

	select {
	case <-done:
		fmt.Fprintln(opts.Out, "Console stopped")
	case <-time.After(5 * time.Second):
		_ = p.Kill()
		fmt.Fprintln(opts.Out, "Console killed")
	}

	_ = os.Remove(pidPath())
	return nil
}

// Status checks whether the console is running.
func Status(ctx context.Context, opts Options) error {
	if opts.Port == 0 {
		opts.Port = defaultPort
	}
	pid, _ := readPID()
	if pid > 0 && isRunning(pid) {
		fmt.Fprintf(opts.Out, "Console running at http://127.0.0.1:%d (pid %d)\n", opts.Port, pid)
	} else {
		fmt.Fprintln(opts.Out, "Console stopped")
	}
	return nil
}

// Open launches the default browser to the console URL.
func Open(ctx context.Context, opts Options) error {
	if opts.Port == 0 {
		opts.Port = defaultPort
	}
	url := fmt.Sprintf("http://127.0.0.1:%d", opts.Port)
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}
	return exec.CommandContext(ctx, cmd, args...).Start()
}

func readPID() (int, error) {
	data, err := os.ReadFile(pidPath())
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

func writePID(pid int) error {
	path := pidPath()
	_ = os.MkdirAll(filepath.Dir(path), 0o700)
	return os.WriteFile(path, []byte(strconv.Itoa(pid)), 0o600)
}

func isRunning(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Unix, FindProcess always succeeds; Signal(0) tests existence.
	err = p.Signal(syscall.Signal(0))
	return err == nil
}

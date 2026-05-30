//go:build integration

package daemon

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/cruxctl/crux/internal/config"
)

func TestE2E_DaemonHealthAndVersion(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.Defaults()
	cfg.Daemon.Port = 0
	cfg.State.Root = t.TempDir()

	addrCh := make(chan string, 1)
	go func() {
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
		if err := RunOnAddr(ctx, cfg, logger, addrCh); err != nil && err != context.Canceled {
			t.Logf("daemon exited: %v", err)
		}
	}()

	addr := <-addrCh
	if addr == "" {
		t.Fatal("daemon did not report address")
	}

	base := "http://" + addr
	waitForReady(t, base)

	resp, err := http.Get(base + "/healthz")
	if err != nil {
		t.Fatalf("healthz: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("healthz: %d %s", resp.StatusCode, body)
	}

	resp, err = http.Get(base + "/v1/version")
	if err != nil {
		t.Fatalf("version: %v", err)
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("version: %d %s", resp.StatusCode, body)
	}
}

func TestE2E_AgentsRefresh(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.Defaults()
	cfg.Daemon.Port = 0
	cfg.State.Root = t.TempDir()

	addrCh := make(chan string, 1)
	go func() {
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
		if err := RunOnAddr(ctx, cfg, logger, addrCh); err != nil && err != context.Canceled {
			t.Logf("daemon exited: %v", err)
		}
	}()

	addr := <-addrCh
	base := "http://" + addr
	waitForReady(t, base)

	resp, err := http.Post(base+"/v1/agents/refresh", "application/json", nil)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("refresh: %d %s", resp.StatusCode, body)
	}
}

func waitForReady(t *testing.T, base string) {
	t.Helper()
	for i := 0; i < 50; i++ {
		resp, err := http.Get(base + "/healthz")
		if err == nil {
			resp.Body.Close()
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("daemon not ready")
}

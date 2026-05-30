package daemon

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/cruxctl/crux/internal/agent"
	"github.com/cruxctl/crux/internal/api"
	"github.com/cruxctl/crux/internal/config"
	"github.com/cruxctl/crux/internal/discovery"
	"github.com/cruxctl/crux/internal/runner"
	"github.com/cruxctl/crux/internal/service"
	"github.com/cruxctl/crux/internal/statepath"
	"github.com/cruxctl/crux/internal/state"
)

func Run(ctx context.Context, cfg config.DaemonConfig, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}
	st, err := store.NewFileStore(cfg.State.Root)
	if err != nil {
		return err
	}
	ds := store.NewDomainStore(st)
	runtime, err := ds.RuntimeConfig(ctx)
	if err != nil {
		return err
	}
	// Load canonical agent specs.
	agentReg, err := loadAgentRegistry()
	if err != nil {
		logger.Warn("agent registry load failed", "error", err)
		agentReg = agent.NewRegistry()
	}

	svc := service.New(ds, runner.NewCommandRunner(), discovery.NewDiscoverer(agentReg), runtime, logger)
	if err := svc.RecoverInterruptedExecutions(ctx); err != nil {
		return fmt.Errorf("recover interrupted executions: %w", err)
	}

	reloadAgents := func() error {
		reg, err := loadAgentRegistry()
		if err != nil {
			return err
		}
		_ = reg
		logger.Info("agent registry reloaded")
		return nil
	}

	srv := api.NewServer(cfg, svc, logger).WithAgentReload(reloadAgents)
	httpServer := &http.Server{
		Addr:         cfg.ListenAddress(),
		Handler:      srv.Handler(),
		ReadTimeout:  time.Duration(cfg.Daemon.ReadTimeoutSeconds) * time.Second,
		WriteTimeout: time.Duration(cfg.Daemon.WriteTimeoutSeconds) * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("cruxd listening", "address", cfg.ListenAddress(), "store", cfg.State.Root)
		err := httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	for {
		select {
		case sig := <-sigCh:
			if sig == syscall.SIGHUP {
				logger.Info("received SIGHUP, reloading agent registry")
				if err := reloadAgents(); err != nil {
					logger.Error("agent reload failed", "error", err)
				}
				continue
			}
			logger.Info("received signal, shutting down", "signal", sig.String())
			shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Daemon.ShutdownGraceSeconds)*time.Second)
			if err := httpServer.Shutdown(shutdownCtx); err != nil {
				cancel()
				return fmt.Errorf("shutdown server: %w", err)
			}
			cancel()
			return <-errCh
		case err := <-errCh:
			return err
		case <-ctx.Done():
			shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Daemon.ShutdownGraceSeconds)*time.Second)
			if err := httpServer.Shutdown(shutdownCtx); err != nil {
				cancel()
				return fmt.Errorf("shutdown server: %w", err)
			}
			cancel()
			return <-errCh
		}
	}
}

// RunOnAddr is like Run but writes the actual bound address to addrCh
// before serving. Used by integration tests with port 0.
func RunOnAddr(ctx context.Context, cfg config.DaemonConfig, logger *slog.Logger, addrCh chan<- string) error {
	if logger == nil {
		logger = slog.Default()
	}
	st, err := store.NewFileStore(cfg.State.Root)
	if err != nil {
		return err
	}
	ds := store.NewDomainStore(st)
	runtime, err := ds.RuntimeConfig(ctx)
	if err != nil {
		return err
	}
	agentReg, err := loadAgentRegistry()
	if err != nil {
		logger.Warn("agent registry load failed", "error", err)
		agentReg = agent.NewRegistry()
	}
	svc := service.New(ds, runner.NewCommandRunner(), discovery.NewDiscoverer(agentReg), runtime, logger)
	addr := cfg.ListenAddress()
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	if addrCh != nil {
		addrCh <- ln.Addr().String()
	}

	reloadAgents := func() error {
		reg, err := loadAgentRegistry()
		if err != nil {
			return err
		}
		_ = reg
		logger.Info("agent registry reloaded")
		return nil
	}

	srv := api.NewServer(cfg, svc, logger).WithAgentReload(reloadAgents)
	hs := &http.Server{Handler: srv.Handler(), ReadTimeout: time.Duration(cfg.Daemon.ReadTimeoutSeconds) * time.Second, WriteTimeout: time.Duration(cfg.Daemon.WriteTimeoutSeconds) * time.Second}
	go func() {
		<-ctx.Done()
		_ = hs.Shutdown(context.Background())
	}()
	if err := hs.Serve(ln); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func loadAgentRegistry() (*agent.Registry, error) {
	defaultsDir := os.Getenv("CRUX_AGENT_SPECS_DIR")
	if defaultsDir == "" {
		defaultsDir = "examples/agents"
	}
	userDir := filepath.Join(statepath.StateRoot(), "agents", "specs")
	return agent.LoadAll(defaultsDir, userDir)
}

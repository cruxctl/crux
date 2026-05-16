package daemon

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cruxctl/crux/internal/api"
	"github.com/cruxctl/crux/internal/config"
	"github.com/cruxctl/crux/internal/discovery"
	"github.com/cruxctl/crux/internal/runner"
	"github.com/cruxctl/crux/internal/service"
	"github.com/cruxctl/crux/internal/store"
)

func Run(ctx context.Context, cfg config.DaemonConfig, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}
	st, err := store.NewFileStore(cfg.Store.Path, cfg.Runtime)
	if err != nil {
		return err
	}
	runtime, err := st.RuntimeConfig(ctx)
	if err != nil {
		return err
	}
	svc := service.New(st, runner.NewCommandRunner(), discovery.DefaultDiscoverer(), runtime, logger)
	httpServer := &http.Server{
		Addr:         cfg.ListenAddress(),
		Handler:      api.NewServer(cfg, svc, logger).Handler(),
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeoutSeconds) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeoutSeconds) * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("cruxd listening", "address", cfg.ListenAddress(), "store", cfg.Store.Path)
		err := httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	signalCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case <-signalCtx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Server.ShutdownGraceSeconds)*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown server: %w", err)
		}
		return <-errCh
	case err := <-errCh:
		return err
	}
}

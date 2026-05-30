//go:build integration

package daemon

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/cruxctl/crux/internal/config"
)

func TestDaemon_StartsAndServesOpenAPI(t *testing.T) {
	t.Setenv("CRUX_HOME", t.TempDir())
	cfg := config.Defaults()
	cfg.Daemon.Port = 0 // ephemeral port; test reads bound port from server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	addrCh := make(chan string, 1)
	errCh := make(chan error, 1)
	go func() {
		err := RunOnAddr(ctx, cfg, nil, addrCh)
		errCh <- err
	}()

	select {
	case addr := <-addrCh:
		resp, err := http.Get("http://" + addr + "/v1/openapi.json")
		if err != nil {
			t.Fatalf("GET: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Fatalf("code = %d", resp.StatusCode)
		}
		body, _ := io.ReadAll(resp.Body)
		var doc map[string]any
		json.Unmarshal(body, &doc)
		if doc["openapi"] != "3.0.3" {
			t.Errorf("openapi field = %v", doc["openapi"])
		}
	case <-time.After(5 * time.Second):
		t.Fatal("daemon did not bind in 5s")
	}
}

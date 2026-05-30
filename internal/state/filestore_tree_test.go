package store

import (
	"context"
	"strings"
	"testing"

	"github.com/cruxctl/crux/internal/statepath"
)

func TestFileStoreTree_PutGetAppendList(t *testing.T) {
	t.Setenv("CRUX_HOME", t.TempDir())
	s, err := NewFileStore(statepath.StateRoot())
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	ctx := context.Background()

	if err := s.Put(ctx, "agents/abc/agent.json", []byte(`{"id":"a"}`)); err != nil {
		t.Fatalf("Put: %v", err)
	}
	got, err := s.Get(ctx, "agents/abc/agent.json")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got) != `{"id":"a"}` {
		t.Errorf("Get = %q", got)
	}

	if err := s.Append(ctx, "aos/events/events-2026-05-18.jsonl", []byte("e1\n")); err != nil {
		t.Fatalf("Append: %v", err)
	}
	if err := s.Append(ctx, "aos/events/events-2026-05-18.jsonl", []byte("e2\n")); err != nil {
		t.Fatalf("Append: %v", err)
	}
	got2, _ := s.Get(ctx, "aos/events/events-2026-05-18.jsonl")
	if string(got2) != "e1\ne2\n" {
		t.Errorf("Append accumulation = %q", got2)
	}

	objs, err := s.List(ctx, "agents/")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(objs) != 1 || !strings.HasSuffix(objs[0].Path, "agents/abc/agent.json") {
		t.Errorf("List = %+v", objs)
	}
}

func TestFileStoreTree_LockReleases(t *testing.T) {
	t.Setenv("CRUX_HOME", t.TempDir())
	s, _ := NewFileStore(statepath.StateRoot())
	ctx := context.Background()
	unlock, err := s.Lock(ctx, "agents/abc")
	if err != nil {
		t.Fatalf("Lock: %v", err)
	}
	unlock()
	// Re-locking the same path should succeed after release.
	unlock2, err := s.Lock(ctx, "agents/abc")
	if err != nil {
		t.Fatalf("Re-Lock: %v", err)
	}
	unlock2()
}

func TestFileStoreTree_Delete(t *testing.T) {
	t.Setenv("CRUX_HOME", t.TempDir())
	s, _ := NewFileStore(statepath.StateRoot())
	ctx := context.Background()
	s.Put(ctx, "agents/abc/agent.json", []byte("x"))
	if err := s.Delete(ctx, "agents/abc/agent.json"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err := s.Get(ctx, "agents/abc/agent.json")
	if err == nil {
		t.Errorf("Get after Delete should fail")
	}
}

func TestFileStoreTree_PathTraversalRejected(t *testing.T) {
	t.Setenv("CRUX_HOME", t.TempDir())
	s, _ := NewFileStore(statepath.StateRoot())
	ctx := context.Background()
	if err := s.Put(ctx, "../escape.txt", []byte("x")); err == nil {
		t.Errorf("Put with .. should be rejected")
	}
}

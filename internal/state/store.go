// Package store defines the persistent state interface and a filesystem-tree
// implementation. State is filesystem-only (no DB) per blueprint §8.
package store

import (
	"context"
	"time"
)

type Store interface {
	Put(ctx context.Context, path string, data []byte) error
	Get(ctx context.Context, path string) ([]byte, error)
	Append(ctx context.Context, path string, data []byte) error
	List(ctx context.Context, prefix string) ([]Object, error)
	Delete(ctx context.Context, path string) error
	Lock(ctx context.Context, path string) (Unlock func(), err error)
}

type Object struct {
	Path    string
	Size    int64
	ModTime time.Time
}

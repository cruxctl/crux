package store

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/cruxctl/crux/internal/statepath"
)

var ErrNotFound = errors.New("not found")

type FileStore struct {
	root  string
	locks sync.Map // path -> *sync.Mutex
}

func NewFileStore(root string) (*FileStore, error) {
	if root == "" {
		root = statepath.StateRoot()
	}
	if err := statepath.EnsureDir(root); err != nil {
		return nil, err
	}
	return &FileStore{root: root}, nil
}

func (s *FileStore) resolve(rel string) (string, error) {
	if strings.Contains(rel, "..") {
		return "", errors.New("path traversal not allowed")
	}
	clean := filepath.Clean("/" + rel) // forces leading /, normalizes
	if strings.Contains(clean, "..") {
		return "", errors.New("path traversal not allowed")
	}
	full := filepath.Join(s.root, strings.TrimPrefix(clean, "/"))
	abs, err := filepath.Abs(full)
	if err != nil {
		return "", err
	}
	rootAbs, _ := filepath.Abs(s.root)
	if !strings.HasPrefix(abs, rootAbs) {
		return "", errors.New("path escapes root")
	}
	return abs, nil
}

func (s *FileStore) Put(ctx context.Context, rel string, data []byte) error {
	full, err := s.resolve(rel)
	if err != nil {
		return err
	}
	if err := statepath.EnsureDir(filepath.Dir(full)); err != nil {
		return err
	}
	return os.WriteFile(full, data, 0o600)
}

func (s *FileStore) Get(ctx context.Context, rel string) ([]byte, error) {
	full, err := s.resolve(rel)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(full)
}

func (s *FileStore) Append(ctx context.Context, rel string, data []byte) error {
	full, err := s.resolve(rel)
	if err != nil {
		return err
	}
	if err := statepath.EnsureDir(filepath.Dir(full)); err != nil {
		return err
	}
	f, err := os.OpenFile(full, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}

func (s *FileStore) List(ctx context.Context, prefix string) ([]Object, error) {
	full, err := s.resolve(prefix)
	if err != nil {
		return nil, err
	}
	var out []Object
	err = filepath.WalkDir(full, func(p string, d fs.DirEntry, e error) error {
		if e != nil {
			if errors.Is(e, os.ErrNotExist) {
				return nil
			}
			return e
		}
		if d.IsDir() {
			return nil
		}
		info, _ := d.Info()
		rel, _ := filepath.Rel(s.root, p)
		out = append(out, Object{Path: rel, Size: info.Size(), ModTime: info.ModTime()})
		return nil
	})
	return out, err
}

func (s *FileStore) Delete(ctx context.Context, rel string) error {
	full, err := s.resolve(rel)
	if err != nil {
		return err
	}
	return os.Remove(full)
}

func (s *FileStore) Lock(ctx context.Context, rel string) (func(), error) {
	full, err := s.resolve(rel)
	if err != nil {
		return nil, err
	}
	if err := statepath.EnsureDir(full); err != nil {
		return nil, err
	}
	lockFile := filepath.Join(full, ".lock")
	f, err := os.OpenFile(lockFile, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		return nil, err
	}
	unlock := func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		f.Close()
	}
	return unlock, nil
}

// ensure io is referenced (build cleanliness if unused above)
var _ = io.EOF
var _ = time.Now

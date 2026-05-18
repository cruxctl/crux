package pty

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	creackpty "github.com/creack/pty"
)

type Factory struct{}

func NewFactory() *Factory {
	return &Factory{}
}

func (f *Factory) Create(ctx context.Context, spec PTYSpec) (*PTYTerminal, error) {
	cmd := exec.CommandContext(ctx, spec.Command, spec.Args...)
	if strings.TrimSpace(spec.WorkDir) != "" {
		cmd.Dir = spec.WorkDir
	}
	cmd.Env = commandEnv(os.Environ(), spec.Command, spec.Env)
	rows := uint16(spec.Rows)
	cols := uint16(spec.Cols)
	if rows == 0 {
		rows = 40
	}
	if cols == 0 {
		cols = 120
	}
	file, err := creackpty.StartWithSize(cmd, &creackpty.Winsize{Rows: rows, Cols: cols})
	if err != nil {
		return nil, err
	}
	return &PTYTerminal{Cmd: cmd, File: file, StartedAt: time.Now().UTC()}, nil
}

func commandEnv(base []string, command string, extra map[string]string) []string {
	envMap := make(map[string]string, len(base)+len(extra)+1)
	for _, entry := range base {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			envMap[key] = value
		}
	}
	for key, value := range extra {
		envMap[key] = value
	}
	if dir := filepath.Dir(command); dir != "." && dir != "" {
		path := envMap["PATH"]
		if path == "" {
			envMap["PATH"] = dir
		} else if !pathHasDir(path, dir) {
			envMap["PATH"] = dir + string(os.PathListSeparator) + path
		}
	}
	out := make([]string, 0, len(envMap))
	for key, value := range envMap {
		out = append(out, key+"="+value)
	}
	sort.Strings(out)
	return out
}

func pathHasDir(pathValue, dir string) bool {
	for _, part := range filepath.SplitList(pathValue) {
		if part == dir {
			return true
		}
	}
	return false
}

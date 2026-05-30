package envpath

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func Lookup(name string) (string, error) {
	path, err := exec.LookPath(name)
	if err == nil {
		return path, nil
	}
	if strings.ContainsAny(name, `/\`) {
		return "", err
	}
	for _, dir := range SearchDirs() {
		candidate := filepath.Join(dir, name)
		if isExecutable(candidate) {
			return candidate, nil
		}
	}
	return "", err
}

func SearchDirs() []string {
	dirs := filepath.SplitList(os.Getenv("PATH"))
	if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		dirs = append(dirs,
			filepath.Join(home, ".local", "bin"),
			filepath.Join(home, "bin"),
			filepath.Join(home, ".cargo", "bin"),
			filepath.Join(home, ".volta", "bin"),
			filepath.Join(home, ".bun", "bin"),
		)
		nodeDirs, _ := filepath.Glob(filepath.Join(home, ".nvm", "versions", "node", "*", "bin"))
		sort.Sort(sort.Reverse(sort.StringSlice(nodeDirs)))
		dirs = append(dirs, nodeDirs...)
	}
	if goBin := strings.TrimSpace(os.Getenv("GOBIN")); goBin != "" {
		dirs = append(dirs, goBin)
	}
	if goPath := strings.TrimSpace(os.Getenv("GOPATH")); goPath != "" {
		for _, path := range filepath.SplitList(goPath) {
			dirs = append(dirs, filepath.Join(path, "bin"))
		}
	}
	return existingUniqueDirs(dirs)
}

func CommandEnv(base []string, commandPath string, extra map[string]string) []string {
	values := make(map[string]string, len(base)+len(extra)+1)
	order := make([]string, 0, len(base)+len(extra)+1)
	for _, item := range base {
		key, value, ok := strings.Cut(item, "=")
		if !ok || key == "" {
			continue
		}
		if _, exists := values[key]; !exists {
			order = append(order, key)
		}
		values[key] = value
	}

	extraKeys := make([]string, 0, len(extra))
	for key := range extra {
		extraKeys = append(extraKeys, key)
	}
	sort.Strings(extraKeys)
	for _, key := range extraKeys {
		if key == "" {
			continue
		}
		if _, exists := values[key]; !exists {
			order = append(order, key)
		}
		values[key] = extra[key]
	}

	commandDir := filepath.Dir(commandPath)
	if commandDir != "." && commandDir != string(filepath.Separator) {
		if _, exists := values["PATH"]; !exists {
			order = append(order, "PATH")
		}
		values["PATH"] = prependPathDir(values["PATH"], commandDir)
	}

	out := make([]string, 0, len(order))
	for _, key := range order {
		out = append(out, key+"="+values[key])
	}
	return out
}

func existingUniqueDirs(dirs []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		clean := strings.TrimSpace(dir)
		if clean == "" {
			continue
		}
		if _, ok := seen[clean]; ok {
			continue
		}
		info, err := os.Stat(clean)
		if err != nil || !info.IsDir() {
			continue
		}
		seen[clean] = struct{}{}
		out = append(out, clean)
	}
	return out
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	return info.Mode().Perm()&0111 != 0
}

func prependPathDir(pathValue, dir string) string {
	if strings.TrimSpace(dir) == "" {
		return pathValue
	}
	for _, existing := range filepath.SplitList(pathValue) {
		if existing == dir {
			return pathValue
		}
	}
	if pathValue == "" {
		return dir
	}
	return dir + string(os.PathListSeparator) + pathValue
}

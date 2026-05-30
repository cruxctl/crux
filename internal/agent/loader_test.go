package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDir_ReadsAllYAMLs(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.yaml", "b.yaml", "ignore.txt"} {
		_ = os.WriteFile(filepath.Join(dir, name), []byte(minimalSpec(name)), 0o600)
	}
	specs, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 2 {
		t.Errorf("expected 2 specs, got %d", len(specs))
	}
}

func TestLoadDir_SkipsInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "good.yaml"), []byte(minimalSpec("good.yaml")), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "bad.yaml"), []byte("this is not: [yaml"), 0o600)
	specs, err := LoadDir(dir)
	if err == nil {
		t.Errorf("expected error from bad.yaml")
	}
	if len(specs) != 1 {
		t.Errorf("expected 1 good spec, got %d", len(specs))
	}
}

func minimalSpec(id string) string {
	return `apiVersion: crux.dev/v1alpha1
kind: CodingAgentSpec
metadata:
  id: ` + id + `
  name: Test
  provider: test
detection:
  binaries: [echo]
launch:
  interactive:
    command: echo
    args: []
    requires_pty: true
`
}

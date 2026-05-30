package statepath

import (
	"strings"
	"testing"
)

func TestID_DeterministicAnd32Hex(t *testing.T) {
	a := ID("agent:/usr/local/bin/claude:1.0.0")
	b := ID("agent:/usr/local/bin/claude:1.0.0")
	if a != b {
		t.Errorf("ID not deterministic: %q vs %q", a, b)
	}
	if len(a) != 32 {
		t.Errorf("len(ID) = %d, want 32", len(a))
	}
	for _, c := range a {
		if !strings.ContainsRune("0123456789abcdef", c) {
			t.Errorf("ID not lowercase hex: %q", a)
			break
		}
	}
}

func TestID_DifferentInputs_DifferentOutputs(t *testing.T) {
	a := ID("agent:claude:1")
	b := ID("agent:claude:2")
	if a == b {
		t.Errorf("ID collision on different inputs: %q", a)
	}
}

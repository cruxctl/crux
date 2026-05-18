package pty

import "testing"

func TestNormalizeProbeInputUsesCarriageReturns(t *testing.T) {
	got := normalizeProbeInput("/usage\n/quit\r\n")
	want := "/usage\r/quit\r"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

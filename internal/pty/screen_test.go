package pty

import (
	"strings"
	"testing"
)

func TestScreen_RendersBasicText(t *testing.T) {
	s := NewScreen(80, 24)
	s.Write([]byte("hello world\n"))
	rendered := s.Render()
	if !strings.Contains(rendered, "hello world") {
		t.Errorf("expected hello world in render: %q", rendered)
	}
}

func TestScreen_HandlesAnsiClearAndCursor(t *testing.T) {
	s := NewScreen(80, 24)
	s.Write([]byte("first line\n"))
	s.Write([]byte("\x1b[2J\x1b[Honly this remains"))
	rendered := s.Render()
	if strings.Contains(rendered, "first line") {
		t.Errorf("clear should have removed first line: %q", rendered)
	}
	if !strings.Contains(rendered, "only this remains") {
		t.Errorf("after clear, second write should be visible: %q", rendered)
	}
}

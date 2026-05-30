package pty

import (
	"strings"

	"github.com/hinshun/vt10x"
)

// Screen wraps vt10x to expose Write/Render for snapshot probes.
type Screen struct {
	term vt10x.Terminal
	rows int
	cols int
}

func NewScreen(cols, rows int) *Screen {
	t := vt10x.New(vt10x.WithSize(cols, rows))
	return &Screen{term: t, rows: rows, cols: cols}
}

func (s *Screen) Write(p []byte) (int, error) {
	return s.term.Write(p)
}

func (s *Screen) Render() string {
	var b strings.Builder
	for row := 0; row < s.rows; row++ {
		for col := 0; col < s.cols; col++ {
			ch := s.term.Cell(col, row).Char
			b.WriteRune(ch)
		}
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), " \n")
}

func (s *Screen) Resize(cols, rows int) {
	s.cols, s.rows = cols, rows
	s.term.Resize(cols, rows)
}

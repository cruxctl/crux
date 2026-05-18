package pty

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

type Normalizer struct{}

func NewNormalizer() *Normalizer {
	return &Normalizer{}
}

var (
	ansiCSIRegex    = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]`)
	ansiOSCRegex    = regexp.MustCompile(`\x1b\][^\x07]*(\x07|\x1b\\)`)
	ansiSingleRegex = regexp.MustCompile(`\x1b(?:[78]|[@-Z\\-_])`)
	spinnerRegex    = regexp.MustCompile(`^\s*[⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏●◐◓◑◒|/\\-]\s*(thinking|loading|processing|working|running)?[. ]*$`)
)

func (n *Normalizer) Normalize(ctx context.Context, input PTYRawOutput, spec NormalizeSpec) (*PTYNormalizedOutput, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ansiText := string(input.RawBytes)
	renderedScreen := renderANSIToScreen(ansiText)
	clean := ansiText
	hasANSI := ansiCSIRegex.MatchString(clean) || ansiOSCRegex.MatchString(clean) || ansiSingleRegex.MatchString(clean)
	if spec.StripANSI {
		clean = ansiOSCRegex.ReplaceAllString(clean, "")
		clean = ansiCSIRegex.ReplaceAllString(clean, "")
		clean = ansiSingleRegex.ReplaceAllString(clean, "")
	}
	if spec.StripControlChars {
		clean = stripControl(clean)
	}
	if spec.NormalizeBoxes {
		clean = normalizeBoxes(clean)
	}
	hadRedraws := strings.Contains(clean, "\r")
	clean = strings.ReplaceAll(clean, "\r\n", "\n")
	clean = strings.ReplaceAll(clean, "\r", "\n")
	lines := strings.Split(clean, "\n")
	if spec.RemoveSpinners || spec.RemoveStatusBars {
		lines = removeNoiseLines(lines)
	}
	if spec.CollapseRedraws {
		lines = collapseRedraws(lines)
	}
	clean = strings.Join(lines, "\n")
	if spec.NormalizeWhitespace {
		clean = normalizeWhitespace(clean)
	}
	clean = strings.TrimSpace(clean)
	finalScreenText := finalScreen(clean)
	if strings.TrimSpace(renderedScreen) != "" {
		screen := renderedScreen
		if spec.NormalizeBoxes {
			screen = normalizeBoxes(screen)
		}
		if spec.NormalizeWhitespace {
			screen = normalizeWhitespace(screen)
		}
		finalScreenText = finalScreen(screen)
	}
	if spec.KeepFinalScreenOnly && finalScreenText != "" {
		clean = finalScreenText
	}
	confidence := "high"
	if hasANSI || hadRedraws {
		confidence = "medium"
	}
	if clean == "" {
		confidence = "low"
	}
	return &PTYNormalizedOutput{
		RawBytes:    append([]byte{}, input.RawBytes...),
		ANSIText:    ansiText,
		CleanText:   clean,
		FinalScreen: finalScreenText,
		HasANSI:     hasANSI,
		HadRedraws:  hadRedraws,
		Confidence:  confidence,
	}, nil
}

func stripControl(value string) string {
	var out []rune
	for _, r := range value {
		switch r {
		case '\n', '\r', '\t':
			out = append(out, r)
		case '\b':
			if len(out) > 0 {
				out = out[:len(out)-1]
			}
		default:
			if r >= 32 || r == unicode.ReplacementChar {
				out = append(out, r)
			}
		}
	}
	return string(out)
}

func normalizeBoxes(value string) string {
	replacer := strings.NewReplacer(
		"┌", " ", "┐", " ", "└", " ", "┘", " ",
		"├", " ", "┤", " ", "┬", " ", "┴", " ", "┼", " ",
		"│", " ", "┃", " ", "─", " ", "━", " ",
		"╭", " ", "╮", " ", "╰", " ", "╯", " ",
		"├", " ", "┄", " ", "┈", " ",
	)
	return replacer.Replace(value)
}

func removeNoiseLines(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			out = append(out, line)
			continue
		}
		if spinnerRegex.MatchString(strings.ToLower(trimmed)) {
			continue
		}
		out = append(out, line)
	}
	return out
}

func collapseRedraws(lines []string) []string {
	out := make([]string, 0, len(lines))
	var prev string
	for _, line := range lines {
		normalized := strings.Join(strings.Fields(line), " ")
		if normalized == "" {
			if prev == "" {
				continue
			}
			prev = ""
			out = append(out, "")
			continue
		}
		if normalized == prev {
			continue
		}
		prev = normalized
		out = append(out, line)
	}
	return out
}

func normalizeWhitespace(value string) string {
	lines := strings.Split(value, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			if len(out) > 0 && out[len(out)-1] != "" {
				out = append(out, "")
			}
			continue
		}
		out = append(out, strings.Join(fields, " "))
	}
	return strings.Join(out, "\n")
}

func finalScreen(value string) string {
	lines := strings.Split(strings.TrimSpace(value), "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			filtered = append(filtered, line)
		}
	}
	if len(filtered) > 80 {
		filtered = filtered[len(filtered)-80:]
	}
	return strings.Join(filtered, "\n")
}

type screenBuffer struct {
	rows     [][]rune
	row      int
	col      int
	savedRow int
	savedCol int
}

func renderANSIToScreen(value string) string {
	buf := &screenBuffer{}
	runes := []rune(value)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == '\x1b' && i+1 < len(runes) {
			next := runes[i+1]
			switch next {
			case '[':
				params, final, nextIndex := readCSI(runes, i+2)
				if final != 0 {
					buf.applyCSI(params, final)
					i = nextIndex
					continue
				}
			case ']':
				i = skipOSC(runes, i+2)
				continue
			default:
				i++
				continue
			}
		}
		buf.writeRune(r)
	}
	return buf.String()
}

func readCSI(runes []rune, start int) (string, rune, int) {
	var params strings.Builder
	for i := start; i < len(runes); i++ {
		r := runes[i]
		if r >= '@' && r <= '~' {
			return params.String(), r, i
		}
		params.WriteRune(r)
	}
	return params.String(), 0, len(runes) - 1
}

func skipOSC(runes []rune, start int) int {
	for i := start; i < len(runes); i++ {
		if runes[i] == '\a' {
			return i
		}
		if runes[i] == '\x1b' && i+1 < len(runes) && runes[i+1] == '\\' {
			return i + 1
		}
	}
	return len(runes) - 1
}

func (s *screenBuffer) writeRune(r rune) {
	switch r {
	case '\n':
		s.row++
		s.col = 0
	case '\r':
		s.col = 0
	case '\b':
		if s.col > 0 {
			s.col--
		}
	case '\t':
		next := ((s.col / 4) + 1) * 4
		for s.col < next {
			s.put(' ')
		}
	default:
		if r >= 32 || r == unicode.ReplacementChar {
			s.put(r)
		}
	}
}

func (s *screenBuffer) put(r rune) {
	if s.row < 0 {
		s.row = 0
	}
	if s.col < 0 {
		s.col = 0
	}
	if s.row > 200 || s.col > 240 {
		return
	}
	s.ensure(s.row, s.col)
	s.rows[s.row][s.col] = r
	s.col++
}

func (s *screenBuffer) ensure(row int, col int) {
	for len(s.rows) <= row {
		s.rows = append(s.rows, nil)
	}
	for len(s.rows[row]) <= col {
		s.rows[row] = append(s.rows[row], ' ')
	}
}

func (s *screenBuffer) applyCSI(params string, final rune) {
	values := csiInts(params)
	n := func(fallback int) int {
		if len(values) == 0 || values[0] <= 0 {
			return fallback
		}
		return values[0]
	}
	switch final {
	case 'A':
		s.row -= n(1)
		if s.row < 0 {
			s.row = 0
		}
	case 'B':
		s.row += n(1)
	case 'C':
		s.col += n(1)
	case 'D':
		s.col -= n(1)
		if s.col < 0 {
			s.col = 0
		}
	case 'G':
		s.col = n(1) - 1
		if s.col < 0 {
			s.col = 0
		}
	case 'H', 'f':
		row, col := 1, 1
		if len(values) > 0 && values[0] > 0 {
			row = values[0]
		}
		if len(values) > 1 && values[1] > 0 {
			col = values[1]
		}
		s.row = row - 1
		s.col = col - 1
	case 'J':
		if len(values) == 0 || values[0] == 0 || values[0] == 2 || values[0] == 3 {
			s.rows = nil
			s.row = 0
			s.col = 0
		}
	case 'K':
		s.clearLineFromCursor()
	case 's':
		s.savedRow = s.row
		s.savedCol = s.col
	case 'u':
		s.row = s.savedRow
		s.col = s.savedCol
	case 'm', 'h', 'l':
		return
	}
}

func (s *screenBuffer) clearLineFromCursor() {
	if s.row < 0 || s.row >= len(s.rows) {
		return
	}
	for i := s.col; i < len(s.rows[s.row]); i++ {
		s.rows[s.row][i] = ' '
	}
}

func (s *screenBuffer) String() string {
	lines := make([]string, 0, len(s.rows))
	for _, row := range s.rows {
		lines = append(lines, strings.TrimRight(string(row), " "))
	}
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}

func csiInts(params string) []int {
	params = strings.TrimLeft(params, "?")
	if params == "" {
		return nil
	}
	parts := strings.Split(params, ";")
	out := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimLeft(part, "?")
		if part == "" {
			out = append(out, 0)
			continue
		}
		value, err := strconv.Atoi(part)
		if err != nil {
			out = append(out, 0)
			continue
		}
		out = append(out, value)
	}
	return out
}

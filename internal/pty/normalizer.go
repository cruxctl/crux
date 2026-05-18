package pty

import (
	"context"
	"regexp"
	"strings"
	"unicode"
)

type Normalizer struct{}

func NewNormalizer() *Normalizer {
	return &Normalizer{}
}

var (
	ansiCSIRegex = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]`)
	ansiOSCRegex = regexp.MustCompile(`\x1b\][^\x07]*(\x07|\x1b\\)`)
	spinnerRegex = regexp.MustCompile(`^\s*[⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏●◐◓◑◒|/\\-]\s*(thinking|loading|processing|working|running)?[. ]*$`)
)

func (n *Normalizer) Normalize(ctx context.Context, input PTYRawOutput, spec NormalizeSpec) (*PTYNormalizedOutput, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ansiText := string(input.RawBytes)
	clean := ansiText
	hasANSI := ansiCSIRegex.MatchString(clean) || ansiOSCRegex.MatchString(clean)
	if spec.StripANSI {
		clean = ansiOSCRegex.ReplaceAllString(clean, "")
		clean = ansiCSIRegex.ReplaceAllString(clean, "")
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
	finalScreen := finalScreen(clean)
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
		FinalScreen: finalScreen,
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

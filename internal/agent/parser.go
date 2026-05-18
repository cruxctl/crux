package agent

import (
	"regexp"
	"strings"
)

func ParseProbeText(probe CommandProbe, text string) map[string]string {
	if strings.TrimSpace(probe.Parser.Type) == "" || strings.TrimSpace(text) == "" {
		return nil
	}
	switch probe.Parser.Type {
	case "regex":
		return parseRegex(probe.Parser.Patterns, text)
	default:
		return nil
	}
}

func parseRegex(patterns map[string]string, text string) map[string]string {
	out := map[string]string{}
	for name, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}
		match := re.FindStringSubmatch(text)
		if len(match) == 0 {
			continue
		}
		if len(match) > 1 {
			names := re.SubexpNames()
			for i := 1; i < len(match); i++ {
				if i < len(names) && names[i] != "" {
					out[names[i]] = strings.TrimSpace(match[i])
					continue
				}
				out[name] = strings.TrimSpace(match[i])
				break
			}
			continue
		}
		out[name] = strings.TrimSpace(match[0])
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

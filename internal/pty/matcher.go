package pty

import (
	"regexp"
	"strings"
	"time"
)

func Match(spec MatcherSpec, screen string) bool {
	strategy := strings.TrimSpace(spec.Strategy)
	if strategy == "" {
		return true
	}
	switch strategy {
	case "screen_contains_any":
		for _, pattern := range spec.Patterns {
			if pattern != "" && strings.Contains(screen, pattern) {
				return true
			}
		}
		return false
	case "screen_contains":
		pattern := firstMatcherPattern(spec)
		return pattern != "" && strings.Contains(screen, pattern)
	case "regex":
		pattern := firstMatcherPattern(spec)
		if pattern == "" {
			return false
		}
		re, err := regexp.Compile(pattern)
		return err == nil && re.MatchString(screen)
	case "screen_stable", "silence_for_duration", "timeout", "prompt_returns":
		return false
	default:
		return false
	}
}

func StableDuration(spec MatcherSpec, fallback time.Duration) time.Duration {
	if spec.StableForMS > 0 {
		return time.Duration(spec.StableForMS) * time.Millisecond
	}
	return fallback
}

func firstMatcherPattern(spec MatcherSpec) string {
	if spec.Pattern != "" {
		return spec.Pattern
	}
	if len(spec.Patterns) > 0 {
		return spec.Patterns[0]
	}
	return ""
}

package agentconfigs

import "embed"

// FS contains the built-in coding-agent specs used by crux discover and
// agent-scoped PTY probes.
//
//go:embed *.yaml
var FS embed.FS

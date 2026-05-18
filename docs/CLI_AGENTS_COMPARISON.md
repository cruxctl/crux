# CLI Agents Comparison

Crux treats coding-agent CLIs as PTY-backed tools. Built-in YAML specs define detection, launch commands, readiness matchers, normalization settings, and probe inputs.

| Agent ID | Binary | Provider | Initial support |
|---|---|---|---|
| `claude-code` | `claude` | Anthropic | discover, describe, usage, conversations, exec |
| `gemini-cli` | `gemini` | Google | discover, describe, usage, conversations, exec |
| `aider` | `aider` | Aider | discover, describe, usage, conversations, exec |
| `codex` | `codex` | OpenAI | discover, describe, usage, conversations, exec |
| `opencode` | `opencode` | OpenCode | discover, describe, usage, conversations, exec |
| `kimi-cli` | `kimi` | Moonshot AI | discover, describe, usage, conversations, exec |
| `cline` | `cline` | Cline | config/file discovery first |

The user-facing command shape is:

```bash
crux discover
crux ps
crux <agent-name> describe
crux <agent-name> usage
crux <agent-name> conversations ls
crux <agent-name> exec
```

Raw PTY streams are written for replay/debugging. Parsers consume normalized `clean_text` or `final_screen` output only.

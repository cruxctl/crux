# CLI Agents Comparison

Crux treats managed CLI agents as command-backed tools. The MVP discovers Claude, Codex, Gemini, and Kimi, then records local execution metadata for each run.

| Agent | Headless command | Auth status surface | MCP support | Crux usage coverage |
|---|---|---|---|---|
| Claude | `claude -p "{prompt}"` | `claude auth status`, `/usage` prompt output | Yes | local runs, success/failure, duration, output bytes, exit codes, event counts |
| Codex | `codex exec "{prompt}"` | `codex login status` | Yes | local runs, success/failure, duration, output bytes, exit codes, event counts |
| Gemini | `gemini -p "{prompt}"` | environment/API-key driven | Yes | local runs, success/failure, duration, output bytes, exit codes, event counts |
| Kimi | `kimi -p "{prompt}"` | `kimi info` | ACP-oriented | local runs, success/failure, duration, output bytes, exit codes, event counts |

`crux agent <name> usage` reports metrics that Crux can prove from daemon state. Provider-side quota, token, billing, and subscription details remain external unless a CLI prints them into captured stdout/stderr or a future adapter normalizes them.


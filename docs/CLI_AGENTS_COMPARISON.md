# CLI Agents Comparison

Crux treats managed CLI agents as TTY-backed tools. The MVP discovers Claude, Codex, Gemini, and Kimi, then records transcript-backed execution metadata for every run, resume, history share, and explicit exec session.

| Agent | TTY exec command | Resume command | Prompt insertion | Crux usage/cost/session coverage |
|---|---|---|---|---|
| Claude | `claude` | `claude --resume <id>`, `claude --continue` | TTY input from `crux agent claude exec "<text>"` or `crux run claude "<text>"` | imported transcripts, success/failure, duration, output bytes, event counts, and any usage/cost text printed by Claude |
| Codex | `codex` | `codex resume --all <id>`, `codex resume --all --last` | TTY input from `crux agent codex exec "<text>"` or `crux run codex "<text>"` | imported transcripts, success/failure, duration, output bytes, event counts, and any usage/cost text printed by Codex |
| Gemini | `gemini --skip-trust` | `gemini --skip-trust --resume <id>` | TTY input from `crux agent gemini exec "<text>"` or `crux run gemini "<text>"` | imported transcripts, success/failure, duration, output bytes, event counts, and any usage/cost text printed by Gemini |
| Kimi | `kimi` | `kimi --session <id>`, `kimi --continue` | TTY input from `crux agent kimi exec "<text>"` or `crux run kimi "<text>"` | imported transcripts, success/failure, duration, output bytes, event counts, and any usage/cost text printed by Kimi |

`cruxd` no longer reads provider-private local session stores or stable noninteractive usage commands. `crux agent <name> sessions`, `usage`, `cost`, and `history` report what Crux can prove from imported TTY transcripts and Crux-owned execution records.

`crux run` is now a thin wrapper over `crux agent <name> exec "<prompt>"`. The daemon returns the provider-specific TUI command through `/v1/agents/{name}/exec/plan`; the CLI opens the local terminal with `expect`, `script`, or direct execution, captures a transcript, and posts it to `/v1/agents/{name}/exec/record`.

Scripted probes can drive the TUI:

```bash
crux agent codex exec --dry-run --resume last -- --no-alt-screen
crux agent claude exec "insert this into the active session"
crux agent gemini exec --send "/help" --send "/quit" --expect "help" --timeout 30
```

History is immutable. "Editing" and sharing history means replaying a prior execution into a new run with an optional edited prompt:

```bash
crux run codex --from exec_abc123 --prompt "improve the previous answer"
crux agent gemini history share exec_abc123 claude --prompt "summarize for release notes"
```

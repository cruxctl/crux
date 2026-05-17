# CLI Agents Comparison

Crux treats managed CLI agents as command-backed tools. The MVP discovers Claude, Codex, Gemini, and Kimi, then records local execution metadata for each run.

| Agent | Headless command | Auth/status surface | Sessions | Resume command | TUI exec command | Crux usage/cost coverage |
|---|---|---|---|---|---|---|
| Claude | `claude -p "{prompt}"` | `claude auth status`, `claude -p /usage` | local `~/.claude/projects` sessions plus Crux history | `claude --resume <id> -p "{prompt}"`, `claude --continue -p "{prompt}"` | `claude`, `claude --resume <id>`, `claude --continue` | local runs, interactive transcripts, success/failure, realtime running state, duration, output bytes, exit codes, event counts, last captured output, subscription text |
| Codex | `codex exec --skip-git-repo-check "{prompt}"` | `codex login status` | local `$CODEX_HOME/sessions` or `~/.codex/sessions` JSONL sessions plus Crux history | `codex exec resume --all <id> --skip-git-repo-check "{prompt}"`, `codex exec resume --all --last --skip-git-repo-check "{prompt}"` | `codex`, `codex resume --all <id>`, `codex resume --all --last` | local runs, interactive transcripts, success/failure, realtime running state, duration, output bytes, exit codes, event counts, last captured output |
| Gemini | `gemini --skip-trust -p "{prompt}"` | environment/API-key driven | `gemini --list-sessions` plus Crux history | `gemini --skip-trust --resume <id> -p "{prompt}"` | `gemini --skip-trust`, `gemini --skip-trust --resume <id>` | local runs, interactive transcripts, success/failure, realtime running state, duration, output bytes, exit codes, event counts, last captured output, provider session evidence |
| Kimi | `kimi --quiet --prompt "{prompt}"` | `kimi info`, membership errors in run output | local `$KIMI_SHARE_DIR/sessions` or `~/.kimi/sessions` sessions plus Crux history; `kimi export` supports explicit session export | `kimi --quiet --session <id> --prompt "{prompt}"`, `kimi --quiet --continue --prompt "{prompt}"` | `kimi`, `kimi --session <id>`, `kimi --continue` | local runs, interactive transcripts, success/failure, realtime running state, duration, output bytes, exit codes, event counts, last captured output, membership evidence |

`crux run` sends the operator's current working directory to `cruxd`, and the daemon uses it unless an agent has a fixed working directory. External sessions created directly in Claude Code, Codex, Gemini CLI, or Kimi CLI are incorporated when their CLI options or local session stores expose resumable IDs. `crux agent <name> sessions` reports the provider session ID, source, original working directory when known, and whether Crux can resume it. `crux agent <name> usage`, `crux agent <name> cost`, and `crux agent <name> history` report metrics that Crux can prove from daemon state plus provider-side evidence from stable noninteractive commands. Provider-side quota, token, billing, and subscription details remain external unless a CLI prints them into captured stdout/stderr or a future adapter normalizes them.

`crux agent <name> exec` is the interactive complement to `crux run`. The daemon returns the provider-specific TUI command through `/v1/agents/{name}/exec/plan`; the CLI opens the local terminal with `expect`, `script`, or direct execution, captures a transcript, and posts it to `/v1/agents/{name}/exec/record`. That import makes interactive sessions visible to `ps`, `trace`, usage, cost, sessions, and history without pretending Crux owns provider-private session databases.

Scripted probes can drive the TUI:

```bash
crux agent codex exec --dry-run --resume last -- --no-alt-screen
crux agent gemini exec --send "/help" --send "/quit" --expect "help" --timeout 30
```

History is immutable. "Editing" and sharing history means replaying a prior execution into a new run with an optional edited prompt:

```bash
crux run codex --from exec_abc123 --prompt "improve the previous answer"
crux agent gemini history share exec_abc123 claude --prompt "summarize for release notes"
```

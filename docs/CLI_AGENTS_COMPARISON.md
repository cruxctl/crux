# CLI Agents Comparison

Crux treats managed CLI agents as command-backed tools. The MVP discovers Claude, Codex, Gemini, and Kimi, then records local execution metadata for each run.

| Agent | Headless command | Auth/status surface | Sessions | Resume command | Crux usage/cost coverage |
|---|---|---|---|---|---|
| Claude | `claude -p "{prompt}"` | `claude auth status`, `claude -p /usage` | Crux history; no stable noninteractive provider list | `claude --resume <id> -p "{prompt}"`, `claude --continue -p "{prompt}"` | local runs, success/failure, realtime running state, duration, output bytes, exit codes, event counts, last captured output, subscription text |
| Codex | `codex exec --skip-git-repo-check "{prompt}"` | `codex login status` | Crux history; no stable noninteractive provider list | `codex exec resume <id> --skip-git-repo-check "{prompt}"`, `codex exec resume --last --skip-git-repo-check "{prompt}"` | local runs, success/failure, realtime running state, duration, output bytes, exit codes, event counts, last captured output |
| Gemini | `gemini --skip-trust -p "{prompt}"` | environment/API-key driven | `gemini --list-sessions` plus Crux history | `gemini --skip-trust --resume <id> -p "{prompt}"` | local runs, success/failure, realtime running state, duration, output bytes, exit codes, event counts, last captured output, provider session evidence |
| Kimi | `kimi --quiet --prompt "{prompt}"` | `kimi info`, membership errors in run output | Crux history; `kimi export` supports explicit session export but no stable list | `kimi --quiet --session <id> --prompt "{prompt}"`, `kimi --quiet --continue --prompt "{prompt}"` | local runs, success/failure, realtime running state, duration, output bytes, exit codes, event counts, last captured output, membership evidence |

`crux run` sends the operator's current working directory to `cruxd`, and the daemon uses it unless an agent has a fixed working directory. `crux agent <name> usage`, `crux agent <name> cost`, `crux agent <name> sessions`, and `crux agent <name> history` report metrics that Crux can prove from daemon state plus provider-side evidence from stable noninteractive commands. Provider-side quota, token, billing, and subscription details remain external unless a CLI prints them into captured stdout/stderr or a future adapter normalizes them.

History is immutable. "Editing" and sharing history means replaying a prior execution into a new run with an optional edited prompt:

```bash
crux run codex --from exec_abc123 --prompt "improve the previous answer"
crux agent gemini history share exec_abc123 claude --prompt "summarize for release notes"
```

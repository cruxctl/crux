# Crux Console

`crux-console` will be the web console for Crux Control.
It is the operator-facing UI for fleet inventory, executions, traces, MCP registry, policies, approvals, cost, and AgBOM review.

Crux Control is a vendor-neutral AI agent control plane.
The console visualizes and operates the control plane; it does not run agents itself.

## Install

Linux / macOS:

```bash
curl -fsSL https://get.cruxcontrol.dev/install.sh | sh
```

Windows (PowerShell):

```powershell
iwr https://get.cruxcontrol.dev/install.ps1 -useb | iex
```

Optional: install the web Console alongside the daemon:

```bash
curl -fsSL https://get.cruxcontrol.dev/install.sh | sh -s -- --with-console
```

See the [full install guide](https://docs.crux.dev/install) for offline installs, checksum verification, and uninstall.

## Planned scope

- Fleet view for managed and custom agents across hosts and teams.
- Execution history and live execution drill-down.
- Trace viewer for model, tool, MCP, handoff, approval, and cost events.
- MCP registry, trust, pinning, scanning, and quarantine views.
- Policy editor, version history, and enforcement log.
- Approval queue and audit history.
- Cost views by agent, team, provider, model, and time range.
- AgBOM list and version diff views.

## Repository layout

```text
src/app/          future application routes
src/features/     domain feature modules
src/shared/       shared UI and API helpers
public/           static assets
docs/views/       planned view specs
```

## Current status

This repository is scaffold-only.

## License

Apache-2.0.

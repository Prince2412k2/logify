# logify

Terminal client for the `log-gateway`. Browse projects, stream live container logs, pick a theme.

This binary will grow into the unified `logify` toolchain (TUI + one-shot CLI + AI-friendly subcommands). For now it ships the TUI.

## Install

**macOS / Linux** (one-liner, downloads the right binary for your OS/arch):

```bash
curl -sSL https://raw.githubusercontent.com/Prince2412k2/logify/main/install.sh | bash
```

**Windows** (PowerShell, Windows Terminal recommended):

```powershell
iwr -useb https://raw.githubusercontent.com/Prince2412k2/logify/main/install.ps1 | iex
```

Or download `install.bat` from the repo and double-click — that bootstraps the same flow without typing a PowerShell pipe.

To track the rolling build of `main` (instead of the newest stable tag):

```bash
LOGIFY_VERSION=nightly curl -sSL .../install.sh | bash
```

```powershell
$env:LOGIFY_VERSION = "nightly"; iwr -useb .../install.ps1 | iex
```

Either installer puts `logify` on your PATH (or tells you how to add it).

### From source

If Go is installed:

```bash
go install github.com/Prince2412k2/logify/cmd/logify@latest
```

Or via the installer:

```bash
INSTALL_MODE=source curl -sSL .../install.sh | bash
```

### Build locally

```bash
cd logify
make build         # ./logify
make install       # → ~/.local/bin/logify
make release       # all platforms → dist/
```

### Uninstall

```bash
bash uninstall.sh                                                       # Linux / macOS
iwr -useb https://raw.githubusercontent.com/Prince2412k2/logify/main/uninstall.ps1 | iex   # Windows
```

## Run

First launch with no config triggers an in-TUI form for gateway URL + API key. Otherwise:

```
./logify --url http://localhost:8089 --token <api-key>
./logify --mock                  # offline demo with synthetic projects + log stream
./logify --theme dracula         # see --themes for the full list
./logify --themes                # list available palettes
```

Env vars also work: `LOGIFY_URL`, `LOGIFY_TOKEN`.

Config persists to `$XDG_CONFIG_HOME/logify/config.toml` (defaults to `~/.config/logify/config.toml`).

## Keys

| Key | Effect |
|---|---|
| `↑` `↓` / `k` `j` | move in focused pane |
| `enter` | open service (from nav) |
| `tab` / `shift+tab` | switch nav ↔ logs |
| `1`–`5` | jump to tab (also toggles level filters when `f` strip is open) |
| `/` | search (logs) / filter (nav) |
| `n` `N` | next / prev match (logs) |
| `f` | toggle level filter strip |
| `space` | pause / resume tail |
| `g` `G` | jump top / bottom |
| `c` | clear buffer |
| `r` | reconnect / retry |
| `t` | open theme picker |
| `?` | help overlay |
| `q` / `ctrl-c` | quit |

## Themes

Ten built-in palettes (all in `internal/theme/theme.go`):

```
amber  tappin  solarizedDark  solarizedLight  dracula
gruvbox  tokyo  nord  contrast  paper
```

Press `t` inside the TUI to cycle, or set `--theme <id>` / `theme = "<id>"` in config.

## Backend requirements

`logify` speaks to a `log-gateway` instance:

- `GET  /api/projects`        — service tree (filtered to allowed containers)
- `GET  /api/containers`      — flat fallback when no Coolify DB is wired
- `WS   /api/logs/{name}`     — live log stream (first-message auth)

Anything beyond that — build logs, deployment status, config, env — is stubbed in the UI with "Coming soon" and the planned endpoint. Those tabs activate as the backend grows.

## Layout

```
internal/api      — REST + WS client
internal/config   — TOML load/save (~/.config/logify/config.toml)
internal/theme    — 10 colour palettes
internal/tui      — Bubble Tea model, screens, grid renderer
internal/mock     — offline data + synthetic log stream
cmd/logify        — entrypoint
```

## Status

v1 ships:

- Project / service tree (sourced from `/api/projects` or fallback)
- Live log streaming over WebSocket with first-message auth
- Search, level filter, pause/resume, paused queue indicator
- 10 themes + in-TUI theme picker
- First-run config form, lifecycle error screens (unreachable / auth / empty)
- Stub tabs for Build, Config, Env, Deploys

Not yet:

- Build logs (needs gateway endpoint)
- Deployment status badges (needs Coolify API client in gateway)
- Config / env / deploys tab bodies (same)
- One-shot CLI subcommands (planned: `logify logs`, `logify status`, `logify diagnose`)
- TUI scrollback (logs anchor to bottom in v1)

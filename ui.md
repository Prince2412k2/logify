# logify TUI — UI Specification

The TUI is a single full-screen Bubble Tea program. Same conceptual layout as the web `/logs/{container}` view, restated for the terminal: services on the left, logs on the right, header up top, help strip on the bottom.

Visual language matches the web theme so users moving between the two feel at home: amber accents on a near-black background, mono font for data, sans for chrome (terminal font does both — lipgloss just picks weights/colors).

## Color palette

| Token | Hex | Use |
|---|---|---|
| `bg` | `#0a0a0a` | terminal default (already dark) |
| `text` | `#ededed` | primary text |
| `muted` | `#a3a3a3` | secondary text, labels |
| `dim` | `#6b6b6b` | timestamps, idle nav items |
| `border` | `#3a3a3a` | pane borders |
| `amber` | `#f59e0b` | selection, active tab, focus ring, matches |
| `green` | `#22c55e` | running / connected |
| `rose` | `#f43f5e` | error / failed |
| `sky` | `#38bdf8` | info |
| `indigo` | `#818cf8` | application badge |
| `emerald` | `#34d399` | service badge |

Status dots reuse: green = running, amber = restarting/deploying (later), rose = exited/failed, dim = unknown.

---

## Overall layout

```
┌─ logify ─────────────────────────────────────── ● connected · prod/api ─┐
│                                                                          │
│ ┌──────────────────────────┬─────────────────────────────────────────┐  │
│ │ FILTER  ____________     │ [ Logs ]  Build·  Config·  Env·  Deploys│  │
│ │                          │ ─────────────────────────────────────── │  │
│ │ ▾ prod                   │ 13:42:01  INFO   starting server         │  │
│ │   ● api          🟢       │ 13:42:01  INFO   listening :8080         │  │
│ │ ▸ web            🟢       │ 13:42:14  WARN   slow query 1.2s         │  │
│ │   ● postgres     🟢       │ 13:42:15  ERROR  connection refused      │  │
│ │ ▾ staging                │ 13:42:15  ERROR    at db.go:88            │  │
│ │   ● api          🔴       │ 13:42:16  INFO   reconnecting…           │  │
│ │   ● web          ⚫       │ ▌                                         │  │
│ └──────────────────────────┴─────────────────────────────────────────┘  │
│                                                                          │
│  ↑↓ nav   enter open   tab switch pane   / search   space pause   ? help │
└──────────────────────────────────────────────────────────────────────────┘
```

- **Width split**: nav pane fixed at `min(36, 30% of width)` cols; log pane fills the rest. Below 80 cols the nav collapses to a top dropdown (later — v1 just requires ≥80 cols).
- **Borders**: 1-line rounded (`lipgloss.RoundedBorder`) in `border` color. The focused pane's border switches to `amber`.
- **Header** (top row): `logify` brand in amber, then current selection path (`prod/api`), then connection state on the right with a colored dot.
- **Help strip** (bottom row): rendered by `help.Model`. Compact by default; full-help overlay on `?`.

---

## Nav pane (left)

Built from a flattened service list with project rows acting as headers (rendered by a custom `list.ItemDelegate`).

```
FILTER  ____________            ← textinput; lights amber when typing
                                 (filter is `list.Model`'s built-in /)

▾ prod                           ← project header (muted, bold)
  ● api          🟢              ← service row: name (text) + dot (status)
  ● web          🟢
  ● postgres     🟢
▾ staging                        ← collapsible later; v1 always expanded
  ● api          🔴
  ● web          ⚫
```

- **Selection**: full-width amber background, black text. Selected item is the source-of-truth for the right pane.
- **Idle items**: name in `text`, status dot at right edge.
- **Project headers**: not selectable; `▾`/`▸` glyphs are cosmetic in v1.
- **Empty state**: if `/api/projects` returns nothing and `/api/containers` is empty too:

  ```
  No services accessible.
  Check that your API key is allowed for at least one container.
  ```

- **No Coolify DB**: if `/api/projects` returns `[]` but `/api/containers` has items, flat-list mode:

  ```
  ▾ containers
    ● log-gateway-1   🟢
    ● redis           🟢
  ```

---

## Right pane — Tabs

Top of the right pane is a lipgloss tab strip. Only **Logs** is functional in v1; the others render a placeholder body explaining what's blocked.

```
[ Logs ]  Build·  Config·  Env·  Deploys·
───────────────────────────────────────────
```

- Active tab: amber underline + bright text. Inactive: muted text, `·` suffix meaning "stub".
- Tab cycle: `tab` / `shift+tab`. Numeric jump: `1` Logs, `2` Build, `3` Config, `4` Env, `5` Deploys.

### Logs tab (v1, functional)

```
13:42:01  INFO   starting server
13:42:01  INFO   listening :8080
13:42:14  WARN   slow query 1.2s
13:42:15  ERROR  connection refused
13:42:15  ERROR    at db.go:88
13:42:16  INFO   reconnecting…
▌
```

- **Layout per line**: `<dim timestamp>  <level badge>  <line>`. Timestamp `HH:MM:SS` (extracted client-side if the line carries one; otherwise the time the line arrived).
- **Level colors**: ERROR rose, WARN amber, INFO sky, DEBUG dim. Plain text default. Background tint on level rows mirrors web (very subtle, 4% alpha).
- **Cursor row**: bottom-anchored `▌` shows "live tail" when following.
- **Wrap**: long lines wrap with a 4-space hanging indent so the level column stays visually aligned.
- **Scrollback**: viewport remembers the full session buffer (capped at e.g. 10k lines, then drops oldest). Manual scroll exits follow mode; `g`/`G` jump start/end; pressing `G` re-engages follow.

#### Search overlay

Press `/` inside the Logs tab:

```
─── search ─────────────────────────────────────────────
/ error db                                              ← textinput, amber border
─── 3 matches · n next  N prev  esc cancel ─────────────
```

- Matches highlight inline with amber background + black foreground (same as web `.match`).
- `n` / `N` jumps to next/previous match (auto-scrolls).
- Esc closes; query persists in nav-pane filter-history (future).

#### Level filter strip (above tab body, hidden until toggled)

Press `f` to reveal:

```
[err] [warn] [info] [dbg]      ← all on by default; toggle with 1/2/3/4 while strip is visible
```

Off levels render with strike-through and the corresponding lines hide from the viewport (filtering rebuilds from the in-memory buffer; no upstream re-fetch).

#### Pause indicator

Press `space` to toggle tail-follow:

```
                                          [ PAUSED · 12 new lines ]
```

A pill renders at the bottom-right corner of the log pane. Resume drains the queued lines.

### Build / Config / Env / Deploys tabs (v1, stubs)

Each renders a centered placeholder body:

```
                          ┌─ Build Logs ─┐
                          │              │
                          │   Coming soon │
                          │              │
                          └──────────────┘

  This view needs the gateway to expose a build-log endpoint
  (planned: GET /api/deployments/{uuid}/build-log). Not yet
  implemented in the backend — runtime logs are available on
  the Logs tab.

  See: ../HANDOFF.md → "Known Issues / Open Threads"
```

The exact text varies per tab but the pattern is identical. The point is to **show where the feature will live**, not hide it.

---

## Header bar

```
 logify ▸ prod/api                              ● connected · tail 100 
```

- Left: `logify` (amber `▸` prefix matching web `.nav-logo`), then breadcrumb of current selection (project/service). If nothing selected: just `logify`.
- Right: connection status — `● connected` (green) / `● connecting…` (amber, spinner) / `● disconnected` (rose). Followed by tail size if logs tab is active.
- 1-line tall, no border, has a 1px bottom rule in `border` color.

---

## Help strip (bottom) and full-help overlay

Compact bottom strip (always visible, generated by `help.Model.ShortHelp`):

```
↑↓ nav · enter open · tab pane · / search · space pause · ? help · q quit
```

Press `?` to open the full overlay (modal, centered):

```
┌─ keybindings ──────────────────────────────────────┐
│                                                    │
│  Navigation                                        │
│    ↑ / ↓ / k / j     move in current pane          │
│    enter             open service (focus logs)     │
│    tab / shift-tab   switch nav ↔ logs             │
│    1..5              jump to tab                   │
│                                                    │
│  Logs                                              │
│    /                 search                        │
│    n / N             next / previous match         │
│    f                 toggle level filter strip     │
│    space             pause / resume tail           │
│    g / G             jump top / bottom             │
│    c                 clear buffer                  │
│                                                    │
│  Global                                            │
│    ?                 toggle this overlay           │
│    r                 reconnect current stream      │
│    q / ctrl-c        quit                          │
│                                                    │
│                              esc to close          │
└────────────────────────────────────────────────────┘
```

---

## Lifecycle screens

### 1. First-run / no config

If neither `--token` flag nor `LOGIFY_TOKEN` env var is set, and no config file exists:

```
┌─ logify · first run ────────────────────────────────┐
│                                                     │
│   Gateway URL  http://localhost:8089            ▌   │
│   API Key      _______________________              │
│                                                     │
│   [ Save & Continue ]   [ Quit ]                    │
│                                                     │
│   Saved to ~/.config/logify/config.toml             │
│                                                     │
└─────────────────────────────────────────────────────┘
```

Two textinputs; enter on the button saves and proceeds. Quitting writes nothing.

### 2. Connecting

Brief state before the first `/api/projects` response:

```
              ● connecting to http://localhost:8089 …
```

Centered. Spinner. Auto-transitions once data arrives.

### 3. Error states

Gateway unreachable:

```
              ✕ Cannot reach gateway

              GET http://localhost:8089/api/projects
              dial tcp: connection refused

              r retry · q quit
```

Auth failure (`401`):

```
              ✕ Authentication failed

              Your API key was rejected by the gateway.
              Edit ~/.config/logify/config.toml and try again.

              q quit
```

Empty access:

```
              No services accessible.
              Ask an admin to grant your API key access
              to at least one container.

              r refresh · q quit
```

Log WS dropped mid-session: the log pane keeps existing lines and shows an inline banner at the bottom:

```
   ▲ stream disconnected · auto-retry in 3s · r retry now
```

Reconnect attempts respect exponential backoff (3s → 6s → 12s → cap 30s).

---

## Keybinds (canonical)

| Key | Action |
|---|---|
| `↑` `↓` / `k` `j` | move within focused pane |
| `enter` | open service (from nav) / no-op (from logs) |
| `tab` / `shift+tab` | switch focus nav ↔ logs |
| `1`–`5` | jump to tab (Logs/Build/Config/Env/Deploys) |
| `/` | open search (logs) or filter input (nav) |
| `n` / `N` | next / previous match |
| `f` | toggle level-filter strip |
| `space` | pause / resume tail |
| `g` / `G` | jump top / bottom |
| `c` | clear log buffer (does not refetch) |
| `r` | reconnect / retry |
| `?` | toggle help overlay |
| `q` / `ctrl+c` | quit |
| `esc` | close overlay / cancel search |

Mouse: scroll wheel scrolls the focused pane; clicking a service selects it. Nothing else.

---

## What v1 deliberately does **not** show

- No deploy-status badges beyond runtime state (backend doesn't expose deployment lifecycle yet).
- No build logs (backend endpoint not yet implemented).
- No config / env panes (same reason; stubs in place).
- No multi-service tail / split view (one stream at a time).
- No theming options (palette is fixed to match the web).
- No mouse drag-resize of the split (fixed proportions).

Each of these has a known unlock path on the backend roadmap. When an unlock lands, the corresponding tab body or badge swaps in without touching layout, keys, or styling.

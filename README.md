# monique

A tiny cross-platform focus tracker. A background daemon watches which window
holds focus, records each stretch as a session in SQLite, and a live terminal
UI shows a real-time activity log plus a weekly usage chart. Events older than
30 days are pruned automatically.

## How it works

```
active window  ──►  collector  ──►  tracker  ──►  SQLite
 (per-OS poll)      (FocusEvent)   (sessions)      │
                                                   ▼
                                    stats  ──►  Bubble Tea UI
```

- **collector** polls the foreground window once a second. Platform logic sits
  behind build tags:
  - **Linux** — Hyprland IPC when available, else X11 via `xprop`
  - **macOS** — AppleScript (`System Events`); window titles need Accessibility permission
  - **Windows** — `user32.dll` (`GetForegroundWindow`)
- **tracker** collapses that stream into sessions: one row per continuous
  period a window held focus. It heartbeats every 30s, recovers any session
  left dangling by a crash, and prunes events older than 30 days (at startup
  and once a day).
- **stats / tui** render two views, refreshed once a second:
  - **live** — a chronological activity log, newest first
    (`started · app · window · duration`); the focused session is marked live.
  - **chart** — a horizontal bar per day of total focused time, each labelled
    with that day's most-active app. Toggle 7/30 days and scroll.

## Usage

```sh
go build -o monique ./cmd/monique

./monique          # track + UI in one process
./monique track    # headless tracker only (for autostart in the background)
./monique ui       # viewer, opens on the live activity log
./monique chart    # viewer, opens on the weekly chart
```

Run the tracker in the background however you like — `./monique track &`,
a systemd user service, or Hyprland `exec-once = monique track` — then open
`ui`/`chart` whenever. SQLite runs in WAL mode, so the viewer reads while the
tracker writes. (Don't run a standalone `track` alongside the combined
`monique` — two trackers would fight over the open session.)

### Keys

- `c` — weekly chart (press again to toggle 7 / 30 days)
- `l` — live activity log
- `↑` / `↓`, `PgUp` / `PgDn` — scroll the chart
- `q` — quit

Both commands read/write `./monique/monique.db` relative to the working
directory.

## Requirements

- Go 1.26+
- Linux: Hyprland or an X11 session with `xprop`
- macOS: grant Accessibility permission for window titles
- No external services — storage is pure-Go SQLite (`modernc.org/sqlite`)

# monique

A tiny cross-platform focus tracker. A background daemon watches which window
holds focus, records each stretch as a session in SQLite, and a live terminal
UI shows where your time went today.

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
  period a window held focus. It heartbeats every 30s and recovers any session
  left dangling by a crash.
- **stats / tui** aggregate focused time per app since midnight and render it
  in a live-refreshing table.

## Usage

```sh
go build -o monique ./cmd/monique

./monique track   # run the tracker (leave it running in the background)
./monique ui      # view today's focus time
```

Both commands read/write `./monique/monique.db` relative to the working
directory.

## Requirements

- Go 1.26+
- Linux: Hyprland or an X11 session with `xprop`
- macOS: grant Accessibility permission for window titles
- No external services — storage is pure-Go SQLite (`modernc.org/sqlite`)

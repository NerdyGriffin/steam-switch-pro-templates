# steam-switch-pro-templates

[![CI](https://github.com/NerdyGriffin/steam-switch-pro-templates/actions/workflows/ci.yml/badge.svg)](https://github.com/NerdyGriffin/steam-switch-pro-templates/actions/workflows/ci.yml)
[![Latest release](https://img.shields.io/github/v/release/NerdyGriffin/steam-switch-pro-templates?sort=semver)](https://github.com/NerdyGriffin/steam-switch-pro-templates/releases/latest)
[![Go report card](https://goreportcard.com/badge/github.com/NerdyGriffin/steam-switch-pro-templates)](https://goreportcard.com/report/github.com/NerdyGriffin/steam-switch-pro-templates)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

**Adds the missing Steam Input controller templates for the Nintendo Switch Pro Controller — including a gyro+mouse template that Valve never shipped — and keeps them installed across Steam client updates.**

The Switch Pro Controller has gyro hardware and Steam Input fully supports it, but Valve only ships three official templates for it: `gamepad_flickstick`, `gamepad_joystick`, and `wasd`. Notably absent: a "Gamepad with Mouse and Gyro" template (the equivalent exists for PS5 / DualSense). This tool ports it over.

Because Valve owns `controller_base/templates/`, any custom template you drop in there gets wiped on the next Steam client update. This tool runs as a small triggered job (HKCU `Run` registry key on Windows, `systemd` path unit on Linux) that detects the missing file and reinstalls it, automatically and silently — no admin / sudo required.

## What it installs

- `controller_switch_pro_gamepad_mouse_gyro.vdf` — Gamepad + Mouse + Gyro template, ported from Valve's PS5 equivalent with Switch-Pro-correct button mappings (ZL as gyro activator, Capture button mapped to screenshot).

Future templates (FPS, gamepad+mouse without gyro) may be added — the architecture supports any number of custom templates.

## Subcommands

| | |
|---|---|
| `sspt apply` | Idempotent install. Compares each embedded template against on-disk content; installs if missing, leaves alone if matching. **Never overwrites unrecognized content** — if Valve eventually ships a real official template at the same filename, the conflict is recorded for `sspt resolve`. |
| `sspt install` | Copies the binary to a stable location and registers the OS trigger. Windows: HKCU `Run` registry key (logon). Linux: systemd user-level path unit watching the templates dir. No admin required on either. |
| `sspt uninstall` | Removes the OS trigger. `--purge` also removes the installed binary. |
| `sspt status` | Reports detected Steam install, per-template hash state, last-installed metadata, trigger health, and any pending conflicts (with Valve-version hint when recognized). |
| `sspt resolve` | Interactive conflict resolution. Shows hash + size + first-seen timestamp for each unresolved conflict and prompts: keep on-disk, overwrite with embedded, or skip. Non-interactive form: `--apply k|o`. |
| `sspt retire` | Happy-path exit for the day Valve ships an official template (or you switch hardware). Adopts on-disk content as new baseline, removes the OS trigger, prints a goodbye. |

### Conflict handling: how the watchdog stays out of your way

When `apply` finds an on-disk file that doesn't match either our embedded copy or what we previously installed, it:

1. **Leaves the file alone.** Default strategy is `preserve` — never destroys content we don't recognize.
2. **Records the conflict** in state (`%LOCALAPPDATA%\sspt\state.json` / `$XDG_STATE_HOME/sspt/state.json`).
3. **Fires a desktop notification** (Windows toast / Linux libnotify) telling you to run `sspt resolve`.
4. **Hashes the disk content against a registry of known Valve releases.** If matched, the message changes from generic "unrecognized content" to "looks like Valve shipped this in client X.Y.Z — consider `sspt retire`."

So the worst case isn't "your custom config got clobbered"; it's "we noticed something changed, here's what it is, you decide."

## Install (Windows)

**Quick install (one line, PowerShell):**

```powershell
irm https://raw.githubusercontent.com/NerdyGriffin/steam-switch-pro-templates/main/scripts/install.ps1 | iex
```

This fetches the latest release, verifies its SHA256 against the published
`SHA256SUMS`, and runs `sspt install`.

**Manual install:**

1. Download `sspt-windows-amd64.exe` from the [latest release](https://github.com/NerdyGriffin/steam-switch-pro-templates/releases/latest)
2. (Optional) Verify with `Get-FileHash sspt-windows-amd64.exe` against `SHA256SUMS`
3. Run: `.\sspt-windows-amd64.exe install`

**What `install` does:**
- Copies the binary to `%LOCALAPPDATA%\Programs\sspt\sspt.exe`
- Writes `HKCU\Software\Microsoft\Windows\CurrentVersion\Run\sspt-apply` so the
  apply step runs at every user logon (no admin / no UAC needed)
- Runs an initial apply

To remove later: `sspt uninstall` (add `--purge` to also delete the binary).
The state file (`%LOCALAPPDATA%\sspt\state.json`) is never touched
automatically — delete it by hand if you want a clean slate.

**SmartScreen warning:** the binary is unsigned, so on first download Windows
will say "Windows protected your PC." Click **More info → Run anyway**. Code
signing for indie open-source projects costs $100–400/yr; we'll add it if the
project gets enough users to justify.

**Why a logon trigger and not Task Scheduler?** Task Scheduler registration
requires admin on many Windows configurations (locked down by Group Policy
even for current-user tasks). The Run key works without elevation. Steam
re-reads `controller_base/templates/` whenever the directory changes, so a
once-per-session trigger is sufficient — your custom template will be present
the next time Steam launches after login.

## Install (Linux)

**Quick install (one line, bash):**

```bash
curl -fsSL https://raw.githubusercontent.com/NerdyGriffin/steam-switch-pro-templates/main/scripts/install.sh | bash
```

This fetches the latest release, verifies SHA256, places the binary at
`~/.local/bin/sspt`, and registers a systemd user-level path unit that watches
`controller_base/templates/` and runs `sspt apply` on any directory change.

**Manual install:** download `sspt-linux-amd64` from the [latest release](https://github.com/NerdyGriffin/steam-switch-pro-templates/releases/latest), verify, place on your `$PATH`, then run:

```bash
sspt install
```

**What `install` does on Linux:**
- Copies the binary to `$XDG_DATA_HOME/sspt/bin/sspt` (default `~/.local/share/sspt/bin/sspt`)
- Writes `~/.config/systemd/user/sspt.service` (oneshot running `sspt apply`)
- Writes `~/.config/systemd/user/sspt.path` (watches `controller_base/templates/`)
- `systemctl --user enable --now sspt.path`

To remove later: `sspt uninstall` (add `--purge` to also delete the binary).

**Inspect:** `systemctl --user status sspt.path sspt.service` &nbsp;·&nbsp; **Logs:** `journalctl --user -u sspt.service`

**Steam install discovery:** searches `$XDG_DATA_HOME/Steam`, `~/.local/share/Steam`, `~/.steam/steam`, `~/.steam/root`, and `~/.steam/debian-installation` (Steam Deck). Override with `--steam-path`.

## Sample `sspt status` output

```
Steam install:
  root:           c:\program files (x86)\steam
  templates dir:  c:\program files (x86)\steam\controller_base\templates

State file:     C:\Users\you\AppData\Local\sspt\state.json

Templates (1 embedded):
  controller_switch_pro_gamepad_mouse_gyro.vdf
    embedded hash: 65bbdfe648bfebd6…
    disk hash:     65bbdfe648bfebd6…
    status:        OK (matches embedded)
    last installed: v0.4.0 on 2026-05-03 21:00 UTC

Trigger:
  installed: yes (sspt apply will run on next user logon)
```

## Why this exists

I wanted a Switch-Pro gyro-mouse template for Horizon Zero Dawn. Steam doesn't ship one. Manually placing the file works, but Steam updates wipe `controller_base/`. So: a tiny watchdog.

## Maintainer notes

When Steam ships a client update, run `sspt scan-valve` on a machine that received it. The output is a Go-source snippet listing the new SHA256 hashes for every Valve template (filtered to exclude any that match a hash we already ship — prevents accidentally classifying our own work as a Valve release). Paste relevant entries into `internal/template/valve.go`'s `ValveHashes` map, set the `FirstSeen` marker to the actual Steam client version, and ship a release.

This keeps the conflict-detection heuristic informative without requiring external network calls or auto-updates.

## Project status

| Phase | What | Released |
|---|---|---|
| 1 | apply + status + conflict-aware state machine + tests | v0.1.0 |
| 2 | Windows install (HKCU Run key trigger) | v0.1.0 |
| 3 | Linux install (systemd user path + service units) | v0.2.0 |
| 4 | GitHub Actions release pipeline + repo public | v0.1.0 / ongoing |
| 5a | Desktop notifications (beeep) + interactive `resolve` | v0.3.0 |
| 5b | `retire` subcommand | v0.3.1 |
| 5c | Valve-hash heuristic + `scan-valve` maintainer tool | v0.4.0 |

## License

MIT. See [LICENSE](LICENSE).

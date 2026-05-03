# steam-switch-pro-templates

**Adds the missing Steam Input controller templates for the Nintendo Switch Pro Controller — including a gyro+mouse template that Valve never shipped — and keeps them installed across Steam client updates.**

The Switch Pro Controller has gyro hardware and Steam Input fully supports it, but Valve only ships three official templates for it: `gamepad_flickstick`, `gamepad_joystick`, and `wasd`. Notably absent: a "Gamepad with Mouse and Gyro" template (the equivalent exists for PS5 / DualSense). This tool ports it over.

Because Valve owns `controller_base/templates/`, any custom template you drop in there gets wiped on the next Steam client update. This tool runs as a small triggered job (Task Scheduler on Windows, `systemd` path unit on Linux) that detects the missing file and reinstalls it, automatically and silently.

## What it installs

- `controller_switch_pro_gamepad_mouse_gyro.vdf` — Gamepad + Mouse + Gyro template, ported from Valve's PS5 equivalent with Switch-Pro-correct button mappings (ZL as gyro activator, Capture button mapped to screenshot).

Future templates (FPS, gamepad+mouse without gyro) may be added — the architecture supports any number of custom templates.

## How it works

1. **`sspt apply`** — idempotent install. Detects your Steam install (Windows registry / Linux filesystem), compares the on-disk template against an embedded canonical copy, and writes it if missing. **Never overwrites unrecognized content** — if Steam shipped a different file with the same name (e.g., Valve finally added an official version), the tool detects the conflict, leaves the file alone, and notifies you.
2. **`sspt install`** — bootstraps the OS-level trigger. On Windows, writes an `HKCU\...\Run` registry value so `sspt apply` fires once per user logon (no admin required). On Linux, writes a systemd user-level path unit that watches `controller_base/templates/` (Phase 3, in progress).
3. **`sspt status`** — reports detected Steam install, per-template state, and trigger health.
4. **`sspt resolve`** — interactive conflict resolution. Shows a diff between your installed custom template and whatever's now on disk, lets you pick which to keep.
5. **`sspt retire`** — for the happy day Valve ships an official template. Disables the trigger, leaves Valve's file alone, prints a goodbye message.

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

## Why this exists

I wanted a Switch-Pro gyro-mouse template for Horizon Zero Dawn. Steam doesn't ship one. Manually placing the file works, but Steam updates wipe `controller_base/`. So: a tiny watchdog.

## Status

**v0.x — under active development.** Phase 1 (apply + state machine) functional; Phase 2 (OS triggers) in progress.

## License

MIT. See [LICENSE](LICENSE).

# steam-switch-pro-templates

**Adds the missing Steam Input controller templates for the Nintendo Switch Pro Controller — including a gyro+mouse template that Valve never shipped — and keeps them installed across Steam client updates.**

The Switch Pro Controller has gyro hardware and Steam Input fully supports it, but Valve only ships three official templates for it: `gamepad_flickstick`, `gamepad_joystick`, and `wasd`. Notably absent: a "Gamepad with Mouse and Gyro" template (the equivalent exists for PS5 / DualSense). This tool ports it over.

Because Valve owns `controller_base/templates/`, any custom template you drop in there gets wiped on the next Steam client update. This tool runs as a small triggered job (Task Scheduler on Windows, `systemd` path unit on Linux) that detects the missing file and reinstalls it, automatically and silently.

## What it installs

- `controller_switch_pro_gamepad_mouse_gyro.vdf` — Gamepad + Mouse + Gyro template, ported from Valve's PS5 equivalent with Switch-Pro-correct button mappings (ZL as gyro activator, Capture button mapped to screenshot).

Future templates (FPS, gamepad+mouse without gyro) may be added — the architecture supports any number of custom templates.

## How it works

1. **`sspt apply`** — idempotent install. Detects your Steam install (Windows registry / Linux filesystem), compares the on-disk template against an embedded canonical copy, and writes it if missing. **Never overwrites unrecognized content** — if Steam shipped a different file with the same name (e.g., Valve finally added an official version), the tool detects the conflict, leaves the file alone, and notifies you.
2. **`sspt install`** — bootstraps the OS-level trigger. On Windows, registers a Task Scheduler task that fires when `steam.exe` starts/stops. On Linux, writes a systemd user-level path unit that watches `controller_base/templates/`.
3. **`sspt status`** — reports detected Steam install, per-template state, and trigger health.
4. **`sspt resolve`** — interactive conflict resolution. Shows a diff between your installed custom template and whatever's now on disk, lets you pick which to keep.
5. **`sspt retire`** — for the happy day Valve ships an official template. Disables the trigger, leaves Valve's file alone, prints a goodbye message.

## Install (Windows)

```powershell
# Download latest release binary, then:
sspt.exe install
```

## Install (Linux)

```bash
# Download latest release binary, then:
./sspt install
```

## Why this exists

I wanted a Switch-Pro gyro-mouse template for Horizon Zero Dawn. Steam doesn't ship one. Manually placing the file works, but Steam updates wipe `controller_base/`. So: a tiny watchdog.

## Status

**v0.x — under active development.** Phase 1 (apply + state machine) functional; Phase 2 (OS triggers) in progress.

## License

MIT. See [LICENSE](LICENSE).

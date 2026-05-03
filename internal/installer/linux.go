//go:build linux

package installer

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"github.com/NerdyGriffin/steam-switch-pro-templates/internal/steam"
)

// New returns the Linux installer.
func New() Installer { return &linuxInstaller{} }

type linuxInstaller struct{}

const (
	pathUnitName    = "sspt.path"
	serviceUnitName = "sspt.service"
)

func (l *linuxInstaller) Install(opts InstallOptions) (InstallReport, error) {
	dst, err := installBinary(opts.SourceBinary)
	if err != nil {
		return InstallReport{}, fmt.Errorf("install binary: %w", err)
	}

	// Need to know which directory to watch — discover Steam, fall back to a
	// placeholder if Steam isn't installed (the unit will simply not fire
	// until Steam is later installed at the watched path).
	watchDir, watchNote := resolveWatchDir()

	unitDir, err := userUnitDir()
	if err != nil {
		return InstallReport{}, err
	}
	if err := os.MkdirAll(unitDir, 0o755); err != nil {
		return InstallReport{}, fmt.Errorf("create %s: %w", unitDir, err)
	}

	servicePath := filepath.Join(unitDir, serviceUnitName)
	pathPath := filepath.Join(unitDir, pathUnitName)

	if err := writeFile(servicePath, renderServiceUnit(dst)); err != nil {
		return InstallReport{}, err
	}
	if err := writeFile(pathPath, renderPathUnit(watchDir)); err != nil {
		return InstallReport{}, err
	}

	if out, err := runSystemctl("--user", "daemon-reload"); err != nil {
		return InstallReport{}, fmt.Errorf("systemctl daemon-reload: %w\n%s", err, out)
	}
	if out, err := runSystemctl("--user", "enable", "--now", pathUnitName); err != nil {
		return InstallReport{}, fmt.Errorf("systemctl enable --now %s: %w\n%s", pathUnitName, err, out)
	}

	notes := []string{
		fmt.Sprintf("Wrote %s and %s", servicePath, pathPath),
		fmt.Sprintf("systemd path-unit watches: %s", watchDir),
		"Fires sspt apply on any change to that directory.",
		"Inspect:  systemctl --user status sspt.path sspt.service",
		"Logs:     journalctl --user -u sspt.service",
	}
	if watchNote != "" {
		notes = append(notes, watchNote)
	}
	return InstallReport{BinaryPath: dst, TriggerName: pathUnitName, Notes: notes}, nil
}

func (l *linuxInstaller) Uninstall(purge bool) (UninstallReport, error) {
	report := UninstallReport{}

	// Disable + stop, ignore "not found" errors.
	if out, err := runSystemctl("--user", "disable", "--now", pathUnitName); err != nil {
		if !bytes.Contains(out, []byte("not loaded")) && !bytes.Contains(out, []byte("does not exist")) {
			report.Notes = append(report.Notes, fmt.Sprintf("disable failed (continuing): %v", err))
		}
	} else {
		report.TriggerRemoved = true
	}

	unitDir, err := userUnitDir()
	if err != nil {
		return report, err
	}

	for _, name := range []string{pathUnitName, serviceUnitName} {
		p := filepath.Join(unitDir, name)
		if err := os.Remove(p); err == nil {
			report.Notes = append(report.Notes, "Removed "+p)
		} else if !os.IsNotExist(err) {
			report.Notes = append(report.Notes, fmt.Sprintf("Remove %s: %v", p, err))
		}
	}

	// daemon-reload to forget the unit files.
	if out, err := runSystemctl("--user", "daemon-reload"); err != nil {
		report.Notes = append(report.Notes, fmt.Sprintf("daemon-reload (continuing): %v\n%s", err, out))
	}

	if purge {
		dir, err := installDir()
		if err == nil {
			if err := os.RemoveAll(dir); err == nil {
				report.BinaryRemoved = true
				report.Notes = append(report.Notes, "Removed "+dir)
			} else {
				report.Notes = append(report.Notes, fmt.Sprintf("Binary remove failed: %v", err))
			}
		}
	}
	return report, nil
}

func (l *linuxInstaller) IsInstalled() (bool, error) {
	out, err := runSystemctl("--user", "is-enabled", pathUnitName)
	if err != nil {
		// `is-enabled` returns nonzero for "disabled" / "not-found"; that
		// isn't a tool error from our perspective.
		if bytes.Contains(out, []byte("Failed")) {
			return false, fmt.Errorf("systemctl is-enabled: %w\n%s", err, out)
		}
		return false, nil
	}
	state := bytes.TrimSpace(out)
	return bytes.Equal(state, []byte("enabled")) || bytes.Equal(state, []byte("static")), nil
}

// --- helpers ---

func userUnitDir() (string, error) {
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return filepath.Join(v, "systemd", "user"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "systemd", "user"), nil
}

// installDir is the canonical user-scope binary location:
// $XDG_DATA_HOME/sspt/bin or ~/.local/share/sspt/bin.
func installDir() (string, error) {
	if v := os.Getenv("XDG_DATA_HOME"); v != "" {
		return filepath.Join(v, "sspt", "bin"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "sspt", "bin"), nil
}

func installBinary(src string) (string, error) {
	dir, err := installDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	dst := filepath.Join(dir, "sspt")

	srcAbs, _ := filepath.Abs(src)
	dstAbs, _ := filepath.Abs(dst)
	if srcAbs == dstAbs {
		return dst, nil
	}

	in, err := os.Open(src)
	if err != nil {
		return "", fmt.Errorf("open source binary %s: %w", src, err)
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return "", fmt.Errorf("create dest binary %s: %w", dst, err)
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return "", fmt.Errorf("copy binary: %w", err)
	}
	if err := out.Close(); err != nil {
		return "", err
	}
	return dst, nil
}

// resolveWatchDir picks the templates directory the path-unit will watch.
// Falls back to the most likely default Steam install path if Steam isn't
// detected — the path-unit will simply sit idle until that directory exists.
func resolveWatchDir() (dir, note string) {
	if install, err := steam.Locate(); err == nil {
		return install.TemplatesDir(), ""
	}
	home, _ := os.UserHomeDir()
	fallback := filepath.Join(home, ".local", "share", "Steam", "controller_base", "templates")
	return fallback, fmt.Sprintf("Steam install not detected; using fallback %s. Re-run `sspt install` after installing Steam to point the unit at the real path.", fallback)
}

func runSystemctl(args ...string) ([]byte, error) {
	cmd := exec.Command("systemctl", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		var ee *exec.Error
		if errors.As(err, &ee) {
			return out, fmt.Errorf("systemctl not available: %w", err)
		}
	}
	return out, err
}

func writeFile(path string, body []byte) error {
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// --- unit file templates ---

func renderServiceUnit(binPath string) []byte {
	tmpl := template.Must(template.New("service").Parse(`[Unit]
Description=Reinstall custom Steam controller templates after Steam updates wipe them
Documentation=https://github.com/NerdyGriffin/steam-switch-pro-templates

[Service]
Type=oneshot
ExecStart={{.Bin}} apply
# apply is idempotent and quick (sub-100 ms in steady state), so no rate limit needed.
# Logs go to journal AND %h/.local/state/sspt/sspt.log via lumberjack.
`))
	var buf bytes.Buffer
	_ = tmpl.Execute(&buf, struct{ Bin string }{Bin: binPath})
	return buf.Bytes()
}

func renderPathUnit(watchDir string) []byte {
	tmpl := template.Must(template.New("path").Parse(`[Unit]
Description=Watch Steam controller_base/templates for changes; trigger sspt-apply
Documentation=https://github.com/NerdyGriffin/steam-switch-pro-templates

[Path]
# PathChanged fires on any modification to the directory contents — Steam
# updates that add, remove, or replace files in this dir all trigger us.
PathChanged={{.WatchDir}}
# Also watch the templates dir's parent so we catch the case where a Steam
# update wipes-and-recreates controller_base/ entirely.
PathExistsGlob={{.WatchDir}}/*.vdf
Unit=sspt.service

[Install]
WantedBy=default.target
`))
	var buf bytes.Buffer
	_ = tmpl.Execute(&buf, struct{ WatchDir string }{WatchDir: watchDir})
	return buf.Bytes()
}

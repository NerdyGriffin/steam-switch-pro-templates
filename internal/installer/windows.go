//go:build windows

package installer

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

// runKeyPath is the per-user Run key. Values written here are executed once
// per interactive logon. Writing requires only the current user's permissions
// (no UAC elevation), which is why we prefer this over Task Scheduler — task
// registration is locked down on many systems.
const (
	runKeyPath  = `Software\Microsoft\Windows\CurrentVersion\Run`
	runValue    = `sspt-apply`
	taskName    = runValue // reused so callers print a stable identifier
	notesHeader = "Logon trigger registered (HKCU Run key)."
)

// New returns the Windows installer.
func New() Installer { return &winInstaller{} }

type winInstaller struct{}

func (w *winInstaller) Install(opts InstallOptions) (InstallReport, error) {
	dst, err := installBinary(opts.SourceBinary)
	if err != nil {
		return InstallReport{}, fmt.Errorf("install binary: %w", err)
	}

	cmdline := fmt.Sprintf(`"%s" apply`, dst)
	if err := writeRunValue(cmdline); err != nil {
		return InstallReport{}, fmt.Errorf("register Run key: %w", err)
	}

	notes := []string{
		notesHeader,
		"reg path: HKCU\\" + runKeyPath + `\` + runValue,
		"command:  " + cmdline,
		"Fires once per user logon. Steam picks up template changes hot, so the template will be present next time you launch Steam after login.",
		"To run manually now:  sspt apply",
	}
	return InstallReport{BinaryPath: dst, TriggerName: taskName, Notes: notes}, nil
}

func (w *winInstaller) Uninstall(purge bool) (UninstallReport, error) {
	report := UninstallReport{}

	removed, err := deleteRunValue()
	if err != nil {
		return report, fmt.Errorf("delete Run key value: %w", err)
	}
	if removed {
		report.TriggerRemoved = true
		report.Notes = append(report.Notes, "Removed HKCU\\"+runKeyPath+`\`+runValue)
	} else {
		report.Notes = append(report.Notes, "Run key value was not present (nothing to remove).")
	}

	if purge {
		dir, err := installDir()
		if err == nil {
			if err := os.RemoveAll(dir); err == nil {
				report.BinaryRemoved = true
				report.Notes = append(report.Notes, "Installed binary removed from "+dir)
			} else {
				report.Notes = append(report.Notes, fmt.Sprintf("Binary remove failed: %v", err))
			}
		}
	}
	return report, nil
}

func (w *winInstaller) IsInstalled() (bool, error) {
	_, err := readRunValue()
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// --- registry helpers ---

func writeRunValue(cmdline string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open %s: %w", runKeyPath, err)
	}
	defer k.Close()
	return k.SetStringValue(runValue, cmdline)
}

func readRunValue() (string, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer k.Close()
	v, _, err := k.GetStringValue(runValue)
	return v, err
}

func deleteRunValue() (bool, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	defer k.Close()
	if err := k.DeleteValue(runValue); err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// --- binary install ---

func installDir() (string, error) {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, "AppData", "Local")
	}
	return filepath.Join(base, "Programs", "sspt"), nil
}

// installBinary copies the running binary into a stable location and returns
// the destination path. If the running binary is already at that path, it is
// reused without copying.
func installBinary(src string) (string, error) {
	dir, err := installDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	dst := filepath.Join(dir, "sspt.exe")

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

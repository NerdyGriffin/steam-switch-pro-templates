//go:build linux

package installer

import "errors"

// New returns the Linux installer.
func New() Installer { return &linuxInstaller{} }

type linuxInstaller struct{}

// errNotImplemented marks Linux trigger registration as still-to-do.
var errNotImplemented = errors.New("linux installer not yet implemented (Phase 3) — for now, run `sspt apply` manually or via your own systemd unit")

func (l *linuxInstaller) Install(opts InstallOptions) (InstallReport, error) {
	return InstallReport{}, errNotImplemented
}

func (l *linuxInstaller) Uninstall(purge bool) (UninstallReport, error) {
	return UninstallReport{}, errNotImplemented
}

func (l *linuxInstaller) IsInstalled() (bool, error) {
	return false, nil
}

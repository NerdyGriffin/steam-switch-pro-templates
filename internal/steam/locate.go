// Package steam discovers a local Steam installation and exposes the paths
// other parts of the tool care about (controller template directory, etc.).
package steam

import (
	"errors"
	"path/filepath"
)

// Install describes a discovered Steam installation.
type Install struct {
	// Root is the Steam install directory, e.g. "C:\Program Files (x86)\Steam"
	// or "/home/user/.local/share/Steam".
	Root string
}

// TemplatesDir returns the controller_base/templates path inside the install.
func (i Install) TemplatesDir() string {
	return filepath.Join(i.Root, "controller_base", "templates")
}

// ErrSteamNotFound is returned by Locate when no Steam install is detected.
var ErrSteamNotFound = errors.New("steam installation not found")

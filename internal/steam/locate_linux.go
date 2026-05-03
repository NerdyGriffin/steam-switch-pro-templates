//go:build linux

package steam

import (
	"os"
	"path/filepath"
)

// Locate finds the Steam install on Linux.
//
// Lookup order, first existing directory wins:
//  1. $XDG_DATA_HOME/Steam
//  2. ~/.local/share/Steam (default XDG_DATA_HOME)
//  3. ~/.steam/steam (legacy symlink target)
//  4. ~/.steam/root (some distros)
//  5. ~/.steam/debian-installation (Steam Deck)
func Locate() (*Install, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	candidates := []string{}
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		candidates = append(candidates, filepath.Join(xdg, "Steam"))
	}
	candidates = append(candidates,
		filepath.Join(home, ".local", "share", "Steam"),
		filepath.Join(home, ".steam", "steam"),
		filepath.Join(home, ".steam", "root"),
		filepath.Join(home, ".steam", "debian-installation"),
	)

	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			// resolve symlinks (~/.steam/steam is usually a symlink)
			resolved, err := filepath.EvalSymlinks(c)
			if err != nil {
				resolved = c
			}
			return &Install{Root: resolved}, nil
		}
	}
	return nil, ErrSteamNotFound
}

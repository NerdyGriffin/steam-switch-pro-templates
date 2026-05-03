// Package state persists what the tool has installed previously, so apply()
// can distinguish "we put this here, content unchanged" from "something else
// (Valve update, manual edit) replaced our file."
package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// State is the on-disk JSON document.
type State struct {
	Templates map[string]TemplateState `json:"templates"`
}

// TemplateState records the most recent successful install of a single template.
type TemplateState struct {
	InstalledHash    string     `json:"installed_hash"`
	InstalledVersion string     `json:"installed_version"`
	InstalledAt      time.Time  `json:"installed_at"`
	ConflictSeenAt   *time.Time `json:"conflict_seen_at,omitempty"`
}

// Path returns the platform-specific state file location.
//
// Windows: %LOCALAPPDATA%\sspt\state.json
// Linux:   $XDG_STATE_HOME/sspt/state.json (default ~/.local/state/sspt/state.json)
func Path() (string, error) {
	dir, err := dataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "state.json"), nil
}

// DataDir returns the directory used for state, logs, and any other tool data.
func DataDir() (string, error) {
	return dataDir()
}

func dataDir() (string, error) {
	if runtime.GOOS == "windows" {
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			base = filepath.Join(home, "AppData", "Local")
		}
		return filepath.Join(base, "sspt"), nil
	}
	if base := os.Getenv("XDG_STATE_HOME"); base != "" {
		return filepath.Join(base, "sspt"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "state", "sspt"), nil
}

// Load reads the state file, returning an empty State if the file is absent.
func Load() (*State, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &State{Templates: map[string]TemplateState{}}, nil
		}
		return nil, fmt.Errorf("read state %s: %w", path, err)
	}
	s := &State{}
	if err := json.Unmarshal(b, s); err != nil {
		return nil, fmt.Errorf("parse state %s: %w", path, err)
	}
	if s.Templates == nil {
		s.Templates = map[string]TemplateState{}
	}
	return s, nil
}

// Save writes the state file atomically (write-temp-then-rename).
func (s *State) Save() error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return fmt.Errorf("write temp state: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename temp state: %w", err)
	}
	return nil
}

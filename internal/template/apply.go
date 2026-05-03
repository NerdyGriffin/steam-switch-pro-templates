package template

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/NerdyGriffin/steam-switch-pro-templates/internal/state"
	"github.com/NerdyGriffin/steam-switch-pro-templates/internal/steam"
)

// Strategy controls how Apply handles the case where the on-disk file exists
// but doesn't match the embedded copy or our previous install record.
type Strategy int

const (
	// StrategyPreserve never overwrites unrecognized content. The conflict is
	// recorded in state and the disk file is left alone. This is the default
	// and the only safe strategy for triggered/headless runs.
	StrategyPreserve Strategy = iota

	// StrategyForce always overwrites. Use only when the user has explicitly
	// chosen to discard whatever is on disk (e.g. via `sspt resolve`).
	StrategyForce

	// StrategyAcceptExisting adopts whatever is currently on disk as the new
	// baseline — records its hash in state but does not modify the file. Used
	// when the user wants to keep the existing content (e.g. Valve shipped
	// their own version) without retiring the tool.
	StrategyAcceptExisting
)

// Outcome describes what Apply did or chose not to do.
type Outcome int

const (
	OutcomeAlreadyCurrent Outcome = iota // disk matches embedded; no-op
	OutcomeInstalled                     // file was absent, now written
	OutcomeUpgraded                      // disk was older version of ours, replaced
	OutcomeConflict                      // unknown content on disk, left alone
	OutcomeForced                        // overwrote unknown content
	OutcomeAccepted                      // adopted disk content as new baseline
)

func (o Outcome) String() string {
	switch o {
	case OutcomeAlreadyCurrent:
		return "already-current"
	case OutcomeInstalled:
		return "installed"
	case OutcomeUpgraded:
		return "upgraded"
	case OutcomeConflict:
		return "conflict"
	case OutcomeForced:
		return "forced"
	case OutcomeAccepted:
		return "accepted-existing"
	default:
		return "unknown"
	}
}

// Result is what Apply returns to the caller for logging / display.
type Result struct {
	Filename string
	Path     string
	Outcome  Outcome
	DiskHash string // sha256 of file currently on disk after Apply (empty if absent)
	Message  string // human-readable summary
}

// Apply runs the conflict-aware install state machine for a single template.
// It mutates `st` (recording installs / conflicts) but does not save it —
// the caller batches saves.
func Apply(install *steam.Install, st *state.State, t Template, strategy Strategy, version string) (Result, error) {
	if install == nil {
		return Result{}, errors.New("steam install is nil")
	}

	dir := install.TemplatesDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Result{}, fmt.Errorf("ensure templates dir: %w", err)
	}

	dst := filepath.Join(dir, t.Filename)
	prev, hadPrev := st.Templates[t.Filename]

	diskBytes, statErr := os.ReadFile(dst)
	switch {
	case os.IsNotExist(statErr):
		// File absent — install.
		return install_(t, dst, st, version)

	case statErr != nil:
		return Result{Filename: t.Filename, Path: dst}, fmt.Errorf("read existing %s: %w", dst, statErr)
	}

	diskHash := HashBytes(diskBytes)

	if diskHash == t.Hash {
		// Disk matches embedded copy — nothing to do, but record it as installed
		// so future runs know it's ours even if state was lost.
		st.Templates[t.Filename] = state.TemplateState{
			InstalledHash:    t.Hash,
			InstalledVersion: version,
			InstalledAt:      timeOrPrev(prev, hadPrev),
		}
		return Result{
			Filename: t.Filename, Path: dst, Outcome: OutcomeAlreadyCurrent, DiskHash: diskHash,
			Message: "already current",
		}, nil
	}

	if hadPrev && diskHash == prev.InstalledHash {
		// Disk matches what *we* installed previously, but the embedded copy
		// has moved forward (new release of this tool). Upgrade silently.
		return install_(t, dst, st, version)
	}

	// Conflict: unknown content on disk.
	switch strategy {
	case StrategyForce:
		res, err := install_(t, dst, st, version)
		if err != nil {
			return res, err
		}
		res.Outcome = OutcomeForced
		res.Message = "overwrote unrecognized content (--force)"
		return res, nil

	case StrategyAcceptExisting:
		now := time.Now().UTC()
		st.Templates[t.Filename] = state.TemplateState{
			InstalledHash:    diskHash,
			InstalledVersion: version,
			InstalledAt:      now,
		}
		return Result{
			Filename: t.Filename, Path: dst, Outcome: OutcomeAccepted, DiskHash: diskHash,
			Message: "adopted existing on-disk content as new baseline",
		}, nil

	default: // StrategyPreserve
		now := time.Now().UTC()
		entry := prev
		if !hadPrev {
			entry = state.TemplateState{}
		}
		entry.ConflictSeenAt = &now
		st.Templates[t.Filename] = entry
		return Result{
			Filename: t.Filename, Path: dst, Outcome: OutcomeConflict, DiskHash: diskHash,
			Message: "unrecognized content on disk; preserved (run `sspt resolve` to choose)",
		}, nil
	}
}

func install_(t Template, dst string, st *state.State, version string) (Result, error) {
	wasPresent := false
	if _, err := os.Stat(dst); err == nil {
		wasPresent = true
	}
	if err := writeAtomic(dst, t.Content); err != nil {
		return Result{Filename: t.Filename, Path: dst}, err
	}
	st.Templates[t.Filename] = state.TemplateState{
		InstalledHash:    t.Hash,
		InstalledVersion: version,
		InstalledAt:      time.Now().UTC(),
	}
	out := Result{Filename: t.Filename, Path: dst, DiskHash: t.Hash}
	if wasPresent {
		out.Outcome = OutcomeUpgraded
		out.Message = "upgraded from previous installed version"
	} else {
		out.Outcome = OutcomeInstalled
		out.Message = "installed"
	}
	return out, nil
}

// writeAtomic writes b to dst via a temp file in the same directory, then
// renames into place — Steam never sees a half-written file.
func writeAtomic(dst string, b []byte) error {
	dir := filepath.Dir(dst)
	tmp, err := os.CreateTemp(dir, ".sspt-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		cleanup()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpName, dst); err != nil {
		cleanup()
		return fmt.Errorf("rename %s to %s: %w", tmpName, dst, err)
	}
	return nil
}

func timeOrPrev(prev state.TemplateState, hadPrev bool) time.Time {
	if hadPrev && !prev.InstalledAt.IsZero() {
		return prev.InstalledAt
	}
	return time.Now().UTC()
}

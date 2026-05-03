// Package installer registers the platform-specific trigger that runs
// `sspt apply` automatically — Task Scheduler on Windows, systemd path/timer
// units on Linux.
package installer

// Installer is the platform-specific implementation.
type Installer interface {
	// Install copies the running binary into a stable location, registers the
	// OS-level trigger to invoke `sspt apply`, and returns the resolved binary
	// install path.
	Install(opts InstallOptions) (InstallReport, error)

	// Uninstall removes the OS trigger. If purge is true, the installed binary
	// is also removed (state is left alone unless the caller deletes it).
	Uninstall(purge bool) (UninstallReport, error)

	// IsInstalled reports whether the OS trigger appears to be registered.
	IsInstalled() (bool, error)
}

// InstallOptions tweaks the install behavior.
type InstallOptions struct {
	// SourceBinary is the path to the running binary (typically os.Executable()).
	SourceBinary string
}

// InstallReport summarizes what Install did.
type InstallReport struct {
	BinaryPath  string // where the binary was placed
	TriggerName string // OS-specific identifier (Task name, systemd unit name)
	Notes       []string
}

// UninstallReport summarizes what Uninstall did.
type UninstallReport struct {
	TriggerRemoved bool
	BinaryRemoved  bool
	Notes          []string
}

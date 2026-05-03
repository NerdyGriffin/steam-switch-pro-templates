// Package logging configures structured slog output with size-rotated files.
package logging

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/NerdyGriffin/steam-switch-pro-templates/internal/state"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

// Setup wires slog to write to both stderr and a rotated file under the data
// dir. `verbose` toggles debug-level output. Returns the resolved log path so
// callers can mention it in --help / status output.
func Setup(verbose bool) (string, error) {
	dir, err := state.DataDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	logPath := filepath.Join(dir, "sspt.log")

	rotator := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    1,  // megabytes per file before rotation
		MaxBackups: 3,  // keep 3 rotated archives
		MaxAge:     30, // days
		Compress:   true,
	}

	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	out := io.MultiWriter(os.Stderr, rotator)
	handler := slog.NewTextHandler(out, &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(handler))
	return logPath, nil
}

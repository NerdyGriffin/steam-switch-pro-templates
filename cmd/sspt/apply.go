package main

import (
	"fmt"
	"log/slog"

	"github.com/NerdyGriffin/steam-switch-pro-templates/internal/logging"
	"github.com/NerdyGriffin/steam-switch-pro-templates/internal/notify"
	"github.com/NerdyGriffin/steam-switch-pro-templates/internal/state"
	"github.com/NerdyGriffin/steam-switch-pro-templates/internal/steam"
	"github.com/NerdyGriffin/steam-switch-pro-templates/internal/template"
	"github.com/spf13/cobra"
)

func newApplyCmd() *cobra.Command {
	var (
		strategyStr string
		steamPath   string
	)
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Install / refresh embedded controller templates (idempotent).",
		Long: `apply runs the conflict-aware install state machine for every embedded
template. It is safe to run repeatedly and is the command invoked by the
Windows Task Scheduler trigger / Linux systemd path unit.

Strategies:
  preserve  (default) Never overwrite unrecognized on-disk content; record the
            conflict so you can resolve it later.
  force     Always overwrite, even if disk content is unrecognized.
  accept    Adopt whatever is on disk as the new baseline (records its hash);
            does not modify the file.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			strategy, err := parseStrategy(strategyStr)
			if err != nil {
				return err
			}

			if _, err := logging.Setup(flagVerbose); err != nil {
				return fmt.Errorf("logging setup: %w", err)
			}

			install, err := resolveInstall(steamPath)
			if err != nil {
				return err
			}
			slog.Info("steam located", "root", install.Root, "templates_dir", install.TemplatesDir())

			st, err := state.Load()
			if err != nil {
				return fmt.Errorf("load state: %w", err)
			}

			templates, err := template.All()
			if err != nil {
				return fmt.Errorf("enumerate embedded templates: %w", err)
			}
			slog.Debug("templates discovered", "count", len(templates))

			conflicts := 0
			for _, t := range templates {
				res, err := template.Apply(install, st, t, strategy, Version)
				if err != nil {
					slog.Error("apply failed", "template", t.Filename, "err", err)
					return err
				}
				logResult(res)
				fmt.Printf("%-55s %-18s %s\n", res.Filename, res.Outcome, res.Message)
				if res.Outcome == template.OutcomeConflict {
					conflicts++
					notify.Conflict(res.Filename)
				}
			}

			if err := st.Save(); err != nil {
				return fmt.Errorf("save state: %w", err)
			}

			if conflicts > 0 {
				fmt.Printf("\n%d conflict(s) detected — run `sspt resolve` to choose.\n", conflicts)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&strategyStr, "strategy", "preserve", "conflict strategy: preserve | force | accept")
	cmd.Flags().StringVar(&steamPath, "steam-path", "", "override Steam install location (skips auto-detection)")
	return cmd
}

func parseStrategy(s string) (template.Strategy, error) {
	switch s {
	case "preserve", "":
		return template.StrategyPreserve, nil
	case "force":
		return template.StrategyForce, nil
	case "accept", "accept-existing":
		return template.StrategyAcceptExisting, nil
	default:
		return 0, fmt.Errorf("unknown strategy %q (want preserve|force|accept)", s)
	}
}

func resolveInstall(override string) (*steam.Install, error) {
	if override != "" {
		return &steam.Install{Root: override}, nil
	}
	install, err := steam.Locate()
	if err != nil {
		return nil, fmt.Errorf("locate steam: %w", err)
	}
	return install, nil
}

func logResult(r template.Result) {
	slog.Info("apply result",
		"template", r.Filename,
		"outcome", r.Outcome.String(),
		"path", r.Path,
		"disk_hash", r.DiskHash,
		"message", r.Message,
	)
}

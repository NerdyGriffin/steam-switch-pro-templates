package main

import (
	"fmt"
	"os"

	"github.com/NerdyGriffin/steam-switch-pro-templates/internal/installer"
	"github.com/NerdyGriffin/steam-switch-pro-templates/internal/logging"
	"github.com/NerdyGriffin/steam-switch-pro-templates/internal/state"
	"github.com/NerdyGriffin/steam-switch-pro-templates/internal/template"
	"github.com/spf13/cobra"
)

func newInstallCmd() *cobra.Command {
	var (
		skipApply bool
		steamPath string
	)
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install logon trigger that re-runs `apply` automatically.",
		Long: `install copies the running sspt binary to a stable location and registers
a logon-time trigger that runs ` + "`sspt apply`" + ` whenever the user logs in.

Windows: writes an HKCU\\...\\Run registry value (no admin needed).

Linux: not yet implemented (Phase 3); see README for manual systemd setup.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := logging.Setup(flagVerbose); err != nil {
				return fmt.Errorf("logging setup: %w", err)
			}

			self, err := os.Executable()
			if err != nil {
				return fmt.Errorf("locate running binary: %w", err)
			}

			inst := installer.New()
			report, err := inst.Install(installer.InstallOptions{
				SourceBinary: self,
			})
			if err != nil {
				return err
			}

			fmt.Println("Install OK.")
			fmt.Println("  binary:  ", report.BinaryPath)
			fmt.Println("  trigger: ", report.TriggerName)
			for _, n := range report.Notes {
				fmt.Println("  note:    ", n)
			}

			if !skipApply {
				fmt.Println("\nRunning initial apply...")
				install, err := resolveInstall(steamPath)
				if err != nil {
					return err
				}
				st, err := state.Load()
				if err != nil {
					return err
				}
				templates, err := template.All()
				if err != nil {
					return err
				}
				for _, t := range templates {
					res, err := template.Apply(install, st, t, template.StrategyPreserve, Version)
					if err != nil {
						return err
					}
					fmt.Printf("  %-55s %-18s %s\n", res.Filename, res.Outcome, res.Message)
				}
				if err := st.Save(); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&skipApply, "skip-apply", false, "register the trigger but skip the initial apply run")
	cmd.Flags().StringVar(&steamPath, "steam-path", "", "override Steam install location for the initial apply")
	return cmd
}

func newUninstallCmd() *cobra.Command {
	var purge bool
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove the OS trigger (and optionally the installed binary).",
		Long: `uninstall removes the OS-level trigger registered by ` + "`sspt install`" + `.

By default the installed binary is left in place; pass --purge to also remove
it. The state file is never touched (delete it manually if desired).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := logging.Setup(flagVerbose); err != nil {
				return fmt.Errorf("logging setup: %w", err)
			}

			inst := installer.New()
			report, err := inst.Uninstall(purge)
			if err != nil {
				return err
			}

			fmt.Println("Uninstall OK.")
			for _, n := range report.Notes {
				fmt.Println("  note:    ", n)
			}
			if !purge {
				fmt.Println("\nInstalled binary left in place. Pass --purge to remove it.")
				fmt.Println("State file (sspt history) left in place. Delete manually if desired.")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&purge, "purge", false, "also remove the installed binary directory")
	return cmd
}

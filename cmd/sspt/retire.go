package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/NerdyGriffin/steam-switch-pro-templates/internal/installer"
	"github.com/NerdyGriffin/steam-switch-pro-templates/internal/logging"
	"github.com/NerdyGriffin/steam-switch-pro-templates/internal/state"
	"github.com/NerdyGriffin/steam-switch-pro-templates/internal/template"
	"github.com/spf13/cobra"
)

func newRetireCmd() *cobra.Command {
	var (
		steamPath string
		purge     bool
		yes       bool
	)
	cmd := &cobra.Command{
		Use:   "retire",
		Short: "Gracefully retire the watchdog (e.g., once Valve ships an official template).",
		Long: `retire is the happy-path exit for this tool.

It assumes some upstream change (Valve shipping an official template at the
same filename, you switching to a different controller, etc.) has made the
watchdog unnecessary. Specifically:

  1. For every embedded template, adopt whatever is currently on disk as the
     new baseline (` + "StrategyAcceptExisting" + `). This records the on-disk
     hash so any future apply run sees no conflict — your file (or Valve's
     replacement) is treated as the authoritative version going forward.
     The on-disk file is NOT modified.
  2. Disable + remove the OS-level trigger (same as ` + "`sspt uninstall`" + `).
  3. Print a goodbye message.

The state file and (by default) the installed binary are left in place so
` + "`sspt status`" + ` still works for verification. Pass --purge to also remove
the installed binary.

This is a one-way action; --yes skips the confirmation prompt.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := logging.Setup(flagVerbose); err != nil {
				return fmt.Errorf("logging setup: %w", err)
			}
			if !yes {
				fmt.Print("Retire sspt watchdog? Templates on disk will be adopted as baseline\n")
				fmt.Print("and the OS trigger will be removed. Continue? [y/N]: ")
				var ans string
				fmt.Scanln(&ans)
				if ans != "y" && ans != "Y" && ans != "yes" {
					fmt.Println("aborted")
					return nil
				}
			}

			install, err := resolveInstall(steamPath)
			if err != nil {
				return err
			}
			st, err := state.Load()
			if err != nil {
				return fmt.Errorf("load state: %w", err)
			}
			templates, err := template.All()
			if err != nil {
				return fmt.Errorf("enumerate embedded templates: %w", err)
			}

			fmt.Println("Adopting on-disk content as baseline:")
			for _, t := range templates {
				dst := filepath.Join(install.TemplatesDir(), t.Filename)
				if _, err := os.Stat(dst); os.IsNotExist(err) {
					fmt.Printf("  %-55s ABSENT (skipping)\n", t.Filename)
					continue
				}
				res, err := template.Apply(install, st, t, template.StrategyAcceptExisting, Version)
				if err != nil {
					return fmt.Errorf("accept-existing %s: %w", t.Filename, err)
				}
				fmt.Printf("  %-55s %s\n", res.Filename, res.Outcome)
			}
			if err := st.Save(); err != nil {
				return fmt.Errorf("save state: %w", err)
			}

			fmt.Println("\nRemoving OS trigger...")
			report, err := installer.New().Uninstall(purge)
			if err != nil {
				return err
			}
			for _, n := range report.Notes {
				fmt.Println("  " + n)
			}

			fmt.Println()
			fmt.Println("✓ sspt retired. Thanks for using us — and thanks Valve (if applicable)")
			fmt.Println("  for shipping an official template. The on-disk content is now the")
			fmt.Println("  authoritative version; this tool will not touch it.")
			if !purge {
				fmt.Println()
				fmt.Println("  Binary and state file left in place. To remove them too:")
				fmt.Println("    sspt retire --purge --yes")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&steamPath, "steam-path", "", "override Steam install location")
	cmd.Flags().BoolVar(&purge, "purge", false, "also remove the installed binary directory")
	cmd.Flags().BoolVar(&yes, "yes", false, "skip the confirmation prompt")
	return cmd
}

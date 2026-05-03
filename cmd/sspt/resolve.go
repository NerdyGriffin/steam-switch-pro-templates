package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/NerdyGriffin/steam-switch-pro-templates/internal/logging"
	"github.com/NerdyGriffin/steam-switch-pro-templates/internal/state"
	"github.com/NerdyGriffin/steam-switch-pro-templates/internal/template"
	"github.com/spf13/cobra"
)

func newResolveCmd() *cobra.Command {
	var (
		steamPath  string
		nonInteractive string
	)
	cmd := &cobra.Command{
		Use:   "resolve",
		Short: "Interactively resolve template conflicts (unrecognized on-disk content).",
		Long: `resolve walks every template that ` + "`sspt apply`" + ` flagged as a conflict
(unknown content on disk that we won't overwrite) and lets you choose:

  k  keep what's currently on disk and adopt it as the new baseline
     (use this if Valve shipped an official template at the same filename
     and you want to switch to it; the watchdog will leave the disk file
     alone going forward)
  o  overwrite with our embedded copy
     (use this if the on-disk content is junk or stale)
  s  skip — leave it as a conflict, decide later

You can also pass --apply k|o to make the choice non-interactively.

After --apply o (or interactive 'o'), consider running ` + "`sspt status`" + ` to
confirm. After --apply k (or interactive 'k'), the watchdog effectively
retires for that one template — see also ` + "`sspt retire`" + ` to also remove
the trigger entirely.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := logging.Setup(flagVerbose); err != nil {
				return fmt.Errorf("logging setup: %w", err)
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

			conflicts := []template.Template{}
			for _, t := range templates {
				dst := filepath.Join(install.TemplatesDir(), t.Filename)
				diskBytes, err := os.ReadFile(dst)
				if err != nil {
					continue // absent or unreadable — `apply` handles install, not us
				}
				diskHash := template.HashBytes(diskBytes)
				if diskHash == t.Hash {
					continue // matches embedded — no conflict
				}
				prev, hadPrev := st.Templates[t.Filename]
				if hadPrev && diskHash == prev.InstalledHash {
					continue // matches our previous install (apply will upgrade)
				}
				conflicts = append(conflicts, t)
			}

			if len(conflicts) == 0 {
				fmt.Println("No conflicts to resolve.")
				return nil
			}

			fmt.Printf("%d conflict(s) detected.\n\n", len(conflicts))

			reader := bufio.NewReader(os.Stdin)
			for _, t := range conflicts {
				dst := filepath.Join(install.TemplatesDir(), t.Filename)
				diskBytes, _ := os.ReadFile(dst)
				diskHash := template.HashBytes(diskBytes)
				prev, hadPrev := st.Templates[t.Filename]

				fmt.Printf("Template: %s\n", t.Filename)
				fmt.Printf("  path:           %s\n", dst)
				fmt.Printf("  embedded sha:   %s\n", short(t.Hash))
				fmt.Printf("  on-disk sha:    %s   (%d bytes)\n", short(diskHash), len(diskBytes))
				if hadPrev {
					fmt.Printf("  last installed: v%s on %s\n", prev.InstalledVersion, prev.InstalledAt.Format("2006-01-02 15:04 UTC"))
					if prev.ConflictSeenAt != nil {
						fmt.Printf("  conflict first seen: %s\n", prev.ConflictSeenAt.Format("2006-01-02 15:04 UTC"))
					}
				}

				choice := strings.ToLower(strings.TrimSpace(nonInteractive))
				if choice == "" {
					fmt.Print("\nChoice — [k]eep on-disk / [o]verwrite with embedded / [s]kip: ")
					line, err := reader.ReadString('\n')
					if err != nil {
						return fmt.Errorf("read input: %w", err)
					}
					choice = strings.ToLower(strings.TrimSpace(line))
				}

				switch choice {
				case "k", "keep":
					if _, err := template.Apply(install, st, t, template.StrategyAcceptExisting, Version); err != nil {
						return fmt.Errorf("accept-existing: %w", err)
					}
					fmt.Println("  → adopted on-disk content as new baseline")
				case "o", "overwrite":
					if _, err := template.Apply(install, st, t, template.StrategyForce, Version); err != nil {
						return fmt.Errorf("force overwrite: %w", err)
					}
					fmt.Println("  → overwrote with embedded copy")
				default:
					fmt.Println("  → skipped, conflict remains recorded")
				}
				fmt.Println()
			}

			if err := st.Save(); err != nil {
				return fmt.Errorf("save state: %w", err)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&steamPath, "steam-path", "", "override Steam install location")
	cmd.Flags().StringVar(&nonInteractive, "apply", "", "non-interactive choice for ALL conflicts: k (keep) or o (overwrite)")
	return cmd
}

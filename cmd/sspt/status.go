package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/NerdyGriffin/steam-switch-pro-templates/internal/state"
	"github.com/NerdyGriffin/steam-switch-pro-templates/internal/steam"
	"github.com/NerdyGriffin/steam-switch-pro-templates/internal/template"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	var steamPath string
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show detected Steam install and per-template state.",
		RunE: func(cmd *cobra.Command, args []string) error {
			install, err := resolveInstall(steamPath)
			if err != nil {
				fmt.Println("Steam install: NOT FOUND")
				fmt.Println("  hint: pass --steam-path /path/to/Steam if installed in a non-standard location")
				return nil
			}
			fmt.Println("Steam install:")
			fmt.Println("  root:          ", install.Root)
			fmt.Println("  templates dir: ", install.TemplatesDir())

			statePath, _ := state.Path()
			fmt.Println("\nState file:    ", statePath)
			st, err := state.Load()
			if err != nil {
				return fmt.Errorf("load state: %w", err)
			}

			templates, err := template.All()
			if err != nil {
				return fmt.Errorf("enumerate embedded templates: %w", err)
			}

			fmt.Printf("\nTemplates (%d embedded):\n", len(templates))
			for _, t := range templates {
				printTemplateStatus(install, st, t)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&steamPath, "steam-path", "", "override Steam install location")
	return cmd
}

func printTemplateStatus(install *steam.Install, st *state.State, t template.Template) {
	dst := filepath.Join(install.TemplatesDir(), t.Filename)
	fmt.Printf("  %s\n", t.Filename)
	fmt.Printf("    embedded hash: %s\n", short(t.Hash))

	diskBytes, err := os.ReadFile(dst)
	if os.IsNotExist(err) {
		fmt.Printf("    disk:          ABSENT\n")
		return
	}
	if err != nil {
		fmt.Printf("    disk:          ERROR (%v)\n", err)
		return
	}
	diskHash := template.HashBytes(diskBytes)
	fmt.Printf("    disk hash:     %s\n", short(diskHash))

	prev, hadPrev := st.Templates[t.Filename]
	switch {
	case diskHash == t.Hash:
		fmt.Printf("    status:        OK (matches embedded)\n")
	case hadPrev && diskHash == prev.InstalledHash:
		fmt.Printf("    status:        OUTDATED (matches our previous install — apply will upgrade)\n")
	case hadPrev && prev.ConflictSeenAt != nil:
		fmt.Printf("    status:        CONFLICT (recorded %s) — run `sspt resolve`\n", prev.ConflictSeenAt.Format("2006-01-02 15:04 UTC"))
	default:
		fmt.Printf("    status:        UNRECOGNIZED CONTENT — run `sspt apply` to record conflict\n")
	}

	if hadPrev {
		fmt.Printf("    last installed: v%s on %s\n", prev.InstalledVersion, prev.InstalledAt.Format("2006-01-02 15:04 UTC"))
	}
}

func short(h string) string {
	if len(h) > 16 {
		return h[:16] + "…"
	}
	return h
}

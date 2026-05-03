// Command sspt installs and persists missing Steam Input controller templates
// across Steam client updates. See https://github.com/NerdyGriffin/steam-switch-pro-templates.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version is set at build time via -ldflags "-X main.Version=...".
var Version = "0.1.0-dev"

var (
	flagVerbose bool
)

func main() {
	root := &cobra.Command{
		Use:           "sspt",
		Short:         "Install and persist Steam Input controller templates across Steam updates.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       Version,
	}
	root.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "enable debug logging")

	root.AddCommand(
		newApplyCmd(),
		newStatusCmd(),
		newInstallCmd(),
		newUninstallCmd(),
		newResolveCmd(),
	)

	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

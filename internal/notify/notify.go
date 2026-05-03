// Package notify wraps cross-platform desktop notifications (beeep) with a
// no-op fallback so the CLI never fails just because the user is on a
// headless box or notifications are disabled.
package notify

import (
	"fmt"
	"log/slog"

	"github.com/gen2brain/beeep"
)

// Conflict fires a desktop notification telling the user a template conflict
// was detected — the body tells them to run `sspt resolve`. Notification
// failures are logged but never propagated; the apply command's correctness
// is not contingent on a working notification daemon.
func Conflict(filename string) {
	title := "sspt: template conflict"
	body := fmt.Sprintf(
		"`%s` on disk doesn't match what we shipped or last installed.\n"+
			"Run `sspt resolve` to choose which version to keep.",
		filename,
	)
	if err := beeep.Notify(title, body, ""); err != nil {
		slog.Warn("desktop notification failed (non-fatal)", "err", err)
	}
}

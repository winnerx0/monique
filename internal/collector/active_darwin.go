//go:build darwin

package collector

import (
	"context"
	"os/exec"
	"strings"

	"monique/internal/domain"
)

// One AppleScript round-trip: the frontmost process name (always available)
// and its front window's title (requires Accessibility permission; comes back
// empty without it, which we tolerate). Output is two lines: name, then title.
const frontmostScript = `
tell application "System Events"
	set procName to name of first application process whose frontmost is true
	set winTitle to ""
	try
		set winTitle to name of front window of (first application process whose frontmost is true)
	end try
end tell
return procName & linefeed & winTitle
`

// activeWindow returns the frontmost app and its window title on macOS.
// ok is false only when osascript fails entirely.
func activeWindow(ctx context.Context) (domain.FocusEvent, bool, error) {
	out, err := exec.CommandContext(ctx, "osascript", "-e", frontmostScript).Output()
	if err != nil {
		return domain.FocusEvent{}, false, nil // login window, no session, etc.
	}
	lines := strings.SplitN(strings.TrimRight(string(out), "\n"), "\n", 2)
	app := strings.TrimSpace(lines[0])
	if app == "" {
		return domain.FocusEvent{}, false, nil
	}
	var title string
	if len(lines) > 1 {
		title = strings.TrimSpace(lines[1])
	}
	return domain.FocusEvent{AppClass: app, Title: title}, true, nil
}

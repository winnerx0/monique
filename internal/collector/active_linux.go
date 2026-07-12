//go:build linux

package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"monique/internal/domain"
)

// activeWindow returns the currently-focused window on Linux. It prefers
// Hyprland's IPC (when running under Hyprland) and otherwise falls back to
// X11 via xprop. ok is false when nothing is focused.
func activeWindow(ctx context.Context) (domain.FocusEvent, bool, error) {
	if os.Getenv("HYPRLAND_INSTANCE_SIGNATURE") != "" {
		return hyprlandActiveWindow(ctx)
	}
	return x11ActiveWindow(ctx)
}

type hyprWindow struct {
	Class string `json:"class"`
	Title string `json:"title"`
	PID   int    `json:"pid"`
}

func hyprlandActiveWindow(ctx context.Context) (domain.FocusEvent, bool, error) {
	out, err := exec.CommandContext(ctx, "hyprctl", "-j", "activewindow").Output()
	if err != nil {
		return domain.FocusEvent{}, false, fmt.Errorf("hyprctl activewindow: %w", err)
	}
	s := strings.TrimSpace(string(out))
	if s == "" || s == "{}" {
		return domain.FocusEvent{}, false, nil // empty workspace, nothing focused
	}
	var w hyprWindow
	if err := json.Unmarshal(out, &w); err != nil {
		return domain.FocusEvent{}, false, fmt.Errorf("parse activewindow json: %w", err)
	}
	return domain.FocusEvent{AppClass: w.Class, Title: w.Title, PID: w.PID}, true, nil
}

var xpropStringRe = regexp.MustCompile(`"([^"]*)"`)

func x11ActiveWindow(ctx context.Context) (domain.FocusEvent, bool, error) {
	// _NET_ACTIVE_WINDOW on the root gives the focused window id.
	rootOut, err := exec.CommandContext(ctx, "xprop", "-root", "_NET_ACTIVE_WINDOW").Output()
	if err != nil {
		return domain.FocusEvent{}, false, fmt.Errorf("xprop -root: %w (is xprop installed?)", err)
	}
	fields := strings.Fields(strings.TrimSpace(string(rootOut)))
	winID := fields[len(fields)-1]
	if winID == "" || winID == "0x0" {
		return domain.FocusEvent{}, false, nil // no active window
	}

	propOut, err := exec.CommandContext(ctx, "xprop", "-id", winID, "_NET_WM_NAME", "WM_NAME", "WM_CLASS").Output()
	if err != nil {
		return domain.FocusEvent{}, false, fmt.Errorf("xprop -id %s: %w", winID, err)
	}

	var title, class string
	for line := range strings.SplitSeq(string(propOut), "\n") {
		vals := xpropStringRe.FindAllStringSubmatch(line, -1)
		switch {
		case strings.HasPrefix(line, "_NET_WM_NAME") && len(vals) > 0:
			title = vals[0][1] // prefer UTF-8 _NET_WM_NAME
		case strings.HasPrefix(line, "WM_NAME") && title == "" && len(vals) > 0:
			title = vals[0][1] // fallback to legacy WM_NAME
		case strings.HasPrefix(line, "WM_CLASS") && len(vals) > 0:
			class = vals[len(vals)-1][1] // WM_CLASS is "instance", "class"; take class
		}
	}

	return domain.FocusEvent{AppClass: class, Title: title}, true, nil
}

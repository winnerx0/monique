//go:build windows

package collector

import (
	"context"
	"path/filepath"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"

	"monique/internal/domain"
)

var (
	user32                       = windows.NewLazySystemDLL("user32.dll")
	procGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
	procGetWindowTextW           = user32.NewProc("GetWindowTextW")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
)

// activeWindow returns the foreground window's title and owning executable
// name on Windows. ok is false when there is no foreground window.
func activeWindow(ctx context.Context) (domain.FocusEvent, bool, error) {
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return domain.FocusEvent{}, false, nil
	}

	title := windowText(hwnd)

	var pid uint32
	procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))

	class := processName(pid) // e.g. "chrome.exe"

	return domain.FocusEvent{AppClass: class, Title: title, PID: int(pid)}, true, nil
}

func windowText(hwnd uintptr) string {
	buf := make([]uint16, 512)
	n, _, _ := procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf[:n])
}

func processName(pid uint32) string {
	if pid == 0 {
		return ""
	}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(h)

	buf := make([]uint16, windows.MAX_PATH)
	size := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &size); err != nil {
		return ""
	}
	return filepath.Base(syscall.UTF16ToString(buf[:size]))
}

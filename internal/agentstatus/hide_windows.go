//go:build windows

package agentstatus

import (
	"os/exec"
	"syscall"
)

// hideWindow prevents the console window of elastic-agent.exe from flashing
// in front of the Wails UI every time the status is refreshed.
func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}

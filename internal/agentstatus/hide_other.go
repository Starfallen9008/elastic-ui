//go:build !windows

package agentstatus

import "os/exec"

func hideWindow(cmd *exec.Cmd) {}

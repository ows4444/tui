//go:build unix

package main

import (
	"os/exec"
	"syscall"
)

// ownGroup puts the command in a process group of its own and makes cancelling
// kill the whole group, so a shell's children (a sleep, a build) stop too and
// release the output pipe instead of keeping the run alive.
func ownGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
}

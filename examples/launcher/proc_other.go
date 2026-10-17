//go:build !unix

package main

import "os/exec"

// ownGroup does nothing here: cancelling stops the shell, and WaitDelay lets
// the run end even if a child still holds the output pipe.
func ownGroup(*exec.Cmd) {}

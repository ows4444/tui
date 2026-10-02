//go:build !(linux || darwin || dragonfly || freebsd || netbsd || openbsd)

package tui

// stopSelf is nil where there is no job control: WithSuspendOnCtrlZ is a
// no-op and Ctrl+Z stays an ordinary key.
var stopSelf func() error

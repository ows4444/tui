//go:build linux || darwin || dragonfly || freebsd || netbsd || openbsd

package tui

import (
	"os"
	"os/signal"
	"syscall"
)

// stopSelf stops this process with SIGSTOP and returns after SIGCONT. The
// SIGCONT subscription is made first so a fast `fg` cannot slip past it.
func stopSelf() error {
	cont := make(chan os.Signal, 1)
	signal.Notify(cont, syscall.SIGCONT)
	defer signal.Stop(cont)
	if err := syscall.Kill(os.Getpid(), syscall.SIGSTOP); err != nil {
		return err
	}
	<-cont
	return nil
}

//go:build !windows

package tui

import (
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestSetWindowTitlePopsOnSignalExit(t *testing.T) {
	p, out, errc := startSpinning(t, WithExitOnSignal(false))
	p.Send(SetWindowTitle("x")())
	time.Sleep(50 * time.Millisecond)
	if err := syscall.Kill(syscall.Getpid(), syscall.SIGHUP); err != nil {
		t.Fatal(err)
	}
	if err := waitRun(t, errc); err != ErrInterrupted {
		t.Fatalf("Run = %v, want ErrInterrupted", err)
	}
	if got := string(out.b); !strings.HasSuffix(got, titlePop) {
		t.Fatalf("signal exit must pop the title; tail %q", got[max(0, len(got)-40):])
	}
}

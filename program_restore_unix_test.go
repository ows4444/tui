//go:build !windows

package tui

import (
	"strings"
	"syscall"
	"testing"

	"github.com/ows4444/tui/ansi"
)

// When SIGHUP arrives while frames are rendered continuously, there is no data
// race (run with -race) and no frame bytes follow the restore sequence.
func TestSIGHUPDuringContinuousRenderingIsRaceFreeAndRestoresLast(t *testing.T) {
	_, out, errc := startSpinning(t, WithExitOnSignal(false))
	if err := syscall.Kill(syscall.Getpid(), syscall.SIGHUP); err != nil {
		t.Fatal(err)
	}
	if err := waitRun(t, errc); err != ErrInterrupted {
		t.Fatalf("Run = %v, want ErrInterrupted", err)
	}
	got := string(out.b)
	if !strings.HasSuffix(got, ansi.AltScreenDisable) {
		t.Fatalf("output does not end with the restore sequence; tail %q", got[max(0, len(got)-80):])
	}
	if strings.Count(got, ansi.AltScreenDisable) != 1 {
		t.Fatalf("restore sequence written %d times", strings.Count(got, ansi.AltScreenDisable))
	}
}

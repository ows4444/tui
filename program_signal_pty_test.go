//go:build darwin || linux

package tui

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/ows4444/tui/internal/ptytest"
)

const sigNoExitHelperEnv = "TUI_SIGTERM_NOEXIT_HELPER"

// Helper body: exits 0 only if Run returned ErrInterrupted.
func TestSigtermNoExit_HelperProcess(t *testing.T) {
	if os.Getenv(sigNoExitHelperEnv) != "1" {
		t.Skip("helper process for TestSigtermNoExitReturnsErrInterrupted")
	}
	tty := os.NewFile(3, "tty")
	_, err := NewProgram(ptyModel{}, WithInput(tty), WithOutput(tty), WithExitOnSignal(false)).Run()
	if errors.Is(err, ErrInterrupted) {
		os.Exit(0)
	}
	os.Exit(3)
}

func TestSigtermNoExitReturnsErrInterrupted(t *testing.T) {
	master, slave := openPTY(t, 80, 24)
	_ = master
	before, err := ptytest.Termios(slave)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestSigtermNoExit_HelperProcess$")
	cmd.Env = append(os.Environ(), sigNoExitHelperEnv+"=1")
	cmd.ExtraFiles = []*os.File{slave}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cmd.Process.Kill() })

	deadline := time.Now().Add(3 * time.Second)
	for {
		during, terr := ptytest.Termios(slave)
		if terr == nil && during != before {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("helper never entered raw mode")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	waitErr := make(chan error, 1)
	go func() { waitErr <- cmd.Wait() }()
	select {
	case err := <-waitErr:
		if err != nil {
			t.Fatalf("helper: Run did not return ErrInterrupted (exit err %v)", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("helper did not return after SIGTERM")
	}
	if after, _ := ptytest.Termios(slave); after != before {
		t.Errorf("terminal not restored: got %+v want %+v", after, before)
	}
}

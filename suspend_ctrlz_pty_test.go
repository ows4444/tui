//go:build darwin || linux

package tui

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/internal/ptytest"
)

const ctrlZChildEnv = "TUI_CTRLZ_CHILD"

// Helper body: runs a Program on the pty passed as fd 3 with the option on.
func TestCtrlZChild_HelperProcess(t *testing.T) {
	if os.Getenv(ctrlZChildEnv) != "1" {
		t.Skip("helper process for TestCtrlZRealStopAndContinue")
	}
	tty := os.NewFile(3, "tty")
	_, err := NewProgram(ptyModel{}, WithInput(tty), WithOutput(tty), WithSuspendOnCtrlZ(true),
		WithKeyboard(KeyboardDisambiguate)).Run()
	if err != nil {
		os.Exit(3)
	}
	os.Exit(0)
}

// wsStopped reports a stop by any signal. WaitStatus.Stopped is false on
// darwin for SIGSTOP itself, the very signal under test.
func wsStopped(ws syscall.WaitStatus) bool { return uint32(ws)&0xff == 0x7f }

// waitState polls the child with wait4 until it is stopped (WUNTRACED) or
// continued (WCONTINUED), or exits.
func waitState(t *testing.T, pid int, stopped bool) syscall.WaitStatus {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var ws syscall.WaitStatus
		wpid, err := syscall.Wait4(pid, &ws, syscall.WUNTRACED|syscall.WCONTINUED|syscall.WNOHANG, nil)
		if err != nil {
			t.Fatalf("wait4: %v", err)
		}
		if wpid == pid && (ws.Exited() || ws.Signaled() || (stopped && wsStopped(ws)) || (!stopped && ws.Continued())) {
			return ws
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("child never became stopped=%v", stopped)
	return 0
}

// The criterion end to end with a real SIGSTOP/SIGCONT: Ctrl+Z restores the
// terminal and the child stops; after SIGCONT it is raw again, has re-entered
// its modes, seen the new size and repainted the whole frame.
func TestCtrlZRealStopAndContinue(t *testing.T) {
	master, slave := openPTY(t, 80, 24)
	before, err := ptytest.Termios(slave)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestCtrlZChild_HelperProcess$")
	cmd.Env = append(cmd.Environ(), ctrlZChildEnv+"=1")
	cmd.ExtraFiles = []*os.File{slave}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	t.Cleanup(func() { syscall.Kill(pid, syscall.SIGKILL) })

	mfd := int(master.Fd())
	if err := syscall.SetNonblock(mfd, true); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	pump := func(d time.Duration, stop func() bool) {
		buf := make([]byte, 4096)
		end := time.Now().Add(d)
		for time.Now().Before(end) {
			n, _ := syscall.Read(mfd, buf)
			if n > 0 {
				out.Write(buf[:n])
			}
			if stop != nil && stop() {
				return
			}
			if n <= 0 {
				time.Sleep(2 * time.Millisecond)
			}
		}
	}
	pump(5*time.Second, func() bool { return strings.Contains(out.String(), "frame:1") })
	if !strings.Contains(out.String(), "frame:1") {
		t.Fatalf("no first frame; output %q", out.String())
	}
	if during, _ := ptytest.Termios(slave); during == before {
		t.Fatal("child not in raw mode")
	}

	if _, err := syscall.Write(mfd, []byte("\x1a")); err != nil {
		t.Fatal(err)
	}
	if ws := waitState(t, pid, true); !wsStopped(ws) {
		t.Fatalf("child did not stop: %v", ws)
	}
	pump(200*time.Millisecond, nil)
	if after, _ := ptytest.Termios(slave); after != before {
		t.Error("terminal not restored while stopped")
	}
	if !strings.HasSuffix(out.String(), ansi.AltScreenDisable) {
		t.Errorf("modes not left before the stop; output %q", out.String())
	}
	mark := out.Len()

	// The window changes while the process is stopped.
	if err := ptytest.SetSize(master, 100, 30); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Kill(pid, syscall.SIGCONT); err != nil {
		t.Fatal(err)
	}
	// (darwin wait4 does not report continue; the repaint below proves it.)
	pump(5*time.Second, func() bool { return strings.Contains(out.String()[mark:], "frame:2") })
	after := out.String()[mark:]
	for _, s := range []string{ansi.AltScreenEnable, ansi.CursorHide, ansi.KittyKeyboardEnableFlags(1), ansi.ClearScreen, "frame:2"} {
		if !strings.Contains(after, s) {
			t.Errorf("after SIGCONT missing %q in %q", s, after)
		}
	}
	if during, _ := ptytest.Termios(slave); during == before {
		t.Error("raw mode not re-entered after SIGCONT")
	}

	// Input works again, and quitting restores the terminal.
	if _, err := syscall.Write(mfd, []byte("q")); err != nil {
		t.Fatal(err)
	}
	ws := waitState(t, pid, false)
	if ws.Continued() {
		// Linux wait4 reports the SIGCONT above first; the exit follows.
		ws = waitState(t, pid, false)
	}
	if !ws.Exited() || ws.ExitStatus() != 0 {
		t.Fatalf("child exit: %v", ws)
	}
	if end, _ := ptytest.Termios(slave); end != before {
		t.Error("terminal not restored after quit")
	}
}

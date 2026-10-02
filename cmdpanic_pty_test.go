//go:build darwin || linux

package tui

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ows4444/tui/ansi"
)

type cmdPanicModel struct{ late bool }

func (m cmdPanicModel) Init() Cmd {
	if m.late {
		// Quit now; a second Cmd panics well after Run has returned.
		return Batch(Quit(), func() Msg {
			time.Sleep(500 * time.Millisecond)
			panic("boom-from-cmd")
		})
	}
	return func() Msg { panic("boom-from-cmd") }
}
func (m cmdPanicModel) Update(Msg) (Model, Cmd) { return m, nil }
func (cmdPanicModel) View() string              { return "x" }

// TestCmdPanicHelper is the helper process: it runs a Program whose Init Cmd
// panics. It does nothing unless launched by runCmdPanicHelper.
func TestCmdPanicHelper(t *testing.T) {
	mode := os.Getenv("TUI_CMD_PANIC_HELPER")
	if mode == "" {
		t.Skip("helper process")
	}
	_, _ = NewProgram(cmdPanicModel{late: mode == "late"}, WithInput(os.Stdin), WithOutput(os.Stdout),
		WithErrOutput(os.Stderr), WithMouse(MouseCellMotion)).Run()
	if mode == "late" {
		// Run has returned; keep the process alive so the late panic can
		// surface. A hang is caught by the parent's timeout.
		time.Sleep(10 * time.Second)
	}
}

// runCmdPanicHelper runs the helper with stdout and stderr on one pty, so the
// captured stream preserves the order restore bytes and panic text appear in.
func runCmdPanicHelper(t *testing.T, mode string) string {
	t.Helper()
	master, slave := openPTY(t, 80, 24)
	cmd := exec.Command(os.Args[0], "-test.run=^TestCmdPanicHelper$")
	cmd.Env = append(cmd.Environ(), "TUI_CMD_PANIC_HELPER="+mode)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = slave, slave, slave
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var out bytes.Buffer
	readDone := make(chan struct{})
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := master.Read(buf)
			mu.Lock()
			out.Write(buf[:n])
			mu.Unlock()
			if err != nil {
				close(readDone)
				return
			}
		}
	}()
	waitErr := make(chan error, 1)
	go func() { waitErr <- cmd.Wait() }()
	select {
	case err := <-waitErr:
		if err == nil {
			t.Fatal("helper exited cleanly; expected a panic exit")
		}
	case <-time.After(8 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("helper did not exit: hang after Cmd panic")
	}
	_ = slave.Close() // lets the reader see EOF/EIO
	select {
	case <-readDone:
	case <-time.After(2 * time.Second):
	}
	mu.Lock()
	defer mu.Unlock()
	return out.String()
}

func TestCmdPanicRestoresTerminal(t *testing.T) {
	got := runCmdPanicHelper(t, "1")
	panicAt := strings.Index(got, "panic: boom-from-cmd")
	if panicAt < 0 || !strings.Contains(got, "goroutine") {
		t.Fatalf("output lacks panic value or trace:\n%q", got)
	}
	for name, seq := range map[string]string{
		"alt-screen off": ansi.AltScreenDisable,
		"cursor show":    ansi.CursorShow,
		"mouse off":      ansi.MouseSGRDisable,
		"paste off":      ansi.BracketedPasteDisable, // default-on, restored on the panic path
	} {
		if !strings.Contains(got[:panicAt], seq) {
			t.Errorf("%s sequence %q not written before the panic text; output %q", name, seq, got)
		}
	}
}

func TestCmdPanicAfterRunReturnedDoesNotHang(t *testing.T) {
	got := runCmdPanicHelper(t, "late")
	panicAt := strings.Index(got, "panic: boom-from-cmd")
	if panicAt < 0 || !strings.Contains(got, "goroutine") {
		t.Fatalf("late panic did not surface:\n%q", got)
	}
	if !strings.Contains(got[:panicAt], ansi.AltScreenDisable) {
		t.Errorf("terminal not restored before the panic: %q", got)
	}
}

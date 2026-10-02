//go:build darwin || linux

package term

import (
	"os"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

// These tests drive a real pseudo-terminal (see openPTY), so they exercise
// the actual termios ioctls and line discipline rather than mocks. They
// skip where a pty can't be opened (some sandboxes and CI containers). The
// windows console path has no equivalent harness and is verified by hand.

// endpoint is one end of a pty, driven with non-blocking raw syscalls and a
// polling deadline: no goroutine is ever left blocked in a Read to steal
// bytes from a later check or straddle a change of line discipline.
type endpoint struct {
	f  *os.File
	fd int
}

func newEndpoint(t *testing.T, f *os.File) *endpoint {
	t.Helper()
	fd := int(f.Fd()) // note: Fd() makes f blocking, so set non-blocking after
	if err := syscall.SetNonblock(fd, true); err != nil {
		t.Fatalf("SetNonblock: %v", err)
	}
	return &endpoint{f: f, fd: fd}
}

func newPTY(t *testing.T) (master, slave *endpoint) {
	t.Helper()
	m, s, err := openPTY()
	if err != nil {
		t.Skipf("cannot open a pty here: %v", err)
	}
	t.Cleanup(func() {
		s.Close()
		m.Close()
	})
	return newEndpoint(t, m), newEndpoint(t, s)
}

func (e *endpoint) write(t *testing.T, s string) {
	t.Helper()
	if n, err := syscall.Write(e.fd, []byte(s)); err != nil || n != len(s) {
		t.Fatalf("write %q: n=%d err=%v", s, n, err)
	}
}

// poll reads whatever is available until d passes or stop returns true.
func (e *endpoint) poll(d time.Duration, stop func(got []byte) bool) []byte {
	var got []byte
	buf := make([]byte, 256)
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		n, err := syscall.Read(e.fd, buf)
		if n > 0 {
			got = append(got, buf[:n]...)
			if stop(got) {
				break
			}
			continue
		}
		if err != nil && err != syscall.EAGAIN {
			break // EIO after hangup, etc.
		}
		time.Sleep(2 * time.Millisecond)
	}
	return got
}

// readN reads at least n bytes or fails the test after a timeout, and
// returns everything that arrived.
func (e *endpoint) readN(t *testing.T, n int) string {
	t.Helper()
	got := e.poll(2*time.Second, func(g []byte) bool { return len(g) >= n })
	if len(got) < n {
		t.Fatalf("timed out waiting for %d bytes, have %q", n, got)
	}
	return string(got)
}

// quiet reports whether nothing arrives within d, discarding anything that does.
func (e *endpoint) quiet(d time.Duration) bool {
	return len(e.poll(d, func([]byte) bool { return true })) == 0
}

// drain discards pending output (e.g. echo) until it goes quiet.
func (e *endpoint) drain(t *testing.T) {
	t.Helper()
	for i := 0; i < 20; i++ {
		if e.quiet(60 * time.Millisecond) {
			return
		}
	}
	t.Fatal("terminal output never went quiet")
}

func setWinsize(t *testing.T, master *endpoint, cols, rows uint16) {
	t.Helper()
	ws := winsize{Row: rows, Col: cols}
	if err := ioctl(master.fd, syscall.TIOCSWINSZ, unsafe.Pointer(&ws)); err != nil {
		t.Fatalf("TIOCSWINSZ: %v", err)
	}
}

func getTermios(t *testing.T, e *endpoint) syscall.Termios {
	t.Helper()
	var tm syscall.Termios
	if err := ioctlGetTermios(e.fd, &tm); err != nil {
		t.Fatalf("get termios: %v", err)
	}
	return tm
}

func TestIsTerminal(t *testing.T) {
	_, slave := newPTY(t)
	if !IsTerminal(slave.fd) {
		t.Error("IsTerminal(pty slave) = false, want true")
	}

	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer pr.Close()
	defer pw.Close()
	null, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer null.Close()

	for name, fd := range map[string]int{
		"pipe read end":  int(pr.Fd()),
		"pipe write end": int(pw.Fd()),
		"/dev/null":      int(null.Fd()),
		"invalid":        -1,
	} {
		if IsTerminal(fd) {
			t.Errorf("IsTerminal(%s) = true, want false", name)
		}
	}
}

func TestGetSize(t *testing.T) {
	master, slave := newPTY(t)
	setWinsize(t, master, 100, 30)
	w, h, err := GetSize(slave.fd)
	if err != nil || w != 100 || h != 30 {
		t.Errorf("GetSize = %d, %d, %v; want 100, 30, nil", w, h, err)
	}

	setWinsize(t, master, 41, 13)
	w, h, err = GetSize(slave.fd)
	if err != nil || w != 41 || h != 13 {
		t.Errorf("after resize GetSize = %d, %d, %v; want 41, 13, nil", w, h, err)
	}

	pr, pw, _ := os.Pipe()
	defer pr.Close()
	defer pw.Close()
	if _, _, err := GetSize(int(pr.Fd())); err == nil {
		t.Error("GetSize(pipe) succeeded, want an error")
	}
	if _, _, err := GetSize(-1); err == nil {
		t.Error("GetSize(-1) succeeded, want an error")
	}
}

func TestMakeRawSetsTermiosFlags(t *testing.T) {
	_, slave := newPTY(t)
	fd := slave.fd

	before := getTermios(t, slave)
	// A fresh pty starts cooked; make sure the test would notice a no-op.
	if uint64(before.Lflag)&uint64(syscall.ICANON) == 0 || uint64(before.Lflag)&uint64(syscall.ECHO) == 0 {
		t.Fatalf("expected a cooked starting state, got Lflag %#x", before.Lflag)
	}

	state, err := MakeRaw(fd)
	if err != nil || state == nil {
		t.Fatalf("MakeRaw = %v, %v", state, err)
	}
	tm := getTermios(t, slave)

	off := func(field string, v uint64, flags map[string]uint64) {
		for name, f := range flags {
			if v&f != 0 {
				t.Errorf("%s: %s still set in raw mode (%s = %#x)", field, name, field, v)
			}
		}
	}
	off("Iflag", uint64(tm.Iflag), map[string]uint64{
		"IGNBRK": uint64(syscall.IGNBRK), "BRKINT": uint64(syscall.BRKINT), "PARMRK": uint64(syscall.PARMRK),
		"ISTRIP": uint64(syscall.ISTRIP), "INLCR": uint64(syscall.INLCR), "IGNCR": uint64(syscall.IGNCR),
		"ICRNL": uint64(syscall.ICRNL), "IXON": uint64(syscall.IXON),
	})
	off("Oflag", uint64(tm.Oflag), map[string]uint64{"OPOST": uint64(syscall.OPOST)})
	off("Lflag", uint64(tm.Lflag), map[string]uint64{
		"ECHO": uint64(syscall.ECHO), "ECHONL": uint64(syscall.ECHONL), "ICANON": uint64(syscall.ICANON),
		"ISIG": uint64(syscall.ISIG), "IEXTEN": uint64(syscall.IEXTEN),
	})
	off("Cflag", uint64(tm.Cflag), map[string]uint64{"PARENB": uint64(syscall.PARENB)})

	if got := uint64(tm.Cflag) & uint64(syscall.CSIZE); got != uint64(syscall.CS8) {
		t.Errorf("character size = %#x, want CS8 (%#x)", got, syscall.CS8)
	}
	if tm.Cc[syscall.VMIN] != 1 || tm.Cc[syscall.VTIME] != 0 {
		t.Errorf("VMIN, VTIME = %d, %d; want 1, 0", tm.Cc[syscall.VMIN], tm.Cc[syscall.VTIME])
	}
}

func TestRawInputIsImmediateUnmodifiedAndNotEchoed(t *testing.T) {
	master, slave := newPTY(t)
	fd := slave.fd

	// Cooked baseline: CR becomes NL, input is line-buffered, and it echoes.
	master.write(t, "x\r")
	if got := slave.readN(t, 2); got != "x\n" {
		t.Fatalf("cooked mode read %q, want %q (CR translated to NL)", got, "x\n")
	}
	if master.quiet(150 * time.Millisecond) {
		t.Fatal("cooked mode did not echo; the echo check below would prove nothing")
	}
	master.drain(t)

	state, err := MakeRaw(fd)
	if err != nil {
		t.Fatal(err)
	}
	defer Restore(fd, state)

	// No newline needed: a single byte is delivered at once.
	master.write(t, "a")
	if got := slave.readN(t, 1); got != "a" {
		t.Errorf("raw read %q, want %q immediately, without a newline", got, "a")
	}
	// CR is not translated.
	master.write(t, "x\r")
	if got := slave.readN(t, 2); got != "x\r" {
		t.Errorf("raw read %q, want %q (no CR->NL)", got, "x\r")
	}
	// Ctrl-C and Ctrl-Z arrive as data instead of raising signals.
	master.write(t, "\x03\x1a")
	if got := slave.readN(t, 2); got != "\x03\x1a" {
		t.Errorf("raw read %q, want the control bytes as data", got)
	}
	// And nothing was echoed back for any of it.
	if !master.quiet(200 * time.Millisecond) {
		t.Error("raw mode echoed input back")
	}
}

func TestRawOutputNewlinesAreNotTranslated(t *testing.T) {
	master, slave := newPTY(t)
	fd := slave.fd

	slave.write(t, "a\n")
	if got := master.readN(t, 3); got != "a\r\n" {
		t.Fatalf("cooked output %q, want %q (NL -> CR LF)", got, "a\r\n")
	}

	state, err := MakeRaw(fd)
	if err != nil {
		t.Fatal(err)
	}
	defer Restore(fd, state)

	slave.write(t, "a\n")
	if got := master.readN(t, 2); got != "a\n" {
		t.Errorf("raw output %q, want %q (no NL -> CR LF)", got, "a\n")
	}
}

func TestRestoreReturnsToCookedMode(t *testing.T) {
	master, slave := newPTY(t)
	fd := slave.fd
	before := getTermios(t, slave)

	state, err := MakeRaw(fd)
	if err != nil {
		t.Fatal(err)
	}
	if getTermios(t, slave) == before {
		t.Fatal("MakeRaw changed nothing")
	}
	if err := Restore(fd, state); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if after := getTermios(t, slave); after != before {
		t.Errorf("termios after Restore differs from the original:\n got  %+v\n want %+v", after, before)
	}

	// Behaviour is cooked again: line-buffered with CR -> NL, and echo.
	master.write(t, "x\r")
	if got := slave.readN(t, 2); got != "x\n" {
		t.Errorf("after Restore read %q, want %q", got, "x\n")
	}
	if master.quiet(150 * time.Millisecond) {
		t.Error("echo did not return after Restore")
	}
}

func TestMakeRawOnNonTerminal(t *testing.T) {
	pr, pw, _ := os.Pipe()
	defer pr.Close()
	defer pw.Close()
	null, _ := os.Open(os.DevNull)
	defer null.Close()

	for name, fd := range map[string]int{"pipe": int(pr.Fd()), "/dev/null": int(null.Fd()), "invalid": -1} {
		state, err := MakeRaw(fd)
		if err == nil || state != nil {
			t.Errorf("MakeRaw(%s) = %v, %v; want nil state and an error", name, state, err)
		}
	}
}

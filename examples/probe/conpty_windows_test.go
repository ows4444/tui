//go:build windows

package main

// This test drives the real probe binary inside a Windows pseudo console
// (ConPTY) and asserts that an arrow key and Ctrl+C are decoded, that a
// resize reaches the program, and that the console input and output modes
// after exit equal the modes before start.
//
// The console modes belong to the pseudo console, not to the test process, so
// the test re-executes its own binary inside the ConPTY as a helper
// (TestConPTYHelper). The helper records the modes, runs the probe on the same
// console, records the modes again, and prints both for the parent to compare.
//
// Everything here uses only package syscall and kernel32; no dependency.

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"github.com/ows4444/tui/ansi"
)

const (
	helperVar   = "TUI_PROBE_CONPTY_HELPER"
	probeExeVar = "TUI_PROBE_CONPTY_EXE"

	procThreadAttributePseudoConsole = 0x00020016
	extendedStartupInfoPresent       = 0x00080000
)

var (
	modKernel32                     = syscall.NewLazyDLL("kernel32.dll")
	procCreatePseudoConsole         = modKernel32.NewProc("CreatePseudoConsole")
	procResizePseudoConsole         = modKernel32.NewProc("ResizePseudoConsole")
	procClosePseudoConsole          = modKernel32.NewProc("ClosePseudoConsole")
	procInitProcThreadAttributeList = modKernel32.NewProc("InitializeProcThreadAttributeList")
	procUpdateProcThreadAttribute   = modKernel32.NewProc("UpdateProcThreadAttribute")
	procDeleteProcThreadAttrList    = modKernel32.NewProc("DeleteProcThreadAttributeList")
	procCreateProcessW              = modKernel32.NewProc("CreateProcessW")
	procGetConsoleModeT             = modKernel32.NewProc("GetConsoleMode")
)

func consoleMode(h syscall.Handle) (uint32, error) {
	var mode uint32
	r, _, err := procGetConsoleModeT.Call(uintptr(h), uintptr(unsafe.Pointer(&mode)))
	if r == 0 {
		return 0, err
	}
	return mode, nil
}

// TestConPTYHelper is not a test: the parent re-executes this binary inside
// the pseudo console with helperVar set. It reports the console modes around
// one run of the probe.
func TestConPTYHelper(t *testing.T) {
	if os.Getenv(helperVar) == "" {
		t.Skip("helper for TestProbeUnderConPTY; runs only inside the pseudo console")
	}
	in, out := syscall.Handle(os.Stdin.Fd()), syscall.Handle(os.Stdout.Fd())
	inBefore, err := consoleMode(in)
	if err != nil {
		t.Fatalf("stdin is not a console: %v", err)
	}
	outBefore, err := consoleMode(out)
	if err != nil {
		t.Fatalf("stdout is not a console: %v", err)
	}
	cmd := exec.Command(os.Getenv(probeExeVar))
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("probe: %v", err)
	}
	inAfter, _ := consoleMode(in)
	outAfter, _ := consoleMode(out)
	fmt.Printf("\nCONMODE in_before=%08x in_after=%08x out_before=%08x out_after=%08x\n",
		inBefore, inAfter, outBefore, outAfter)
}

func coord(x, y int) uintptr { return uintptr(uint16(x)) | uintptr(uint16(y))<<16 }

// lockedBuf is a bytes.Buffer safe for one writer and one polling reader.
type lockedBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (l *lockedBuf) write(p []byte) { l.mu.Lock(); l.b.Write(p); l.mu.Unlock() }
func (l *lockedBuf) text() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return ansi.StripANSI(l.b.String())
}

func waitFor(t *testing.T, buf *lockedBuf, what string, d time.Duration) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if strings.Contains(buf.text(), what) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %q; output so far:\n%s", what, buf.text())
}

func TestProbeUnderConPTY(t *testing.T) {
	dir := t.TempDir()
	probeExe := filepath.Join(dir, "probe.exe")
	if out, err := exec.Command("go", "build", "-o", probeExe, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build probe: %v\n%s", err, out)
	}
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(helperVar, "1")
	t.Setenv(probeExeVar, probeExe)

	// Pipes: the test writes keystrokes to inW and reads screen output from outR.
	var inR, inW, outR, outW syscall.Handle
	if err := syscall.CreatePipe(&inR, &inW, nil, 0); err != nil {
		t.Fatal(err)
	}
	if err := syscall.CreatePipe(&outR, &outW, nil, 0); err != nil {
		t.Fatal(err)
	}
	var hpc uintptr
	if r, _, err := procCreatePseudoConsole.Call(coord(80, 30), uintptr(inR), uintptr(outW), 0, uintptr(unsafe.Pointer(&hpc))); r != 0 {
		t.Fatalf("CreatePseudoConsole: hresult %#x (%v)", r, err)
	}
	// The pseudo console owns these ends now.
	_ = syscall.CloseHandle(inR)
	_ = syscall.CloseHandle(outW)

	var buf lockedBuf
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		chunk := make([]byte, 4096)
		for {
			var n uint32
			if err := syscall.ReadFile(outR, chunk, &n, nil); err != nil || n == 0 {
				return
			}
			buf.write(chunk[:n])
		}
	}()

	// Attribute list carrying the pseudo console.
	var size uintptr
	_, _, _ = procInitProcThreadAttributeList.Call(0, 1, 0, uintptr(unsafe.Pointer(&size)))
	attrs := make([]byte, size)
	if r, _, err := procInitProcThreadAttributeList.Call(uintptr(unsafe.Pointer(&attrs[0])), 1, 0, uintptr(unsafe.Pointer(&size))); r == 0 {
		t.Fatalf("InitializeProcThreadAttributeList: %v", err)
	}
	defer procDeleteProcThreadAttrList.Call(uintptr(unsafe.Pointer(&attrs[0])))
	if r, _, err := procUpdateProcThreadAttribute.Call(uintptr(unsafe.Pointer(&attrs[0])), 0, procThreadAttributePseudoConsole, hpc, unsafe.Sizeof(hpc), 0, 0); r == 0 {
		t.Fatalf("UpdateProcThreadAttribute: %v", err)
	}

	type startupInfoEx struct {
		syscall.StartupInfo
		attributeList uintptr
	}
	var si startupInfoEx
	si.Cb = uint32(unsafe.Sizeof(si))
	si.attributeList = uintptr(unsafe.Pointer(&attrs[0]))
	var pi syscall.ProcessInformation
	cmdline, err := syscall.UTF16PtrFromString(fmt.Sprintf(`"%s" -test.run=^TestConPTYHelper$ -test.count=1`, self))
	if err != nil {
		t.Fatal(err)
	}
	if r, _, err := procCreateProcessW.Call(0, uintptr(unsafe.Pointer(cmdline)), 0, 0, 0, extendedStartupInfoPresent, 0, 0,
		uintptr(unsafe.Pointer(&si)), uintptr(unsafe.Pointer(&pi))); r == 0 {
		t.Fatalf("CreateProcessW: %v", err)
	}
	defer syscall.CloseHandle(pi.Process)
	defer syscall.CloseHandle(pi.Thread)

	send := func(s string) {
		t.Helper()
		var n uint32
		if err := syscall.WriteFile(inW, []byte(s), &n, nil); err != nil {
			t.Fatalf("write %q: %v", s, err)
		}
	}

	waitFor(t, &buf, "Last key:", 60*time.Second) // the probe has rendered
	send("\x1b[A")                                // Up arrow, as Windows Terminal sends it
	time.Sleep(500 * time.Millisecond)            // the diff renderer may split the line; the exit summary is what is asserted

	if r, _, err := procResizePseudoConsole.Call(hpc, coord(100, 40)); r != 0 {
		t.Fatalf("ResizePseudoConsole: hresult %#x (%v)", r, err)
	}
	time.Sleep(1500 * time.Millisecond) // the resize watcher polls every 250ms

	send("\x03") // Ctrl+C
	ev, _ := syscall.WaitForSingleObject(pi.Process, 30000)
	if ev != syscall.WAIT_OBJECT_0 {
		t.Fatalf("probe did not exit after Ctrl+C (wait=%d); output:\n%s", ev, buf.text())
	}
	var code uint32
	_ = syscall.GetExitCodeProcess(pi.Process, &code)
	_, _, _ = procClosePseudoConsole.Call(hpc)
	select {
	case <-readDone:
	case <-time.After(10 * time.Second):
	}
	_ = syscall.CloseHandle(inW)
	_ = syscall.CloseHandle(outR)

	out := buf.text()
	t.Logf("screen output:\n%s", out)
	if code != 0 {
		t.Errorf("helper exit code = %d", code)
	}
	for _, want := range []string{"probe: keys=up,ctrl+c", "probe: size=100x40"} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q", want)
		}
	}
	m := regexp.MustCompile(`CONMODE in_before=(\w+) in_after=(\w+) out_before=(\w+) out_after=(\w+)`).FindStringSubmatch(out)
	if m == nil {
		t.Fatal("helper did not report console modes")
	}
	if m[1] != m[2] {
		t.Errorf("console input mode after exit %s != before start %s", m[2], m[1])
	}
	if m[3] != m[4] {
		t.Errorf("console output mode after exit %s != before start %s", m[4], m[3])
	}
}

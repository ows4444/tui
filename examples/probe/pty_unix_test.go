//go:build darwin || linux

package main

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/internal/ptytest"
)

// The cluster check that runs before the Program reads the terminal's answers
// from stdin. On a terminal that never answers, its reader must be gone when
// the Program starts: every key typed afterwards reaches the Program.
func TestKeysAfterTheClusterCheckReachTheProgram(t *testing.T) {
	exe := filepath.Join(t.TempDir(), "probe")
	if out, err := exec.Command("go", "build", "-o", exe, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	master, slave, err := ptytest.Open()
	if err != nil {
		t.Skipf("cannot open a pty here: %v", err)
	}
	defer master.Close()
	if err := ptytest.SetSize(master, 100, 40); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = slave, slave, slave
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	slave.Close()
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()
	t.Cleanup(func() { _ = cmd.Process.Kill() })

	mfd := int(master.Fd())
	if err := syscall.SetNonblock(mfd, true); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	text := func() string { return ansi.StripANSI(out.String()) }
	pump := func(d time.Duration, done func() bool) bool {
		buf := make([]byte, 4096)
		for end := time.Now().Add(d); time.Now().Before(end); {
			if n, _ := syscall.Read(mfd, buf); n > 0 {
				out.Write(buf[:n])
				continue
			}
			if done() {
				return true
			}
			time.Sleep(2 * time.Millisecond)
		}
		return done()
	}
	shows := func(s string) func() bool { return func() bool { return strings.Contains(text(), s) } }

	// The test never answers the cluster check's queries, like a silent terminal.
	if !pump(20*time.Second, shows("Last key:")) {
		t.Fatalf("the probe never drew its screen; output:\n%s", text())
	}
	for _, key := range []string{"x", "y", "q"} {
		if _, err := master.Write([]byte(key)); err != nil {
			t.Fatal(err)
		}
		pump(150*time.Millisecond, func() bool { return false })
	}
	select {
	case <-exited:
	case <-time.After(5 * time.Second):
		t.Fatalf("the probe did not quit on q; output:\n%s", text())
	}
	pump(200*time.Millisecond, func() bool { return false })
	if !strings.Contains(text(), "probe: keys=x,y,q") {
		t.Fatalf("keys were lost before the Program got them; output tail:\n%s", text()[max(0, len(text())-400):])
	}
}

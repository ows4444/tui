//go:build darwin || linux

package tui

import (
	"bufio"
	"errors"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/ows4444/tui/internal/ptytest"
)

const suspendChildEnv = "TUI_SUSPEND_CHILD"

// Helper body: reads one line from stdin (the pty, cooked while suspended)
// and prints it. Skipped in an ordinary test run.
func TestSuspendChild_HelperProcess(t *testing.T) {
	if os.Getenv(suspendChildEnv) != "1" {
		t.Skip("helper process for TestSuspendChildReadsAllTypedBytes")
	}
	os.Stdout.WriteString("ready\n")
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	os.Stdout.WriteString("got:" + strings.TrimSpace(line) + "\n")
	os.Exit(0)
}

// #31: a child started by Suspend reads every byte typed while it runs, and
// the app's reader takes none of them. The bytes are typed before the child
// reads, so a reader that was still alive would have taken them.
func TestSuspendChildReadsAllTypedBytes(t *testing.T) {
	master, slave := openPTY(t, 80, 24)
	typed, fnDone := make(chan struct{}), make(chan struct{})
	var childOut string
	var fnErr error
	fn := func() error {
		defer close(fnDone)
		cmd := exec.Command(os.Args[0], "-test.run=^TestSuspendChild_HelperProcess$")
		cmd.Env = append(os.Environ(), suspendChildEnv+"=1")
		cmd.Stdin, cmd.Stderr = slave, os.Stderr
		out, err := cmd.StdoutPipe()
		if err != nil {
			return err
		}
		if err := cmd.Start(); err != nil {
			return err
		}
		sc := bufio.NewScanner(out)
		sc.Scan() // "ready"
		close(typed)
		time.Sleep(100 * time.Millisecond) // let a stray reader take them
		for sc.Scan() {
			childOut += sc.Text()
		}
		fnErr = cmd.Wait()
		return fnErr
	}
	m := suspendRec{mu: &sync.Mutex{}, keys: new([]rune), fn: fn}
	s := attachPTY(t, master, slave, m)
	s.waitFor("x")
	s.send("e")
	select {
	case <-typed:
	case <-time.After(5 * time.Second):
		t.Fatal("child never became ready")
	}
	s.send("hello\n")
	select {
	case <-fnDone:
	case <-time.After(5 * time.Second):
		t.Fatal("Suspend never returned")
	}
	s.pump(300*time.Millisecond, nil)
	s.send("q")
	res := s.finish()
	if res.err != nil {
		t.Fatalf("Run: %v", res.err)
	}
	if fnErr != nil {
		t.Fatalf("child: %v", fnErr)
	}
	if childOut != "got:hello" {
		t.Errorf("child output %q, want %q", childOut, "got:hello")
	}
	if k := m.got(); k != "eq" {
		t.Errorf("app saw keys %q, want %q (it stole or lost bytes)", k, "eq")
	}
	if after, _ := ptytest.Termios(s.slave); after != s.before {
		t.Errorf("terminal not restored after Run")
	}
}

// #30: SIGTERM while a Suspend's fn is running restores the terminal without
// racing suspend's termState update (run under -race), and suspend does not
// put the terminal back into raw mode afterwards.
func TestSigtermDuringSuspendRestoresWithoutRace(t *testing.T) {
	master, slave := openPTY(t, 80, 24)
	inFn, release := make(chan struct{}), make(chan struct{})
	fn := func() error {
		close(inFn)
		<-release
		return nil
	}
	m := suspendRec{mu: &sync.Mutex{}, keys: new([]rune), fn: fn}
	s := attachPTY(t, master, slave, m, WithExitOnSignal(false))
	s.waitFor("x")
	s.send("e")
	select {
	case <-inFn:
	case <-time.After(5 * time.Second):
		t.Fatal("suspend never ran fn")
	}
	if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)
	close(release)
	res := s.finish()
	if !errors.Is(res.err, ErrInterrupted) {
		t.Fatalf("Run err = %v, want ErrInterrupted", res.err)
	}
	if after, _ := ptytest.Termios(s.slave); after != s.before {
		t.Errorf("terminal left in raw mode after SIGTERM during suspend")
	}
}

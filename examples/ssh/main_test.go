package main

import (
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/internal/vtscreen"
)

// client is an SSH client with a pty whose output is drawn on an emulated
// screen.
type client struct {
	t      *testing.T
	sess   *ssh.Session
	stdin  io.WriteCloser
	mu     sync.Mutex
	screen *vtscreen.Screen
	done   chan error
}

// dial starts the server on a free port and connects a client with a pty of
// cols x rows.
func dial(t *testing.T, cols, rows int) *client {
	t.Helper()
	cfg, err := serverConfig()
	if err != nil {
		t.Fatal(err)
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = l.Close() })
	go func() { _ = serve(l, cfg, func(remote string) tui.Model { return newGreeter(remote) }) }()
	conn, err := ssh.Dial("tcp", l.Addr().String(), &ssh.ClientConfig{
		User: "demo", HostKeyCallback: ssh.InsecureIgnoreHostKey(), // #nosec G106 -- a test against our own server
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	sess, err := conn.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	c := &client{t: t, sess: sess, screen: vtscreen.NewScreen(cols, rows), done: make(chan error, 1)}
	if c.stdin, err = sess.StdinPipe(); err != nil {
		t.Fatal(err)
	}
	out, err := sess.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := sess.RequestPty("xterm-256color", rows, cols, ssh.TerminalModes{}); err != nil {
		t.Fatal(err)
	}
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := out.Read(buf)
			if n > 0 {
				c.mu.Lock()
				c.screen.Write(buf[:n])
				c.mu.Unlock()
			}
			if err != nil {
				return
			}
		}
	}()
	if err := sess.Shell(); err != nil {
		t.Fatal(err)
	}
	go func() { c.done <- sess.Wait() }()
	t.Cleanup(func() { _ = sess.Close() })
	return c
}

func (c *client) text() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return strings.Join(c.screen.Lines(), "\n")
}

func (c *client) waitFor(text string) {
	c.t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(c.text(), text) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	c.t.Fatalf("never saw %q; screen:\n%s", text, c.text())
}

func TestSessionRunsTheProgramWithTheClientsSize(t *testing.T) {
	c := dial(t, 80, 24)
	c.waitFor("tui over SSH")
	c.waitFor("size 80x24")
	c.waitFor("keys 0")
	if !strings.Contains(c.text(), "you are 127.0.0.1:") {
		t.Errorf("the greeting does not name the client:\n%s", c.text())
	}
}

func TestKeysReachTheProgramAndResizesFollow(t *testing.T) {
	c := dial(t, 80, 24)
	c.waitFor("keys 0")
	if _, err := io.WriteString(c.stdin, "abc"); err != nil {
		t.Fatal(err)
	}
	c.waitFor("keys 3")
	if err := c.sess.WindowChange(30, 100); err != nil {
		t.Fatal(err)
	}
	c.waitFor("size 100x30")
}

func TestQuitEndsTheSessionWithStatusZero(t *testing.T) {
	c := dial(t, 80, 24)
	c.waitFor("press q to leave")
	if _, err := io.WriteString(c.stdin, "q"); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-c.done:
		if err != nil {
			t.Fatalf("the session ended with %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the session did not end after q")
	}
}

func TestTwoSessionsAreIndependent(t *testing.T) {
	a := dial(t, 80, 24)
	b := dial(t, 60, 20)
	a.waitFor("size 80x24")
	b.waitFor("size 60x20")
	if _, err := io.WriteString(a.stdin, "xx"); err != nil {
		t.Fatal(err)
	}
	a.waitFor("keys 2")
	if !strings.Contains(b.text(), "keys 0") {
		t.Errorf("a key in one session reached the other:\n%s", b.text())
	}
}

func TestShellWithoutAPtyIsRefused(t *testing.T) {
	cfg, err := serverConfig()
	if err != nil {
		t.Fatal(err)
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	go func() { _ = serve(l, cfg, func(remote string) tui.Model { return newGreeter(remote) }) }()
	conn, err := ssh.Dial("tcp", l.Addr().String(), &ssh.ClientConfig{User: "x", HostKeyCallback: ssh.InsecureIgnoreHostKey(), Timeout: 5 * time.Second}) // #nosec G106 -- a test against our own server
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	sess, err := conn.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	if err := sess.Shell(); err == nil {
		t.Fatal("a shell without a pty was accepted")
	}
}

func TestOtherChannelTypesAreRejected(t *testing.T) {
	cfg, err := serverConfig()
	if err != nil {
		t.Fatal(err)
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	go func() { _ = serve(l, cfg, func(remote string) tui.Model { return newGreeter(remote) }) }()
	conn, err := ssh.Dial("tcp", l.Addr().String(), &ssh.ClientConfig{User: "x", HostKeyCallback: ssh.InsecureIgnoreHostKey(), Timeout: 5 * time.Second}) // #nosec G106 -- a test against our own server
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, _, err := conn.OpenChannel("direct-tcpip", nil); err == nil {
		t.Fatal("a port-forwarding channel was accepted")
	}
}

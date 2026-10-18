package main

import (
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/ows4444/tui/internal/vtscreen"
)

// startServer serves a new lobby on a free port and returns its address.
func startServer(t *testing.T, password string, maxUsers int) string {
	t.Helper()
	cfg, err := serverConfig(password)
	if err != nil {
		t.Fatal(err)
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = l.Close() })
	go func() { _ = serve(l, cfg, lobbyApp(newHub(maxUsers))) }()
	return l.Addr().String()
}

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

func sshConn(t *testing.T, addr, user, password string) *ssh.Client {
	t.Helper()
	cc := &ssh.ClientConfig{User: user, HostKeyCallback: ssh.InsecureIgnoreHostKey(), Timeout: 5 * time.Second} // #nosec G106 -- a test against our own server
	if password != "" {
		cc.Auth = []ssh.AuthMethod{ssh.Password(password)}
	}
	conn, err := ssh.Dial("tcp", addr, cc)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

// dial connects as user with a pty of cols x rows and starts a shell.
func dial(t *testing.T, addr, user string, cols, rows int) *client {
	t.Helper()
	sess, err := sshConn(t, addr, user, "").NewSession()
	if err != nil {
		t.Fatal(err)
	}
	c := &client{t: t, sess: sess, screen: vtscreen.NewScreen(max(cols, 120), max(rows, 40)), done: make(chan error, 1)}
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

func (c *client) type_(s string) {
	c.t.Helper()
	if _, err := io.WriteString(c.stdin, s); err != nil {
		c.t.Fatal(err)
	}
}

func (c *client) waitEnd() error {
	select {
	case err := <-c.done:
		return err
	case <-time.After(10 * time.Second):
		c.t.Fatal("the session did not end")
		return nil
	}
}

func TestSessionShowsTheLobbyToTheNamedUser(t *testing.T) {
	c := dial(t, startServer(t, "", 50), "ada", 80, 24)
	c.waitFor("lobby")
	c.waitFor("you are ada from 127.0.0.1:")
	c.waitFor("Online (1)")
	c.waitFor("ada joined")
}

func TestKeysReachTheProgramAndResizesFollow(t *testing.T) {
	c := dial(t, startServer(t, "", 50), "ada", 50, 20)
	c.waitFor("say something")
	if strings.Contains(c.text(), "Online (1)") {
		t.Fatalf("a 50-column window should have no side list:\n%s", c.text())
	}
	c.type_("hello")
	c.waitFor("ada› hello")
	if err := c.sess.WindowChange(30, 100); err != nil {
		t.Fatal(err)
	}
	c.waitFor("Online (1)")
}

func TestQuitCommandEndsTheSessionWithStatusZero(t *testing.T) {
	c := dial(t, startServer(t, "", 50), "ada", 80, 24)
	c.waitFor("say something")
	c.type_("/quit\r")
	if err := c.waitEnd(); err != nil {
		t.Fatalf("the session ended with %v", err)
	}
}

func TestTwoSessionsShareTheRoom(t *testing.T) {
	addr := startServer(t, "", 50)
	a := dial(t, addr, "ada", 80, 24)
	a.waitFor("Online (1)")
	b := dial(t, addr, "bob", 80, 24)
	b.waitFor("Online (2)")
	a.waitFor("Online (2)")
	a.waitFor("bob joined")
	a.type_("hi bob\r")
	b.waitFor("ada: hi bob")
	a.waitFor("ada: hi bob")
	b.type_("/me waves\r")
	a.waitFor("* bob waves")
	a.type_("/quit\r")
	b.waitFor("ada left")
	b.waitFor("Online (1)")
}

func TestSameNameGetsANumber(t *testing.T) {
	addr := startServer(t, "", 50)
	a := dial(t, addr, "sam", 80, 24)
	a.waitFor("you are sam from")
	b := dial(t, addr, "sam", 80, 24)
	b.waitFor("you are sam2 from")
}

func TestAFullLobbyRefusesTheSession(t *testing.T) {
	addr := startServer(t, "", 1)
	a := dial(t, addr, "ada", 80, 24)
	a.waitFor("Online (1)")
	b := dial(t, addr, "bob", 80, 24)
	b.waitFor("the lobby is full")
	if err := b.waitEnd(); err == nil {
		t.Fatal("a refused session should end with a non-zero status")
	}
}

func TestPasswordIsAskedForWhenSet(t *testing.T) {
	addr := startServer(t, "s3cret", 50)
	if _, err := ssh.Dial("tcp", addr, &ssh.ClientConfig{User: "x", HostKeyCallback: ssh.InsecureIgnoreHostKey(), Timeout: 5 * time.Second}); err == nil { // #nosec G106 -- a test against our own server
		t.Fatal("a client with no password got in")
	}
	if _, err := ssh.Dial("tcp", addr, &ssh.ClientConfig{User: "x", Auth: []ssh.AuthMethod{ssh.Password("nope")}, HostKeyCallback: ssh.InsecureIgnoreHostKey(), Timeout: 5 * time.Second}); err == nil { // #nosec G106 -- a test against our own server
		t.Fatal("a wrong password got in")
	}
	conn := sshConn(t, addr, "ok", "s3cret")
	if conn == nil {
		t.Fatal("the right password was refused")
	}
}

func TestShellWithoutAPtyIsRefused(t *testing.T) {
	conn := sshConn(t, startServer(t, "", 50), "x", "")
	sess, err := conn.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	if err := sess.Shell(); err == nil {
		t.Fatal("a shell without a pty was accepted")
	}
}

func TestOtherChannelTypesAreRejected(t *testing.T) {
	conn := sshConn(t, startServer(t, "", 50), "x", "")
	if _, _, err := conn.OpenChannel("direct-tcpip", nil); err == nil {
		t.Fatal("a port-forwarding channel was accepted")
	}
}

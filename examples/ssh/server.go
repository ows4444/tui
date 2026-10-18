package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"io"
	"net"
	"sync"

	"golang.org/x/crypto/ssh"

	"github.com/ows4444/tui"
)

// app makes the program of one session. send delivers a message to that
// session's program from any goroutine; the returned function is called when the
// session ends. An error means the session is refused, and is shown to the client.
type app func(user, remote string, send func(tui.Msg) bool) (tui.Model, func(), error)

// serverConfig makes a server configuration with a new host key. A real server
// keeps its host key in a file, so clients are not warned on every restart.
func serverConfig(password string) (*ssh.ServerConfig, error) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		return nil, err
	}
	cfg := &ssh.ServerConfig{NoClientAuth: password == ""}
	if password != "" {
		cfg.PasswordCallback = func(_ ssh.ConnMetadata, got []byte) (*ssh.Permissions, error) {
			if subtle.ConstantTimeCompare(got, []byte(password)) == 1 {
				return nil, nil
			}
			return nil, errors.New("wrong password")
		}
	}
	cfg.AddHostKey(signer)
	return cfg, nil
}

// serve accepts connections until the listener closes.
func serve(l net.Listener, cfg *ssh.ServerConfig, newApp app) error {
	for {
		nc, err := l.Accept()
		if err != nil {
			return err
		}
		go handle(nc, cfg, newApp)
	}
}

// handle serves one connection: each "session" channel it opens is a program.
func handle(nc net.Conn, cfg *ssh.ServerConfig, newApp app) {
	defer nc.Close()
	conn, chans, reqs, err := ssh.NewServerConn(nc, cfg)
	if err != nil {
		return
	}
	defer conn.Close()
	go ssh.DiscardRequests(reqs)
	for nch := range chans {
		if nch.ChannelType() != "session" {
			_ = nch.Reject(ssh.UnknownChannelType, "only sessions")
			continue
		}
		ch, creqs, err := nch.Accept()
		if err != nil {
			return
		}
		go session(ch, creqs, conn.User(), conn.RemoteAddr().String(), newApp)
	}
}

// term is the Terminal of one SSH session: the size the client last sent.
type term struct {
	mu      sync.Mutex
	w, h    int
	resizes chan struct{}
}

func (t *term) IsTerminal() bool { return true }

func (t *term) Size() (int, int, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.w, t.h, t.w > 0 && t.h > 0
}

func (t *term) MakeRaw(bool) (func() error, error)    { return func() error { return nil }, nil }
func (t *term) EnableOutputVT() (func() error, error) { return nil, nil }

// Resizes implements tui.ResizeNotifier.
func (t *term) Resizes() <-chan struct{} { return t.resizes }

// set records the client's size and tells the Program, without blocking.
func (t *term) set(w, h int) {
	t.mu.Lock()
	t.w, t.h = w, h
	t.mu.Unlock()
	select {
	case t.resizes <- struct{}{}:
	default:
	}
}

// ptyReq is the payload of an SSH "pty-req" request (RFC 4254, section 6.2).
type ptyReq struct {
	Term          string
	Cols, Rows    uint32
	Width, Height uint32
	Modes         string
}

// windowChange is the payload of a "window-change" request (section 6.7).
type windowChange struct {
	Cols, Rows    uint32
	Width, Height uint32
}

// session answers the requests of one channel and runs the program when the
// client asks for a shell.
func session(ch ssh.Channel, reqs <-chan *ssh.Request, user, remote string, newApp app) {
	defer ch.Close()
	t := &term{resizes: make(chan struct{}, 1)}
	started := false
	for req := range reqs {
		switch req.Type {
		case "pty-req":
			var p ptyReq
			if ssh.Unmarshal(req.Payload, &p) != nil || p.Cols == 0 || p.Rows == 0 || p.Cols > 1<<15 || p.Rows > 1<<15 {
				_ = req.Reply(false, nil)
				continue
			}
			t.set(int(p.Cols), int(p.Rows))
			_ = req.Reply(true, nil)
		case "window-change":
			var w windowChange
			if ssh.Unmarshal(req.Payload, &w) == nil && w.Cols > 0 && w.Rows > 0 && w.Cols <= 1<<15 && w.Rows <= 1<<15 {
				t.set(int(w.Cols), int(w.Rows))
			}
		case "shell":
			if _, _, ok := t.Size(); started || !ok { // a shell needs a pty, and runs once
				_ = req.Reply(false, nil)
				continue
			}
			started = true
			_ = req.Reply(true, nil)
			go func() {
				run(ch, t, user, remote, newApp)
				_ = ch.Close()
			}()
		default:
			_ = req.Reply(false, nil)
		}
	}
}

// run is the Program of one session, ended by the model quitting or the client
// going away.
func run(ch ssh.Channel, t *term, user, remote string, newApp app) {
	status := uint32(0)
	// The Program does not exist until the model does, and the model is already
	// in the hub, so what arrives in between is held and handed over at the start.
	var mu sync.Mutex
	var prog *tui.Program
	var held []tui.Msg
	send := func(msg tui.Msg) bool {
		mu.Lock()
		defer mu.Unlock()
		if prog == nil {
			held = append(held, msg)
			return true
		}
		return prog.TrySend(msg)
	}
	m, done, err := newApp(user, remote, send)
	if err != nil {
		_, _ = io.WriteString(ch, "\r\n"+err.Error()+"\r\n")
		_, _ = ch.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{1}))
		return
	}
	defer done()
	// The Program may close what it reads from when it ends; the channel must stay
	// open for the exit status, so it is handed over as a plain Reader.
	p := tui.NewProgram(m, tui.WithAltScreen(true), tui.WithMouse(tui.MouseAllMotion), tui.WithBracketedPaste(true),
		tui.WithInput(struct{ io.Reader }{ch}), tui.WithOutput(ch), tui.WithTerminal(t))
	mu.Lock()
	prog = p
	for _, msg := range held {
		p.TrySend(msg)
	}
	held = nil
	mu.Unlock()
	if _, err := p.Run(); err != nil {
		status = 1
	}
	_, _ = ch.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{status}))
}

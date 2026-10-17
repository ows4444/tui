// Command ssh serves a tui program over SSH, so anyone who connects gets their
// own copy of it in their own terminal, the way Charm's wish does for Bubble
// Tea:
//
//	go run . -addr :2222
//	ssh -p 2222 localhost
//
// It is a separate module (see go.mod) because golang.org/x/crypto/ssh is the
// one dependency it needs, and the tui module has none. Each session is a
// tui.Program whose input and output are the SSH channel and whose Terminal is
// answered from the pty request and the window-change requests the client
// sends; the client's own terminal is already in raw mode, so MakeRaw does
// nothing here.
//
// This demo lets anyone in (ssh.ServerConfig.NoClientAuth). Do not expose it
// as it is: set PublicKeyCallback or PasswordCallback first.
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:2222", "address to listen on")
	flag.Parse()
	cfg, err := serverConfig()
	if err != nil {
		log.Fatal(err)
	}
	l, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprintf(os.Stderr, "listening on %s: ssh -p %s localhost\n", l.Addr(), portOf(l.Addr()))
	log.Fatal(serve(l, cfg, func(remote string) tui.Model { return newGreeter(remote) }))
}

func portOf(a net.Addr) string {
	_, p, _ := net.SplitHostPort(a.String())
	return p
}

// serverConfig makes a server configuration with a new host key. A real server
// keeps its host key in a file, so clients are not warned on every restart.
func serverConfig() (*ssh.ServerConfig, error) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		return nil, err
	}
	cfg := &ssh.ServerConfig{NoClientAuth: true}
	cfg.AddHostKey(signer)
	return cfg, nil
}

// serve accepts connections until the listener closes.
func serve(l net.Listener, cfg *ssh.ServerConfig, newModel func(remote string) tui.Model) error {
	for {
		nc, err := l.Accept()
		if err != nil {
			return err
		}
		go handle(nc, cfg, newModel)
	}
}

// handle serves one connection: each "session" channel it opens is a program.
func handle(nc net.Conn, cfg *ssh.ServerConfig, newModel func(string) tui.Model) {
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
		go session(ch, creqs, conn.RemoteAddr().String(), newModel)
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
func session(ch ssh.Channel, reqs <-chan *ssh.Request, remote string, newModel func(string) tui.Model) {
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
				run(ch, t, newModel(remote))
				_ = ch.Close()
			}()
		default:
			_ = req.Reply(false, nil)
		}
	}
}

// run is the Program of one session, ended by the model quitting or the client
// going away.
func run(ch ssh.Channel, t *term, m tui.Model) {
	status := uint32(0)
	// The Program may close what it reads from when it ends; the channel must stay
	// open for the exit status, so it is handed over as a plain Reader.
	p := tui.NewProgram(m, tui.WithAltScreen(true), tui.WithInput(struct{ io.Reader }{ch}), tui.WithOutput(ch), tui.WithTerminal(t))
	if _, err := p.Run(); err != nil {
		status = 1
	}
	_, _ = ch.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{status}))
}

// greeter is the program every session runs: who connected, how big their
// terminal is, and a count of the keys they pressed.
type greeter struct {
	remote string
	w, h   int
	keys   int
}

func newGreeter(remote string) greeter { return greeter{remote: remote} }

func (greeter) Init() tui.Cmd { return nil }

func (g greeter) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		g.w, g.h = msg.Width, msg.Height
	case tui.Key:
		if msg.Type == tui.KeyCtrlC || msg.Type == tui.KeyRunes && msg.Text == "q" {
			return g, tui.Quit()
		}
		g.keys++
	}
	return g, nil
}

func (g greeter) View() string {
	title := ansi.NewStyle().Bold().Foreground(ansi.BrightCyan).Render("tui over SSH")
	body := strings.Join([]string{
		title,
		"",
		"you are " + ansi.Sanitize(g.remote),
		fmt.Sprintf("size %dx%d", g.w, g.h),
		fmt.Sprintf("keys %d", g.keys),
		"",
		"press q to leave",
	}, "\n")
	node := layout.BoxNode(layout.NewBox().Border(layout.RoundedBorder()).Padding(0, 2, 0, 2), layout.Block(body))
	return layout.Draw(node, layout.Constraints{MaxW: max(g.w, 1), MaxH: max(g.h, 1)})
}

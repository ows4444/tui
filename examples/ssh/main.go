// Command ssh serves a chat lobby over SSH, so anyone who connects gets their
// own copy of the program in their own terminal and everyone sees the same room,
// the way Charm's wish does for Bubble Tea:
//
//	go run . -addr :2222
//	ssh -p 2222 yourname@localhost
//
// Each session is a tui.Program whose input and output are the SSH channel and
// whose Terminal is answered from the pty request and the window-change requests
// the client sends. The sessions share one hub (hub.go): it holds who is online
// and the last lines, hands each session a name that is unique, limits how fast a
// session may talk and how many may join, and cuts escape sequences out of
// everything people type before anyone else sees it. A session learns of what
// others do through Program.TrySend, the way any outside event reaches Update.
//
// In the lobby: Enter sends, /me, /nick, /who and /help are commands, click a name
// or press Tab to mention someone, PgUp and PgDn scroll, Ctrl+C or /quit leaves.
//
// It is a separate module (see go.mod) because golang.org/x/crypto/ssh is the
// one dependency it needs, and the tui module has none. By default it lets anyone
// in; pass -password to ask for one. A host key made at start is thrown away on
// exit, so clients are warned again next time; a real server keeps its key in a
// file. Do not put it on the open internet as it is.
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/ows4444/tui"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:2222", "address to listen on")
	password := flag.String("password", "", "ask clients for this password (empty: let anyone in)")
	maxUsers := flag.Int("max", 50, "most sessions at once")
	flag.Parse()
	cfg, err := serverConfig(*password)
	if err != nil {
		log.Fatal(err)
	}
	l, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprintf(os.Stderr, "listening on %s: ssh -p %s name@localhost\n", l.Addr(), portOf(l.Addr()))
	log.Fatal(serve(l, cfg, lobbyApp(newHub(*maxUsers))))
}

func portOf(a net.Addr) string {
	_, p, _ := net.SplitHostPort(a.String())
	return p
}

// lobbyApp makes the program of each session: a lobby on the shared hub.
func lobbyApp(h *hub) app {
	return func(user, remote string, send func(tui.Msg) bool) (tui.Model, func(), error) {
		l, err := newLobby(h, user, remote, send)
		if err != nil {
			return nil, nil, err
		}
		return l, func() { h.leave(l.id) }, nil
	}
}

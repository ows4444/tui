package main

import (
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
)

type eventKind int

const (
	evJoin eventKind = iota
	evLeave
	evSay
	evMe
	evNick
)

const (
	maxNick    = 16
	maxText    = 300
	maxHistory = 200
	recent     = 20 // lines a new arrival sees
	minGap     = 300 * time.Millisecond
)

var (
	errFull    = errors.New("the lobby is full")
	errSlow    = errors.New("slow down: one message every 300 ms")
	errNick    = errors.New("that name is taken or not allowed")
	errNothing = errors.New("nothing to send")
)

// event is one thing that happened in the lobby.
type event struct {
	kind eventKind
	who  string
	text string // the message, or the old name for evNick
	at   time.Time
}

// hubMsg is what a session's program receives when something happens: the event
// and who is online after it.
type hubMsg struct {
	ev    event
	users []string
}

type sub struct {
	nick string
	send func(tui.Msg) bool
	last time.Time
}

// hub is the room every session shares. It is the only state the sessions have
// in common, and everything it touches is behind one lock.
type hub struct {
	mu      sync.Mutex
	subs    map[int]*sub
	next    int
	history []event
	max     int
	now     func() time.Time
}

func newHub(maxUsers int) *hub {
	return &hub{subs: map[int]*sub{}, max: maxUsers, now: time.Now}
}

// cleanNick makes a name safe to show and to type: no escapes, no spaces, cut
// to maxNick characters. It is empty if nothing printable is left.
func cleanNick(s string) string {
	s = strings.Join(strings.Fields(ansi.Sanitize(s)), "")
	if r := []rune(s); len(r) > maxNick {
		s = string(r[:maxNick])
	}
	return s
}

func cleanText(s string) string {
	s = strings.Join(strings.Fields(strings.ReplaceAll(ansi.Sanitize(s), "\n", " ")), " ")
	if r := []rune(s); len(r) > maxText {
		s = string(r[:maxText])
	}
	return s
}

// unique returns nick, or nick with the smallest number after it that nobody
// has. h.mu must be held.
func (h *hub) unique(nick, except string) string {
	taken := func(n string) bool {
		for _, s := range h.subs {
			if strings.EqualFold(s.nick, n) && !strings.EqualFold(n, except) {
				return true
			}
		}
		return false
	}
	if !taken(nick) {
		return nick
	}
	for i := 2; ; i++ {
		if n := nick + strconv.Itoa(i); !taken(n) {
			return n
		}
	}
}

// join adds a session and tells everyone else. It returns the id, the name it
// was given (made unique), the recent lines, which end with its own arrival, and
// who is online.
func (h *hub) join(nick string, send func(tui.Msg) bool) (int, string, []event, []string, error) {
	h.mu.Lock()
	if len(h.subs) >= h.max {
		h.mu.Unlock()
		return 0, "", nil, nil, errFull
	}
	nick = cleanNick(nick)
	if nick == "" {
		nick = "guest"
	}
	nick = h.unique(nick, "")
	h.next++
	id := h.next
	h.subs[id] = &sub{nick: nick, send: send}
	ev := event{kind: evJoin, who: nick, at: h.now()}
	users, out := h.record(ev, id)
	past := append([]event(nil), h.history[max(0, len(h.history)-recent):]...)
	h.mu.Unlock()
	deliver(out, hubMsg{ev: ev, users: users})
	return id, nick, past, users, nil
}

func (h *hub) leave(id int) {
	h.mu.Lock()
	s, ok := h.subs[id]
	if !ok {
		h.mu.Unlock()
		return
	}
	delete(h.subs, id)
	ev := event{kind: evLeave, who: s.nick, at: h.now()}
	users, out := h.record(ev, 0)
	h.mu.Unlock()
	deliver(out, hubMsg{ev: ev, users: users})
}

// record keeps ev in the history and returns who is online and the senders to
// tell. h.mu must be held.
func (h *hub) record(ev event, except int) ([]string, []func(tui.Msg) bool) {
	h.history = append(h.history, ev)
	if len(h.history) > maxHistory {
		h.history = h.history[len(h.history)-maxHistory:]
	}
	var users []string
	var out []func(tui.Msg) bool
	for id, s := range h.subs {
		users = append(users, s.nick)
		if id != except {
			out = append(out, s.send)
		}
	}
	sort.Strings(users)
	return users, out
}

func deliver(out []func(tui.Msg) bool, m hubMsg) {
	for _, send := range out {
		if send != nil {
			send(m)
		}
	}
}

// say sends a line to everyone; me makes it an action ("* ada waves").
func (h *hub) say(id int, text string, me bool) error {
	text = cleanText(text)
	if text == "" {
		return errNothing
	}
	h.mu.Lock()
	s, ok := h.subs[id]
	if !ok {
		h.mu.Unlock()
		return errNothing
	}
	now := h.now()
	if !s.last.IsZero() && now.Sub(s.last) < minGap {
		h.mu.Unlock()
		return errSlow
	}
	s.last = now
	ev := event{kind: evSay, who: s.nick, text: text, at: now}
	if me {
		ev.kind = evMe
	}
	users, out := h.record(ev, 0)
	h.mu.Unlock()
	deliver(out, hubMsg{ev: ev, users: users})
	return nil
}

// rename changes a session's name and returns the one it now has.
func (h *hub) rename(id int, nick string) (string, error) {
	nick = cleanNick(nick)
	if nick == "" {
		return "", errNick
	}
	h.mu.Lock()
	s, ok := h.subs[id]
	if !ok {
		h.mu.Unlock()
		return "", errNick
	}
	for _, o := range h.subs {
		if o != s && strings.EqualFold(o.nick, nick) {
			h.mu.Unlock()
			return "", errNick
		}
	}
	old := s.nick
	s.nick = nick
	ev := event{kind: evNick, who: nick, text: old, at: h.now()}
	users, out := h.record(ev, 0)
	h.mu.Unlock()
	deliver(out, hubMsg{ev: ev, users: users})
	return nick, nil
}

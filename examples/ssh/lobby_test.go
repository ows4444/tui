package main

import (
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/examples/internal/uitest"
	"github.com/ows4444/tui/layout"
)

// room is a hub with two sessions whose programs are driven by hand.
type room struct {
	h    *hub
	now  *time.Time
	a, b tui.Model
	ain  *inbox
	bin  *inbox
}

func newRoom(t *testing.T) *room {
	t.Helper()
	h, now := testHub(10)
	r := &room{h: h, now: now, ain: &inbox{}, bin: &inbox{}}
	la, err := newLobby(h, "ada", "10.0.0.1:1", r.ain.send)
	if err != nil {
		t.Fatal(err)
	}
	lb, err := newLobby(h, "bob", "10.0.0.2:2", r.bin.send)
	if err != nil {
		t.Fatal(err)
	}
	r.a, _ = la.Update(tui.ResizeMsg{Width: 100, Height: 24})
	r.b, _ = lb.Update(tui.ResizeMsg{Width: 100, Height: 24})
	r.pump()
	return r
}

// pump hands each program the messages the hub queued for it.
func (r *room) pump() {
	for len(r.ain.got) > 0 || len(r.bin.got) > 0 {
		a, b := r.ain.got, r.bin.got
		r.ain.got, r.bin.got = nil, nil
		for _, m := range a {
			r.a, _ = r.a.Update(m)
		}
		for _, m := range b {
			r.b, _ = r.b.Update(m)
		}
	}
}

func typeInto(m tui.Model, s string) tui.Model {
	for _, r := range s {
		m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: string(r)})
	}
	return m
}

func enter(m tui.Model) (tui.Model, tui.Cmd) { return m.Update(tui.Key{Type: tui.KeyEnter}) }

func plain(m tui.Model) string { return ansi.StripANSI(m.View()) }

func (r *room) tick() { *r.now = r.now.Add(time.Second) }

func TestLobbyShowsTheRoomAndWhoIsHere(t *testing.T) {
	r := newRoom(t)
	out := plain(r.a)
	for _, want := range []string{"lobby  2 online", "Online (2)", "▸ ada (you)", "  bob", "ada joined", "bob joined", "you are ada from 10.0.0.1:1", "say something", "[enter] send"} {
		if !strings.Contains(out, want) {
			t.Errorf("screen lacks %q:\n%s", want, out)
		}
	}
}

func TestMessagesReachBothSessions(t *testing.T) {
	r := newRoom(t)
	r.tick()
	r.a = typeInto(r.a, "hello there")
	if !strings.Contains(plain(r.a), "ada› hello there") {
		t.Fatalf("\n%s", plain(r.a))
	}
	r.a, _ = enter(r.a)
	r.pump()
	for _, m := range []tui.Model{r.a, r.b} {
		if out := plain(m); !strings.Contains(out, "12:00 ada: hello there") {
			t.Errorf("\n%s", out)
		}
	}
	if len(r.a.(*lobby).prompt) != 0 {
		t.Fatal("the prompt was not cleared")
	}
}

func TestCommands(t *testing.T) {
	r := newRoom(t)
	r.tick()
	r.a = typeInto(r.a, "/me waves")
	r.a, _ = enter(r.a)
	r.pump()
	if !strings.Contains(plain(r.b), "* ada waves") {
		t.Fatalf("\n%s", plain(r.b))
	}
	r.a = typeInto(r.a, "/nick countess")
	r.a, _ = enter(r.a)
	r.pump()
	if !strings.Contains(plain(r.b), "ada is now countess") || !strings.Contains(plain(r.b), "  countess") || !strings.Contains(plain(r.a), "▸ countess (you)") {
		t.Fatalf("\n%s", plain(r.b))
	}
	r.a = typeInto(r.a, "/nick bob")
	r.a, _ = enter(r.a)
	if !strings.Contains(plain(r.a), errNick.Error()) {
		t.Fatalf("\n%s", plain(r.a))
	}
	r.a = typeInto(r.a, "/who")
	r.a, _ = enter(r.a)
	if !strings.Contains(plain(r.a), "2 online: bob, countess") {
		t.Fatalf("\n%s", plain(r.a))
	}
	r.a = typeInto(r.a, "/nope")
	r.a, _ = enter(r.a)
	if !strings.Contains(plain(r.a), "Unknown command /nope") {
		t.Fatalf("\n%s", plain(r.a))
	}
	r.a = typeInto(r.a, "/help")
	r.a, _ = enter(r.a)
	if !strings.Contains(plain(r.a), "mention someone") {
		t.Fatalf("\n%s", plain(r.a))
	}
	r.a, _ = r.a.Update(tui.Key{Type: tui.KeyEsc})
	if strings.Contains(plain(r.a), "mention someone") {
		t.Fatal("a key should close help")
	}
	r.a = typeInto(r.a, "/quit")
	if _, cmd := enter(r.a); cmd == nil {
		t.Fatal("/quit should quit")
	}
}

func TestSlowDownIsShownToTheSender(t *testing.T) {
	r := newRoom(t)
	r.tick()
	r.a = typeInto(r.a, "one")
	r.a, _ = enter(r.a)
	r.a = typeInto(r.a, "two")
	r.a, _ = enter(r.a)
	if !strings.Contains(plain(r.a), "slow down") {
		t.Fatalf("\n%s", plain(r.a))
	}
}

func TestClickingANameMentionsThem(t *testing.T) {
	r := newRoom(t)
	rect, ok := layout.RectOf(r.a.(*lobby).screen(), r.a.(*lobby).size(), "user-1")
	if !ok {
		t.Fatalf("no user-1:\n%s", plain(r.a))
	}
	r.a, _ = r.a.Update(tui.MouseEvent{X: rect.X + 2, Y: rect.Y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if string(r.a.(*lobby).prompt) != "@bob " {
		t.Fatalf("prompt %q", string(r.a.(*lobby).prompt))
	}
	rect, _ = layout.RectOf(r.a.(*lobby).screen(), r.a.(*lobby).size(), "user-0")
	r.a, _ = r.a.Update(tui.MouseEvent{X: rect.X, Y: rect.Y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if string(r.a.(*lobby).prompt) != "@bob " {
		t.Fatalf("clicking yourself should do nothing: %q", string(r.a.(*lobby).prompt))
	}
}

func TestTabCompletesAMention(t *testing.T) {
	r := newRoom(t)
	r.a, _ = r.a.Update(tui.Key{Type: tui.KeyTab})
	if string(r.a.(*lobby).prompt) != "@bob " {
		t.Fatalf("%q", string(r.a.(*lobby).prompt))
	}
	r.a = typeInto(r.a, "@b")
	r.a.(*lobby).prompt, r.a.(*lobby).cur = []rune("@bo"), 3
	r.a, _ = r.a.Update(tui.Key{Type: tui.KeyTab})
	if string(r.a.(*lobby).prompt) != "@bob " {
		t.Fatalf("%q", string(r.a.(*lobby).prompt))
	}
	solo, _ := newLobby(newHub(5), "solo", "x", (&inbox{}).send)
	m, _ := solo.Update(tui.Key{Type: tui.KeyTab})
	if !strings.Contains(m.(*lobby).msg, "No one") {
		t.Fatal("completing with nobody here should say so")
	}
}

func TestPasteAndEscapesAreCleaned(t *testing.T) {
	r := newRoom(t)
	r.a, _ = r.a.Update(tui.PasteEvent{Text: "a\x1b[31m b\nc"})
	if got := string(r.a.(*lobby).prompt); got != "a b c" {
		t.Fatalf("%q", got)
	}
	r.a, _ = r.a.Update(tui.Key{Type: tui.KeyRunes, Text: "\x1b[2J"})
	if strings.ContainsRune(string(r.a.(*lobby).prompt), 0x1b) {
		t.Fatal("an escape got into the prompt")
	}
}

func TestHistoryAndScrolling(t *testing.T) {
	r := newRoom(t)
	for _, s := range []string{"first", "second"} {
		r.tick()
		r.a = typeInto(r.a, s)
		r.a, _ = enter(r.a)
	}
	r.a, _ = r.a.Update(tui.Key{Type: tui.KeyUp})
	if string(r.a.(*lobby).prompt) != "second" {
		t.Fatalf("%q", string(r.a.(*lobby).prompt))
	}
	r.a, _ = r.a.Update(tui.Key{Type: tui.KeyUp})
	r.a, _ = r.a.Update(tui.Key{Type: tui.KeyDown})
	r.a, _ = r.a.Update(tui.Key{Type: tui.KeyDown})
	if len(r.a.(*lobby).prompt) != 0 {
		t.Fatal("Down past the newest should empty the prompt")
	}
	r.a, _ = r.a.Update(tui.ResizeMsg{Width: 100, Height: 8})
	for i := 0; i < 12; i++ {
		r.tick()
		r.b = typeInto(r.b, "line"+itoa(i))
		r.b, _ = enter(r.b)
		r.pump()
	}
	r.a, _ = r.a.Update(tui.Key{Type: tui.KeyPgUp})
	if l := r.a.(*lobby); l.off == 0 || !strings.Contains(plain(r.a), "lines below") {
		t.Fatalf("off %d\n%s", l.off, plain(r.a))
	}
	off := r.a.(*lobby).off
	r.tick()
	r.b = typeInto(r.b, "late")
	r.b, _ = enter(r.b)
	r.pump()
	if r.a.(*lobby).off != off+1 {
		t.Fatalf("a scrolled reader should stay put: %d -> %d", off, r.a.(*lobby).off)
	}
	for i := 0; i < 5; i++ {
		r.a, _ = r.a.Update(tui.Key{Type: tui.KeyPgDown})
	}
	if r.a.(*lobby).off != 0 || !strings.Contains(plain(r.a), "bob: late") {
		t.Fatalf("\n%s", plain(r.a))
	}
	r.a, _ = r.a.Update(tui.MouseEvent{Button: tui.MouseButtonWheelUp, Action: tui.MouseActionPress})
	if r.a.(*lobby).off != 3 {
		t.Fatalf("wheel: %d", r.a.(*lobby).off)
	}
}

func TestSmallWindowHasNoSideListButKeepsThePrompt(t *testing.T) {
	r := newRoom(t)
	r.a, _ = r.a.Update(tui.ResizeMsg{Width: 40, Height: 10})
	lines := strings.Split(plain(r.a), "\n")
	if len(lines) != 10 || strings.Contains(plain(r.a), "Online") || !strings.Contains(lines[8], "ada›") || !strings.Contains(lines[9], "[enter]") {
		t.Fatalf("\n%s", plain(r.a))
	}
}

func TestHelpAndQuitKeys(t *testing.T) {
	r := newRoom(t)
	r.a, _ = r.a.Update(tui.Key{Type: tui.KeyRunes, Text: "?"})
	if !strings.Contains(plain(r.a), "Press any key to close") {
		t.Fatalf("\n%s", plain(r.a))
	}
	r.a, _ = r.a.Update(tui.Key{Type: tui.KeyEsc})
	r.a = typeInto(r.a, "x")
	r.a, _ = r.a.Update(tui.Key{Type: tui.KeyRunes, Text: "?"})
	if string(r.a.(*lobby).prompt) != "x?" {
		t.Fatalf("a ? in a sentence is text: %q", string(r.a.(*lobby).prompt))
	}
	if _, cmd := r.a.Update(tui.Key{Type: tui.KeyCtrlC}); cmd == nil {
		t.Fatal("Ctrl+C should leave")
	}
	if _, cmd := r.a.Update(tui.Key{Type: tui.KeyCtrl, Code: 'd'}); cmd == nil {
		t.Fatal("Ctrl+D should leave")
	}
}

func TestLobbyFollowsTheThemeAndSpeaksPlainly(t *testing.T) {
	uitest.Check(t, func() tui.Model {
		l, err := newLobby(newHub(5), "ada", "10.0.0.1:1", (&inbox{}).send)
		if err != nil {
			t.Fatal(err)
		}
		return l
	})
}

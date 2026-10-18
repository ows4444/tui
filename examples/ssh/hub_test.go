package main

import (
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui"
)

// inbox collects what a session would be sent.
type inbox struct{ got []hubMsg }

func (b *inbox) send(m tui.Msg) bool {
	b.got = append(b.got, m.(hubMsg))
	return true
}

func (b *inbox) last() hubMsg { return b.got[len(b.got)-1] }

func testHub(max int) (*hub, *time.Time) {
	h := newHub(max)
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	h.now = func() time.Time { return now }
	return h, &now
}

func join(t *testing.T, h *hub, nick string) (int, string, *inbox) {
	t.Helper()
	in := &inbox{}
	id, name, _, _, err := h.join(nick, in.send)
	if err != nil {
		t.Fatal(err)
	}
	return id, name, in
}

func TestJoinNamesAreCleanedAndUnique(t *testing.T) {
	h, _ := testHub(10)
	for in, want := range map[string]string{"ada": "ada", "ada ": "ada2", "ADA": "ADA3", "": "guest", "  ": "guest2", "a b c": "abc", "\x1b[31mred\x1b[0m": "red"} {
		if _, got, _ := join(t, h, in); got != want {
			t.Errorf("join(%q) = %q, want %q", in, got, want)
		}
	}
	if _, name, _ := join(t, h, strings.Repeat("x", 40)); len([]rune(name)) != maxNick {
		t.Errorf("long name %q", name)
	}
}

func TestEveryoneIsToldWhoJoinedAndLeft(t *testing.T) {
	h, _ := testHub(10)
	aid, _, a := join(t, h, "ada")
	_, _, b := join(t, h, "bob")
	if m := a.last(); m.ev.kind != evJoin || m.ev.who != "bob" || strings.Join(m.users, ",") != "ada,bob" {
		t.Fatalf("%+v", m)
	}
	if len(b.got) != 0 && b.got[0].ev.who == "bob" {
		t.Fatalf("the newcomer is not told of their own arrival: %+v", b.got)
	}
	h.leave(aid)
	if m := b.last(); m.ev.kind != evLeave || m.ev.who != "ada" || strings.Join(m.users, ",") != "bob" {
		t.Fatalf("%+v", m)
	}
	h.leave(aid)
	if len(b.got) != 1 {
		t.Fatalf("leaving twice told people again: %d messages", len(b.got))
	}
}

func TestNewArrivalsSeeTheRecentLines(t *testing.T) {
	h, now := testHub(10)
	id, _, _ := join(t, h, "ada")
	for i := 0; i < 30; i++ {
		*now = now.Add(time.Second)
		if err := h.say(id, "line "+itoa(i), false); err != nil {
			t.Fatal(err)
		}
	}
	_, _, past, _, err := h.join("bob", (&inbox{}).send)
	if err != nil || len(past) != recent || past[len(past)-2].text != "line 29" || past[len(past)-1].kind != evJoin || past[len(past)-1].who != "bob" {
		t.Fatalf("%d lines, err %v", len(past), err)
	}
}

func TestSayCleansAndLimitsText(t *testing.T) {
	h, now := testHub(10)
	id, _, in := join(t, h, "ada")
	*now = now.Add(time.Second)
	if err := h.say(id, "hi \x1b[2Jthere\n friend", false); err != nil || in.last().ev.text != "hi there friend" {
		t.Fatalf("%q %v", in.last().ev.text, err)
	}
	*now = now.Add(time.Second)
	if err := h.say(id, "   ", false); err != errNothing {
		t.Fatalf("empty: %v", err)
	}
	*now = now.Add(time.Second)
	h.say(id, strings.Repeat("é", 500), false)
	if n := len([]rune(in.last().ev.text)); n != maxText {
		t.Fatalf("long text kept %d runes", n)
	}
	*now = now.Add(time.Second)
	h.say(id, "waves", true)
	if in.last().ev.kind != evMe {
		t.Fatal("not an action")
	}
}

func TestRateLimit(t *testing.T) {
	h, now := testHub(10)
	id, _, _ := join(t, h, "ada")
	*now = now.Add(time.Second)
	if err := h.say(id, "one", false); err != nil {
		t.Fatal(err)
	}
	*now = now.Add(100 * time.Millisecond)
	if err := h.say(id, "two", false); err != errSlow {
		t.Fatalf("a second line 100 ms later: %v", err)
	}
	*now = now.Add(250 * time.Millisecond)
	if err := h.say(id, "two", false); err != nil {
		t.Fatalf("after the gap: %v", err)
	}
	other, _, _ := join(t, h, "bob")
	if err := h.say(other, "mine", false); err != nil {
		t.Fatalf("the limit is per session: %v", err)
	}
}

func TestRename(t *testing.T) {
	h, _ := testHub(10)
	a, _, ain := join(t, h, "ada")
	join(t, h, "bob")
	if _, err := h.rename(a, "Bob"); err != errNick {
		t.Fatalf("taking another's name: %v", err)
	}
	if _, err := h.rename(a, "  "); err != errNick {
		t.Fatalf("empty name: %v", err)
	}
	name, err := h.rename(a, "countess")
	if err != nil || name != "countess" {
		t.Fatalf("%q %v", name, err)
	}
	if m := ain.last(); m.ev.kind != evNick || m.ev.who != "countess" || m.ev.text != "ada" || strings.Join(m.users, ",") != "bob,countess" {
		t.Fatalf("%+v", m)
	}
	if _, err := h.rename(a, "COUNTESS"); err != nil {
		t.Fatalf("changing the case of your own name: %v", err)
	}
}

func TestTheLobbyHoldsOnlySoManyAndHistoryIsCapped(t *testing.T) {
	h, now := testHub(2)
	a, _, _ := join(t, h, "a")
	join(t, h, "b")
	if _, _, _, _, err := h.join("c", (&inbox{}).send); err != errFull {
		t.Fatalf("%v", err)
	}
	h.leave(a)
	if _, _, _, _, err := h.join("c", (&inbox{}).send); err != nil {
		t.Fatalf("a place should free up: %v", err)
	}
	id, _, _ := join2(h)
	for i := 0; i < maxHistory+40; i++ {
		*now = now.Add(time.Second)
		_ = h.say(id, "x", false)
	}
	if len(h.history) != maxHistory {
		t.Fatalf("history %d", len(h.history))
	}
}

func join2(h *hub) (int, string, error) {
	h.mu.Lock()
	h.max = 99
	h.mu.Unlock()
	id, name, _, _, err := h.join("zed", (&inbox{}).send)
	return id, name, err
}

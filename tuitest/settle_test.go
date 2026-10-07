package tuitest

import (
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui"
)

type nudge struct{}

type later string

// slow shows the keys it received, and "ready" 80ms after an "r": output that
// arrives on its own time.
type slow struct{ keys, shown string }

func (slow) Init() tui.Cmd { return nil }

func (m slow) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.Key:
		m.keys += msg.String() + " "
		switch msg.String() {
		case "r":
			return m, func() tui.Msg { time.Sleep(80 * time.Millisecond); return later("ready") }
		case "q":
			return m, tui.Quit()
		}
	case later:
		m.shown = string(msg)
	}
	return m, nil
}

func (m slow) View() string { return "keys: " + m.keys + "\n" + m.shown }

// A lone ESC reaches the model only after the escape timeout. An unrelated
// message handled in that gap is an Update too, and Keys must not take it for
// the key: it used to, and returned before the key had arrived.
func TestKeysWaitsForTheKeyNotForAnotherUpdate(t *testing.T) {
	s := New(slow{}, 30, 3, tui.WithEscTimeout(300*time.Millisecond))
	defer s.Close()
	go func() {
		time.Sleep(20 * time.Millisecond)
		s.prog.Send(nudge{})
	}()
	s.Keys("esc")
	if row := s.Screen()[0]; !strings.Contains(row, "esc") {
		t.Fatalf("Keys returned before the model received the key: %q", row)
	}
}

func TestWaitForText(t *testing.T) {
	s := New(slow{}, 30, 3)
	defer s.Close()
	s.Keys("r")
	if s.hasText("ready") {
		t.Fatal("the delayed text was already drawn; the test no longer covers waiting")
	}
	if !s.WaitForText("ready", 5*time.Second) {
		t.Fatalf("WaitForText did not see the text: %q", s.Screen())
	}
	if s.WaitForText("never drawn", 30*time.Millisecond) {
		t.Fatal("WaitForText reported text that is not on the screen")
	}
}

// Once the program has exited nothing more will be drawn, so WaitForText
// answers from the screen it left without waiting out the timeout.
func TestWaitForTextReturnsWhenTheProgramExits(t *testing.T) {
	s := New(slow{}, 30, 3)
	defer s.Close()
	s.Keys("q")
	for !s.Done() {
		time.Sleep(time.Millisecond)
	}
	start := time.Now()
	if s.WaitForText("never drawn", time.Minute) {
		t.Fatal("WaitForText reported text that is not on the screen")
	}
	if d := time.Since(start); d > 5*time.Second {
		t.Fatalf("WaitForText waited %v after the program had exited", d)
	}
}

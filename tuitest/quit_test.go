package tuitest_test

import (
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/tuitest"
)

// slowOut is a terminal that takes d to accept each write, as a loaded
// machine's does.
type slowOut struct{ d time.Duration }

func (w slowOut) Write(p []byte) (int, error) {
	time.Sleep(w.d)
	return len(p), nil
}

// A key that makes the model quit is settled only once the program has
// exited. Leaving the screen takes several writes; when they were slow, Keys
// saw 25ms without an Update, took the program for idle and returned while it
// was still shutting down, so Done was false straight after.
func TestKeysWaitsForAQuittingProgramToExit(t *testing.T) {
	s := tuitest.New(echo{}, 30, 6, tui.WithOutput(slowOut{20 * time.Millisecond}))
	defer s.Close()
	s.Keys("q")
	if !s.Done() {
		t.Fatal("Keys returned before the program it told to quit had exited")
	}
}

// The same for a quit sent as a message.
func TestSendWaitsForAQuittingProgramToExit(t *testing.T) {
	s := tuitest.New(echo{}, 30, 6, tui.WithOutput(slowOut{20 * time.Millisecond}))
	defer s.Close()
	s.Send(tui.Key{Type: tui.KeyRunes, Text: "q"})
	if !s.Done() {
		t.Fatal("Send returned before the program it told to quit had exited")
	}
}

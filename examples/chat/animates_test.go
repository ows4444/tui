package main

import (
	"testing"
	"time"

	"github.com/ows4444/tui/tuitest"
)

// The program is run as main runs it, from Init on, and the reply must type
// itself out to its end. A reply that Init starts on a copy of the model is
// never marked running in the model the Program keeps, and it ignores its
// own ticks. "exit." is the reply's last word; it wraps onto a row of its own.
func TestTheReplyTypesOutInTheRunningProgram(t *testing.T) {
	m := initialModel()
	m.reply.Interval = time.Millisecond
	s := tuitest.New(m, 100, 40)
	defer s.Close()
	if !s.WaitForText(" exit.", 3*time.Second) {
		t.Fatalf("the reply did not finish typing:\n%v", s.Screen())
	}
}

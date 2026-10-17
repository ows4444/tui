package main

import (
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui/tuitest"
)

func session(t *testing.T) *tuitest.Session {
	t.Helper()
	s := tuitest.New(initialModel(), 80, 24)
	t.Cleanup(s.Close)
	return s
}

func screen(s *tuitest.Session) string { return strings.Join(s.Screen(), "\n") }

func TestEnterGreets(t *testing.T) {
	s := session(t)
	s.Keys("A", "d", "a", "enter")
	if !s.WaitForText("Hello, Ada!", time.Second) {
		t.Fatalf("no greeting:\n%s", screen(s))
	}
}

func TestClickGreetButton(t *testing.T) {
	s := session(t)
	s.Keys("B", "o", "b")
	s.ClickText("Greet")
	if !s.WaitForText("Hello, Bob!", time.Second) {
		t.Fatalf("clicking Greet did not greet:\n%s", screen(s))
	}
}

func TestEmptyNameIsExplained(t *testing.T) {
	s := session(t)
	s.ClickText("Greet")
	if !s.WaitForText("type your name first", time.Second) {
		t.Fatalf("an empty name was not explained:\n%s", screen(s))
	}
}

func TestClickFieldFocusesIt(t *testing.T) {
	s := session(t)
	s.Keys("tab") // focus Greet
	s.ClickText("Your name")
	s.Keys("Z")
	if !s.WaitForText("Z", time.Second) {
		t.Fatalf("typing after clicking the field did not reach it:\n%s", screen(s))
	}
}

func TestTabMovesFocusToButtons(t *testing.T) {
	s := session(t)
	s.Keys("x", "tab", "enter")
	if !s.WaitForText("Hello, x!", time.Second) {
		t.Fatalf("Tab then Enter did not press Greet:\n%s", screen(s))
	}
}

func TestQuit(t *testing.T) {
	s := session(t)
	s.ClickText("Quit")
	deadline := time.Now().Add(time.Second)
	for !s.Done() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if !s.Done() {
		t.Fatal("clicking Quit did not end the program")
	}
}

func TestCtrlCQuits(t *testing.T) {
	s := session(t)
	s.Keys("ctrl+c")
	deadline := time.Now().Add(time.Second)
	for !s.Done() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if !s.Done() {
		t.Fatal("ctrl+c did not end the program")
	}
}

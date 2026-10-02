package main

import (
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/form"
)

func init() { authDelay = 0 }

func send(m model, msg tui.Msg) (model, tui.Cmd) {
	next, cmd := m.Update(msg)
	return next.(model), cmd
}

func typed(m model, s string) model {
	for _, r := range s {
		m, _ = send(m, tui.Key{Type: tui.KeyRunes, Text: string(r)})
	}
	return m
}

func key(m model, t tui.KeyType) (model, tui.Cmd) { return send(m, tui.Key{Type: t}) }

// outcome runs cmd, which may be a cursor-blink tick that only fires later,
// and returns what it produced if that is quick.
func outcome(cmd tui.Cmd) tui.Msg {
	if cmd == nil {
		return nil
	}
	ch := make(chan tui.Msg, 1)
	go func() { ch <- cmd() }()
	select {
	case msg := <-ch:
		return msg
	case <-time.After(100 * time.Millisecond):
		return nil
	}
}

// submit presses Enter and feeds any SubmittedMsg back, as the Program would.
// It returns the model in the authenticating state (or still editing when the
// form was invalid).
func submit(m model) model {
	m, cmd := key(m, tui.KeyEnter)
	if msg, ok := outcome(cmd).(form.SubmittedMsg); ok {
		m, _ = send(m, msg)
	}
	return m
}

// signIn fills the form, submits it and delivers the authentication result.
func signIn(m model, user, password string) model {
	if user != "" { // otherwise focus is already on the password
		m = typed(m, user)
		m, _ = key(m, tui.KeyTab)
	}
	m = typed(m, password)
	m = submit(m)
	if m.state != authenticating {
		return m
	}
	m, _ = send(m, authenticate(m.user, password, m.remember)())
	return m
}

func plain(m model) string { return ansi.StripANSI(m.View()) }

func TestInitFocusesTheUsername(t *testing.T) {
	m := initialModel()
	if m.Init() == nil {
		t.Error("Init returned no Cmd, want the first field's cursor blink")
	}
	if m.form.Current() != "user" || !m.form.Focused() {
		t.Errorf("focus = %q focused=%v, want the user field", m.form.Current(), m.form.Focused())
	}
}

func TestEmptySubmitShowsRequiredErrors(t *testing.T) {
	m := submit(initialModel())
	if m.state != editing {
		t.Fatalf("state = %v, want editing", m.state)
	}
	if m.form.Err("user") == "" || m.form.Err("password") == "" {
		t.Errorf("want required errors, got user=%q password=%q", m.form.Err("user"), m.form.Err("password"))
	}
}

func TestValidSubmitShowsSpinnerAndIgnoresTyping(t *testing.T) {
	m := initialModel()
	m = typed(m, demoUser)
	m, _ = key(m, tui.KeyTab)
	m = typed(m, demoPassword)
	m = submit(m)
	if m.state != authenticating {
		t.Fatalf("state = %v, want authenticating", m.state)
	}
	if !strings.Contains(plain(m), "Signing in") {
		t.Errorf("view should show the spinner, got:\n%s", plain(m))
	}
	before := plain(m)
	m = typed(m, "x")
	if plain(m) != before {
		t.Error("typing changed the view while authenticating")
	}
}

func TestCorrectCredentialsSignIn(t *testing.T) {
	m := signIn(initialModel(), demoUser, demoPassword)
	if m.state != signedIn {
		t.Fatalf("state = %v, want signedIn", m.state)
	}
	view := plain(m)
	if !strings.Contains(view, "Welcome back, "+demoUser) {
		t.Errorf("view should greet the user, got:\n%s", view)
	}
	if strings.Contains(view, demoPassword) {
		t.Error("the signed-in view shows the password")
	}
}

func TestWrongPasswordKeepsUserClearsPasswordAndFocusesIt(t *testing.T) {
	m := signIn(initialModel(), demoUser, "nope")
	if m.state != editing {
		t.Fatalf("state = %v, want editing", m.state)
	}
	v := m.form.Values()
	if v["user"] != demoUser || v["password"] != "" {
		t.Errorf("values = %v, want the user kept and the password cleared", v)
	}
	if m.form.Current() != "password" {
		t.Errorf("focus = %q, want password", m.form.Current())
	}
	if !strings.Contains(plain(m), "Invalid username or password (1 of 3") {
		t.Errorf("view should show the error, got:\n%s", plain(m))
	}
}

func TestThreeFailuresLock(t *testing.T) {
	m := initialModel()
	m = signIn(m, demoUser, "a")
	m = signIn(m, "", "b") // the username is kept from the first try
	m = signIn(m, "", "c")
	if m.state != locked {
		t.Fatalf("state = %v, want locked", m.state)
	}
	if !strings.Contains(plain(m), "Too many failed attempts") {
		t.Errorf("view should say locked, got:\n%s", plain(m))
	}
}

func TestEscQuits(t *testing.T) {
	_, cmd := key(initialModel(), tui.KeyEsc)
	if _, ok := outcome(cmd).(tui.QuitMsg); !ok {
		t.Error("Esc did not quit")
	}
}

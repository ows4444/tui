package main

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/input"
	"github.com/ows4444/tui/layout"
)

func update(m model, msg tui.Msg) (model, tui.Cmd) {
	next, cmd := m.Update(msg)
	return next.(model), cmd
}

func send(m model, msg tui.Msg) model {
	m, _ = update(m, msg)
	return m
}

func key(m model, t tui.KeyType) model { return send(m, tui.Key{Type: t}) }

func typed(m model, s string) model {
	for _, r := range s {
		m = send(m, tui.Key{Type: tui.KeyRunes, Text: string(r)})
	}
	return m
}

// ready is a model at 80x24 with reduced motion, so a sign-in settles at once.
func ready() model { return send(newModel(true), tui.ResizeMsg{Width: 80, Height: 24}) }

func text(m model) string { return ansi.StripANSI(m.View()) }

func click(t testing.TB, m model, id string) model {
	t.Helper()
	r, ok := layout.RectOf(m.screen(), m.size(), id)
	if !ok {
		t.Fatalf("no target %q on screen:\n%s", id, text(m))
	}
	return send(m, tui.MouseEvent{X: r.X + r.W/2, Y: r.Y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
}

func isQuit(cmd tui.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tui.QuitMsg)
	return ok
}

func login(m model, user, pass string) model {
	m = typed(m, user)
	m = key(m, tui.KeyEnter) // username -> password
	m = typed(m, pass)
	return key(m, tui.KeyEnter)
}

func TestStartsOnTheUsername(t *testing.T) {
	m := ready()
	out := text(m)
	for _, want := range []string{"Sign in", "Username", "Password", "show", "[ ] Remember me", "ada / lovelace1815"} {
		if !strings.Contains(out, want) {
			t.Errorf("screen lacks %q:\n%s", want, out)
		}
	}
	if m.focus != onUser {
		t.Fatal("focus does not start on the username")
	}
}

func TestPasswordIsMaskedUntilShown(t *testing.T) {
	m := typed(key(typed(ready(), "ada"), tui.KeyEnter), "secret1")
	if out := text(m); strings.Contains(out, "secret1") || !strings.Contains(out, "•••••••") {
		t.Fatalf("password not masked:\n%s", out)
	}
	m = click(t, m, "reveal")
	if out := text(m); !strings.Contains(out, "secret1") || !strings.Contains(out, "hide") {
		t.Fatalf("show did not reveal:\n%s", out)
	}
	m = send(m, tui.Key{Type: tui.KeyCtrl, Code: 't'})
	if strings.Contains(text(m), "secret1") {
		t.Fatal("Ctrl+T did not hide it again")
	}
}

func TestGoodCredentialsSignIn(t *testing.T) {
	m := login(ready(), "ada", demoPassword)
	if m.phase != signedIn || !strings.Contains(text(m), "Welcome back, ada.") {
		t.Fatalf("not signed in:\n%s", text(m))
	}
}

func TestWrongPasswordKeepsUserClearsPasswordAndFocusesIt(t *testing.T) {
	m := login(ready(), "ada", "nope")
	if m.phase != signingIn || m.user.Value() != "ada" || m.pass.Value() != "" || m.focus != onPass {
		t.Fatalf("after a wrong password: phase=%v user=%q pass=%q focus=%v", m.phase, m.user.Value(), m.pass.Value(), m.focus)
	}
	if !strings.Contains(text(m), "✗ Wrong username or password") {
		t.Fatalf("no error:\n%s", text(m))
	}
}

func TestEmptyFieldsAreExplained(t *testing.T) {
	m := key(typed(ready(), "ada"), tui.KeyTab) // to the password
	m = key(m, tui.KeyEnter)
	if !strings.Contains(text(m), "Enter your password") || m.busy || m.focus != onPass {
		t.Fatalf("\n%s", text(m))
	}
	m = key(ready(), tui.KeyTab)
	m = key(m, tui.KeyEnter)
	if !strings.Contains(text(m), "Enter your username") || m.focus != onUser {
		t.Fatalf("\n%s", text(m))
	}
	if m = key(ready(), tui.KeyEnter); m.focus != onPass {
		t.Fatal("Enter on the username did not move to the password")
	}
}

// wrong tries the demo user with a wrong password, from any focus.
func wrong(m model) model {
	m.user.SetValue(demoUser)
	next, _ := m.focusOn(onPass)
	m = typed(next.(model), "bad")
	return key(m, tui.KeyEnter)
}

func TestThreeFailuresLockThenUnlock(t *testing.T) {
	m := ready()
	for i := 0; i < maxTries; i++ {
		m = wrong(m)
	}
	if m.lock != lockSeconds || !strings.Contains(text(m), "Too many tries. Try again in 5s") {
		t.Fatalf("not locked: lock=%d\n%s", m.lock, text(m))
	}
	m.user.SetValue(demoUser)
	m = typed(m, demoPassword)
	if m = key(m, tui.KeyEnter); m.phase == signedIn {
		t.Fatal("signed in while locked")
	}
	for i := 0; i < lockSeconds; i++ {
		m = send(m, lockMsg{m.seq})
	}
	if m.lock != 0 || m.fails != 0 {
		t.Fatalf("lock=%d fails=%d after the countdown", m.lock, m.fails)
	}
	m.pass.Reset()
	m = typed(m, demoPassword)
	if m = key(m, tui.KeyEnter); m.phase != signedIn {
		t.Fatalf("could not sign in after the lock ended:\n%s", text(m))
	}
}

func TestStaleTimersAreIgnored(t *testing.T) {
	m := ready()
	m.lock, m.seq = 3, 4
	if m = send(m, lockMsg{seq: 1}); m.lock != 3 {
		t.Fatal("a stale lock tick counted down")
	}
	m.shake = len(shakeOffsets)
	if m = send(m, shakeMsg{seq: 1, step: 2}); m.shake != len(shakeOffsets) {
		t.Fatal("a stale shake step moved the panel")
	}
}

func TestShakeRunsThenSettles(t *testing.T) {
	m := send(newModel(false), tui.ResizeMsg{Width: 80, Height: 24})
	m = typed(key(typed(m, "ada"), tui.KeyEnter), "bad")
	m, cmd := update(m, tui.Key{Type: tui.KeyEnter})
	if cmd == nil || !m.busy {
		t.Fatal("sign-in did not start")
	}
	m = send(m, authMsg{m.seq, false})
	if m.shake != 0 {
		t.Fatalf("shake = %d after a failure, want 0", m.shake)
	}
	for step := 1; step <= len(shakeOffsets); step++ {
		m = send(m, shakeMsg{m.seq, step})
	}
	if m.shake != len(shakeOffsets) {
		t.Fatalf("the panel did not settle: shake = %d", m.shake)
	}
}

func TestReducedMotionDoesNotShake(t *testing.T) {
	m := login(ready(), "ada", "bad")
	if m.shake != len(shakeOffsets) {
		t.Fatalf("shake = %d with reduced motion", m.shake)
	}
}

func TestInputIsIgnoredWhileSigningIn(t *testing.T) {
	m := send(newModel(false), tui.ResizeMsg{Width: 80, Height: 24})
	m = typed(key(typed(m, "ada"), tui.KeyEnter), demoPassword)
	m = key(m, tui.KeyEnter)
	m = typed(m, "x")
	if !m.busy || m.pass.Value() != demoPassword {
		t.Fatalf("typing during sign-in changed the password: busy=%v", m.busy)
	}
	m = send(m, authMsg{m.seq + 5, true}) // not this sign-in's answer
	if m.phase == signedIn {
		t.Fatal("an answer for another attempt signed in")
	}
	m = send(m, authMsg{m.seq, true})
	if m.phase != signedIn {
		t.Fatal("the real answer did not sign in")
	}
}

func TestCapsLockHint(t *testing.T) {
	m := key(typed(ready(), "ada"), tui.KeyEnter)
	if m = typed(m, "AB"); strings.Contains(text(m), "Caps Lock") {
		t.Fatal("hint shown for two letters")
	}
	if m = typed(m, "C"); !strings.Contains(text(m), "⚠ Caps Lock may be on") {
		t.Fatalf("no hint for three capitals:\n%s", text(m))
	}
	if m = typed(m, "d"); strings.Contains(text(m), "Caps Lock") {
		t.Fatal("hint stays with a lowercase letter")
	}
}

func TestRememberMeKeepsTheUsernameAfterSignOut(t *testing.T) {
	m := click(t, ready(), "remember")
	if !m.remember || !strings.Contains(text(m), "[x] Remember me") {
		t.Fatal("clicking the box did not tick it")
	}
	m = login(click(t, m, "user"), "ada", demoPassword)
	m = click(t, m, "signout")
	if m.phase != signingIn || m.user.Value() != "ada" || m.focus != onPass || !m.remember {
		t.Fatalf("after sign-out: phase=%v user=%q focus=%v", m.phase, m.user.Value(), m.focus)
	}
	m = login(newModel(true), "ada", demoPassword)
	m = key(m, tui.KeyEnter)
	if m.user.Value() != "" {
		t.Fatal("without Remember me the username was kept")
	}
}

func TestTabAndShiftTab(t *testing.T) {
	m := ready()
	for _, want := range []focus{onPass, onRemember, onButton, onUser} {
		if m = key(m, tui.KeyTab); m.focus != want {
			t.Fatalf("Tab went to %v, want %v", m.focus, want)
		}
	}
	m = send(m, tui.Key{Type: tui.KeyTab, Mod: input.ModShift})
	if m.focus != onButton {
		t.Fatalf("Shift+Tab from the first field went to %v, want the button", m.focus)
	}
}

func TestSpaceTicksTheBoxAndPressesTheButton(t *testing.T) {
	m := key(key(ready(), tui.KeyTab), tui.KeyTab)
	if m = typed(m, " "); !m.remember {
		t.Fatal("Space did not tick the box")
	}
	b := key(key(typed(key(typed(ready(), "ada"), tui.KeyEnter), demoPassword), tui.KeyTab), tui.KeyTab)
	if b.focus != onButton {
		t.Fatal("not on the button")
	}
	if b = typed(b, " "); b.phase != signedIn {
		t.Fatal("Space on the button did not sign in")
	}
}

func TestClickFieldsAndButton(t *testing.T) {
	m := click(t, ready(), "pass")
	if m.focus != onPass {
		t.Fatal("clicking the password did not focus it")
	}
	m = typed(m, demoPassword)
	m = click(t, click(t, m, "user"), "user")
	m = typed(m, "ada")
	m = click(t, m, "button")
	if m.phase != signedIn {
		t.Fatalf("the Sign in button did not sign in:\n%s", text(m))
	}
}

func TestEscClearsThePassword(t *testing.T) {
	m := login(ready(), "ada", "bad")
	m = typed(m, "x")
	m = key(m, tui.KeyEsc)
	if m.pass.Value() != "" || m.err != "" {
		t.Fatal("Esc left the password or the error")
	}
}

func TestHelpAndQuit(t *testing.T) {
	m := key(ready(), tui.KeyF1)
	if !strings.Contains(text(m), "clear the password") {
		t.Fatalf("\n%s", text(m))
	}
	if m = key(m, tui.KeyEsc); m.help {
		t.Fatal("Esc did not close help")
	}
	if _, cmd := update(ready(), tui.Key{Type: tui.KeyCtrlC}); !isQuit(cmd) {
		t.Fatal("Ctrl+C did not quit")
	}
	if _, cmd := update(typed(ready(), "q"), tui.Key{Type: tui.KeyRunes, Text: "q"}); isQuit(cmd) {
		t.Fatal("typing q quit")
	}
	s := login(ready(), "ada", demoPassword)
	if _, cmd := update(s, tui.Key{Type: tui.KeyRunes, Text: "q"}); !isQuit(cmd) {
		t.Fatal("q on the welcome screen did not quit")
	}
}

func TestHover(t *testing.T) {
	m := ready()
	r, _ := layout.RectOf(m.screen(), m.size(), "reveal")
	m = send(m, tui.MouseEvent{X: r.X, Y: r.Y, Action: tui.MouseActionMotion})
	if m.hover != "reveal" {
		t.Fatalf("hover = %q", m.hover)
	}
}

func TestTinyTerminalsStayInside(t *testing.T) {
	m := login(ready(), "ada", "bad")
	for _, sz := range [][2]int{{40, 10}, {20, 6}, {10, 4}} {
		m = send(m, tui.ResizeMsg{Width: sz[0], Height: sz[1]})
		for i, line := range strings.Split(m.View(), "\n") {
			if w := ansi.Width(line); w > sz[0] {
				t.Fatalf("%dx%d: line %d is %d wide: %q", sz[0], sz[1], i, w, line)
			}
		}
	}
}

package main

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/form"
	"github.com/ows4444/tui/layout"
)

func update(m model, msg tui.Msg) (model, tui.Cmd) {
	next, cmd := m.Update(msg)
	return next.(model), cmd
}

// send delivers msg and, like the Program, feeds back any message the form's
// Cmd produces (the submit).
func send(m model, msg tui.Msg) model {
	m, cmd := update(m, msg)
	if cmd != nil {
		if out := cmd(); out != nil {
			if sub, ok := out.(form.SubmittedMsg); ok {
				m, _ = update(m, sub)
			}
		}
	}
	return m
}

func key(m model, t tui.KeyType) model { return send(m, tui.Key{Type: t}) }

func typed(m model, s string) model {
	for _, r := range s {
		m = send(m, tui.Key{Type: tui.KeyRunes, Text: string(r)})
	}
	return m
}

func ready() model { return send(initialModel(), tui.ResizeMsg{Width: 80, Height: 24}) }

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

// fillAccount fills step one with valid values and moves on.
func fillAccount(m model) model {
	m = typed(m, "Ada")
	m = key(m, tui.KeyTab)
	m = typed(m, "ada@example.com")
	m = key(m, tui.KeyTab)
	m = typed(m, "longenough")
	return key(m, tui.KeyEnter)
}

func TestFirstStepShowsTheStepBar(t *testing.T) {
	out := text(ready())
	for _, want := range []string{"1 Account", "2 Plan", "3 Review", "Name", "Email", "Password"} {
		if !strings.Contains(out, want) {
			t.Errorf("screen lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Billing") {
		t.Errorf("the billing step shows before a paid plan is chosen:\n%s", out)
	}
}

func TestStepIsCheckedBeforeMovingOn(t *testing.T) {
	m := key(typed(ready(), "Ada"), tui.KeyEnter)
	if m.current() != 0 {
		t.Fatal("Enter moved on with empty fields")
	}
	if out := text(m); !strings.Contains(out, "Email") || !strings.Contains(strings.ToLower(out), "required") {
		t.Fatalf("no error shown:\n%s", out)
	}
}

func TestFreePlanSkipsBilling(t *testing.T) {
	m := fillAccount(ready())
	if m.current() != 1 {
		t.Fatalf("step = %d after the account step, want 1", m.current())
	}
	if got := m.steps(); len(got) != 3 {
		t.Fatalf("steps = %v, want Account, Plan, Review", got)
	}
}

func TestPaidPlanAddsBilling(t *testing.T) {
	m := fillAccount(ready())
	m = send(m, tui.Key{Type: tui.KeyRight}) // Free -> Pro
	if got := m.steps(); len(got) != 4 || got[2] != "Billing" {
		t.Fatalf("steps = %v, want a Billing step before Review", got)
	}
}

// walk goes through every step on the Pro plan and stops on the review.
func walk(t *testing.T) model {
	t.Helper()
	m := fillAccount(ready())
	m = send(m, tui.Key{Type: tui.KeyRight})
	m = key(m, tui.KeyTab) // newsletter
	m = key(m, tui.KeyTab) // terms
	m = typed(m, " ")
	m = key(m, tui.KeyEnter)
	if m.current() != 2 {
		t.Fatalf("step = %d after the plan step, want 2 (billing):\n%s", m.current(), text(m))
	}
	m = typed(m, "Engines Ltd")
	m = key(m, tui.KeyTab)
	m = send(m, tui.Key{Type: tui.KeyRight}) // 5 seats
	m = key(m, tui.KeyEnter)
	if m.phase != reviewing {
		t.Fatalf("not on the review:\n%s", text(m))
	}
	return m
}

func TestReviewListsTheAnswersWithoutThePassword(t *testing.T) {
	out := text(walk(t))
	for _, want := range []string{"Ada", "ada@example.com", "10 characters, not shown", "Pro", "Engines Ltd", "5", "Create account"} {
		if !strings.Contains(out, want) {
			t.Errorf("review lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "longenough") {
		t.Fatal("the review shows the password")
	}
}

func TestCreateFinishes(t *testing.T) {
	m := key(walk(t), tui.KeyEnter)
	if m.phase != finished || !strings.Contains(text(m), "Account created") {
		t.Fatalf("not finished:\n%s", text(m))
	}
	if _, cmd := update(m, tui.Key{Type: tui.KeyEnter}); !isQuit(cmd) {
		t.Fatal("Enter on the last screen did not quit")
	}
}

func TestClickAFinishedStepGoesBack(t *testing.T) {
	m := click(t, walk(t), "step-0")
	if m.phase != editing || m.current() != 0 {
		t.Fatalf("phase=%v step=%d, want editing at step 0:\n%s", m.phase, m.current(), text(m))
	}
	if v := m.form.Values()["name"]; v != "Ada" {
		t.Fatalf("going back lost the answers: name = %q", v)
	}
}

func TestClickingAFutureStepDoesNothing(t *testing.T) {
	m := click(t, ready(), "step-2")
	if m.current() != 0 {
		t.Fatalf("a click skipped ahead to step %d", m.current())
	}
}

func TestBackButtonAndEsc(t *testing.T) {
	m := click(t, walk(t), "back")
	if m.current() != 2 || m.phase != editing {
		t.Fatalf("Back from the review: step=%d phase=%v", m.current(), m.phase)
	}
	m = key(m, tui.KeyEsc)
	if m.current() != 1 {
		t.Fatalf("Esc: step = %d, want 1", m.current())
	}
	m = key(key(m, tui.KeyPgUp), tui.KeyEsc) // already at the first step
	if m.current() != 0 {
		t.Fatalf("step = %d at the first step", m.current())
	}
}

func TestNextButtonChecksTheStep(t *testing.T) {
	m := click(t, ready(), "next")
	if m.current() != 0 || !strings.Contains(strings.ToLower(text(m)), "required") {
		t.Fatalf("Next with empty fields:\n%s", text(m))
	}
}

func TestClickAFieldFocusesIt(t *testing.T) {
	m := ready()
	y := -1
	for i, line := range strings.Split(text(m), "\n") {
		if strings.Contains(line, "Email:") {
			y = i
		}
	}
	x := strings.Index(strings.Split(text(m), "\n")[y], "Email:") + 8
	m = send(m, tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if got := m.form.Current(); got != "email" {
		t.Fatalf("clicking the Email row focused %q:\n%s", got, text(m))
	}
}

func TestHelpToggles(t *testing.T) {
	m := key(ready(), tui.KeyF1)
	if !strings.Contains(text(m), "check the step") {
		t.Fatalf("help not shown:\n%s", text(m))
	}
	if m = key(m, tui.KeyEsc); m.help {
		t.Fatal("Esc did not close help")
	}
}

func TestQuit(t *testing.T) {
	if _, cmd := update(ready(), tui.Key{Type: tui.KeyCtrlC}); !isQuit(cmd) {
		t.Error("Ctrl+C did not quit")
	}
	m := typed(ready(), "q")
	if _, cmd := update(m, tui.Key{Type: tui.KeyRunes, Text: "q"}); isQuit(cmd) {
		t.Error("typing q quit")
	}
}

func TestTinyTerminalStaysInside(t *testing.T) {
	m := walk(t)
	for _, sz := range [][2]int{{40, 10}, {20, 6}, {10, 4}} {
		m = send(m, tui.ResizeMsg{Width: sz[0], Height: sz[1]})
		for i, line := range strings.Split(m.View(), "\n") {
			if w := ansi.Width(line); w > sz[0] {
				t.Fatalf("%dx%d: line %d is %d wide", sz[0], sz[1], i, w)
			}
		}
	}
}

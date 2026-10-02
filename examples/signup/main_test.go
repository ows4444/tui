package main

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/form"
	"github.com/ows4444/tui/layout"
)

var updateGolden = flag.Bool("update", false, "rewrite the golden files")

const secret = "correct horse battery"

func send(m model, msg tui.Msg) (model, tui.Cmd) {
	next, cmd := m.Update(msg)
	return next.(model), cmd
}

func typed(m model, s string) model {
	for _, r := range s {
		m, _ = send(m, tui.Key{Type: tui.KeyRunes, Text: string([]rune{r})})
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

// fillExceptTerms fills every field but leaves the terms box unticked, with the
// Pro plan chosen; focus ends on the plan field.
func fillExceptTerms(m model) model {
	m = typed(m, "Ada Lovelace")
	m, _ = key(m, tui.KeyTab)
	m = typed(m, "ada@example.com")
	m, _ = key(m, tui.KeyTab)
	m = typed(m, secret)
	m, _ = key(m, tui.KeyTab) // plan
	m, _ = key(m, tui.KeyRight)
	return m
}

// fill fills every field and ticks the terms box.
func fill(m model) model {
	m = fillExceptTerms(m)
	m, _ = key(m, tui.KeyTab) // terms
	m, _ = send(m, tui.Key{Type: tui.KeySpace})
	return m
}

// submit presses Enter and feeds any SubmittedMsg back, as the Program would.
func submit(m model) model {
	m, cmd := key(m, tui.KeyEnter)
	if msg, ok := outcome(cmd).(form.SubmittedMsg); ok {
		m, _ = send(m, msg)
	}
	return m
}

func plain(m model) string { return ansi.StripANSI(m.View()) }

func TestInitFocusesTheFirstField(t *testing.T) {
	m := initialModel()
	if m.Init() == nil {
		t.Error("Init returned no Cmd, want the first field's cursor blink")
	}
	if m.form.Current() != "name" || !m.form.Focused() {
		t.Errorf("focus = %q focused=%v, want the name field", m.form.Current(), m.form.Focused())
	}
}

func TestTypingAndTabFillTheFields(t *testing.T) {
	m := fill(initialModel())
	v := m.form.Values()
	if v["name"] != "Ada Lovelace" || v["email"] != "ada@example.com" || v["password"] != secret ||
		v["plan"] != "Pro" || v["terms"] != "true" {
		t.Errorf("values = %v", v)
	}
}

func TestInvalidSubmitShowsEveryErrorAndFocusesTheFirstInvalidField(t *testing.T) {
	m := initialModel()
	m, _ = key(m, tui.KeyTab)
	m, _ = key(m, tui.KeyTab) // on the password field
	m = typed(m, "short")
	m = submit(m)
	if m.done {
		t.Fatal("an invalid form was accepted")
	}
	if e := m.form.Err("terms"); e != "you must accept the terms" {
		t.Errorf("terms error = %q, want the terms message", e)
	}
	if m.form.Current() != "name" {
		t.Errorf("focus = %q, want the first invalid field, name", m.form.Current())
	}
	view := plain(m)
	for _, name := range []string{"name", "email", "password"} {
		e := m.form.Err(name)
		if e == "" || !strings.Contains(view, e) {
			t.Errorf("%s error %q is not on screen:\n%s", name, e, view)
		}
	}
}

func TestValidSubmitShowsTheSummaryWithoutThePassword(t *testing.T) {
	m := submit(fill(initialModel()))
	if !m.done {
		t.Fatalf("a valid form was not accepted:\n%s", plain(m))
	}
	view := plain(m)
	for _, want := range []string{"Account created", "Ada Lovelace", "ada@example.com", "Plan:     Pro", "21 characters, not shown"} {
		if !strings.Contains(view, want) {
			t.Errorf("summary is missing %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "correct") || strings.Contains(view, "horse") {
		t.Errorf("the summary shows the password:\n%s", view)
	}
}

// The password never reaches the screen or the accessible text, before or
// after submitting.
func TestThePasswordIsNeverShown(t *testing.T) {
	m := fill(initialModel())
	check := func(when string, m model) {
		t.Helper()
		for what, out := range map[string]string{"View": m.View(), "Linearize": m.form.Linearize()} {
			if strings.Contains(ansi.StripANSI(out), "correct") || strings.Contains(ansi.StripANSI(out), "horse") {
				t.Errorf("%s: %s shows the password: %q", when, what, out)
			}
		}
	}
	check("while typing", m)
	// A failed submit with the password filled in: name and email are empty.
	failed := initialModel()
	failed, _ = key(failed, tui.KeyTab)
	failed, _ = key(failed, tui.KeyTab)
	failed = submit(typed(failed, secret))
	if failed.done || failed.form.Err("name") == "" {
		t.Fatal("test premise changed: the incomplete form was accepted")
	}
	check("after a failed submit", failed)
	check("after success", submit(m))
}

func TestQuitKeys(t *testing.T) {
	quits := func(m model, k tui.Key) bool {
		_, cmd := send(m, k)
		if cmd == nil {
			return false
		}
		_, ok := outcome(cmd).(tui.QuitMsg)
		return ok
	}
	if !quits(initialModel(), tui.Key{Type: tui.KeyCtrlC}) || !quits(initialModel(), tui.Key{Type: tui.KeyEsc}) {
		t.Error("ctrl+c and esc must quit the form")
	}
	done := submit(fill(initialModel()))
	if !quits(done, tui.Key{Type: tui.KeyEnter}) || !quits(done, tui.Key{Type: tui.KeyRunes, Text: "q"}) {
		t.Error("enter and q must quit the summary")
	}
	if quits(initialModel(), tui.Key{Type: tui.KeyRunes, Text: "q"}) {
		t.Error("typing q in the form must not quit")
	}
}

// At any terminal size the screen fills exactly the space it is given, in
// every state.
func TestScreenRendersExactlyTheRequestedSize(t *testing.T) {
	invalid := submit(initialModel())
	states := map[string]model{"empty": initialModel(), "errors": invalid, "filled": fill(initialModel()), "done": submit(fill(initialModel()))}
	for name, m := range states {
		for _, s := range []layout.Size{{W: 40, H: 10}, {W: 80, H: 24}, {W: 300, H: 80}, {W: 5, H: 3}} {
			lines := strings.Split(m.screen().Render(s), "\n")
			if len(lines) != s.H {
				t.Fatalf("%s %v: %d rows, want %d", name, s, len(lines), s.H)
			}
			for i, l := range lines {
				if w := ansi.Width(l); w != s.W {
					t.Errorf("%s %v: row %d is %d wide, want %d", name, s, i, w, s.W)
				}
			}
		}
	}
}

// The natural view has no row that spills past the box's right border.
func TestNaturalViewHasNoOverflowingRow(t *testing.T) {
	for name, m := range map[string]model{"empty": initialModel(), "errors": submit(initialModel()), "done": submit(fill(initialModel()))} {
		lines := strings.Split(m.View(), "\n")
		want := ansi.Width(lines[0])
		for i, l := range lines {
			if w := ansi.Width(l); w != want {
				t.Errorf("%s: row %d is %d wide, the top border is %d: %q", name, i, w, want, ansi.StripANSI(l))
			}
		}
	}
}

func TestScreenUsesLayoutNode(t *testing.T) {
	b, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(b)
	if !strings.Contains(src, "layout.Draw(") || !strings.Contains(src, "LayoutNode()") {
		t.Error("main.go does not draw the form through layout.Draw and LayoutNode")
	}
	for _, banned := range []string{"JoinHorizontal", "JoinVertical"} {
		if strings.Contains(src, banned) {
			t.Errorf("main.go still uses the string layout: %s", banned)
		}
	}
}

// Golden views, ANSI stripped so they stay readable. Regenerate with -update.
func TestViewGolden(t *testing.T) {
	for name, m := range map[string]model{
		"empty":  initialModel(),
		"errors": submit(initialModel()),
		"done":   submit(fill(initialModel())),
	} {
		got := ansi.StripANSI(m.View())
		golden := filepath.Join("testdata", name+".golden")
		if *updateGolden {
			if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		want, err := os.ReadFile(golden)
		if err != nil {
			t.Fatal(err)
		}
		if got != string(want) {
			t.Errorf("view differs from %s (go test -update to regenerate):\n%s", golden, got)
		}
	}
}

// Without the terms ticked the form is not accepted: the terms error is shown
// and focus moves to that field, although every other field is valid.
func TestSubmitWithoutTheTermsShowsTheErrorAndFocusesTheBox(t *testing.T) {
	m := submit(fillExceptTerms(initialModel()))
	if m.done {
		t.Fatalf("the form was accepted without the terms:\n%s", plain(m))
	}
	if m.form.Current() != "terms" {
		t.Errorf("focus = %q, want the terms field", m.form.Current())
	}
	if view := plain(m); !strings.Contains(view, "you must accept the terms") {
		t.Errorf("the terms error is not on screen:\n%s", view)
	}
	for _, name := range []string{"name", "email", "password", "plan"} {
		if e := m.form.Err(name); e != "" {
			t.Errorf("valid field %s shows the error %q", name, e)
		}
	}
	// Ticking the box clears the error, and the form can then be submitted.
	m, _ = send(m, tui.Key{Type: tui.KeySpace})
	if e := m.form.Err("terms"); e != "" {
		t.Errorf("ticking the box left the error %q", e)
	}
	if m = submit(m); !m.done {
		t.Errorf("the form was not accepted after ticking the terms:\n%s", plain(m))
	}
}

// The plan can be changed with the arrow keys and lands in the summary.
func TestThePlanChoiceReachesTheSummary(t *testing.T) {
	m := fillExceptTerms(initialModel()) // focus on the plan, Pro chosen
	m, _ = key(m, tui.KeyLeft)           // Free
	m, _ = key(m, tui.KeyTab)
	m, _ = send(m, tui.Key{Type: tui.KeySpace})
	m = submit(m)
	if !m.done {
		t.Fatalf("the form was not accepted:\n%s", plain(m))
	}
	if view := plain(m); !strings.Contains(view, "Plan:     Free") {
		t.Errorf("the summary does not show the chosen plan:\n%s", view)
	}
}

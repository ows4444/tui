package form

import (
	"regexp"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/input"
)

func typed(m Model, s string) Model {
	for _, r := range s {
		m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: string([]rune{r})})
	}
	return m
}

func tab(m Model) Model {
	m, _ = m.Update(tui.Key{Type: tui.KeyTab})
	return m
}

func shiftTab(m Model) Model {
	m, _ = m.Update(tui.Key{Type: tui.KeyTab, Mod: input.ModShift})
	return m
}

func plain(m Model) string { return ansi.StripANSI(m.View()) }

func signup() Model {
	m := New(
		Field{Name: "name", Label: "Name", Validators: []Validator{Required()}},
		Field{Name: "email", Label: "Email", Validators: []Validator{Required(), Email()}},
		Field{Name: "password", Label: "Password", Secret: true, Validators: []Validator{Required(), MinLen(8)}},
	)
	m.Focus()
	return m
}

// When every field passes, Submit yields SubmittedMsg carrying the values.
func TestSubmitWithValidFieldsEmitsSubmittedMsg(t *testing.T) {
	m := signup()
	m = typed(m, "Ada")
	m = tab(m)
	m = typed(m, "ada@example.com")
	m = tab(m)
	m = typed(m, "correct horse")

	m, cmd := m.Submit()
	if cmd == nil {
		t.Fatal("Submit returned no Cmd for a valid form")
	}
	got, ok := cmd().(SubmittedMsg)
	if !ok {
		t.Fatalf("Cmd produced %T, want SubmittedMsg", cmd())
	}
	want := map[string]string{"name": "Ada", "email": "ada@example.com", "password": "correct horse"}
	for k, v := range want {
		if got.Values[k] != v {
			t.Errorf("Values[%q] = %q, want %q", k, got.Values[k], v)
		}
	}
	if len(got.Values) != len(want) {
		t.Errorf("Values = %v, want exactly %v", got.Values, want)
	}
	for _, name := range []string{"name", "email", "password"} {
		if e := m.Err(name); e != "" {
			t.Errorf("Err(%q) = %q after a valid submit", name, e)
		}
	}
}

// When any field fails, every failing field's first error is shown, the first
// invalid field is focused and no SubmittedMsg is emitted.
func TestSubmitWithInvalidFieldsShowsErrorsAndFocusesTheFirst(t *testing.T) {
	m := signup()
	m = tab(m) // focus starts on the last...first; move away so refocusing is observable
	m = tab(m)
	m = typed(m, "short")

	m, cmd := m.Submit()
	if cmd != nil {
		if _, ok := cmd().(SubmittedMsg); ok {
			t.Fatal("an invalid form emitted SubmittedMsg")
		}
	}
	if m.Current() != "name" {
		t.Errorf("focused field = %q, want the first invalid field %q", m.Current(), "name")
	}
	if m.Err("name") == "" || m.Err("email") == "" || m.Err("password") == "" {
		t.Errorf("errors name=%q email=%q password=%q, want all three set",
			m.Err("name"), m.Err("email"), m.Err("password"))
	}
	view := plain(m)
	for _, name := range []string{"name", "email", "password"} {
		if !strings.Contains(view, m.Err(name)) {
			t.Errorf("view does not show the %s error %q:\n%s", name, m.Err(name), view)
		}
	}
}

// Validators run in order and stop at the first error.
func TestValidatorsStopAtTheFirstError(t *testing.T) {
	var calls []string
	track := func(name string, fail bool) Validator {
		return func(string) error {
			calls = append(calls, name)
			if fail {
				return errString(name + " failed")
			}
			return nil
		}
	}
	m := New(Field{Name: "f", Label: "F", Validators: []Validator{track("a", false), track("b", true), track("c", false)}})
	m, _ = m.Submit()
	if got := strings.Join(calls, ","); got != "a,b" {
		t.Errorf("validators ran %q, want a,b", got)
	}
	if m.Err("f") != "b failed" {
		t.Errorf("Err = %q, want %q", m.Err("f"), "b failed")
	}
}

type errString string

func (e errString) Error() string { return string(e) }

// After an error is shown, editing the field re-checks it: the error stays
// while the value is still invalid and clears once it validates.
func TestEditingClearsAnErrorOnceTheValueValidates(t *testing.T) {
	m := New(Field{Name: "pw", Label: "Password", Validators: []Validator{MinLen(3)}})
	m.Focus()
	m = typed(m, "ab")
	if m.Err("pw") != "" {
		t.Fatalf("an error appeared before Submit: %q", m.Err("pw"))
	}
	m, _ = m.Submit()
	if m.Err("pw") == "" {
		t.Fatal("Submit did not report MinLen(3) for \"ab\"")
	}
	m = typed(m, "c") // "abc" validates
	if m.Err("pw") != "" {
		t.Errorf("error %q was not cleared after the value became valid", m.Err("pw"))
	}

	m = New(Field{Name: "pw", Label: "Password", Validators: []Validator{Required(), MinLen(3)}})
	m.Focus()
	m, _ = m.Submit()
	if m.Err("pw") != "required" {
		t.Fatalf("Err = %q, want required", m.Err("pw"))
	}
	m = typed(m, "a") // no longer empty, but still shorter than 3
	if m.Err("pw") != "must be at least 3 characters" {
		t.Errorf("Err = %q, want the MinLen error to replace the Required error", m.Err("pw"))
	}
}

// Tab and Shift+Tab move focus to the next and previous field, wrapping.
func TestTabAndShiftTabMoveFocusAndWrap(t *testing.T) {
	m := signup()
	steps := []struct {
		move func(Model) Model
		want string
	}{
		{tab, "email"}, {tab, "password"}, {tab, "name"},
		{shiftTab, "password"}, {shiftTab, "email"},
	}
	if m.Current() != "name" {
		t.Fatalf("initial focus = %q, want name", m.Current())
	}
	for i, s := range steps {
		m = s.move(m)
		if m.Current() != s.want {
			t.Fatalf("step %d: focus = %q, want %q", i, m.Current(), s.want)
		}
	}
}

// Typing goes to the focused field only.
func TestTypingReachesOnlyTheFocusedField(t *testing.T) {
	m := signup()
	m = typed(m, "a")
	m = tab(m)
	m = typed(m, "b")
	v := m.Values()
	if v["name"] != "a" || v["email"] != "b" || v["password"] != "" {
		t.Errorf("values = %v, want name=a email=b password empty", v)
	}
}

// Enter submits.
func TestEnterSubmits(t *testing.T) {
	m := New(Field{Name: "x", Label: "X", Value: "ok"})
	m.Focus()
	_, cmd := m.Update(tui.Key{Type: tui.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter returned no Cmd")
	}
	if _, ok := cmd().(SubmittedMsg); !ok {
		t.Fatalf("Enter produced %T, want SubmittedMsg", cmd())
	}
}

// The built-in validators accept and reject what their names imply.
func TestBuiltInValidators(t *testing.T) {
	re := Match(regexp.MustCompile(`^\d+$`), "digits only")
	cases := []struct {
		name string
		v    Validator
		in   string
		ok   bool
	}{
		{"Required empty", Required(), "", false},
		{"Required spaces", Required(), "   ", false},
		{"Required value", Required(), "x", true},
		{"MinLen short", MinLen(3), "ab", false},
		{"MinLen exact", MinLen(3), "abc", true},
		{"MinLen counts runes", MinLen(3), "äöü", true},
		{"MinLen empty passes", MinLen(3), "", true},
		{"MaxLen long", MaxLen(3), "abcd", false},
		{"MaxLen exact", MaxLen(3), "abc", true},
		{"MaxLen counts runes", MaxLen(3), "äöü", true},
		{"Email ok", Email(), "ada@example.com", true},
		{"Email no at", Email(), "ada.example.com", false},
		{"Email no domain dot", Email(), "ada@example", false},
		{"Email with name", Email(), "Ada <ada@example.com>", false},
		{"Email empty passes", Email(), "", true},
		{"Match ok", re, "123", true},
		{"Match bad", re, "12a", false},
		{"Match empty passes", re, "", true},
	}
	for _, c := range cases {
		if err := c.v(c.in); (err == nil) != c.ok {
			t.Errorf("%s: %q gave error %v, want ok=%v", c.name, c.in, err, c.ok)
		}
	}
	if err := re("12a"); err == nil || err.Error() != "digits only" {
		t.Errorf("Match error = %v, want the supplied message", err)
	}
}

// A Secret field is masked in View; its real value is still returned by Values.
func TestSecretFieldIsMaskedInView(t *testing.T) {
	m := signup()
	m = tab(m)
	m = tab(m)
	m = typed(m, "hunter2hunter2")
	if strings.Contains(m.View(), "hunter2") {
		t.Errorf("View contains the secret value:\n%s", m.View())
	}
	if !strings.Contains(plain(m), "•") {
		t.Errorf("View shows no mask characters:\n%s", plain(m))
	}
	if m.Values()["password"] != "hunter2hunter2" {
		t.Errorf("Values()[password] = %q", m.Values()["password"])
	}
}

// Update on a copy must not change the model it was called on.
func TestUpdateDoesNotChangeTheReceiver(t *testing.T) {
	m := signup()
	before := m.Values()["name"]
	_ = typed(m, "zzz")
	if m.Values()["name"] != before {
		t.Errorf("receiver changed: name = %q", m.Values()["name"])
	}
	m2, _ := m.Submit()
	if m.Err("name") != "" || m2.Err("name") == "" {
		t.Errorf("Submit leaked into the receiver: old=%q new=%q", m.Err("name"), m2.Err("name"))
	}
}

func TestInitialValuesAndBlurredFormIgnoresInput(t *testing.T) {
	m := New(Field{Name: "a", Label: "A", Value: "start"})
	if m.Values()["a"] != "start" {
		t.Fatalf("initial value = %q, want start", m.Values()["a"])
	}
	m = typed(m, "x") // not focused
	if m.Values()["a"] != "start" {
		t.Errorf("an unfocused form accepted input: %q", m.Values()["a"])
	}
	if m.Focused() {
		t.Error("New returned a focused form")
	}
	m.Focus()
	if !m.Focused() {
		t.Error("Focus did not focus the form")
	}
	m.Blur()
	if m.Focused() {
		t.Error("Blur did not blur the form")
	}
}

func TestEmptyFormSubmitsEmptyValues(t *testing.T) {
	m := New()
	m.Focus()
	_, cmd := m.Submit()
	msg, ok := cmd().(SubmittedMsg)
	if !ok || len(msg.Values) != 0 {
		t.Fatalf("empty form Submit = %#v", msg)
	}
	_, _ = m.Update(tui.Key{Type: tui.KeyTab}) // must not panic
}

package form

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
)

func key(m Model, t tui.KeyType) Model {
	m, _ = m.Update(tui.Key{Type: t})
	return m
}

func space(m Model) Model {
	m, _ = m.Update(tui.Key{Type: tui.KeySpace})
	return m
}

func kindsForm() Model {
	m := New(
		Field{Name: "name", Label: "Name"},
		Field{Name: "plan", Label: "Plan", Kind: FieldSelect, Options: []string{"Free", "Pro", "Team"}},
		Field{Name: "terms", Label: "Accept terms", Kind: FieldCheckbox, Validators: []Validator{mustBeTrue}},
	)
	m.Focus()
	return m
}

func mustBeTrue(v string) error {
	if v != "true" {
		return errString("you must accept the terms")
	}
	return nil
}

// A select starts on its initial value (or the first option) and Right/Down
// advance, Left/Up go back, staying within the options.
func TestSelectMovesWithinItsOptions(t *testing.T) {
	m := kindsForm()
	m = key(m, tui.KeyTab) // on the select
	if m.Current() != "plan" || m.Values()["plan"] != "Free" {
		t.Fatalf("focus %q value %q, want plan / Free", m.Current(), m.Values()["plan"])
	}
	steps := []struct {
		k    tui.KeyType
		want string
	}{
		{tui.KeyRight, "Pro"}, {tui.KeyDown, "Team"}, {tui.KeyRight, "Team"}, // clamped at the end
		{tui.KeyLeft, "Pro"}, {tui.KeyUp, "Free"}, {tui.KeyLeft, "Free"}, // clamped at the start
	}
	for i, s := range steps {
		m = key(m, s.k)
		if got := m.Values()["plan"]; got != s.want {
			t.Fatalf("step %d: plan = %q, want %q", i, got, s.want)
		}
	}
}

func TestSelectInitialValue(t *testing.T) {
	opts := []string{"Free", "Pro", "Team"}
	for _, c := range []struct{ initial, want string }{{"Team", "Team"}, {"Pro", "Pro"}, {"", "Free"}, {"nope", "Free"}} {
		m := New(Field{Name: "p", Kind: FieldSelect, Options: opts, Value: c.initial})
		if got := m.Values()["p"]; got != c.want {
			t.Errorf("initial %q: value %q, want %q", c.initial, got, c.want)
		}
	}
	if got := New(Field{Name: "p", Kind: FieldSelect}).Values()["p"]; got != "" {
		t.Errorf("a select with no options has value %q, want empty", got)
	}
}

// Space toggles a checkbox between "true" and "false"; other keys leave it.
func TestSpaceTogglesACheckbox(t *testing.T) {
	m := kindsForm()
	m = key(key(m, tui.KeyTab), tui.KeyTab) // on the checkbox
	if m.Current() != "terms" || m.Values()["terms"] != "false" {
		t.Fatalf("focus %q value %q, want terms / false", m.Current(), m.Values()["terms"])
	}
	m = space(m)
	if m.Values()["terms"] != "true" {
		t.Errorf("after one Space the value is %q, want true", m.Values()["terms"])
	}
	for _, k := range []tui.KeyType{tui.KeyLeft, tui.KeyRight, tui.KeyUp, tui.KeyDown, tui.KeyBackspace} {
		m = key(m, k)
	}
	if m.Values()["terms"] != "true" {
		t.Errorf("arrow keys changed the checkbox to %q", m.Values()["terms"])
	}
	m = space(m)
	if m.Values()["terms"] != "false" {
		t.Errorf("after a second Space the value is %q, want false", m.Values()["terms"])
	}
	// A space typed as a rune toggles too.
	m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: " "})
	if m.Values()["terms"] != "true" {
		t.Errorf("a ' ' rune did not toggle: %q", m.Values()["terms"])
	}
}

func TestCheckboxInitialValue(t *testing.T) {
	for _, c := range []struct{ initial, want string }{{"true", "true"}, {"TRUE", "true"}, {"", "false"}, {"false", "false"}, {"yes", "false"}} {
		m := New(Field{Name: "c", Kind: FieldCheckbox, Value: c.initial})
		if got := m.Values()["c"]; got != c.want {
			t.Errorf("initial %q: value %q, want %q", c.initial, got, c.want)
		}
	}
}

// Typing into the form only edits the focused field's own kind: a rune sent
// to a select or a checkbox does not become text anywhere.
func TestRunesDoNotLeakIntoOtherFields(t *testing.T) {
	m := kindsForm()
	m = key(m, tui.KeyTab)
	m = typed(m, "zzz") // on the select: ignored
	if m.Values()["plan"] != "Free" || m.Values()["name"] != "" {
		t.Errorf("values after typing on a select: %v", m.Values())
	}
}

// Submit puts select and checkbox values into SubmittedMsg as strings.
func TestSubmitIncludesSelectAndCheckboxValues(t *testing.T) {
	m := kindsForm()
	m = typed(m, "Ada")
	m = key(m, tui.KeyTab)
	m = key(m, tui.KeyRight) // Pro
	m = key(m, tui.KeyTab)
	m = space(m) // accept
	m, cmd := m.Submit()
	msg, ok := cmd().(SubmittedMsg)
	if !ok {
		t.Fatalf("Submit produced %T, want SubmittedMsg; errors: %q", cmd(), m.Err("terms"))
	}
	want := map[string]string{"name": "Ada", "plan": "Pro", "terms": "true"}
	for k, v := range want {
		if msg.Values[k] != v {
			t.Errorf("Values[%q] = %q, want %q", k, msg.Values[k], v)
		}
	}
	if len(msg.Values) != len(want) {
		t.Errorf("Values = %v, want exactly %v", msg.Values, want)
	}
}

// A validator rejecting a checkbox value shows its error, focuses the field
// and stops the submit; ticking it clears the error.
func TestValidatorRejectsACheckboxAndTickingClearsIt(t *testing.T) {
	m := kindsForm()
	m, cmd := m.Submit()
	if cmd != nil {
		if _, ok := cmd().(SubmittedMsg); ok {
			t.Fatal("an unticked required checkbox let the form submit")
		}
	}
	if m.Err("terms") != "you must accept the terms" || m.Current() != "terms" {
		t.Fatalf("Err(terms)=%q Current=%q, want the error and focus on terms", m.Err("terms"), m.Current())
	}
	if !strings.Contains(plain(m), "you must accept the terms") {
		t.Errorf("the error is not shown:\n%s", plain(m))
	}
	m = space(m)
	if m.Err("terms") != "" {
		t.Errorf("ticking the box left the error %q", m.Err("terms"))
	}
}

func TestValidatorsRunOnASelectValue(t *testing.T) {
	notFree := func(v string) error {
		if v == "Free" {
			return errString("pick a paid plan")
		}
		return nil
	}
	m := New(Field{Name: "plan", Label: "Plan", Kind: FieldSelect, Options: []string{"Free", "Pro"}, Validators: []Validator{notFree}})
	m.Focus()
	m, _ = m.Submit()
	if m.Err("plan") != "pick a paid plan" {
		t.Fatalf("Err = %q, want the validator's message", m.Err("plan"))
	}
	m = key(m, tui.KeyRight)
	if m.Err("plan") != "" {
		t.Errorf("choosing Pro left the error %q", m.Err("plan"))
	}
}

// Tab still walks every field, whatever its kind, and wraps.
func TestTabWalksMixedKinds(t *testing.T) {
	m := kindsForm()
	var seen []string
	for i := 0; i < 4; i++ {
		seen = append(seen, m.Current())
		m = key(m, tui.KeyTab)
	}
	if got := strings.Join(seen, ","); got != "name,plan,terms,name" {
		t.Errorf("focus order = %s", got)
	}
}

// Update on a copy must not change the receiver, for the new kinds too.
func TestUpdateDoesNotChangeTheReceiverForNewKinds(t *testing.T) {
	m := kindsForm()
	m = key(m, tui.KeyTab)
	before := m.Values()["plan"]
	_ = key(m, tui.KeyRight)
	if m.Values()["plan"] != before {
		t.Errorf("receiver changed: plan = %q", m.Values()["plan"])
	}
	m2 := key(m, tui.KeyTab)
	c := m2
	_ = space(c)
	if m2.Values()["terms"] != "false" {
		t.Errorf("Space changed the receiver's checkbox to %q", m2.Values()["terms"])
	}
}

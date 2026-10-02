package main

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/confirm"
)

func pressEnter(m model) (model, tui.Cmd) {
	next, cmd := m.Update(tui.Key{Type: tui.KeyEnter})
	return next.(model), cmd
}

func pressTab(m model) (model, tui.Cmd) {
	next, cmd := m.Update(tui.Key{Type: tui.KeyTab})
	return next.(model), cmd
}

func pressEsc(m model) (model, tui.Cmd) {
	next, cmd := m.Update(tui.Key{Type: tui.KeyEsc})
	return next.(model), cmd
}

// Tab moves focus across the three fields without advancing the wizard step.
func TestTabMovesFocusAcrossFields(t *testing.T) {
	m := initialModel()
	if m.focused != fieldName {
		t.Fatalf("focused = %v, want fieldName", m.focused)
	}
	m, _ = pressTab(m)
	if m.focused != fieldEmail {
		t.Fatalf("focused = %v, want fieldEmail", m.focused)
	}
	m, _ = pressTab(m)
	if m.focused != fieldLanguage {
		t.Fatalf("focused = %v, want fieldLanguage", m.focused)
	}
	if got := m.wizard.Current(); got != stepEditing {
		t.Fatalf("wizard.Current() = %d, want stepEditing (tab shouldn't advance the wizard)", got)
	}
}

// Enter on the name and email fields advances focus but not the wizard step;
// enter on language (with the dropdown closed) advances the wizard to Confirm.
func TestEnterAdvancesToConfirmStep(t *testing.T) {
	m := initialModel()

	m.name.SetValue("Ada Lovelace")
	m, _ = pressEnter(m) // name -> email
	if m.focused != fieldEmail {
		t.Fatalf("focused = %v, want fieldEmail", m.focused)
	}
	if got := m.wizard.Current(); got != stepEditing {
		t.Fatalf("wizard.Current() = %d, want stepEditing", got)
	}

	m.email.SetValue("ada@example.com")
	m, _ = pressEnter(m) // email -> language
	if m.focused != fieldLanguage {
		t.Fatalf("focused = %v, want fieldLanguage", m.focused)
	}
	if got := m.wizard.Current(); got != stepEditing {
		t.Fatalf("wizard.Current() = %d, want stepEditing before language is submitted", got)
	}

	m, _ = pressEnter(m) // language -> confirm
	if got := m.wizard.Current(); got != stepConfirm {
		t.Fatalf("wizard.Current() = %d, want stepConfirm", got)
	}
	if m.done {
		t.Fatal("model should not be done yet")
	}
}

// Esc from the confirm step goes back to editing (wizard.Back), not quit.
func TestEscFromConfirmGoesBackToEditing(t *testing.T) {
	m := initialModel()
	m, _ = pressEnter(m) // name -> email
	m, _ = pressEnter(m) // email -> language
	m, _ = pressEnter(m) // language -> confirm
	if got := m.wizard.Current(); got != stepConfirm {
		t.Fatalf("wizard.Current() = %d, want stepConfirm", got)
	}

	m, _ = pressEsc(m)
	if got := m.wizard.Current(); got != stepEditing {
		t.Fatalf("wizard.Current() = %d, want stepEditing after esc from confirm", got)
	}
	if m.done {
		t.Fatal("model should not be done after esc")
	}
}

// A "yes" ConfirmedMsg marks the model done; the wizard step is untouched.
func TestConfirmedYesMarksDone(t *testing.T) {
	m := initialModel()
	next, _ := m.Update(confirm.ConfirmedMsg{Yes: true})
	m = next.(model)
	if !m.done {
		t.Fatal("model should be done after a yes confirmation")
	}
}

// A "no" ConfirmedMsg retreats the wizard to editing rather than completing.
func TestConfirmedNoGoesBack(t *testing.T) {
	m := initialModel()
	m, _ = pressEnter(m)
	m, _ = pressEnter(m)
	m, _ = pressEnter(m) // now at stepConfirm

	next, _ := m.Update(confirm.ConfirmedMsg{Yes: false})
	m = next.(model)
	if m.done {
		t.Fatal("model should not be done after a no confirmation")
	}
	if got := m.wizard.Current(); got != stepEditing {
		t.Fatalf("wizard.Current() = %d, want stepEditing after declining", got)
	}
}

// Esc while editing (not confirming, not done) quits.
func TestEscWhileEditingQuits(t *testing.T) {
	m := initialModel()
	_, cmd := pressEsc(m)
	if cmd == nil {
		t.Fatal("esc while editing should return a quit Cmd")
	}
}

package form

import (
	"testing"

	"github.com/ows4444/tui"
)

// A pasted newline is text for the focused field, not Enter: it must not
// submit the form, so a required field stays without an error.
func TestPasteWithNewlineDoesNotSubmit(t *testing.T) {
	m := New(Field{Name: "name", Label: "Name", Validators: []Validator{Required()}})
	m.Focus()

	m, cmd := m.Update(tui.PasteEvent{Text: "\n"})
	if cmd != nil {
		if _, ok := cmd().(SubmittedMsg); ok {
			t.Fatal("paste submitted the form")
		}
	}
	if err := m.Err("name"); err != "" {
		t.Fatalf("paste ran validation (Submit): %q", err)
	}

	m, _ = m.Update(tui.PasteEvent{Text: "ada\nlovelace"})
	if got := m.Values()["name"]; got != "adalovelace" {
		t.Fatalf("value %q, want the paste with its newline dropped", got)
	}
	if m.Err("name") != "" {
		t.Fatal("paste alone showed a validation error")
	}
}

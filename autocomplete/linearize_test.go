package autocomplete

import (
	"testing"

	"github.com/ows4444/tui"
)

var _ tui.Linearizer = Model{}

func TestLinearize(t *testing.T) {
	m := New("apple", "apricot", "banana")
	m.Input.Prompt = "Fruit: "
	m.Blur()
	if got := m.Linearize(); got != "Fruit, text field, empty" {
		t.Errorf("closed dropdown = %q", got)
	}
	m.Input.SetValue("ap")
	want := "Fruit, text field, value: ap\n2 suggestions\napple, option 1 of 2, selected\napricot, option 2 of 2"
	if got := m.Linearize(); got != want {
		t.Errorf("open =\n%s\nwant\n%s", got, want)
	}
	m.Input.SetValue("ban")
	if got := m.Linearize(); got != "Fruit, text field, value: ban\n1 suggestion\nbanana, option 1 of 1, selected" {
		t.Errorf("one suggestion = %q", got)
	}
	m.Input.SetValue("zzz")
	if got := m.Linearize(); got != "Fruit, text field, value: zzz" {
		t.Errorf("no match = %q", got)
	}
}

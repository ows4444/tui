package form

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/ows4444/tui"
)

type level int

func TestSelectOfReadsTheTypedValue(t *testing.T) {
	f := SelectOf("level", "Level",
		Option[level]{"Low", 1}, Option[level]{"High", 10})
	f.Value = "High"
	m := New(f)
	m.Focus()
	if v, ok := Value[level](m, "level"); !ok || v != 10 {
		t.Errorf("Value = %v, %v; want 10", v, ok)
	}
	if got := m.Values()["level"]; got != "High" {
		t.Errorf("string value = %q, want the label", got)
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyLeft})
	if v, _ := Value[level](m, "level"); v != 1 {
		t.Errorf("after Left: %v, want 1", v)
	}
	if _, ok := Value[string](m, "level"); ok {
		t.Error("a different T must not match")
	}
	if _, ok := Value[level](m, "nope"); ok {
		t.Error("unknown field matched")
	}
}

func TestValueIsNotOkForAHiddenGroup(t *testing.T) {
	m := NewGroups(
		Group{Fields: []Field{{Name: "on", Label: "On", Kind: FieldCheckbox}}},
		Group{Fields: []Field{SelectOf("n", "N", Option[int]{"One", 1})},
			Hide: func(v map[string]string) bool { return v["on"] != "true" }},
	)
	if _, ok := Value[int](m, "n"); ok {
		t.Error("Value for a hidden group's field is ok")
	}
}

func TestBoolAndInt(t *testing.T) {
	m := New(
		Field{Name: "ok", Label: "OK", Kind: FieldCheckbox, Value: "true"},
		Field{Name: "n", Label: "N", Value: " 42 "},
		Field{Name: "bad", Label: "Bad", Value: "x"},
	)
	if !Bool(m, "ok") || Bool(m, "n") || Bool(m, "nope") {
		t.Error("Bool wrong")
	}
	if n, err := Int(m, "n"); err != nil || n != 42 {
		t.Errorf("Int = %d, %v", n, err)
	}
	if _, err := Int(m, "bad"); err == nil {
		t.Error("Int of text did not fail")
	}
	if _, err := Int(m, "nope"); err == nil || !strings.Contains(err.Error(), "nope") {
		t.Errorf("Int of an unknown field: %v", err)
	}
	var target *unknownFieldError
	if _, err := Int(m, "nope"); !errors.As(err, &target) {
		t.Error("unknown field error type")
	}
}

func cityForm() Model {
	cities := map[string][]string{
		"Pakistan": {"Lahore", "Karachi"},
		"Japan":    {"Tokyo", "Osaka", "Kyoto"},
	}
	m := New(
		Field{Name: "country", Label: "Country", Kind: FieldSelect, Options: []string{"Pakistan", "Japan"}},
		Field{Name: "city", Label: "City", Kind: FieldSelect, Value: "Osaka",
			OptionsFunc: func(v map[string]string) []string { return cities[v["country"]] }},
	)
	m.Focus()
	return m
}

func TestOptionsFuncRecomputesFromAnotherField(t *testing.T) {
	m := cityForm()
	// Pakistan does not offer Osaka, so the first city is chosen.
	if got := m.Values()["city"]; got != "Lahore" {
		t.Fatalf("initial city = %q, want Lahore", got)
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyRight}) // country -> Japan
	m = tab(m)
	m, _ = m.Update(tui.Key{Type: tui.KeyRight})
	if got := m.Values()["city"]; got != "Osaka" {
		t.Errorf("after Japan, Right: city = %q, want Osaka", got)
	}
	if v := plain(m); !strings.Contains(v, "Osaka") {
		t.Errorf("view:\n%s", v)
	}
	m = shiftTab(m)
	m, _ = m.Update(tui.Key{Type: tui.KeyLeft}) // back to Pakistan
	if got := m.Values()["city"]; got != "Lahore" {
		t.Errorf("after Pakistan: city = %q, want Lahore (Osaka not offered)", got)
	}
}

func TestOptionsFuncKeepsTheChoiceWhileOffered(t *testing.T) {
	m := New(
		Field{Name: "a", Label: "A", Value: "x"},
		Field{Name: "b", Label: "B", Kind: FieldSelect, Value: "two",
			OptionsFunc: func(v map[string]string) []string { return []string{"one", "two", "three"} }},
	)
	if got := m.Values()["b"]; got != "two" {
		t.Errorf("initial b = %q, want two", got)
	}
	m.Focus()
	m = typed(m, "yz")
	if got := m.Values()["b"]; got != "two" {
		t.Errorf("after editing a: b = %q, want two", got)
	}
}

func TestOptionsFuncLeavesTheReceiverAlone(t *testing.T) {
	m := cityForm()
	before := reflect.ValueOf(m.inputs[1].options).Pointer()
	next, _ := m.Update(tui.Key{Type: tui.KeyRight})
	if got := m.Values()["city"]; got != "Lahore" {
		t.Errorf("receiver changed: city = %q", got)
	}
	if reflect.ValueOf(m.inputs[1].options).Pointer() != before {
		t.Error("receiver's options slice was replaced")
	}
	if got := next.Values()["city"]; got != "Tokyo" {
		t.Errorf("new model: city = %q, want Tokyo", got)
	}
}

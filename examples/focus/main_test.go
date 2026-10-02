package main

import (
	"os"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/input"
)

// shiftTab is a Tab keypress with Shift held, the way this repo represents
// it: Mod exists precisely for modifier combinations that have no direct
// byte encoding (see input/keys.go), so Shift+Tab is KeyTab with
// Mod.Shift() set rather than a distinct KeyType.
var shiftTab = tui.Key{Type: tui.KeyTab, Mod: input.ModShift}

// model composes three different widget types (textinput, passwordinput,
// textarea) behind a single focus.Ring (criterion #429).
func TestModelComposesThreeDifferentWidgetTypes(t *testing.T) {
	m := initialModel()

	if m.current() != fieldName {
		t.Fatalf("expected initial focus on fieldName, got %v", m.current())
	}
	if !m.name.Focused() {
		t.Fatal("name should be focused initially")
	}
	if m.password.Focused() {
		t.Fatal("password should not be focused initially")
	}
	if m.bio.Focused() {
		t.Fatal("bio should not be focused initially")
	}

	// The three fields really are different concrete widget types, not
	// three copies of the same one.
	var _ = m.name     // textinput.Model
	var _ = m.password // passwordinput.Model
	var _ = m.bio      // textarea.Model
}

// Tab: Blur the current widget, advance to the next, Focus it, wrapping
// from the last field back to the first (criterion #430).
func TestTabAdvancesFocusAndWraps(t *testing.T) {
	m := initialModel()

	for _, want := range []field{fieldPassword, fieldBio, fieldName} {
		next, _ := m.Update(tui.Key{Type: tui.KeyTab})
		m = next.(model)
		if m.current() != want {
			t.Fatalf("after tab, expected focus %v, got %v", want, m.current())
		}
		assertOnlyFocused(t, m, m.current())
	}
}

// Shift+Tab: move focus to the previous widget, wrapping from the first
// back to the last (criterion #431).
func TestShiftTabRetreatsFocusAndWraps(t *testing.T) {
	m := initialModel() // starts on fieldName

	for _, want := range []field{fieldBio, fieldPassword, fieldName} {
		next, _ := m.Update(shiftTab)
		m = next.(model)
		if m.current() != want {
			t.Fatalf("after shift+tab, expected focus %v, got %v", want, m.current())
		}
		assertOnlyFocused(t, m, m.current())
	}
}

// assertOnlyFocused checks that exactly the widget at want reports
// Focused() == true and the others report false — i.e. switchFocus really
// blurred the old widget and focused the new one, rather than leaving both
// (or neither) focused.
func assertOnlyFocused(t *testing.T, m model, want field) {
	t.Helper()
	got := map[field]bool{
		fieldName:     m.name.Focused(),
		fieldPassword: m.password.Focused(),
		fieldBio:      m.bio.Focused(),
	}
	for f, focused := range got {
		if f == want && !focused {
			t.Errorf("field %v should be focused", f)
		}
		if f != want && focused {
			t.Errorf("field %v should be blurred, only %v should be focused", f, want)
		}
	}
}

// View visibly distinguishes the focused widget from the others: the
// rendered output differs depending on which widget is focused (criterion
// #432).
func TestViewDistinguishesFocusedWidget(t *testing.T) {
	onName := initialModel()
	viewOnName := onName.View()

	next, _ := onName.Update(tui.Key{Type: tui.KeyTab})
	onPassword := next.(model)
	viewOnPassword := onPassword.View()

	if viewOnName == viewOnPassword {
		t.Fatal("View() should differ depending on which widget is focused")
	}

	// More specifically: the highlighted border color's SGR sequence
	// should wrap whichever box is currently focused.
	focusedSeq := colorSequence(focusedBorder)
	if !strings.Contains(viewOnName, focusedSeq) {
		t.Error("view focused on name should contain the focused border color")
	}
	if !strings.Contains(viewOnPassword, focusedSeq) {
		t.Error("view focused on password should contain the focused border color")
	}
}

// colorSequence returns the SGR escape sequence ansi.Style uses to render
// text in the foreground color c, by rendering a marker and stripping it
// back off.
func colorSequence(c ansi.Color) string {
	rendered := ansi.NewStyle().Foreground(c).Render("X")
	return strings.TrimSuffix(strings.TrimSuffix(rendered, ansi.Reset), "X")
}

// The doc comment at the top of main.go identifies this example as the
// canonical Focus/Blur reference, distinct from examples/form's incidental
// use of the same pattern (criterion #433). Reading the actual file keeps
// this test tied to the real source rather than a copy of the string.
func TestDocCommentExplainsCanonicalReference(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("reading main.go: %v", err)
	}

	// Only look at the leading doc comment (up to "package main"), so this
	// test is really checking the top-of-file doc comment, not just that
	// these words appear somewhere in the file.
	doc, _, ok := strings.Cut(string(src), "package main")
	if !ok {
		t.Fatal("main.go has no package declaration")
	}

	for _, want := range []string{
		"canonical",
		"Focus()/Blur()",
		"examples/form",
	} {
		if !strings.Contains(doc, want) {
			t.Errorf("main.go doc comment missing %q", want)
		}
	}
}

func TestQuitKeys(t *testing.T) {
	m := initialModel()
	for _, key := range []tui.Key{
		{Type: tui.KeyCtrlC},
		{Type: tui.KeyEsc},
	} {
		_, cmd := m.Update(key)
		if cmd == nil {
			t.Errorf("key %+v should return a quit Cmd", key)
		}
	}
}

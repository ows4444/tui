package colorpicker

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
)

// TestCursorSwatchHighlightNotClobbered proves criterion #566 empirically,
// the same way widgets.TestToolCallCursorHighlightNotClobbered verifies
// accordion's title highlight: build the real rendered View, then confirm
// the cursor swatch's highlight covers its ENTIRE rendered block rather
// than being asserted by construction.
func TestCursorSwatchHighlightNotClobbered(t *testing.T) {
	m := New(ansi.Red, ansi.Green, ansi.Blue)
	m.cursor = 1 // Green is the cursor swatch.

	out := m.View()

	wantStyle := ansi.NewStyle().Background(ansi.Green).Bold().Foreground(m.Theme.TextInverse)
	wantBlock := wantStyle.Render(swatch)

	idx := strings.Index(out, wantBlock)
	if idx == -1 {
		t.Fatalf("View() = %q, want it to contain the whole cursor swatch wrapped in one combined-style Render call: %q", out, wantBlock)
	}

	// The combined block itself must carry exactly one reset (the single
	// outer Render call's own trailing Reset) — proving no inner
	// separately-Render-ed sub-style snuck an early reset into the middle
	// of the swatch, which would leave part of the swatch unstyled.
	if got := strings.Count(wantBlock, ansi.Reset); got != 1 {
		t.Fatalf("wantBlock = %q contains %d resets, want exactly 1 (single combined style, single Render call)", wantBlock, got)
	}

	// Nothing between the swatch's opening escape sequence and its own
	// single reset should be a second reset (which would mean the
	// highlight was clobbered partway through by a nested Render).
	openIdx := strings.Index(wantBlock, ansi.CSI)
	closeIdx := strings.Index(wantBlock, ansi.Reset)
	if openIdx == -1 || closeIdx == -1 || openIdx >= closeIdx {
		t.Fatalf("wantBlock = %q, want an opening escape sequence before the single trailing reset", wantBlock)
	}
	inner := wantBlock[openIdx+len(ansi.CSI) : closeIdx]
	if strings.Contains(inner, ansi.Reset) {
		t.Errorf("found a reset inside the swatch body %q, highlight was clobbered partway through", wantBlock)
	}

	// The neighboring (non-cursor) swatches must NOT carry the cursor's
	// bold/inverse styling, confirming the highlight is scoped to exactly
	// the cursor swatch.
	plainRed := ansi.NewStyle().Background(ansi.Red).Render(swatch)
	plainBlue := ansi.NewStyle().Background(ansi.Blue).Render(swatch)
	if !strings.Contains(out, plainRed) {
		t.Errorf("View() = %q, want the non-cursor Red swatch rendered with its plain (non-highlighted) style: %q", out, plainRed)
	}
	if !strings.Contains(out, plainBlue) {
		t.Errorf("View() = %q, want the non-cursor Blue swatch rendered with its plain (non-highlighted) style: %q", out, plainBlue)
	}
}

// TestLeftRightMovesCursor proves criterion #567: Left/Right move cursor
// among Palette entries, clamped at the ends.
func TestLeftRightMovesCursor(t *testing.T) {
	m := New(ansi.Red, ansi.Green, ansi.Blue)

	if m.Cursor() != 0 {
		t.Fatalf("initial Cursor() = %d, want 0", m.Cursor())
	}

	m, _ = m.Update(tui.Key{Type: tui.KeyLeft})
	if m.Cursor() != 0 {
		t.Errorf("Left at start: Cursor() = %d, want 0 (clamped)", m.Cursor())
	}

	m, _ = m.Update(tui.Key{Type: tui.KeyRight})
	if m.Cursor() != 1 {
		t.Fatalf("after Right: Cursor() = %d, want 1", m.Cursor())
	}

	m, _ = m.Update(tui.Key{Type: tui.KeyRight})
	if m.Cursor() != 2 {
		t.Fatalf("after second Right: Cursor() = %d, want 2", m.Cursor())
	}

	m, _ = m.Update(tui.Key{Type: tui.KeyRight})
	if m.Cursor() != 2 {
		t.Errorf("Right past end: Cursor() = %d, want 2 (clamped)", m.Cursor())
	}

	m, _ = m.Update(tui.Key{Type: tui.KeyLeft})
	if m.Cursor() != 1 {
		t.Errorf("after Left: Cursor() = %d, want 1", m.Cursor())
	}
}

// TestEnterConfirmsPaletteSwatch proves criterion #568: Enter on a palette
// swatch (not hex-focused) returns a Cmd delivering SelectedMsg with the
// currently-highlighted color.
func TestEnterConfirmsPaletteSwatch(t *testing.T) {
	m := New(ansi.Red, ansi.Green, ansi.Blue)
	m, _ = m.Update(tui.Key{Type: tui.KeyRight}) // cursor -> Green

	_, cmd := m.Update(tui.Key{Type: tui.KeyEnter})
	if cmd == nil {
		t.Fatal("Update(Enter) returned a nil Cmd, want one delivering SelectedMsg")
	}

	msg := cmd()
	sel, ok := msg.(SelectedMsg)
	if !ok {
		t.Fatalf("Cmd delivered %T, want SelectedMsg", msg)
	}
	if sel.Color != ansi.Color(ansi.Green) {
		t.Errorf("SelectedMsg.Color = %v, want ansi.Green", sel.Color)
	}
}

// TestTabTogglesHexFocusAndForwardsKeys proves the Tab-toggle and
// key-forwarding half of criterion #569.
func TestTabTogglesHexFocusAndForwardsKeys(t *testing.T) {
	m := New(ansi.Red, ansi.Green, ansi.Blue)

	if m.HexFocused() {
		t.Fatal("HexFocused() = true before any Tab, want false")
	}

	m, _ = m.Update(tui.Key{Type: tui.KeyTab})
	if !m.HexFocused() {
		t.Fatal("HexFocused() = false after Tab, want true")
	}

	// While hex-focused, a Left press must be forwarded to HexInput
	// (moving its internal cursor / accepting input) instead of moving the
	// palette cursor.
	cursorBefore := m.Cursor()
	m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: "#"})
	if m.Cursor() != cursorBefore {
		t.Errorf("palette Cursor() changed while hex-focused: got %d, want unchanged %d", m.Cursor(), cursorBefore)
	}
	if m.HexInput.Value() != "#" {
		t.Errorf("HexInput.Value() = %q after forwarded rune, want %q (key forwarded to HexInput)", m.HexInput.Value(), "#")
	}

	m, _ = m.Update(tui.Key{Type: tui.KeyTab})
	if m.HexFocused() {
		t.Fatal("HexFocused() = true after second Tab, want false (toggled back)")
	}

	// Now that focus is back on the palette, Left/Right should move the
	// cursor again rather than being forwarded.
	m, _ = m.Update(tui.Key{Type: tui.KeyRight})
	if m.Cursor() != cursorBefore+1 {
		t.Errorf("Cursor() = %d after Right post-toggle-back, want %d", m.Cursor(), cursorBefore+1)
	}
}

// TestEnterOnValidHexConfirms proves the valid-hex half of criterion #569.
func TestEnterOnValidHexConfirms(t *testing.T) {
	m := New(ansi.Red, ansi.Green, ansi.Blue)
	m, _ = m.Update(tui.Key{Type: tui.KeyTab}) // focus hex input
	m.HexInput.SetValue("#1A2B3C")

	m, cmd := m.Update(tui.Key{Type: tui.KeyEnter})
	if cmd == nil {
		t.Fatal("Update(Enter) with a valid hex value returned a nil Cmd, want one delivering SelectedMsg")
	}

	msg := cmd()
	sel, ok := msg.(SelectedMsg)
	if !ok {
		t.Fatalf("Cmd delivered %T, want SelectedMsg", msg)
	}
	want := ansi.RGB{R: 0x1A, G: 0x2B, B: 0x3C}
	if sel.Color != want {
		t.Errorf("SelectedMsg.Color = %#v, want %#v", sel.Color, want)
	}
	_ = m
}

// TestEnterOnInvalidHexIsNoop proves the invalid-hex half of criterion
// #569: no panic, no Cmd.
func TestEnterOnInvalidHexIsNoop(t *testing.T) {
	invalid := []string{"", "123456", "#12345", "#GGGGGG", "not a color", "#1234567"}

	for _, v := range invalid {
		t.Run(v, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Update panicked on invalid hex %q: %v", v, r)
				}
			}()

			m := New(ansi.Red, ansi.Green, ansi.Blue)
			m, _ = m.Update(tui.Key{Type: tui.KeyTab})
			m.HexInput.SetValue(v)

			m, cmd := m.Update(tui.Key{Type: tui.KeyEnter})
			if cmd != nil {
				t.Errorf("Update(Enter) with invalid hex %q returned a non-nil Cmd, want nil (no-op)", v)
			}
			_ = m
		})
	}
}

// TestParseHex is a direct table-driven check of the hex-parsing helper.
func TestParseHex(t *testing.T) {
	tests := []struct {
		in   string
		want ansi.RGB
		ok   bool
	}{
		{"#000000", ansi.RGB{R: 0, G: 0, B: 0}, true},
		{"#FFFFFF", ansi.RGB{R: 255, G: 255, B: 255}, true},
		{"#ffffff", ansi.RGB{R: 255, G: 255, B: 255}, true},
		{"#1A2B3C", ansi.RGB{R: 0x1A, G: 0x2B, B: 0x3C}, true},
		{"", ansi.RGB{}, false},
		{"123456", ansi.RGB{}, false},
		{"#12345", ansi.RGB{}, false},
		{"#1234567", ansi.RGB{}, false},
		{"#GGGGGG", ansi.RGB{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, ok := parseHex(tt.in)
			if ok != tt.ok {
				t.Fatalf("parseHex(%q) ok = %v, want %v", tt.in, ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Errorf("parseHex(%q) = %#v, want %#v", tt.in, got, tt.want)
			}
		})
	}
}

package textarea

import (
	"testing"

	"github.com/ows4444/tui"
)

// viModel is a focused Vi textarea holding text, the cursor at its start.
func viModel(text string) Model {
	m := New()
	m.Vi = true
	m.SetValue(text)
	m.SetCursor(0)
	m.Focus()
	return m
}

// typeVi sends keys to m: each rune is a key, \x1b is Esc and \x12 is Ctrl+R.
func typeVi(m *Model, keys string) (cmds int) {
	for _, r := range keys {
		var k tui.Key
		switch r {
		case 0x1b:
			k = tui.Key{Type: tui.KeyEsc}
		case 0x12:
			k = tui.Key{Type: tui.KeyCtrl, Code: 'r'}
		case '\r':
			k = tui.Key{Type: tui.KeyEnter}
		default:
			k = tui.Key{Type: tui.KeyRunes, Text: string(r), Code: r}
		}
		next, cmd := m.Update(k)
		*m = next
		if cmd != nil {
			cmds++
		}
	}
	return cmds
}

func TestViStartsInNormalMode(t *testing.T) {
	m := viModel("abc")
	typeVi(&m, "zq")
	if m.Value() != "abc" || m.ViMode() != ViNormal {
		t.Fatalf("value %q, mode %v; letters must not type in Normal mode", m.Value(), m.ViMode())
	}
	typeVi(&m, "iX\x1b")
	if m.Value() != "Xabc" || m.ViMode() != ViNormal {
		t.Fatalf("value %q, mode %v", m.Value(), m.ViMode())
	}
}

func TestViOffTypesLetters(t *testing.T) {
	m := viModel("abc")
	m.Vi = false
	typeVi(&m, "x")
	if m.Value() != "xabc" {
		t.Fatalf("value %q", m.Value())
	}
	if m.ViMode() != ViNormal {
		t.Fatalf("ViMode = %v with Vi off", m.ViMode())
	}
}

func TestViCommands(t *testing.T) {
	for _, c := range []struct{ in, keys, want string }{
		{"one two three", "dw", "two three"},
		{"one two three", "wcwX\x1b", "one X three"},
		{"a\nb\nc", "jdd", "a\nc"},
		{"a\nb", "yyjp", "a\nb\na"},
		{"one two", "wyiwP", "one twotwo"},
		{"abc", "A!\x1b", "abc!"},
		{"abc", "ox\x1b", "abc\nx"},
		{"f(a, b)", "fadi(", "f()"},
		{"abcd", "x.", "cd"},
		{"abcd", "3x", "d"},
		{"one two", "$bD", "one "},
		{"a\nb\nc", "VjdP", "a\nb\nc"},
	} {
		m := viModel(c.in)
		typeVi(&m, c.keys)
		if m.Value() != c.want {
			t.Errorf("%q then %q = %q; want %q", c.in, c.keys, m.Value(), c.want)
		}
	}
}

func TestViUndoIsOneChange(t *testing.T) {
	m := viModel("one two")
	typeVi(&m, "cwhello world\x1b")
	if m.Value() != "hello world two" {
		t.Fatalf("value %q", m.Value())
	}
	typeVi(&m, "u")
	if m.Value() != "one two" {
		t.Fatalf("u left %q; want the whole change undone", m.Value())
	}
	typeVi(&m, "\x12")
	if m.Value() != "hello world two" {
		t.Fatalf("Ctrl+R gave %q", m.Value())
	}
	typeVi(&m, "uu")
	if m.Value() != "one two" {
		t.Fatalf("value %q after undoing past the start", m.Value())
	}
}

func TestViSeparateInsertsUndoSeparately(t *testing.T) {
	m := viModel("")
	typeVi(&m, "iab\x1bAcd\x1b")
	if m.Value() != "abcd" {
		t.Fatalf("value %q", m.Value())
	}
	typeVi(&m, "u")
	if m.Value() != "ab" {
		t.Fatalf("u gave %q; want only the second insert undone", m.Value())
	}
	typeVi(&m, "u")
	if m.Value() != "" {
		t.Fatalf("u gave %q", m.Value())
	}
}

func TestViCtrlKeysKeepTheirMeaning(t *testing.T) {
	m := viModel("abc")
	typeVi(&m, "x")
	press(&m, ctrl('z'))
	if m.Value() != "abc" {
		t.Fatalf("Ctrl+Z gave %q", m.Value())
	}
}

func TestViYankWritesTheClipboard(t *testing.T) {
	m := viModel("one two")
	if n := typeVi(&m, "yw"); n != 1 {
		t.Fatalf("yw returned %d commands; want the clipboard write", n)
	}
	if n := typeVi(&m, "x"); n != 0 {
		t.Fatalf("x returned %d commands", n)
	}
}

func TestViVisualSelects(t *testing.T) {
	m := viModel("one two")
	typeVi(&m, "vee")
	if got := m.SelectedText(); got != "one tw" {
		t.Fatalf("selected %q", got)
	}
	if m.ViMode() != ViVisual {
		t.Fatalf("mode %v", m.ViMode())
	}
	typeVi(&m, "\x1b")
	if _, _, ok := m.Selection(); ok || m.ViMode() != ViNormal {
		t.Fatalf("Esc left a selection, mode %v", m.ViMode())
	}
	typeVi(&m, "Vj")
	if m.ViMode() != ViVisualLine {
		t.Fatalf("mode %v", m.ViMode())
	}
}

func TestViPasteIsIgnoredInNormalMode(t *testing.T) {
	m := viModel("abc")
	next, _ := m.Update(tui.PasteEvent{Text: "zzz"})
	if next.Value() != "abc" {
		t.Fatalf("paste gave %q in Normal mode", next.Value())
	}
	typeVi(&m, "i")
	next, _ = m.Update(tui.PasteEvent{Text: "zzz"})
	if next.Value() != "zzzabc" {
		t.Fatalf("paste gave %q in Insert mode", next.Value())
	}
}

func TestViSetModeAndPending(t *testing.T) {
	m := viModel("")
	m.SetViMode(ViInsert)
	typeVi(&m, "hi")
	if m.Value() != "hi" || m.ViMode() != ViInsert {
		t.Fatalf("value %q mode %v", m.Value(), m.ViMode())
	}
	m.SetViMode(ViVisual) // not allowed: ignored
	if m.ViMode() != ViInsert {
		t.Fatalf("SetViMode(ViVisual) changed the mode to %v", m.ViMode())
	}
	typeVi(&m, "\x1b2d")
	if m.ViPending() != "2d" {
		t.Fatalf("ViPending = %q", m.ViPending())
	}
}

func TestViCursorStaysOnACharacter(t *testing.T) {
	m := viModel("abc")
	typeVi(&m, "$")
	if m.Cursor() != 2 {
		t.Fatalf("$ put the cursor at %d; want on the last character", m.Cursor())
	}
	typeVi(&m, "ix\x1b")
	if m.Cursor() != 2 || m.Value() != "abxc" {
		t.Fatalf("cursor %d value %q", m.Cursor(), m.Value())
	}
}

func TestViArrowsMoveInNormalMode(t *testing.T) {
	m := viModel("ab\ncd")
	press(&m, tui.Key{Type: tui.KeyRight})
	press(&m, tui.Key{Type: tui.KeyDown})
	if m.Cursor() != 4 {
		t.Fatalf("cursor %d", m.Cursor())
	}
}

package textinput

import (
	"testing"

	"github.com/ows4444/tui"
)

func viField(text string) Model {
	m := New()
	m.Vi = true
	m.SetValue(text)
	m.SetCursor(0)
	m.Focus()
	return m
}

// typeVi sends keys: each rune is a key, \x1b is Esc and \x12 is Ctrl+R.
func typeVi(m *Model, keys string) (cmds int) {
	for _, r := range keys {
		k := tui.Key{Type: tui.KeyRunes, Text: string(r), Code: r}
		switch r {
		case 0x1b:
			k = tui.Key{Type: tui.KeyEsc}
		case 0x12:
			k = tui.Key{Type: tui.KeyCtrl, Code: 'r'}
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
	m := viField("abc")
	typeVi(&m, "zq")
	if m.Value() != "abc" || m.ViMode() != ViNormal {
		t.Fatalf("value %q mode %v; letters must not type in Normal mode", m.Value(), m.ViMode())
	}
	typeVi(&m, "iX\x1b")
	if m.Value() != "Xabc" || m.ViMode() != ViNormal {
		t.Fatalf("value %q mode %v", m.Value(), m.ViMode())
	}
}

func TestViOffTypesLetters(t *testing.T) {
	m := viField("abc")
	m.Vi = false
	typeVi(&m, "x")
	if m.Value() != "xabc" || m.ViMode() != ViNormal {
		t.Fatalf("value %q", m.Value())
	}
}

func TestViCommands(t *testing.T) {
	for _, c := range []struct{ in, keys, want string }{
		{"one two three", "dw", "two three"},
		{"one two three", "wcwX\x1b", "one X three"},
		{"one two", "yyP", "one twoone two"},
		{"one two", "wyiwP", "one twotwo"},
		{"abc", "A!\x1b", "abc!"},
		{"abc", "ox\x1b", "abcx"},
		{"f(a, b)", "fadi(", "f()"},
		{"abcd", "x.", "cd"},
		{"abcd", "3x", "d"},
		{"one two", "$bD", "one "},
		{"héllo wörld", "wx", "héllo örld"},
		{"abc", "vlld", ""},
		{"abc", "dd", ""},
	} {
		m := viField(c.in)
		typeVi(&m, c.keys)
		if m.Value() != c.want {
			t.Errorf("%q then %q = %q; want %q", c.in, c.keys, m.Value(), c.want)
		}
		if bytesHaveNewline(m.Value()) {
			t.Errorf("%q then %q put a newline in a single-line field", c.in, c.keys)
		}
	}
}

func bytesHaveNewline(s string) bool {
	for _, r := range s {
		if r == '\n' {
			return true
		}
	}
	return false
}

func TestViUndoIsOneChange(t *testing.T) {
	m := viField("one two")
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
}

func TestViHistoryStillWorks(t *testing.T) {
	m := viField("")
	m.History = []string{"older", "newest"}
	press := func(k tui.Key) {
		next, _ := m.Update(k)
		m = next
	}
	press(tui.Key{Type: tui.KeyUp})
	if m.Value() != "newest" {
		t.Fatalf("Up gave %q in Normal mode", m.Value())
	}
	typeVi(&m, "i")
	press(tui.Key{Type: tui.KeyUp})
	if m.Value() != "older" {
		t.Fatalf("Up gave %q in Insert mode", m.Value())
	}
}

func TestViYankAndDisableCopy(t *testing.T) {
	m := viField("one two")
	if n := typeVi(&m, "yw"); n != 1 {
		t.Fatalf("yw returned %d commands; want the clipboard write", n)
	}
	m.DisableCopy = true
	if n := typeVi(&m, "yw"); n != 0 {
		t.Fatalf("yw returned %d commands with DisableCopy", n)
	}
}

func TestViVisualSelectsAndPasteIsIgnored(t *testing.T) {
	m := viField("one two")
	typeVi(&m, "vee")
	if got := m.ed.SelectedText(); got != "one tw" {
		t.Fatalf("selected %q", got)
	}
	typeVi(&m, "\x1b")
	if m.ed.SelectedText() != "" || m.ViMode() != ViNormal {
		t.Fatalf("Esc left %q selected, mode %v", m.ed.SelectedText(), m.ViMode())
	}
	next, _ := m.Update(tui.PasteEvent{Text: "zzz"})
	if next.Value() != "one two" {
		t.Fatalf("paste gave %q in Normal mode", next.Value())
	}
}

func TestViSetModeAndCursor(t *testing.T) {
	m := viField("")
	m.SetViMode(ViInsert)
	typeVi(&m, "hi")
	if m.Value() != "hi" || m.ViMode() != ViInsert {
		t.Fatalf("value %q mode %v", m.Value(), m.ViMode())
	}
	typeVi(&m, "\x1b")
	if m.Cursor() != 1 {
		t.Fatalf("Esc left the cursor at %d; want on the last character", m.Cursor())
	}
	typeVi(&m, "2d")
	if m.ViPending() != "2d" {
		t.Fatalf("ViPending = %q", m.ViPending())
	}
}

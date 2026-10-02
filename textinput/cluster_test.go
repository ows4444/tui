package textinput

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/input"
)

const fam = "👨‍👩‍👧"

// #33
func TestViewNeverWiderThanWidthWithWideRunes(t *testing.T) {
	for _, w := range []int{4, 5, 10} {
		m := focused()
		m.Width = w
		m.SetValue(strings.Repeat("世", 12))
		for pos := 0; pos <= 12; pos++ {
			m.SetCursor(pos)
			if got := ansi.Width(m.View()); got > w {
				t.Fatalf("Width %d cursor %d: view is %d columns", w, pos, got)
			}
		}
	}
}

// #34
func TestBackspaceAndDeleteRemoveWholeCluster(t *testing.T) {
	m := focused()
	m.SetValue("a" + fam)
	m, _ = m.Update(tui.Key{Type: tui.KeyBackspace})
	if m.Value() != "a" {
		t.Errorf("backspace left %q", m.Value())
	}
	m.SetValue(fam + "b")
	m.SetCursor(0)
	m, _ = m.Update(tui.Key{Type: tui.KeyDelete})
	if m.Value() != "b" {
		t.Errorf("delete left %q", m.Value())
	}
}

// #35
func TestLeftRightMoveOneCluster(t *testing.T) {
	m := focused()
	m.SetValue(fam + "é")
	m, _ = m.Update(tui.Key{Type: tui.KeyLeft})
	if m.Cursor() != 1 {
		t.Errorf("left: cursor %d", m.Cursor())
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyLeft})
	m, _ = m.Update(tui.Key{Type: tui.KeyRight})
	if m.Cursor() != 1 {
		t.Errorf("right: cursor %d", m.Cursor())
	}
}

// #36
func TestCtrlAndAltArrowsMoveOneWord(t *testing.T) {
	for _, mod := range []input.Mod{input.ModCtrl, input.ModAlt} {
		m := focused()
		m.SetValue("foo bar")
		m, _ = m.Update(tui.Key{Type: tui.KeyLeft, Mod: mod})
		if m.Cursor() != 4 {
			t.Errorf("mod %v left: cursor %d", mod, m.Cursor())
		}
		m, _ = m.Update(tui.Key{Type: tui.KeyLeft, Mod: mod})
		if m.Cursor() != 0 {
			t.Errorf("mod %v left 2: cursor %d", mod, m.Cursor())
		}
		m, _ = m.Update(tui.Key{Type: tui.KeyRight, Mod: mod})
		if m.Cursor() != 3 {
			t.Errorf("mod %v right: cursor %d", mod, m.Cursor())
		}
	}
}

// #37
func TestLargePasteIsOneOperation(t *testing.T) {
	m := focused()
	big := strings.Repeat("x", 200000) + "\n" + fam
	m, _ = m.Update(tui.PasteEvent{Text: big})
	if m.Cursor() != 200001 || !strings.HasSuffix(m.Value(), "x"+fam) {
		t.Errorf("cursor %d", m.Cursor())
	}
}

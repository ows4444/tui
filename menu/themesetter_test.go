package menu

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/theme"
)

// SetTheme reaches levels that already exist and levels opened afterwards.
func TestSetThemeAppliesToOpenedLevels(t *testing.T) {
	m := New([]Item{{Label: "a", Children: []Item{{Label: "b"}}}})
	th := theme.DraculaTheme()
	m = m.SetTheme(th)
	if m.stack[0].Theme.Primary != th.Primary {
		t.Error("root level not themed")
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyEnter})
	if len(m.stack) != 2 {
		t.Fatalf("stack = %d levels, want 2", len(m.stack))
	}
	if m.stack[1].Theme.Primary != th.Primary {
		t.Error("level opened after SetTheme not themed")
	}
}

func TestSetThemeDoesNotMutateTheCopy(t *testing.T) {
	m := New([]Item{{Label: "a"}})
	before := m.stack[0].Theme.Primary
	_ = m.SetTheme(theme.DraculaTheme())
	if m.stack[0].Theme.Primary != before {
		t.Error("SetTheme changed the receiver's level")
	}
}

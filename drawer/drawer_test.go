package drawer

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/internal/boxdraw"
	"github.com/ows4444/tui/layout"
)

func key(t tui.KeyType) tui.Key { return tui.Key{Type: t} }

// grid returns a w x h rectangle of '.' characters, one row per line, a
// fixed-size base to test Drawer placement against.
func grid(w, h int) string {
	row := strings.Repeat(".", w)
	rows := make([]string, h)
	for i := range rows {
		rows[i] = row
	}
	return strings.Join(rows, "\n")
}

// TestDrawerLifecycleMirrorsPopover proves criterion #542: New returns an
// already-open Model, Show/Hide/Open work as in popover.Model/dialog.Model,
// and Update dismisses on Enter/Esc while open and no-ops when closed.
func TestDrawerLifecycleMirrorsPopover(t *testing.T) {
	m := New("content")
	if !m.Open() {
		t.Fatal("New() should return an already-open drawer")
	}
	if m.Edge != EdgeRight {
		t.Errorf("New() Edge = %v, want EdgeRight (the default)", m.Edge)
	}

	m.Hide()
	if m.Open() {
		t.Error("Hide() should close the drawer")
	}
	m.Show()
	if !m.Open() {
		t.Error("Show() should open the drawer")
	}

	next, cmd := m.Update(key(tui.KeyEnter))
	if next.Open() {
		t.Error("Enter should close the drawer")
	}
	if cmd == nil {
		t.Fatal("Enter should return a non-nil Cmd")
	}
	if _, ok := cmd().(DismissedMsg); !ok {
		t.Errorf("Cmd produced %T, want DismissedMsg", cmd())
	}

	m2 := New("content")
	next2, cmd2 := m2.Update(key(tui.KeyEsc))
	if next2.Open() {
		t.Error("Esc should close the drawer")
	}
	if cmd2 == nil {
		t.Fatal("Esc should return a non-nil Cmd")
	}
	if _, ok := cmd2().(DismissedMsg); !ok {
		t.Errorf("Cmd produced %T, want DismissedMsg", cmd2())
	}

	m3 := New("content")
	next3, cmd3 := m3.Update(key(tui.KeyDown))
	if !next3.Open() {
		t.Error("an unrelated key should not close the drawer")
	}
	if cmd3 != nil {
		t.Error("an unrelated key should not return a Cmd")
	}

	m4 := New("content")
	m4.Hide()
	next4, cmd4 := m4.Update(key(tui.KeyEnter))
	if next4.Open() {
		t.Error("Update on a closed drawer should stay closed")
	}
	if cmd4 != nil {
		t.Error("Update on a closed drawer should not return a Cmd, even for Enter/Esc")
	}
}

// TestRenderReturnsBaseUnchangedWhenClosed proves criterion #543: Render
// returns base unchanged while closed.
func TestRenderReturnsBaseUnchangedWhenClosed(t *testing.T) {
	base := "some\nbackground\ncontent"
	m := New("hi")
	m.Hide()
	if got := m.Render(base); got != base {
		t.Errorf("Render() with a closed drawer = %q, want base unchanged", got)
	}
}

// TestRenderCompositesBorderedBoxSizedByEdge proves criterion #544: Render
// composites a bordered box holding Content, sized by Width for
// EdgeLeft/EdgeRight or Height for EdgeTop/EdgeBottom.
func TestRenderCompositesBorderedBoxSizedByEdge(t *testing.T) {
	tests := []struct {
		name string
		edge Edge
	}{
		{"left sized by width", EdgeLeft},
		{"right sized by width", EdgeRight},
		{"top sized by height", EdgeTop},
		{"bottom sized by height", EdgeBottom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := grid(60, 30)
			m := New("line one\nline two")
			m.Edge = tt.edge
			m.Width = 20
			m.Height = 8
			got := m.Render(base)

			visible := ansi.StripANSI(got)
			if !strings.Contains(visible, "line one") || !strings.Contains(visible, "line two") {
				t.Fatalf("Render() should contain multi-line Content: %q", visible)
			}

			b := layout.NewBox().BorderColor(m.Theme.BorderColor)
			var box string
			switch tt.edge {
			case EdgeLeft, EdgeRight:
				box = boxdraw.Draw(b, m.Theme.Border, 1, m.Width, m.Content)
				bw, _ := extent(box)
				if want := m.Width + 2 + 2; bw != want { // border (1+1) + padding (1+1)
					t.Errorf("box width = %d, want %d (Width %d + border/padding)", bw, want, m.Width)
				}
			case EdgeTop, EdgeBottom:
				box = boxdraw.Draw(b, m.Theme.Border, 1, 0, padToHeight(m.Content, m.Height))
				_, bh := extent(box)
				if want := m.Height + 2 + 2; bh != want { // border (1+1) + padding (1+1)
					t.Errorf("box height = %d, want %d (Height %d + border/padding)", bh, want, m.Height)
				}
			}
		})
	}
}

// TestRenderPositionsBoxAtEdge proves criterion #545: Render positions the
// box at the edge corresponding to m.Edge: x=0 for Left, x=baseWidth-
// boxWidth for Right, y=0 for Top, y=baseHeight-boxHeight for Bottom.
func TestRenderPositionsBoxAtEdge(t *testing.T) {
	base := grid(60, 30)

	newDrawer := func(edge Edge) Model {
		m := New("hi")
		m.Edge = edge
		m.Width = 20
		m.Height = 8
		return m
	}

	boxFor := func(m Model) string {
		b := layout.NewBox().BorderColor(m.Theme.BorderColor)
		switch m.Edge {
		case EdgeLeft, EdgeRight:
			return boxdraw.Draw(b, m.Theme.Border, 1, m.Width, m.Content)
		default:
			return boxdraw.Draw(b, m.Theme.Border, 1, 0, padToHeight(m.Content, m.Height))
		}
	}

	t.Run("EdgeLeft places box at x=0", func(t *testing.T) {
		m := newDrawer(EdgeLeft)
		got := m.Render(base)
		box := boxFor(m)
		_, bh := extent(box)
		_, baseH := extent(base)
		wantY := (baseH - bh) / 2
		want := layout.Overlay(base, box, 0, wantY)
		if got != want {
			t.Errorf("Render() with EdgeLeft did not place box at x=0")
		}
	})

	t.Run("EdgeRight places box at x=baseWidth-boxWidth", func(t *testing.T) {
		m := newDrawer(EdgeRight)
		got := m.Render(base)
		box := boxFor(m)
		bw, bh := extent(box)
		baseW, baseH := extent(base)
		wantX := baseW - bw
		wantY := (baseH - bh) / 2
		want := layout.Overlay(base, box, wantX, wantY)
		if got != want {
			t.Errorf("Render() with EdgeRight did not place box at x=baseWidth-boxWidth")
		}
	})

	t.Run("EdgeTop places box at y=0", func(t *testing.T) {
		m := newDrawer(EdgeTop)
		got := m.Render(base)
		box := boxFor(m)
		bw, _ := extent(box)
		baseW, _ := extent(base)
		wantX := (baseW - bw) / 2
		want := layout.Overlay(base, box, wantX, 0)
		if got != want {
			t.Errorf("Render() with EdgeTop did not place box at y=0")
		}
	})

	t.Run("EdgeBottom places box at y=baseHeight-boxHeight", func(t *testing.T) {
		m := newDrawer(EdgeBottom)
		got := m.Render(base)
		box := boxFor(m)
		bw, bh := extent(box)
		baseW, baseH := extent(base)
		wantX := (baseW - bw) / 2
		wantY := baseH - bh
		want := layout.Overlay(base, box, wantX, wantY)
		if got != want {
			t.Errorf("Render() with EdgeBottom did not place box at y=baseHeight-boxHeight")
		}
	})
}

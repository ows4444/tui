package virtuallist

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/theme"
)

// #518: only the visible window is rendered, never all items.
func TestView_OnlyRendersVisibleWindow(t *testing.T) {
	calls := 0
	m := New(10000, 10, func(i int) string {
		calls++
		return fmt.Sprintf("item %d", i)
	})

	out := m.View()

	if calls >= 10000 {
		t.Fatalf("RenderItem called %d times, want far fewer than ItemCount=10000", calls)
	}
	if calls != m.Height {
		t.Fatalf("RenderItem called %d times, want exactly Height=%d", calls, m.Height)
	}
	if strings.Count(out, "\n")+1 != calls {
		t.Fatalf("View() has %d lines, want %d matching RenderItem calls", strings.Count(out, "\n")+1, calls)
	}
}

// #26: Height 10, Overscan 2, offset 50 renders 10 rows, 50 through 59.
func TestView_ExactlyHeightRowsFromOffsetDespiteOverscan(t *testing.T) {
	m := New(1000, 10, func(i int) string { return fmt.Sprintf("row %d", i) })
	m.Overscan = 2
	m.LineDown(50)
	if m.Offset() != 50 || m.Overscan != 2 {
		t.Fatalf("setup: offset=%d overscan=%d", m.Offset(), m.Overscan)
	}
	rows := strings.Split(m.View(), "\n")
	if len(rows) != 10 {
		t.Fatalf("View has %d rows, want 10: %q", len(rows), rows)
	}
	for i, r := range rows {
		if want := fmt.Sprintf("row %d", 50+i); r != want {
			t.Errorf("row %d = %q, want %q", i, r, want)
		}
	}
}

// The same holds at the ends: the last window is exactly Height rows too.
func TestView_ExactlyHeightRowsAtBottom(t *testing.T) {
	m := New(100, 10, func(i int) string { return fmt.Sprintf("row %d", i) })
	m.LineDown(1000)
	rows := strings.Split(m.View(), "\n")
	if len(rows) != 10 || rows[0] != "row 90" || rows[9] != "row 99" {
		t.Fatalf("got %q", rows)
	}
}

// #518, scrolled: window follows offset and stays small regardless of position.
func TestView_ScrolledWindowStaysSmall(t *testing.T) {
	calls := 0
	m := New(10000, 10, func(i int) string {
		calls++
		return fmt.Sprintf("item %d", i)
	})
	m.LineDown(5000)
	m.View()

	if calls >= 10000 {
		t.Fatalf("RenderItem called %d times after scrolling, want far fewer than ItemCount=10000", calls)
	}
	if calls != m.Height {
		t.Fatalf("RenderItem called %d times, want exactly %d", calls, m.Height)
	}
}

// #519: LineUp/LineDown move offset clamped to [0, ItemCount-Height].
func TestLineUpDown_ClampsOffset(t *testing.T) {
	m := New(100, 10, func(i int) string { return "" })

	m.LineUp(50) // can't go below 0
	if m.Offset() != 0 {
		t.Fatalf("Offset() = %d, want 0 after LineUp past top", m.Offset())
	}

	m.LineDown(1000) // can't go past ItemCount-Height = 90
	if got, want := m.Offset(), 90; got != want {
		t.Fatalf("Offset() = %d, want %d after LineDown past bottom", got, want)
	}

	m.LineUp(1000)
	if m.Offset() != 0 {
		t.Fatalf("Offset() = %d, want 0 after LineUp past top again", m.Offset())
	}

	m.LineDown(5)
	if m.Offset() != 5 {
		t.Fatalf("Offset() = %d, want 5", m.Offset())
	}
}

func TestUpdate_ArrowKeysScroll(t *testing.T) {
	m := New(100, 10, func(i int) string { return "" })

	m, _ = m.Update(tui.Key{Type: tui.KeyDown})
	if m.Offset() != 1 {
		t.Fatalf("Offset() = %d, want 1 after KeyDown", m.Offset())
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyUp})
	if m.Offset() != 0 {
		t.Fatalf("Offset() = %d, want 0 after KeyUp", m.Offset())
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyUp})
	if m.Offset() != 0 {
		t.Fatalf("Offset() = %d, want 0 (clamped) after KeyUp past top", m.Offset())
	}
}

// #520: ItemCount <= Height fits on one screen; offset stays 0 and every
// item is rendered exactly once.
func TestView_FitsWithoutScrolling(t *testing.T) {
	calls := 0
	var seen []int
	m := New(5, 10, func(i int) string {
		calls++
		seen = append(seen, i)
		return fmt.Sprintf("item %d", i)
	})

	m.LineDown(100) // attempt to scroll; should have no effect since it all fits
	if m.Offset() != 0 {
		t.Fatalf("Offset() = %d, want 0 when ItemCount <= Height", m.Offset())
	}

	m.View()
	if calls != 5 {
		t.Fatalf("RenderItem called %d times, want exactly ItemCount=5", calls)
	}
	for i := 0; i < 5; i++ {
		found := false
		for _, s := range seen {
			if s == i {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("item %d never rendered", i)
		}
	}
}

// #521: ItemCount <= 0 or Height <= 0 renders "" without calling RenderItem
// or panicking.
func TestView_EmptyOrInvalidDimensions(t *testing.T) {
	tests := []struct {
		name      string
		itemCount int
		height    int
	}{
		{"zero ItemCount", 0, 10},
		{"negative ItemCount", -5, 10},
		{"zero Height", 100, 0},
		{"negative Height", 100, -3},
		{"both zero", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			m := New(tt.itemCount, tt.height, func(i int) string {
				calls++
				return "x"
			})

			var out string
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("View() panicked: %v", r)
					}
				}()
				out = m.View()
			}()

			if out != "" {
				t.Fatalf("View() = %q, want empty string", out)
			}
			if calls != 0 {
				t.Fatalf("RenderItem called %d times, want 0", calls)
			}
		})
	}
}

// TestNewDefaults proves New's documented zero-opts defaults: ItemHeight
// 1, Overscan 2, Theme theme.DarkTheme().
func TestNewDefaults(t *testing.T) {
	m := New(10, 5, func(i int) string { return "" })
	if m.ItemHeight != 1 {
		t.Errorf("ItemHeight = %d, want 1", m.ItemHeight)
	}
	if m.Overscan != defaultOverscan {
		t.Errorf("Overscan = %d, want %d", m.Overscan, defaultOverscan)
	}
	if m.Theme != theme.DarkTheme() {
		t.Errorf("Theme = %+v, want theme.DarkTheme()", m.Theme)
	}
}

func TestView_NilRenderItemDoesNotPanic(t *testing.T) {
	m := Model{ItemCount: 10, Height: 5}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("View() panicked with nil RenderItem: %v", r)
		}
	}()
	if got := m.View(); got != "" {
		t.Fatalf("View() = %q, want empty string with nil RenderItem", got)
	}
}

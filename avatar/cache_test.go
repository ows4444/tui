package avatar

import (
	"testing"

	"github.com/ows4444/tui/theme"
)

func countDraws(t *testing.T) *int {
	t.Helper()
	n := 0
	drawHook = func() { n++ }
	t.Cleanup(func() { drawHook = nil })
	return &n
}

// A Model from New draws once and then returns the cached View until a field
// the View depends on changes.
func TestViewIsCachedUntilAFieldChanges(t *testing.T) {
	draws := countDraws(t)
	m := New("alain00")
	first := m.View()
	for range 5 {
		if m.View() != first {
			t.Fatal("the cached View differs")
		}
	}
	if *draws != 1 {
		t.Fatalf("an unchanged Model drew %d times, want 1", *draws)
	}

	changes := map[string]func(*Model){
		"Name":       func(m *Model) { m.Name = "tove" },
		"Raw":        func(m *Model) { m.Name, m.Raw = "ALAIN00", true },
		"Width":      func(m *Model) { m.Width, m.Height = 12, 6 },
		"Height":     func(m *Model) { m.Height = 5 },
		"Background": func(m *Model) { m.Background = BackgroundCircle },
		"LookX":      func(m *Model) { m.LookX = 1 },
		"LookY":      func(m *Model) { m.Width, m.Height, m.LookY = 24, 12, -1 },
		"blink":      func(m *Model) { m.Blink() },
		"Expression": func(m *Model) { m.Width, m.Height, m.Expression = 12, 6, ExpressionSurprised },
		"Theme":      func(m *Model) { m.Theme = theme.DarkTheme().ASCII() },
	}
	for field, change := range changes {
		c := New("alain00")
		before := c.View()
		*draws = 0
		change(&c)
		fresh := c
		fresh.cache = nil
		want := fresh.View() // drawn without a cache
		*draws = 0
		if got := c.View(); got != want {
			t.Errorf("%s: the cached Model returned a stale View", field)
		}
		if *draws != 1 {
			t.Errorf("%s: changing it drew %d times, want 1", field, *draws)
		}
		if c.View() == before && field != "Raw" {
			t.Errorf("%s: the change did not alter the View, so the case proves nothing", field)
		}
	}
}

// A look outside [-1, 1] is clamped before it is drawn, so it shares the
// cached View of the clamped value.
func TestCacheKeyUsesTheClampedLook(t *testing.T) {
	draws := countDraws(t)
	m := New("alain00")
	m.LookX = 1
	full := m.View()
	m.LookX = 7
	if m.View() != full || *draws != 1 {
		t.Errorf("a look of 7 drew again (%d draws)", *draws)
	}
}

// A struct literal has no cache and draws every time, with the same result.
func TestStructLiteralDrawsEveryTime(t *testing.T) {
	draws := countDraws(t)
	m := Model{Name: "alain00", Width: DefaultWidth, Height: DefaultHeight}
	if m.View() != m.View() || m.View() == "" {
		t.Error("an uncached Model does not draw the same View")
	}
	if *draws != 3 {
		t.Errorf("an uncached Model drew %d times, want 3", *draws)
	}
	if got, want := m.View(), New("alain00").View(); got != want {
		t.Error("the uncached and cached Views differ")
	}
}

// Copies of a Model share one cache, and each still gets its own View.
func TestCopiesShareTheCacheSafely(t *testing.T) {
	a := New("alain00")
	b := a
	b.Name = "tove"
	wantA, wantB := a.View(), b.View()
	done := make(chan struct{})
	for _, m := range []Model{a, b} {
		go func() {
			defer func() { done <- struct{}{} }()
			want := wantA
			if m.Name == "tove" {
				want = wantB
			}
			for range 30 {
				if m.View() != want {
					t.Error("a copy got another copy's View")
					return
				}
			}
		}()
	}
	<-done
	<-done
}

func BenchmarkView(b *testing.B) {
	for _, size := range [][2]int{{DefaultWidth, DefaultHeight}, {24, 12}} {
		m := New("alain00")
		m.Width, m.Height = size[0], size[1]
		uncached := m
		uncached.cache = nil
		b.Run(sizeName(size)+"/cached", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = m.View()
			}
		})
		b.Run(sizeName(size)+"/uncached", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = uncached.View()
			}
		})
	}
}

func sizeName(s [2]int) string {
	return string(rune('0'+s[0]/10)) + string(rune('0'+s[0]%10)) + "x" + string(rune('0'+s[1]/10)) + string(rune('0'+s[1]%10))
}

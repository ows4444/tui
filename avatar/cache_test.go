package avatar

import (
	"bytes"
	"image/png"
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

// BenchmarkAvatarView measures View at the default and a large size, with
// and without the cache. CI runs it in the benchmark smoke pass.
func BenchmarkAvatarView(b *testing.B) {
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

// A Model from New draws its PNG once and returns that same slice until the
// avatar or the size changes; a struct literal draws every time.
func TestPNGIsCachedUntilTheAvatarChanges(t *testing.T) {
	draws := countDraws(t)
	m := New("alain00")
	first := m.PNG(64)
	for range 4 {
		if again := m.PNG(64); &again[0] != &first[0] {
			t.Fatal("an unchanged Model returned another slice")
		}
	}
	if *draws != 1 {
		t.Fatalf("an unchanged Model drew its PNG %d times, want 1", *draws)
	}
	// The View's cache is separate: drawing cells does not drop the image.
	_ = m.View()
	if again := m.PNG(64); &again[0] != &first[0] {
		t.Error("drawing the View dropped the cached PNG")
	}
	for what, change := range map[string]func(*Model) int{
		"size":       func(m *Model) int { return 96 },
		"expression": func(m *Model) int { m.Expression = ExpressionSad; return 64 },
		"look":       func(m *Model) int { m.LookX = 1; return 64 },
		"blink":      func(m *Model) int { m.Blink(); return 64 },
		"name":       func(m *Model) int { m.Name = "tove"; return 64 },
	} {
		c := New("alain00")
		before := c.PNG(64)
		*draws = 0
		size := change(&c)
		after := c.PNG(size)
		if *draws != 1 || &after[0] == &before[0] {
			t.Errorf("%s: a change drew %d times, or returned the old slice", what, *draws)
		}
		fresh := c
		fresh.cache = nil
		if string(fresh.PNG(size)) != string(after) {
			t.Errorf("%s: the cached PNG is stale", what)
		}
	}
	literal := Model{Name: "alain00"}
	*draws = 0
	a, b := literal.PNG(32), literal.PNG(32)
	if *draws != 2 || &a[0] == &b[0] || string(a) != string(b) {
		t.Errorf("a struct literal drew %d times", *draws)
	}
}

// PNG samples a pixel finely only where an outline crosses it. The picture
// must be the one full sampling of every pixel gives.
func TestPNGMatchesFullSampling(t *testing.T) {
	const size = 48
	for _, name := range []string{"ada", "linus", "grace", "ken", "user-21"} {
		for _, e := range []Expression{ExpressionNone, ExpressionHappy, ExpressionMad, ExpressionScared} {
			m := Model{Name: name, Expression: e}
			img, err := png.Decode(bytes.NewReader(m.PNG(size)))
			if err != nil {
				t.Fatal(err)
			}
			f := layoutFigure(m.traits())
			s := newScene(f, nil)
			from, to, at := m.poses()
			s.setEyes(f.posed(from, to, at, 0, 0, 0, 1, 1))
			s.setLift(lift(from, to, at), 0)
			cx, cy, side := s.bounds()
			unit := side / size
			for j := 0; j < size; j++ {
				for i := 0; i < size; i++ {
					hit := 0
					for sb := 0; sb < samples; sb++ {
						for sa := 0; sa < samples; sa++ {
							x := cx - side/2 + (float64(i)+(float64(sa)+0.5)/samples)*unit
							y := cy - side/2 + (float64(j)+(float64(sb)+0.5)/samples)*unit
							if s.at(x, y) != layerNone {
								hit++
							}
						}
					}
					want := uint32(shade(255*float64(hit)/(samples*samples))) * 0x101
					if _, _, _, a := img.At(i, j).RGBA(); a != want {
						t.Fatalf("%q %v: pixel (%d, %d) has alpha %#x, full sampling gives %#x", name, e, i, j, a, want)
					}
				}
			}
		}
	}
}

// An animation that returns to frames it has shown draws each once: a second
// blink, and every cycle of a tremble or a rock after the first, draw
// nothing new, for the View and for the PNG.
func TestRepeatedAnimationFramesAreNotRedrawn(t *testing.T) {
	draws := countDraws(t)
	show := func(m Model) { _, _ = m.View(), m.PNG(64) }

	m := New("alain00")
	show(m)
	blink := func() {
		m.Blink()
		for m.Blinking() {
			show(m)
			m, _ = m.Update(tickMsg{owner: m.owner})
		}
		show(m)
	}
	blink()
	first := *draws
	blink()
	if *draws != first {
		t.Errorf("a second blink drew %d more pictures", *draws-first)
	}

	for _, e := range []Expression{ExpressionMad, ExpressionThinking} {
		w, _ := wobbling(t, e)
		cycle := func() {
			for range wobbleCycle {
				show(w)
				w, _ = step(w)
			}
		}
		*draws = 0
		cycle()
		once := *draws
		cycle()
		cycle()
		if *draws != once {
			t.Errorf("%v: later cycles drew %d more pictures", e, *draws-once)
		}
		want := 2 * wobbleCycle // a View and a PNG for each frame of a rock
		if e == ExpressionMad {
			want = 2 * 2 // a tremble has two pictures
		}
		if once != want {
			t.Errorf("%v: the first cycle drew %d pictures, want %d", e, once, want)
		}
	}
}

// The store is bounded: past frameMemory pictures the oldest is dropped and
// drawn again when it is next shown, and what is returned is always what an
// uncached Model returns.
func TestFrameMemoryIsBounded(t *testing.T) {
	draws := countDraws(t)
	m := New("alain00")
	m.Width, m.Height = 24, 12
	look := func(i int) Model {
		c := m
		c.LookX = float64(i) / float64(2*frameMemory)
		return c
	}
	for i := 0; i <= frameMemory; i++ { // one more than fits
		c := look(i)
		fresh := c
		fresh.cache = nil
		if c.View() != fresh.View() || string(c.PNG(48)) != string(fresh.PNG(48)) {
			t.Fatalf("look %d: the cached picture is not the uncached one", i)
		}
	}
	if n := len(m.cache.views.at); n != frameMemory || len(m.cache.views.order) != frameMemory || len(m.cache.pngs.at) != frameMemory {
		t.Fatalf("the store holds %d views and %d images, want %d of each", n, len(m.cache.pngs.at), frameMemory)
	}
	*draws = 0
	last := look(frameMemory)
	_, _ = last.View(), last.PNG(48)
	if *draws != 0 {
		t.Errorf("the newest picture was drawn again (%d draws)", *draws)
	}
	oldest := look(0)
	_, _ = oldest.View(), oldest.PNG(48)
	if *draws != 2 {
		t.Errorf("the dropped picture was drawn %d times on its return, want 2 (a View and a PNG)", *draws)
	}
}

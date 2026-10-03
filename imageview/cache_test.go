package imageview

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

// gradientPNG is a w by h image with many colours, so Sixel has work to do.
func gradientPNG(tb testing.TB, w, h int) []byte {
	tb.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), uint8(x ^ y), 255})
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		tb.Fatal(err)
	}
	return b.Bytes()
}

// countEncodes reports how many times View encoded while f ran.
func countEncodes(t *testing.T, f func()) int {
	t.Helper()
	n := 0
	encodeHook = func() { n++ }
	defer func() { encodeHook = nil }()
	f()
	return n
}

// A second View of the same image, size and mode does not encode again, in
// either graphics mode, and returns the same string.
func TestViewEncodesOncePerInput(t *testing.T) {
	for _, mode := range []string{"kitty", "sixel"} {
		m := New(gradientPNG(t, 64, 64), 10, 4, "alt")
		m.Kitty, m.Sixel = mode == "kitty", mode == "sixel"
		m.Getenv = func(string) string { return "" }
		var first, second string
		if n := countEncodes(t, func() { first, second = m.View(), m.View() }); n != 1 {
			t.Errorf("%s: two Views encoded %d times, want 1", mode, n)
		}
		if first != second || first == "" {
			t.Errorf("%s: the cached View differs from the first", mode)
		}
		// A copy of the Model, as Update returns one, keeps the cache.
		cp := m
		if n := countEncodes(t, func() { _ = cp.View() }); n != 0 {
			t.Errorf("%s: a copy of the Model encoded again", mode)
		}
	}
}

// Changing the image, the size, the mode or a field the output depends on
// gives the output for the new input, the same a fresh Model gives.
func TestViewCacheFollowsItsInputs(t *testing.T) {
	noEnv := func(string) string { return "" }
	a, b := gradientPNG(t, 64, 64), gradientPNG(t, 48, 32)
	m := New(a, 10, 4, "alt")
	m.Kitty, m.Getenv = true, noEnv
	_ = m.View()

	fresh := func(m Model) string { m.cache = nil; return m.View() }
	changes := map[string]func(*Model){
		"image":       func(m *Model) { m.PNG = b },
		"width":       func(m *Model) { m.Width = 12 },
		"height":      func(m *Model) { m.Height = 5 },
		"mode":        func(m *Model) { m.Kitty, m.Sixel = false, true },
		"id":          func(m *Model) { m.ID = 7 },
		"multiplexer": func(m *Model) { m.Getenv = func(k string) string { return map[string]string{"TMUX": "1"}[k] } },
		"placeholder": func(m *Model) { m.Kitty = false },
		"cell size": func(m *Model) {
			m.Kitty, m.Sixel = false, true
			_ = m.View()
			m.CellWidth, m.CellHeight = 6, 12
		},
	}
	for name, change := range changes {
		next := m
		change(&next)
		if got, want := next.View(), fresh(next); got != want {
			t.Errorf("after changing the %s, View is stale", name)
		}
		if next.View() == m.View() {
			t.Errorf("changing the %s did not change View", name)
		}
	}
}

// A struct literal has no cache: it still renders, encoding each time.
func TestViewWithoutCache(t *testing.T) {
	m := Model{PNG: gradientPNG(t, 16, 16), Width: 4, Height: 2, Kitty: true, Getenv: func(string) string { return "" }}
	if n := countEncodes(t, func() { _, _ = m.View(), m.View() }); n != 2 {
		t.Errorf("a struct literal encoded %d times for two Views, want 2", n)
	}
}

func benchView(b *testing.B, cached, kitty bool) {
	m := New(gradientPNG(b, 320, 240), 40, 20, "alt")
	m.Kitty, m.Sixel = kitty, !kitty
	m.Getenv = func(string) string { return "" }
	if !cached {
		m.cache = nil // what every View cost before the cache
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}

func BenchmarkViewRepeatedKitty(b *testing.B)         { benchView(b, true, true) }
func BenchmarkViewRepeatedKittyUncached(b *testing.B) { benchView(b, false, true) }
func BenchmarkViewRepeatedSixel(b *testing.B)         { benchView(b, true, false) }
func BenchmarkViewRepeatedSixelUncached(b *testing.B) { benchView(b, false, false) }

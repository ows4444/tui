package imageview

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
)

// Inline is one OSC 1337 File sequence that carries the PNG as it is and
// sizes it in cells.
func TestInlineSequence(t *testing.T) {
	png := gradientPNG(t, 16, 16)
	seq := Inline(png, 8, 4)
	want := "\x1b]1337;File=inline=1;size=" + itoa(len(png)) + ";width=8;height=4;preserveAspectRatio=0:"
	if !strings.HasPrefix(seq, want) || !strings.HasSuffix(seq, "\a") {
		t.Fatalf("unexpected sequence: %.80q...", seq)
	}
	body := strings.TrimSuffix(strings.TrimPrefix(seq, want), "\a")
	got, err := base64.StdEncoding.DecodeString(body)
	if err != nil || string(got) != string(png) {
		t.Errorf("the payload is not the PNG (err %v)", err)
	}
	if strings.Count(seq, "\x1b]1337;File=") != 1 {
		t.Error("more than one sequence")
	}
	for _, bad := range [][3]int{{0, 4, 1}, {8, 0, 1}, {-1, 4, 1}, {8, 4, 0}} {
		data := png
		if bad[2] == 0 {
			data = nil
		}
		if got := Inline(data, bad[0], bad[1]); got != "" {
			t.Errorf("Inline with cols %d rows %d and %d bytes = %.30q", bad[0], bad[1], len(data), got)
		}
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for ; n > 0; n /= 10 {
		b = append([]byte{byte('0' + n%10)}, b...)
	}
	return string(b)
}

// A Model draws through the first protocol the terminal has: kitty, then
// inline images, then Sixel; with none it draws the placeholder.
func TestProtocolOrder(t *testing.T) {
	png := gradientPNG(t, 16, 16)
	const kitty, inline, sixel = "\x1b_G", "\x1b]1337;File=", "\x1bP"
	for name, c := range map[string]struct {
		k, i, s bool
		want    string
	}{
		"kitty only":       {true, false, false, kitty},
		"inline only":      {false, true, false, inline},
		"sixel only":       {false, false, true, sixel},
		"kitty and inline": {true, true, false, kitty},
		"inline and sixel": {false, true, true, inline},
		"all three":        {true, true, true, kitty},
		"none":             {false, false, false, ""},
	} {
		m := New(png, 8, 4, "alt")
		m.Kitty, m.Inline, m.Sixel = c.k, c.i, c.s
		m.Getenv = func(string) string { return "" }
		view := m.View()
		for _, p := range []string{kitty, inline, sixel} {
			if has := strings.Contains(view, p); has != (p == c.want) {
				t.Errorf("%s: sequence %q present=%v", name, p, has)
			}
		}
		rows := strings.Split(view, "\n")
		if len(rows) != 4 {
			t.Fatalf("%s: %d rows", name, len(rows))
		}
		for i, r := range rows {
			if w := ansi.Width(r); w != 8 {
				t.Errorf("%s: row %d is %d cells wide", name, i, w)
			}
		}
	}
}

// The inline View is encoded once while the Model is unchanged, and again
// when the protocol, the size or the image changes.
func TestInlineViewIsCached(t *testing.T) {
	encodes := 0
	encodeHook = func() { encodes++ }
	t.Cleanup(func() { encodeHook = nil })
	m := New(gradientPNG(t, 16, 16), 8, 4, "alt")
	m.Inline = true
	first := m.View()
	for range 3 {
		if m.View() != first {
			t.Fatal("an unchanged Model drew differently")
		}
	}
	if encodes != 1 {
		t.Fatalf("an unchanged Model encoded %d times", encodes)
	}
	m.Width = 10
	if m.View() == first || encodes != 2 {
		t.Errorf("a new size did not encode again (%d encodes)", encodes)
	}
	m.Inline, m.Sixel = false, true
	if strings.Contains(m.View(), "1337") || encodes != 3 {
		t.Errorf("switching to Sixel reused the inline View (%d encodes)", encodes)
	}
	m.Sixel, m.Inline = false, true
	m.PNG = gradientPNG(t, 8, 8)
	if !strings.Contains(m.View(), "1337") || encodes != 4 {
		t.Errorf("a new image did not encode again (%d encodes)", encodes)
	}
}

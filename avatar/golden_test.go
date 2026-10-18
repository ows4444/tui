package avatar

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ows4444/tui/cellbuf"
	"github.com/ows4444/tui/theme"
)

// goldenCases are the pictures kept in testdata/views: a few names at three
// sizes on each background, in colour and under an ASCII glyph set. Run with
// AVATAR_UPDATE_GOLDEN=1 to rewrite them after a deliberate change.
func goldenCases() map[string]Model {
	cases := map[string]Model{}
	for _, name := range []string{"ada", "grace", "linus", "ken"} {
		for _, size := range [][2]int{{8, 4}, {16, 8}, {32, 16}} {
			for bg, bgName := range map[Background]string{BackgroundNone: "none", BackgroundCircle: "circle", BackgroundSquare: "square"} {
				for _, th := range []struct {
					name string
					th   theme.Theme
				}{{"dark", theme.DarkTheme()}, {"light", theme.LightTheme()}, {"ascii", theme.DarkTheme().ASCII()}} {
					m := New(name)
					m.Width, m.Height, m.Background, m.Theme = size[0], size[1], bg, th.th
					cases[fmt.Sprintf("%s-%dx%d-%s-%s", name, size[0], size[1], bgName, th.name)] = m
				}
			}
		}
	}
	return cases
}

// escapes shows the control bytes of a view, so a golden file diffs as text.
func escapes(s string) string { return strings.ReplaceAll(s, "\x1b", "\\e") }

func TestViewsMatchTheirGoldenFiles(t *testing.T) {
	dir := filepath.Join("testdata", "views")
	update := os.Getenv("AVATAR_UPDATE_GOLDEN") != ""
	if update {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatal(err)
		}
	}
	for name, m := range goldenCases() {
		path := filepath.Join(dir, name+".txt")
		got := escapes(m.View()) + "\n"
		if update {
			if err := os.WriteFile(path, []byte(got), 0o600); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(path) // #nosec G304 -- a fixed directory of test data
		if err != nil {
			t.Fatalf("%s: %v (AVATAR_UPDATE_GOLDEN=1 writes the files)", name, err)
		}
		if got != string(want) {
			t.Errorf("%s: the picture changed; AVATAR_UPDATE_GOLDEN=1 rewrites the golden files after a deliberate change", name)
		}
	}
}

// FuzzAvatar draws any name every way and expects no panic, rows of the right
// width, and an SVG with nothing in it but numbers and colours.
func FuzzAvatar(f *testing.F) {
	for _, s := range []string{"", "ada", " Ada ", "é", "é", "\x1b[31m", "日本語", "\xff\xfe", strings.Repeat("x", 300)} {
		f.Add(s, uint8(8), uint8(4))
	}
	f.Fuzz(func(t *testing.T, name string, w, h uint8) {
		m := New(name)
		m.Width, m.Height = int(w%40), int(h%20)
		m.Background = Background(w % 4)
		m.Expression = Expression(h % 14)
		v := m.View()
		if m.Width > 0 && m.Height > 0 && strings.Count(v, "\n") != m.Height-1 {
			t.Fatalf("%q: %d rows, want %d", name, strings.Count(v, "\n")+1, m.Height)
		}
		if m.Width > 0 && m.Height > 0 && m.Width <= 40 {
			if got, want := drawn(m), cellsOf(m); got != want {
				t.Fatalf("%q: DrawCells differs from View", name)
			}
		}
		_ = m.Linearize()
		m.Colors()
		// Markup characters of the name never reach the SVG.
		if svg := m.SVG(); strings.ContainsAny(name, "<>\"&'") && strings.ContainsAny(strings.NewReplacer("<path", "", "<circle", "", "<g", "", "<svg", "", "</", "", "/>", "", ">", "").Replace(svg), "<>&'") {
			t.Fatalf("the name leaked into the SVG: %q", name)
		}
		if w%8 == 0 {
			_ = m.PNG(int(w%32) + 1)
		}
	})
}

// drawn is m drawn by DrawCells into a grid of its own size.
func drawn(m Model) string {
	buf := cellbuf.New(m.Width, m.Height)
	m.DrawCells(buf, buf.Bounds())
	return buf.String()
}

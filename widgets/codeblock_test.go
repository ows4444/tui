package widgets

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

func plainLines(s string) []string { return strings.Split(ansi.StripANSI(s), "\n") }

func TestCodeBlockZeroWidth(t *testing.T) {
	for _, w := range []int{0, -2} {
		if got := CodeBlock("x", w, true, theme.DarkTheme()); got != "" {
			t.Errorf("CodeBlock width %d = %q, want empty", w, got)
		}
	}
}

func TestCodeBlockEveryLineIsExactlyWidth(t *testing.T) {
	codes := map[string]string{
		"short":     "a\nb",
		"long":      strings.Repeat("x", 200),
		"wide":      "你好世界你好世界你好世界",
		"tabs":      "\tif x {\n\t\treturn\n\t}",
		"empty":     "",
		"blank mid": "a\n\nb",
		"styled":    ansi.NewStyle().Bold().Render("bold") + " plain",
	}
	for name, code := range codes {
		for _, w := range []int{5, 6, 10, 24, 80} {
			for _, nums := range []bool{false, true} {
				for i, l := range strings.Split(CodeBlock(code, w, nums, theme.DarkTheme()), "\n") {
					if got := ansi.Width(l); got != w {
						t.Errorf("%s w=%d nums=%v line %d: Width = %d, want %d: %q", name, w, nums, i, got, w, l)
					}
				}
			}
		}
	}
}

func TestCodeBlockTruncatesWithEllipsis(t *testing.T) {
	got := plainLines(CodeBlock("abcdefghijklmnop\nshort", 12, false, theme.DarkTheme()))
	// width 12 = border(2) + padding(2) + 8 content columns.
	if want := "│ abcdefg… │"; got[1] != want {
		t.Errorf("long line = %q, want %q", got[1], want)
	}
	if want := "│ short    │"; got[2] != want {
		t.Errorf("short line = %q, want %q", got[2], want)
	}
	if len(got) != 4 {
		t.Errorf("got %d lines, want 4 (no wrapping)", len(got))
	}
}

func TestCodeBlockTruncatesWideRuneWhole(t *testing.T) {
	got := plainLines(CodeBlock("你好世界", 9, false, theme.DarkTheme()))
	// 5 content columns: "你好" (4) + ellipsis, never a split rune.
	if want := "│ 你好… │"; got[1] != want {
		t.Errorf("line = %q, want %q", got[1], want)
	}
}

func TestCodeBlockBorderTheme(t *testing.T) {
	th := theme.DarkTheme()
	th.BorderColor = ansi.Red
	lines := strings.Split(CodeBlock("hi", 10, false, th), "\n")
	bc := ansi.NewStyle().Foreground(ansi.Red)
	wantTop := bc.Render(th.Border.TopLeft + strings.Repeat(th.Border.Top, 8) + th.Border.TopRight)
	wantBot := bc.Render(th.Border.BottomLeft + strings.Repeat(th.Border.Bottom, 8) + th.Border.BottomRight)
	if lines[0] != wantTop {
		t.Errorf("top = %q, want %q", lines[0], wantTop)
	}
	if lines[len(lines)-1] != wantBot {
		t.Errorf("bottom = %q, want %q", lines[len(lines)-1], wantBot)
	}
	if !strings.HasPrefix(lines[1], bc.Render(th.Border.Left)+" ") || !strings.HasSuffix(lines[1], " "+bc.Render(th.Border.Right)) {
		t.Errorf("side border/padding wrong: %q", lines[1])
	}
}

func TestCodeBlockDifferentBorders(t *testing.T) {
	th := theme.DarkTheme()
	th.Border.TopLeft = "+"
	th.Border.TopRight = "+"
	if p := plainLines(CodeBlock("x", 8, false, th)); !strings.HasPrefix(p[0], "+") || !strings.HasSuffix(p[0], "+") {
		t.Errorf("top row %q does not use the theme's corners", p[0])
	}
}

func TestCodeBlockNormalisesInput(t *testing.T) {
	tests := []struct {
		name string
		code string
		want []string // content rows, without border/padding
	}{
		{"tab", "\tx", []string{"    x"}},
		{"crlf", "a\r\nb\r\n", []string{"a", "b"}},
		{"one trailing newline", "a\n", []string{"a"}},
		{"two trailing newlines keep one blank", "a\n\n", []string{"a", ""}},
		{"empty", "", []string{""}},
	}
	for _, tt := range tests {
		got := plainLines(CodeBlock(tt.code, 20, false, theme.DarkTheme()))
		if len(got) != len(tt.want)+2 {
			t.Errorf("%s: %d lines, want %d", tt.name, len(got), len(tt.want)+2)
			continue
		}
		for i, w := range tt.want {
			row := strings.TrimSuffix(strings.TrimPrefix(got[i+1], "│ "), " │")
			if strings.TrimRight(row, " ") != w {
				t.Errorf("%s row %d = %q, want %q", tt.name, i, strings.TrimRight(row, " "), w)
			}
		}
	}
}

func TestCodeBlockLineNumbers(t *testing.T) {
	var code []string
	for i := 0; i < 12; i++ {
		code = append(code, "x")
	}
	out := CodeBlock(strings.Join(code, "\n"), 20, true, theme.DarkTheme())
	p := plainLines(out)
	if want := "│  1 │ x"; !strings.HasPrefix(p[1], want) {
		t.Errorf("first row = %q, want prefix %q", p[1], want)
	}
	if want := "│ 12 │ x"; !strings.HasPrefix(p[12], want) {
		t.Errorf("last row = %q, want prefix %q", p[12], want)
	}

	// Gutter width follows the last number: 3 lines -> one digit.
	p = plainLines(CodeBlock("a\nb\nc", 20, true, theme.DarkTheme()))
	if want := "│ 1 │ a"; !strings.HasPrefix(p[1], want) {
		t.Errorf("one-digit gutter row = %q, want prefix %q", p[1], want)
	}

	// The gutter (number and separator) is one Muted span.
	if want := ansi.NewStyle().Foreground(theme.DarkTheme().Muted).Render("1 │ "); !strings.Contains(CodeBlock("a", 20, true, theme.DarkTheme()), want) {
		t.Errorf("gutter not rendered in Muted (want span %q)", want)
	}

	// Without numbers there is no gutter.
	p = plainLines(CodeBlock("a", 20, false, theme.DarkTheme()))
	if want := "│ a"; !strings.HasPrefix(p[1], want) {
		t.Errorf("no-gutter row = %q, want prefix %q", p[1], want)
	}
}

func TestCodeBlockDropsNumbersWhenNoRoom(t *testing.T) {
	// width 8 -> 4 content columns; a 1-digit gutter needs 4, leaving 0.
	if got, want := CodeBlock("abc", 8, true, theme.DarkTheme()), CodeBlock("abc", 8, false, theme.DarkTheme()); got != want {
		t.Errorf("numbers not dropped when they leave no room:\n%q\nvs\n%q", got, want)
	}
}

func TestCodeBlockNarrowIsBorderless(t *testing.T) {
	for w := 1; w <= 4; w++ {
		got := CodeBlock("abcdefgh\nxy", w, true, theme.DarkTheme())
		for i, l := range plainLines(got) {
			if ansi.Width(l) > w {
				t.Errorf("w=%d line %d = %q wider than width", w, i, l)
			}
			if strings.ContainsAny(l, "│┌┐└┘─") {
				t.Errorf("w=%d line %d has border: %q", w, i, l)
			}
		}
		if n := len(plainLines(got)); n != 2 {
			t.Errorf("w=%d: %d lines, want 2", w, n)
		}
	}
}

func TestCodeBlockTextColourOnly(t *testing.T) {
	th := theme.DarkTheme()
	th.Text = ansi.Yellow
	out := CodeBlock("hello", 20, false, th)
	if want := ansi.NewStyle().Foreground(ansi.Yellow).Render("hello"); !strings.Contains(out, want) {
		t.Errorf("code not rendered in Text colour: %q", out)
	}
	// Visible code text equals the input.
	src := "func main() {\n    x := 1\n}"
	var got []string
	for _, l := range plainLines(CodeBlock(src, 40, false, th))[1:4] {
		got = append(got, strings.TrimRight(strings.TrimSuffix(strings.TrimPrefix(l, "│ "), " │"), " "))
	}
	if strings.Join(got, "\n") != src {
		t.Errorf("visible text = %q, want %q", strings.Join(got, "\n"), src)
	}
}

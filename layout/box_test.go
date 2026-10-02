package layout

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
)

func TestBoxRenderNoBorderNoPadding(t *testing.T) {
	got := NewBox().Render("hi")
	if got != "hi" {
		t.Errorf("Render() = %q, want %q", got, "hi")
	}
}

func TestBoxRenderPadsLinesToCommonWidth(t *testing.T) {
	got := NewBox().Render("a\nbb\nc")
	lines := strings.Split(got, "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want 3: %q", len(lines), got)
	}
	for _, l := range lines {
		if ansi.Width(l) != 2 {
			t.Errorf("line %q has width %d, want 2 (widest line is %q)", l, ansi.Width(l), "bb")
		}
	}
}

func TestBoxRenderPadding(t *testing.T) {
	got := NewBox().PaddingAll(1).Render("x")
	lines := strings.Split(got, "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want 3 (1 pad + 1 content + 1 pad): %q", len(lines), got)
	}
	if strings.TrimSpace(lines[0]) != "" || strings.TrimSpace(lines[2]) != "" {
		t.Errorf("padding rows should be blank, got %q / %q", lines[0], lines[2])
	}
	if lines[1] != " x " {
		t.Errorf("content row = %q, want %q", lines[1], " x ")
	}
}

func TestBoxRenderBorder(t *testing.T) {
	got := NewBox().Border(NormalBorder()).Render("hi")
	want := "┌──┐\n│hi│\n└──┘"
	if got != want {
		t.Errorf("Render() =\n%q\nwant\n%q", got, want)
	}
}

func TestBoxRenderASCIIBorder(t *testing.T) {
	got := NewBox().Border(ASCIIBorder()).Render("hi")
	want := "+--+\n|hi|\n+--+"
	if got != want {
		t.Errorf("Render() =\n%q\nwant\n%q", got, want)
	}
}

func TestBoxRenderBorderWithPadding(t *testing.T) {
	got := NewBox().Border(RoundedBorder()).PaddingAll(1).Render("x")
	want := "╭───╮\n│   │\n│ x │\n│   │\n╰───╯"
	if got != want {
		t.Errorf("Render() =\n%s\nwant\n%s", got, want)
	}
}

func TestBoxRenderFixedWidth(t *testing.T) {
	got := NewBox().Width(5).Border(NormalBorder()).Render("hi")
	lines := strings.Split(got, "\n")
	if ansi.Width(lines[0]) != 7 { // width(5) + 2 border chars
		t.Errorf("top border width = %d, want 7 for content width 5", ansi.Width(lines[0]))
	}
}

// TestBoxRenderIsANSIWidthAware is the key correctness property for
// combining layout with ansi.Style: a styled line's escape codes must not
// be counted toward its visible width, or borders around colored text
// would misalign.
func TestBoxRenderIsANSIWidthAware(t *testing.T) {
	plain := "hi"
	styled := ansi.NewStyle().Bold().Foreground(ansi.Red).Render("hi")

	gotPlain := NewBox().Border(NormalBorder()).Render(plain)
	gotStyled := NewBox().Border(NormalBorder()).Render(styled)

	plainLines := strings.Split(gotPlain, "\n")
	styledLines := strings.Split(gotStyled, "\n")

	if ansi.Width(plainLines[0]) != ansi.Width(styledLines[0]) {
		t.Errorf("border width differs between plain and styled content: %d vs %d",
			ansi.Width(plainLines[0]), ansi.Width(styledLines[0]))
	}
	if !strings.Contains(gotStyled, styled) {
		t.Errorf("Render() dropped the style codes from the content: %q", gotStyled)
	}
}

func TestJoinHorizontal(t *testing.T) {
	left := "a\nbb"
	right := "1"
	got := joinHorizontal(1, left, right)
	want := "a  1\nbb  "
	if got != want {
		t.Errorf("joinHorizontal() =\n%q\nwant\n%q", got, want)
	}
}

func TestJoinHorizontalEmpty(t *testing.T) {
	if got := joinHorizontal(1); got != "" {
		t.Errorf("joinHorizontal() with no blocks = %q, want empty", got)
	}
}

func TestJoinVerticalEmpty(t *testing.T) {
	if got := joinVertical(0); got != "" {
		t.Errorf("joinVertical with no blocks = %q, want empty", got)
	}
	if got := joinVertical(3); got != "" {
		t.Errorf("joinVertical(3) with no blocks = %q, want empty", got)
	}
}

func TestJoinVerticalPadsToWidestLine(t *testing.T) {
	got := joinVertical(0, "ab\nabcd", "x", "abcdef\ny")
	want := strings.Join([]string{
		"ab    ",
		"abcd  ",
		"x     ",
		"abcdef",
		"y     ",
	}, "\n")
	if got != want {
		t.Errorf("joinVertical =\n%q\nwant\n%q", got, want)
	}
}

func TestJoinVerticalLeavesWidestLinesUnchanged(t *testing.T) {
	if got := joinVertical(0, "abc", "abc"); got != "abc\nabc" {
		t.Errorf("got %q, want no padding when every line is already the widest", got)
	}
}

func TestJoinVerticalGap(t *testing.T) {
	got := joinVertical(2, "aaa", "b")
	if want := "aaa\n   \n   \nb  "; got != want {
		t.Errorf("gap 2 = %q, want %q", got, want)
	}
	for _, g := range []int{0, -1} {
		if got, want := joinVertical(g, "aaa", "b"), "aaa\nb  "; got != want {
			t.Errorf("gap %d = %q, want %q (no blank lines)", g, got, want)
		}
	}
	// No gap after the last block, none before the first, none for one block.
	if got := joinVertical(3, "a"); got != "a" {
		t.Errorf("single block with gap = %q, want %q", got, "a")
	}
}

func TestJoinVerticalEmptyBlockIsOneBlankLine(t *testing.T) {
	if got, want := joinVertical(0, "abc", "", "de"), "abc\n   \nde "; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got, want := joinVertical(0, ""), ""; got != want {
		t.Errorf("single empty block = %q, want %q", got, want)
	}
}

func TestJoinVerticalSingleBlockPadsToOwnWidth(t *testing.T) {
	if got, want := joinVertical(0, "a\nabc"), "a  \nabc"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestJoinVerticalIsANSIAndWideAware(t *testing.T) {
	styled := ansi.NewStyle().Bold().Render("hi")
	wide := "你好" // 4 columns, 2 runes
	got := joinVertical(1, styled, wide, "abcdef")
	lines := strings.Split(got, "\n")
	if len(lines) != 5 {
		t.Fatalf("got %d lines, want 5: %q", len(lines), got)
	}
	for i, l := range lines {
		if w := ansi.Width(l); w != 6 {
			t.Errorf("line %d Width = %d, want 6: %q", i, w, l)
		}
	}
	// Content bytes untouched; only unstyled spaces appended.
	if want := styled + "    "; lines[0] != want {
		t.Errorf("styled line = %q, want %q", lines[0], want)
	}
	if want := wide + "  "; lines[2] != want {
		t.Errorf("wide line = %q, want %q", lines[2], want)
	}
}

func TestBoxBorderColorPaintsEveryBorderSegment(t *testing.T) {
	red := ansi.NewStyle().Foreground(ansi.Red)
	got := NewBox().Border(NormalBorder()).Padding(0, 1, 0, 1).BorderColor(ansi.Red).Render("ab\ncd")
	lines := strings.Split(got, "\n")
	want := []string{
		red.Render("┌────┐"),
		red.Render("│") + " ab " + red.Render("│"),
		red.Render("│") + " cd " + red.Render("│"),
		red.Render("└────┘"),
	}
	if len(lines) != len(want) {
		t.Fatalf("got %d lines, want %d: %q", len(lines), len(want), got)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, lines[i], want[i])
		}
	}
	// Every line is self-contained: it ends with no style left open.
	for i, l := range lines {
		if !strings.HasSuffix(l, ansi.Reset) {
			t.Errorf("line %d does not end with Reset: %q", i, l)
		}
	}
}

func TestBoxBorderColorIsOptIn(t *testing.T) {
	plain := NewBox().Border(RoundedBorder()).PaddingAll(1).Width(6).Render("hi\nthere")
	if got := NewBox().Border(RoundedBorder()).PaddingAll(1).Width(6).BorderColor(nil).Render("hi\nthere"); got != plain {
		t.Errorf("nil colour changed the output:\n got  %q\n want %q", got, plain)
	}
	if strings.Contains(plain, ansi.CSI) {
		t.Errorf("a box without BorderColor emitted escape codes: %q", plain)
	}
}

func TestBoxBorderColorWithoutBorderHasNoEffect(t *testing.T) {
	plain := NewBox().PaddingAll(1).Render("hi")
	if got := NewBox().PaddingAll(1).BorderColor(ansi.Red).Render("hi"); got != plain {
		t.Errorf("colour applied to a borderless box: %q vs %q", got, plain)
	}
}

func TestBoxBorderColorKeepsDimensionsAndText(t *testing.T) {
	for _, b := range []Border{NormalBorder(), RoundedBorder(), DoubleBorder(), ThickBorder(), ASCIIBorder()} {
		base := NewBox().Border(b).Padding(1, 2, 1, 2).Width(9)
		plain := strings.Split(base.Render("wide 你好\nx"), "\n")
		coloured := strings.Split(base.BorderColor(ansi.Green).Render("wide 你好\nx"), "\n")
		if len(plain) != len(coloured) {
			t.Fatalf("line count changed: %d vs %d", len(plain), len(coloured))
		}
		for i := range plain {
			if ansi.Width(plain[i]) != ansi.Width(coloured[i]) {
				t.Errorf("line %d width %d vs %d", i, ansi.Width(plain[i]), ansi.Width(coloured[i]))
			}
			if ansi.StripANSI(coloured[i]) != plain[i] {
				t.Errorf("line %d visible text changed: %q vs %q", i, ansi.StripANSI(coloured[i]), plain[i])
			}
		}
	}
}

func TestBoxBorderColorIsImmutableBuilder(t *testing.T) {
	base := NewBox().Border(NormalBorder())
	_ = base.BorderColor(ansi.Red)
	if strings.Contains(base.Render("x"), ansi.CSI) {
		t.Error("BorderColor mutated the receiver")
	}
}

// TestJoinHorizontalAlign covers criteria #309-#311: top (AlignStart) pads
// blank rows below a block's content, center vertically centers it, and
// bottom (AlignEnd) pads blank rows above it, within the tallest block's row
// height.
func TestJoinHorizontalAlign(t *testing.T) {
	tests := []struct {
		name  string
		align Align
		want  string
	}{
		{
			// "a" is one row; the tallest block ("bb\ncc\ndd") is three.
			name:  "top pads blank rows below content (AlignStart)",
			align: AlignStart,
			want:  "a bb\n  cc\n  dd",
		},
		{
			// missing=2, offset = 2/2 = 1: one blank row above, one below.
			name:  "center vertically centers rows",
			align: AlignCenter,
			want:  "  bb\na cc\n  dd",
		},
		{
			name:  "bottom pads blank rows above content (AlignEnd)",
			align: AlignEnd,
			want:  "  bb\n  cc\na dd",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := joinHorizontalAlign(1, tt.align, "a", "bb\ncc\ndd")
			if got != tt.want {
				t.Errorf("joinHorizontalAlign(1, %v, ...) =\n%q\nwant\n%q", tt.align, got, tt.want)
			}
		})
	}
}

// TestJoinHorizontalAlignOddPaddingGoesAfter checks that when the missing
// row count is odd, center places the extra blank row after the content
// (matching AlignStart's placement of the single extra row).
func TestJoinHorizontalAlignOddPaddingGoesAfter(t *testing.T) {
	// tallest block has 4 rows, "a" has 1: missing=3, offset=3/2=1 above, 2 below.
	got := joinHorizontalAlign(1, AlignCenter, "a", "w\nx\ny\nz")
	want := "  w\na x\n  y\n  z"
	if got != want {
		t.Errorf("got\n%q\nwant\n%q", got, want)
	}
}

func TestJoinHorizontalAlignMatchesJoinHorizontal(t *testing.T) {
	left, right := "a\nbb", "1"
	if got, want := joinHorizontalAlign(1, AlignStart, left, right), joinHorizontal(1, left, right); got != want {
		t.Errorf("joinHorizontalAlign(AlignStart) = %q, want joinHorizontal() = %q", got, want)
	}
}

// TestJoinVerticalAlign covers criteria #312-#314: left (AlignStart)
// right-pads each line to the common width, center horizontally centers it,
// and right (AlignEnd) left-pads it.
func TestJoinVerticalAlign(t *testing.T) {
	tests := []struct {
		name  string
		align Align
		want  string
	}{
		{
			name:  "left right-pads to common width (AlignStart)",
			align: AlignStart,
			want:  "ab   \nabcde",
		},
		{
			// pad=3, left=1, right=2.
			name:  "center horizontally centers each line",
			align: AlignCenter,
			want:  " ab  \nabcde",
		},
		{
			name:  "right left-pads to common width (AlignEnd)",
			align: AlignEnd,
			want:  "   ab\nabcde",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := joinVerticalAlign(0, tt.align, "ab", "abcde")
			if got != tt.want {
				t.Errorf("joinVerticalAlign(0, %v, ...) =\n%q\nwant\n%q", tt.align, got, tt.want)
			}
		})
	}
}

func TestJoinVerticalAlignMatchesJoinVertical(t *testing.T) {
	if got, want := joinVerticalAlign(1, AlignStart, "ab", "abcde"), joinVertical(1, "ab", "abcde"); got != want {
		t.Errorf("joinVerticalAlign(AlignStart) = %q, want joinVertical() = %q", got, want)
	}
}

// TestExistingCallSitesUnchanged is a regression test for criterion #315:
// joinHorizontal and joinVertical keep their original signatures and
// behaviour, so existing call sites (dialog, toast, examples) build and
// behave identically without changes.
func TestExistingCallSitesUnchanged(t *testing.T) {
	if got, want := joinHorizontal(1, "a\nbb", "1"), "a  1\nbb  "; got != want {
		t.Errorf("joinHorizontal() = %q, want %q", got, want)
	}
	if got, want := joinVertical(0, "ab\nabcd", "x"), "ab  \nabcd\nx   "; got != want {
		t.Errorf("joinVertical() = %q, want %q", got, want)
	}
}

// #39: a fixed Width clips wider content, so every row is exactly as wide as
// the box was asked to be.
func TestBoxWidthClipsContent(t *testing.T) {
	out := NewBox().Border(NormalBorder()).Width(5).Render("hello world")
	lines := strings.Split(out, "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d lines: %q", len(lines), out)
	}
	for i, l := range lines {
		if w := ansi.Width(l); w != 7 {
			t.Errorf("line %d is %d columns, want 7: %q", i, w, l)
		}
	}
	if !strings.Contains(lines[1], "hello") || strings.Contains(lines[1], "world") {
		t.Errorf("content row %q, want 'hello' clipped from 'hello world'", lines[1])
	}
}

func TestBoxWidthClipsEveryLineStyledAndWide(t *testing.T) {
	content := "\x1b[31mabcdefghij\x1b[0m\n你好你好你好\nshort"
	for _, pad := range []int{0, 2} {
		out := NewBox().Border(NormalBorder()).PaddingAll(pad).Width(6).Render(content)
		for i, l := range strings.Split(out, "\n") {
			if w := ansi.Width(l); w != 6+2*pad+2 {
				t.Errorf("pad %d line %d is %d columns, want %d: %q", pad, i, w, 6+2*pad+2, l)
			}
		}
	}
}

func TestBoxWithoutWidthStillSizesToContent(t *testing.T) {
	out := NewBox().Border(NormalBorder()).Render("hello world")
	if w := ansi.Width(strings.Split(out, "\n")[1]); w != 13 {
		t.Fatalf("auto-width row is %d columns, want 13", w)
	}
}

func boxLines(s string) []string { return strings.Split(s, "\n") }

func TestBoxTitleCentredOnTopBorder(t *testing.T) {
	out := NewBox().Border(NormalBorder()).Width(20).Title("Logs", AlignCenter).Render("hi")
	lines := boxLines(out)
	top := lines[0]
	if ansi.Width(top) != 22 {
		t.Fatalf("top border is %d columns, want 22: %q", ansi.Width(top), top)
	}
	at := strings.Index(top, "Logs")
	if at < 0 {
		t.Fatalf("top border %q has no title", top)
	}
	if lead, trail := strings.Count(top[:at], "─"), strings.Count(top[at+4:], "─"); lead != trail {
		t.Fatalf("title off-centre: %d dashes before, %d after in %q", lead, trail, top)
	}
	for _, l := range lines {
		if ansi.Width(l) != 22 {
			t.Fatalf("row %q is %d columns, want 22", l, ansi.Width(l))
		}
	}
}

func TestBoxTitleAlignAndClip(t *testing.T) {
	start := boxLines(NewBox().Border(NormalBorder()).Width(20).Title("Logs", AlignStart).Render("x"))[0]
	end := boxLines(NewBox().Border(NormalBorder()).Width(20).Title("Logs", AlignEnd).Render("x"))[0]
	if !strings.HasPrefix(start, "┌─ Logs ") || !strings.HasSuffix(end, " Logs ─┐") {
		t.Fatalf("start %q end %q", start, end)
	}
	long := boxLines(NewBox().Border(NormalBorder()).Width(6).Title("a very long title", AlignStart).Render("x"))[0]
	if ansi.Width(long) != 8 {
		t.Fatalf("long title widened the border to %d: %q", ansi.Width(long), long)
	}
	auto := boxLines(NewBox().Border(NormalBorder()).Title("Settings", AlignStart).Render("x"))[0]
	if !strings.Contains(auto, "Settings") {
		t.Fatalf("auto-sized box clipped its title: %q", auto)
	}
	if got := NewBox().Title("T", AlignStart).Render("x"); got != "x" {
		t.Fatalf("borderless box drew a title: %q", got)
	}
}

func TestBoxHeightPadsAndCuts(t *testing.T) {
	short := boxLines(NewBox().Border(NormalBorder()).Width(3).Height(4).Render("a"))
	if len(short) != 6 {
		t.Fatalf("got %d rows, want 6 (4 + border)", len(short))
	}
	cut := boxLines(NewBox().Height(2).Render("a\nb\nc\nd"))
	if len(cut) != 2 || cut[1] != "b" {
		t.Fatalf("cut = %q, want 2 rows a,b", cut)
	}
}

func TestBoxMargin(t *testing.T) {
	out := boxLines(NewBox().Border(NormalBorder()).Width(3).Margin(1, 2, 1, 4).Render("abc"))
	if len(out) != 5 {
		t.Fatalf("got %d rows, want 5", len(out))
	}
	for _, l := range out {
		if ansi.Width(l) != 3+2+4+2 {
			t.Fatalf("row %q is %d columns, want 11", l, ansi.Width(l))
		}
	}
	if strings.TrimSpace(out[0]) != "" || !strings.HasPrefix(out[1], "    ┌") || !strings.HasSuffix(out[2], "│  ") {
		t.Fatalf("margin misplaced: %q", out)
	}
}

func TestBoxBorderSides(t *testing.T) {
	out := boxLines(NewBox().Border(NormalBorder()).Width(3).BorderSides(true, false, true, false).Render("abc"))
	if len(out) != 3 || out[0] != "───" || out[2] != "───" || out[1] != "abc" {
		t.Fatalf("got %q", out)
	}
	got := boxLines(NewBox().Border(NormalBorder()).Width(2).BorderSides(false, false, true, true).Render("ab"))
	if len(got) != 2 || got[0] != "│ab" || got[1] != "└──" {
		t.Fatalf("got %q", got)
	}
}

func TestBoxBackgroundFillsInterior(t *testing.T) {
	c := ansi.RGB{R: 10, G: 20, B: 30}
	plain := NewBox().Border(NormalBorder()).Width(6).Padding(0, 1, 0, 1)
	out := plain.Background(c).Render("hi")
	if ansi.StripANSI(out) != plain.Render("hi") {
		t.Fatalf("background changed the layout: %q", out)
	}
	if !strings.Contains(out, "\x1b[48") {
		t.Fatalf("no background escape in %q", out)
	}
	styled := NewBox().Width(6).Background(c).Render(ansi.NewStyle().Bold().Render("hi") + "x")
	open, _, _ := strings.Cut(ansi.NewStyle().Background(c).Render("X"), "X")
	if !strings.Contains(styled, "\x1b[0m"+open+"x") {
		t.Fatalf("background not restored after reset: %q", styled)
	}
}

package fx

import (
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/cellbuf"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/motion"
)

// picture is a buffer holding the rows, white on blue.
func picture(t *testing.T, rows ...string) *cellbuf.Buffer {
	t.Helper()
	b := cellbuf.New(len(rows[0]), len(rows))
	st := b.StyleID(cellbuf.Style{FG: cellbuf.RGB(200, 200, 200), BG: cellbuf.RGB(0, 0, 100)})
	for y, r := range rows {
		b.SetString(0, y, r, st)
	}
	return b
}

func text(b *cellbuf.Buffer) string { return ansi.StripANSI(b.String()) }

func TestProgress(t *testing.T) {
	ms := time.Millisecond
	for _, c := range []struct {
		elapsed, d time.Duration
		ease       func(float64) float64
		pref       motion.Preference
		want       float64
	}{
		{0, 100 * ms, nil, motion.Normal, 0},
		{-5 * ms, 100 * ms, nil, motion.Normal, 0},
		{25 * ms, 100 * ms, nil, motion.Normal, 0.25},
		{100 * ms, 100 * ms, nil, motion.Normal, 1},
		{500 * ms, 100 * ms, nil, motion.Normal, 1},
		{10 * ms, 0, nil, motion.Normal, 1},
		{25 * ms, 100 * ms, nil, motion.Reduced, 1},
		{50 * ms, 100 * ms, func(t float64) float64 { return t * t }, motion.Normal, 0.25},
		{50 * ms, 100 * ms, func(t float64) float64 { return t * 10 }, motion.Normal, 1},
	} {
		if got := Progress(c.elapsed, c.d, c.ease, c.pref); got != c.want {
			t.Errorf("Progress(%v, %v, %v) = %v; want %v", c.elapsed, c.d, c.pref, got, c.want)
		}
	}
}

func TestEveryEffectLeavesTheFinishedPictureAlone(t *testing.T) {
	for name, e := range map[string]Effect{
		"fade": Fade{}, "dissolve": Dissolve{Seed: 7}, "slide": Slide{From: FromRight},
		"sweep": Sweep{Soft: 3}, "out": Out(Fade{}),
	} {
		b := picture(t, "hello wor", "ld and mo")
		want := b.String()
		p := 1.0
		if name == "out" {
			p = 0
		}
		e.Apply(b, b.Bounds(), p)
		if b.String() != want {
			t.Errorf("%s changed the picture at its end", name)
		}
	}
}

func TestFadeBlendsTowardTheBackground(t *testing.T) {
	b := picture(t, "ab")
	Fade{To: ansi.RGB{R: 0, G: 0, B: 100}}.Apply(b, b.Bounds(), 0)
	st := b.Style(b.At(0, 0).Style)
	if st.FG != cellbuf.RGB(0, 0, 100) || st.BG != cellbuf.RGB(0, 0, 100) {
		t.Fatalf("at p=0 the style is %+v; want the text the colour of the background", st)
	}
	if text(b) != "ab" {
		t.Fatalf("text %q", text(b))
	}
	b = picture(t, "ab")
	Fade{To: ansi.RGB{}}.Apply(b, b.Bounds(), 0.5)
	if st := b.Style(b.At(1, 0).Style); st.FG != cellbuf.RGB(100, 100, 100) || st.BG != cellbuf.RGB(0, 0, 50) {
		t.Fatalf("half way the style is %+v", st)
	}
}

func TestFadeReadsBasicAndDefaultColours(t *testing.T) {
	b := cellbuf.New(2, 1)
	b.SetString(0, 0, "x", b.StyleID(cellbuf.Style{FG: cellbuf.Basic(1)}))
	b.SetString(1, 0, "y", 0)
	Fade{}.Apply(b, b.Bounds(), 0.5)
	if got := b.Style(b.At(0, 0).Style).FG; got != cellbuf.RGB(103, 0, 0) {
		t.Errorf("a basic red faded to %v", got)
	}
	if got := b.Style(b.At(1, 0).Style).FG; got != cellbuf.RGB(115, 115, 115) {
		t.Errorf("a default colour faded to %v", got)
	}
}

func TestDissolveIsMonotoneAndDeterministic(t *testing.T) {
	row := strings.Repeat("x", 40)
	shown := func(seed uint64, p float64) map[int]bool {
		b := picture(t, row, row)
		Dissolve{Seed: seed}.Apply(b, b.Bounds(), p)
		got := map[int]bool{}
		for y := 0; y < 2; y++ {
			for x := 0; x < 40; x++ {
				if b.At(x, y).Cluster == "x" {
					got[y*40+x] = true
				}
			}
		}
		return got
	}
	lo, hi := shown(1, 0.3), shown(1, 0.7)
	for k := range lo {
		if !hi[k] {
			t.Fatalf("cell %d showed at 0.3 and was hidden at 0.7", k)
		}
	}
	if n := len(lo); n < 10 || n > 40 {
		t.Errorf("%d of 80 cells shown at 0.3", n)
	}
	if n := len(hi); n < 40 || n > 70 {
		t.Errorf("%d of 80 cells shown at 0.7", n)
	}
	if fmtSet(shown(2, 0.5)) == fmtSet(shown(1, 0.5)) {
		t.Error("two seeds dissolve the same cells")
	}
	if fmtSet(shown(1, 0.5)) != fmtSet(shown(1, 0.5)) {
		t.Error("the same seed dissolved differently twice")
	}
	if n := len(shown(1, 0)); n != 0 {
		t.Errorf("%d cells shown at 0", n)
	}
}

func fmtSet(m map[int]bool) string {
	var b strings.Builder
	for i := 0; i < 200; i++ {
		if m[i] {
			b.WriteByte('1')
		} else {
			b.WriteByte('0')
		}
	}
	return b.String()
}

func TestDissolveKeepsWideClustersWhole(t *testing.T) {
	for p := 0.0; p < 1; p += 0.1 {
		c := cellbuf.New(8, 1)
		c.SetString(0, 0, "日本語x", 0)
		Dissolve{Seed: 3}.Apply(c, c.Bounds(), p)
		if strings.Contains(text(c), "�") {
			t.Fatalf("p=%v split a wide rune: %q", p, text(c))
		}
		for x := 0; x < 6; x += 2 {
			h, cont := c.At(x, 0), c.At(x+1, 0)
			if (h.Width == 2) != (cont.Width == 0 && cont.Cluster == "") {
				t.Fatalf("p=%v left half a cluster at %d: %+v %+v", p, x, h, cont)
			}
		}
	}
}

func TestSlideMovesFromEachEdge(t *testing.T) {
	wide := []string{"abcdefgh", "ijklmnop"}
	tall := []string{"abcdefgh", "ijklmnop", "qrstuvwx", "yzABCDEF"}
	for _, c := range []struct {
		name string
		from Direction
		p    float64
		in   []string
		want string
	}{
		{"left", FromLeft, 0.5, wide, "efgh    \nmnop    "},
		{"right", FromRight, 0.5, wide, "    abcd\n    ijkl"},
		{"top", FromTop, 0.5, tall, "qrstuvwx\nyzABCDEF\n        \n        "},
		{"bottom", FromBottom, 0.5, tall, "        \n        \nabcdefgh\nijklmnop"},
		{"left at 0", FromLeft, 0, wide, "        \n        "},
		{"left a quarter in", FromLeft, 0.75, wide, "cdefgh  \nklmnop  "},
	} {
		b := picture(t, c.in...)
		Slide{From: c.from}.Apply(b, b.Bounds(), c.p)
		if got := text(b); got != c.want {
			t.Errorf("%s: got\n%s\nwant\n%s", c.name, got, c.want)
		}
	}
}

func TestSlideStaysInsideItsRegion(t *testing.T) {
	b := picture(t, "aaaaaaaa", "bbbbbbbb", "cccccccc")
	Slide{From: FromLeft}.Apply(b, cellbuf.Rect{X: 2, Y: 1, W: 4, H: 1}, 0)
	want := "aaaaaaaa\nbb    bb\ncccccccc"
	if got := text(b); got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
}

func TestSweepRevealsAndFeathers(t *testing.T) {
	b := picture(t, "abcdefgh")
	Sweep{}.Apply(b, b.Bounds(), 0.5)
	if got := text(b); got != "abcd    " {
		t.Fatalf("hard edge: %q", got)
	}
	b = picture(t, "abcdefgh")
	Sweep{}.Apply(b, b.Bounds(), 0)
	if got := text(b); strings.TrimSpace(got) != "" {
		t.Fatalf("at 0 something shows: %q", got)
	}
	b = picture(t, "abcdefgh")
	Sweep{Soft: 4}.Apply(b, b.Bounds(), 0.5) // the edge is at 6: a-b whole, c-f fading, g-h gone
	if got := text(b); got != "abcdef  " {
		t.Fatalf("soft edge: %q", got)
	}
	whole := b.Style(b.At(0, 0).Style)
	soft := b.Style(b.At(4, 0).Style)
	if whole.FG != cellbuf.RGB(200, 200, 200) || soft.FG == whole.FG {
		t.Fatalf("the gradient did not fade: %+v then %+v", whole, soft)
	}
	for _, from := range []Direction{FromRight, FromTop, FromBottom} {
		b = picture(t, "abcd", "efgh", "ijkl", "mnop")
		Sweep{From: from}.Apply(b, b.Bounds(), 0.5)
		got := strings.Split(text(b), "\n")
		shown := 0
		for _, l := range got {
			shown += len(strings.TrimSpace(l))
		}
		if shown != 8 {
			t.Errorf("from %v half the cells should show, %d do:\n%s", from, shown, text(b))
		}
	}
}

func TestOutRunsBackwards(t *testing.T) {
	a, b := picture(t, "abcdefgh"), picture(t, "abcdefgh")
	Out(Sweep{}).Apply(a, a.Bounds(), 0.25)
	Sweep{}.Apply(b, b.Bounds(), 0.75)
	if a.String() != b.String() {
		t.Fatalf("%q != %q", a.String(), b.String())
	}
}

func TestEffectsClipToTheBufferAndHandleEmptyRegions(t *testing.T) {
	for _, e := range []Effect{Fade{}, Dissolve{}, Slide{}, Sweep{Soft: 2}} {
		b := picture(t, "abcd", "efgh")
		before := b.String()
		e.Apply(b, cellbuf.Rect{X: 10, Y: 10, W: 5, H: 5}, 0)
		e.Apply(b, cellbuf.Rect{}, 0)
		e.Apply(b, cellbuf.Rect{X: -5, Y: -5, W: 2, H: 2}, 0)
		if b.String() != before {
			t.Errorf("%T changed cells outside its region", e)
		}
		e.Apply(b, cellbuf.Rect{X: -2, Y: -2, W: 100, H: 100}, 0.5) // must not panic
	}
}

func TestWrapDrawsTheNodeThroughTheEffect(t *testing.T) {
	node := layout.Block("abcdefgh\nijklmnop")
	s := layout.Size{W: 8, H: 2}
	if got := fxRender(node, Sweep{}, 1, s); got != "abcdefgh\nijklmnop" {
		t.Fatalf("p=1: %q", got)
	}
	if got := fxRender(node, Sweep{}, 0.5, s); got != "abcd    \nijkl    " {
		t.Fatalf("p=0.5: %q", got)
	}
	if got := fxRender(node, Sweep{}, 0, s); strings.TrimSpace(got) != "" {
		t.Fatalf("p=0: %q", got)
	}
	// Drawing into a cell grid gives the same screen as Render.
	b := cellbuf.New(8, 2)
	Wrap(node, Sweep{}, 0.5).(layout.CellNode).DrawCells(cellbuf.Layout(b), layout.Rect{W: 8, H: 2})
	if got := text(b); got != "abcd    \nijkl    " {
		t.Fatalf("DrawCells: %q", got)
	}
	if got := Wrap(node, Fade{}, 0.5).Measure(layout.Unconstrained()); got.W != 8 || got.H != 2 {
		t.Fatalf("Measure = %v", got)
	}
	if Wrap(node, Fade{}, 0.5).Render(layout.Size{}) != "" {
		t.Error("an empty size should render nothing")
	}
}

func fxRender(n layout.Node, e Effect, p float64, s layout.Size) string {
	return ansi.StripANSI(Wrap(n, e, p).Render(s))
}

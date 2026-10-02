package faces

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
)

func TestCatalogHasFiftyUniqueFaces(t *testing.T) {
	if Count() != 50 {
		t.Fatalf("Count() = %d, want 50", Count())
	}
	seen := map[string]bool{}
	for _, f := range All() {
		key := strings.ToLower(f.Name)
		if f.Name == "" || seen[key] {
			t.Errorf("face name %q is empty or duplicated", f.Name)
		}
		seen[key] = true
	}
}

var bothSizes = []Size{Small, Large}

func TestEveryFaceLoopsFourEightOrTwelveFrames(t *testing.T) {
	for _, f := range All() {
		switch n := len(f.Frames(Small)); n {
		case 4, 8, 12:
		default:
			t.Errorf("%s has %d frames, want 4, 8 or 12", f.Name, n)
		}
		if len(f.Frames(Small)) != len(f.Frames(Large)) {
			t.Errorf("%s has %d small but %d large frames", f.Name, len(f.Frames(Small)), len(f.Frames(Large)))
		}
		if f.Interval <= 0 {
			t.Errorf("%s has no interval", f.Name)
		}
	}
}

func TestHeadsAreSixByThreeAndTenByFiveCells(t *testing.T) {
	for sz, want := range map[Size][2]int{Small: {6, 3}, Large: {10, 5}} {
		if w, h := sz.HeadCells(); w != want[0] || h != want[1] {
			t.Errorf("size %d head = %dx%d cells, want %dx%d", sz, w, h, want[0], want[1])
		}
		g := sz.geom()
		if g.head != want[0]*2 || g.head != want[1]*4 {
			t.Errorf("size %d head is %d dots, want %dx%d", sz, g.head, want[0]*2, want[1]*4)
		}
		// The head sits on whole cells, so its edges are crisp Braille.
		if g.ox%2 != 0 || g.oy%4 != 0 || g.w%2 != 0 || g.h%4 != 0 {
			t.Errorf("size %d geometry is not cell-aligned: %+v", sz, g)
		}
	}
}

// Every frame of every face is the same block of cells at a size, so
// swapping faces or playing a shaking one never moves the layout.
func TestEveryFrameIsTheSameSize(t *testing.T) {
	for _, sz := range bothSizes {
		w, h := sz.Cells()
		for _, f := range All() {
			for i, fr := range f.Frames(sz) {
				rows := strings.Split(fr, "\n")
				if len(rows) != h {
					t.Errorf("%s size %d frame %d has %d rows, want %d", f.Name, sz, i, len(rows), h)
					continue
				}
				for r, row := range rows {
					if got := ansi.Width(row); got != w {
						t.Errorf("%s size %d frame %d row %d is %d columns, want %d: %q", f.Name, sz, i, r, got, w, row)
					}
				}
			}
		}
	}
}

// Sprites are Braille dots (U+2801..U+28FF) and plain spaces, all one column wide.
func TestFramesAreOnlyBrailleAndSpaces(t *testing.T) {
	for _, sz := range bothSizes {
		for _, f := range All() {
			for i, fr := range f.Frames(sz) {
				for _, r := range strings.ReplaceAll(fr, "\n", "") {
					if r != ' ' && (r < 0x2801 || r > 0x28ff) {
						t.Errorf("%s size %d frame %d uses %q (U+%04X)", f.Name, sz, i, r, r)
					}
					if ansi.Width(string(r)) != 1 {
						t.Errorf("%s size %d frame %d: %q is not one column wide", f.Name, sz, i, r)
					}
				}
			}
		}
	}
}

// The head is a solid silhouette with the face cut out, like a filled
// blob with holes, not an outline. Pointed chins and round corners leave
// a few bounding-box cells empty, but an outline would fill under half.
func TestHeadsAreSolidWithFeaturesCutOut(t *testing.T) {
	for _, sz := range bothSizes {
		g := sz.geom()
		for _, f := range All() {
			rest := f.Frames(sz)[0]
			lit, total := 0, 0
			for r, row := range strings.Split(rest, "\n") {
				if r*4 < g.oy || r*4 >= g.oy+g.head {
					continue
				}
				cells := []rune(row)
				for c := g.ox / 2; c < (g.ox+g.head)/2; c++ {
					total++
					if cells[c] != ' ' {
						lit++
					}
				}
			}
			if lit*10 < total*8 {
				t.Errorf("%s size %d: only %d of %d head cells have dots; heads should be solid", f.Name, sz, lit, total)
			}
		}
	}
}

// Pip's small head is the reference look, taken from the design sample: a
// filled blob with one-dot-wide eyes and a two-dot mouth cut out.
func TestPipSmallMatchesTheReferenceLook(t *testing.T) {
	pip, _ := ByName("Pip")
	want := []string{
		"            ",
		"   ⣠⣾⣿⣿⣷⣄   ",
		"   ⣿⣧⣿⣿⣼⣿   ",
		"   ⠙⢿⣯⣽⡿⠋   ",
		"            ",
	}
	if got := strings.Split(pip.Frames(Small)[0], "\n"); !reflect.DeepEqual(got, want) {
		t.Errorf("Pip small frame 0:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestEveryFaceActuallyAnimates(t *testing.T) {
	for _, sz := range bothSizes {
		sigs := map[string]string{}
		for _, f := range All() {
			distinct := map[string]bool{}
			for _, fr := range f.Frames(sz) {
				distinct[fr] = true
			}
			if len(distinct) < 3 {
				t.Errorf("%s size %d has only %d distinct frames", f.Name, sz, len(distinct))
			}
			sig := strings.Join(f.Frames(sz), "|")
			if other, dup := sigs[sig]; dup {
				t.Errorf("%s and %s are the same animation at size %d", f.Name, other, sz)
			}
			sigs[sig] = f.Name
		}
	}
}

func TestGetWrapsAndByNameIgnoresCase(t *testing.T) {
	if Get(-1).Name != Get(Count()-1).Name || Get(Count()).Name != Get(0).Name {
		t.Error("Get should wrap in both directions")
	}
	f, ok := ByName("pIp")
	if !ok || f.Name != "Pip" {
		t.Errorf("ByName(pIp) = %q, %v", f.Name, ok)
	}
	if _, ok := ByName("nobody"); ok {
		t.Error("ByName should miss an unknown name")
	}
}

func TestAllReturnsACopy(t *testing.T) {
	a := All()
	a[0].Name = "changed"
	if Get(0).Name == "changed" {
		t.Error("mutating All()'s slice changed the catalog")
	}
}

func TestRenderClipsEffectsAtTheCanvasEdge(t *testing.T) {
	p := pose{eye: pair(eyeOpen), mouth: mSmile, dx: -9, dy: 9,
		fx: []fx{{fxHeart, -99, -99}, {fxZBig, 99, 99}, {fxSpark, 3, -80}}}
	for _, sz := range bothSizes {
		w, h := sz.Cells()
		out := headRound.render(sz, look{}, p)
		if rows := strings.Split(out, "\n"); len(rows) != h || ansi.Width(rows[0]) != w {
			t.Errorf("size %d clipped render is %d rows / %d cols, want %dx%d", sz, len(rows), ansi.Width(rows[0]), h, w)
		}
	}
}

// --- Model ---

func TestNewIsStoppedOnFirstFrame(t *testing.T) {
	m := New()
	if m.Running() || m.Frame() != 0 || m.Index() != 0 {
		t.Errorf("New() = running %v frame %d index %d, want stopped at 0/0", m.Running(), m.Frame(), m.Index())
	}
	if got := ansi.StripANSI(m.View()); got != Get(0).Frames(Small)[0] {
		t.Errorf("View before Start should be frame 0, got:\n%s", got)
	}
}

func TestStartTickAdvancesWrapsAndReschedules(t *testing.T) {
	m := New()
	cmd := m.Start()
	if !m.Running() || cmd == nil {
		t.Fatal("Start should run and return a Cmd")
	}
	if _, ok := tui.RunCmd(context.Background(), cmd).(tickMsg); !ok {
		t.Fatalf("Start's Cmd produced %T, want tickMsg", tui.RunCmd(context.Background(), cmd))
	}
	n := len(m.frames())
	for i := 1; i <= n; i++ {
		var next tui.Cmd
		m, next = m.Update(tickMsg{gen: m.gen})
		if next == nil {
			t.Fatal("a running tick should reschedule")
		}
		if want := i % n; m.Frame() != want {
			t.Errorf("after %d ticks frame = %d, want %d", i, m.Frame(), want)
		}
	}
}

func TestStoppedOrForeignMsgsDoNothing(t *testing.T) {
	m := New()
	if next, cmd := m.Update(tickMsg{}); next.Frame() != 0 || cmd != nil {
		t.Error("a tick while stopped should be ignored")
	}
	m.Start()
	if next, cmd := m.Update(tui.Key{}); next.Frame() != 0 || cmd != nil {
		t.Error("a foreign Msg should be ignored")
	}
	m.Stop()
	if _, cmd := m.Update(tickMsg{gen: m.gen}); cmd != nil {
		t.Error("a tick after Stop should not reschedule")
	}
}

// A tick from before a Stop/Start pair must not start a second chain, or
// the animation would play at double speed.
func TestStaleTickAfterRestartIsIgnored(t *testing.T) {
	m := New()
	stale := tui.RunCmd(context.Background(), m.Start()).(tickMsg) // the tick the first run scheduled
	m.Stop()
	m.Start()
	if next, cmd := m.Update(stale); next.Frame() != 0 || cmd != nil {
		t.Errorf("a tick from an earlier run advanced the frame to %d / rescheduled", next.Frame())
	}
}

func TestSetNextPrevWrapAndRestartTheLoop(t *testing.T) {
	m := New()
	m.Start()
	m, _ = m.Update(tickMsg{})
	m.Next()
	if m.Index() != 1 || m.Frame() != 0 {
		t.Errorf("Next: index %d frame %d, want 1/0", m.Index(), m.Frame())
	}
	m.Prev()
	m.Prev()
	if m.Index() != Count()-1 {
		t.Errorf("Prev past the start = %d, want %d", m.Index(), Count()-1)
	}
	m.Set(Count() + 2)
	if m.Index() != 2 {
		t.Errorf("Set should wrap, got %d", m.Index())
	}
}

func TestIntervalDefaultsToTheFacesOwnPace(t *testing.T) {
	m := New()
	if m.delay() != m.Face().Interval {
		t.Errorf("delay = %v, want the face's %v", m.delay(), m.Face().Interval)
	}
	m.Interval = 5 * time.Millisecond
	if m.delay() != 5*time.Millisecond {
		t.Errorf("an explicit Interval should win, got %v", m.delay())
	}
}

func TestShowLabelNamesTheFaceAndFrame(t *testing.T) {
	m := New()
	m.ShowLabel = true
	got := ansi.StripANSI(m.View())
	f := m.Face()
	want := f.Name + " · " + f.Anim
	if !strings.Contains(got, want) || !strings.Contains(got, "1/") {
		t.Errorf("label missing %q and frame counter in:\n%s", want, got)
	}
	if _, h := m.Size.Cells(); strings.Count(got, "\n") != h {
		t.Errorf("labelled view should be %d rows plus the label", h)
	}
}

func TestPlayOnceRunsOneLoopThenRestsOnTheFirstFrame(t *testing.T) {
	m := New()
	m.Set(1)
	cmd := m.PlayOnce()
	if !m.Running() || cmd == nil {
		t.Fatal("PlayOnce should run and return a Cmd")
	}
	n := len(m.frames())
	for i := 1; i < n; i++ {
		var next tui.Cmd
		m, next = m.Update(tickMsg{gen: m.gen})
		if next == nil || m.Frame() != i {
			t.Fatalf("tick %d: frame %d, reschedule %v; want frame %d and true", i, m.Frame(), next != nil, i)
		}
	}
	m, next := m.Update(tickMsg{gen: m.gen})
	if m.Frame() != 0 || m.Running() || next != nil {
		t.Errorf("after the last frame: frame %d running %v reschedule %v; want 0, stopped, none", m.Frame(), m.Running(), next != nil)
	}
	if _, next := m.Update(tickMsg{gen: m.gen}); next != nil {
		t.Error("a finished single loop must not tick again")
	}
}

func TestPlayOnceMidLoopRestartsAndStartLoopsForever(t *testing.T) {
	m := New()
	m.PlayOnce()
	m, _ = m.Update(tickMsg{gen: m.gen})
	m, _ = m.Update(tickMsg{gen: m.gen})
	m.PlayOnce()
	if m.Frame() != 0 {
		t.Errorf("PlayOnce mid-loop should restart, frame = %d", m.Frame())
	}

	m.Start() // a looping Start cancels the one-shot behaviour
	for i := 0; i < len(m.frames())+2; i++ {
		var next tui.Cmd
		if m, next = m.Update(tickMsg{gen: m.gen}); next == nil {
			t.Fatalf("Start should keep looping past the first wrap (stopped at tick %d)", i+1)
		}
	}
}

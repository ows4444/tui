// Package carousel shows one slide of several at a time, with a row of dots
// under it that says which. Left and Right go to the slide before and after,
// Home and End to the first and last. With the mouse on, a click on a dot
// goes to its slide and a click on either half of the slide goes back or on.
//
// A slide is text the caller has already drawn, of any number of lines; its
// colours and attributes are kept and every other escape sequence is
// dropped.
//
// Stability: experimental. Its API may change in any minor release.
package carousel

import (
	"strconv"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/theme"
)

// Model is one carousel. The zero value has no slides; build one with New.
type Model struct {
	// ID is carried by ChangedMsg, to tell one carousel's change from
	// another's.
	ID string
	// Slides are the slides, each already drawn by the caller.
	Slides []string
	// Width and Height are the size of the slide area in cells; a slide
	// that is larger is cut, and one that is smaller is padded on the right
	// and below. Zero or less fits the widest and the tallest slide, so the
	// carousel keeps one size whichever slide shows. The width is never
	// less than the row under the slide needs, which is at most 7 cells
	// for up to nine slides.
	Width, Height int
	// Loop makes Right on the last slide go to the first, and Left on the
	// first go to the last. Off (the default) stops at the ends.
	Loop  bool
	Theme theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens
	// KeyMap holds the keys for each action. New fills it with DefaultKeyMap;
	// a Model built as a struct literal with a zero KeyMap behaves as if it
	// held DefaultKeyMap.
	KeyMap KeyMap

	// Mouse, when true, makes Update handle tui.MouseEvent: a left press on
	// a dot goes to its slide, and one on the left or right half of the
	// slide goes to the slide before or after. Off (the default) ignores
	// the mouse.
	Mouse bool
	// Bounds is the screen rectangle where the app draws the carousel.
	Bounds hittest.Rect

	focused bool
	index   int
}

// New returns a carousel of the given slides, showing the first.
func New(slides ...string) Model {
	return Model{Slides: slides, Theme: theme.DarkTheme(), KeyMap: DefaultKeyMap()}
}

// KeyMap names the keys of each action of a Model.
type KeyMap struct {
	Prev  keymap.Binding // go to the slide before
	Next  keymap.Binding // go to the slide after
	First keymap.Binding // go to the first slide
	Last  keymap.Binding // go to the last slide
}

// DefaultKeyMap returns Left and Right, and Home and End.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Prev:  keymap.NewBinding("previous", "left"),
		Next:  keymap.NewBinding("next", "right"),
		First: keymap.NewBinding("first", "home"),
		Last:  keymap.NewBinding("last", "end"),
	}
}

func (m Model) keys() KeyMap {
	km := m.KeyMap
	if len(km.Prev.Keys)+len(km.Next.Keys)+len(km.First.Keys)+len(km.Last.Keys) == 0 {
		return DefaultKeyMap()
	}
	return km
}

// Bindings returns the actions the carousel honours, with descriptions, for
// help text.
func (m Model) Bindings() []keymap.Binding {
	km := m.keys()
	return []keymap.Binding{km.Prev, km.Next, km.First, km.Last}
}

// ChangedMsg is delivered (via the Cmd Update returns) when another slide
// shows, by key or by pointer.
type ChangedMsg struct {
	// ID is the carousel's ID.
	ID string
	// Index is the position in Slides of the slide now showing.
	Index int
}

// Index returns the position in Slides of the slide showing, or -1 when
// there are none.
func (m Model) Index() int {
	return min(max(m.index, 0), len(m.Slides)-1)
}

// SetIndex shows slide i, held within the slides there are. It delivers no
// ChangedMsg.
func (m *Model) SetIndex(i int) {
	m.index = min(max(i, 0), max(len(m.Slides)-1, 0))
}

// Focus gives the carousel keyboard focus. It returns no Cmd; the result is
// there so a carousel can stand where any focusable widget does.
func (m *Model) Focus() tui.Cmd {
	m.focused = true
	return nil
}

// Blur takes keyboard focus away.
func (m *Model) Blur() { m.focused = false }

// Focused reports whether the carousel has keyboard focus.
func (m Model) Focused() bool { return m.focused }

// show goes to slide i and reports it, or does nothing when i is the slide
// already showing.
func (m Model) show(i int) (Model, tui.Cmd) {
	if i == m.Index() {
		return m, nil
	}
	m.index = i
	msg := ChangedMsg{ID: m.ID, Index: i}
	return m, func() tui.Msg { return msg }
}

// step goes d slides on, wrapping when Loop is set and stopping at the ends
// when it is not.
func (m Model) step(d int) (Model, tui.Cmd) {
	n := len(m.Slides)
	i := m.Index() + d
	if m.Loop {
		i = ((i % n) + n) % n
	}
	return m.show(min(max(i, 0), n-1))
}

// Update goes to another slide on Left, Right, Home and End when the
// carousel has focus. With Mouse on, a left press on a dot goes to its
// slide, and one on the slide goes back from its left half and on from its
// right.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	n := len(m.Slides)
	if n == 0 {
		return m, nil
	}
	if ev, ok := msg.(tui.MouseEvent); ok {
		if !m.Mouse || ev.Action != tui.MouseActionPress || ev.Button != tui.MouseButtonLeft || !m.Bounds.Contains(ev.X, ev.Y) {
			return m, nil
		}
		x, y := ev.X-m.Bounds.X, ev.Y-m.Bounds.Y
		w, h := m.size()
		switch {
		case x >= w || y > h:
		case y == h:
			if i := m.dotAt(x); i >= 0 {
				return m.show(i)
			}
		case x < w/2:
			return m.step(-1)
		default:
			return m.step(1)
		}
		return m, nil
	}
	if !m.focused {
		return m, nil
	}
	km := m.keys()
	switch {
	case keymap.Matches(msg, km.Prev):
		return m.step(-1)
	case keymap.Matches(msg, km.Next):
		return m.step(1)
	case keymap.Matches(msg, km.First):
		return m.show(0)
	case keymap.Matches(msg, km.Last):
		return m.show(n - 1)
	}
	return m, nil
}

// lines returns slide i as its rows, made safe to draw.
func (m Model) lines(i int) []string {
	return strings.Split(ansi.SanitizeKeepSGR(ansi.ExpandTabs(m.Slides[i])), "\n")
}

// size is the width and height of the slide area. The width is never less
// than the narrower form of the indicator needs.
func (m Model) size() (w, h int) {
	w, h = m.Width, m.Height
	if w <= 0 || h <= 0 {
		fw, fh := 0, 0
		for i := range m.Slides {
			ls := m.lines(i)
			fh = max(fh, len(ls))
			for _, l := range ls {
				fw = max(fw, ansi.Width(l))
			}
		}
		if w <= 0 {
			w = fw
		}
		if h <= 0 {
			h = fh
		}
	}
	return max(w, min(2*len(m.Slides)+1, ansi.Width(m.count())+2)), h
}

// count is the indicator for a carousel with more slides than its width has
// room to give a dot each: "3 / 12".
func (m Model) count() string {
	return strconv.Itoa(m.Index()+1) + " / " + strconv.Itoa(len(m.Slides))
}

// dots reports whether the indicator is a row of dots: one per slide with a
// blank between, when that and its two end cells fit in w cells.
func (m Model) dots(w int) bool { return 2*len(m.Slides)+1 <= w }

// indicator returns the text between the indicator's two end cells and the
// column it starts at, centred under a slide area w cells wide.
func (m Model) indicator(w int) (text string, left int) {
	if m.dots(w) {
		g := m.themed().GlyphSet()
		marks := make([]string, len(m.Slides))
		for i := range marks {
			marks[i] = g.DotEmpty
			if i == m.Index() {
				marks[i] = g.Dot
			}
		}
		text = strings.Join(marks, " ")
	} else {
		text = m.count()
	}
	return text, (w - ansi.Width(text) - 2) / 2
}

// dotAt returns the slide whose dot is drawn at local column x of the
// indicator row, or -1.
func (m Model) dotAt(x int) int {
	w, _ := m.size()
	if !m.dots(w) {
		return -1
	}
	_, left := m.indicator(w)
	// The dots stand at every other column after the opening end cell.
	if k := x - left - 1; k >= 0 && k%2 == 0 && k/2 < len(m.Slides) {
		return k / 2
	}
	return -1
}

// Size is the number of cells and of rows View takes: the slide area and the
// indicator row under it. Both are 0 with no slides.
func (m Model) Size() (w, h int) {
	if len(m.Slides) == 0 {
		return 0, 0
	}
	w, h = m.size()
	return w, h + 1
}

// View renders the slide showing, cut or padded to the slide area, and under
// it a centred row with a dot for each slide, the one showing filled:
// " ○ ● ○ ". A carousel that has focus puts that row in angle brackets,
// "<○ ● ○>", so focus reads without colour and does not change the width.
// With more slides than there is room for dots the row reads " 3 / 12 ". A
// carousel with no slides renders "".
func (m Model) View() string {
	if len(m.Slides) == 0 {
		return ""
	}
	t := m.themed()
	w, h := m.size()
	ls := m.lines(m.Index())
	var b strings.Builder
	for row := range h {
		l := ""
		if row < len(ls) {
			l = ansi.Truncate(ls[row], w)
		}
		b.WriteString(l + strings.Repeat(" ", w-ansi.Width(l)) + "\n")
	}
	text, left := m.indicator(w)
	open, shut, style := " ", " ", ansi.NewStyle().Foreground(t.Primary)
	if m.focused {
		open, shut, style = "<", ">", t.ResolvedStates().Focus.Bold()
	}
	b.WriteString(strings.Repeat(" ", left) + style.Render(open+text+shut))
	b.WriteString(strings.Repeat(" ", w-left-ansi.Width(text)-2))
	return b.String()
}

// Compile-time proof that Model satisfies tui.Component[Model] — see
// tui.Component's doc comment for what this contract means and why.
var _ tui.Component[Model] = Model{}

// Linearize renders the carousel as plain text for accessible output (see
// tui.Linearizer): "Slide 2 of 3" and then the slide's own text without its
// styling. A carousel with no slides reads "No slides".
func (m Model) Linearize() string {
	if len(m.Slides) == 0 {
		return "No slides"
	}
	head := "Slide " + strconv.Itoa(m.Index()+1) + " of " + strconv.Itoa(len(m.Slides))
	return head + "\n" + ansi.StripANSI(strings.Join(m.lines(m.Index()), "\n"))
}

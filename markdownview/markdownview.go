// Package markdownview is a Markdown document you read in a window: it scrolls,
// knows its headings and links, jumps between them, follows in-page links
// ("#section") with a back and forward history, and reports other links to the
// program with a LinkMsg. It is markdown.RenderWith inside a viewport.
//
// Tab and Shift+Tab move through the links (the active one is named on a status
// row under the text and scrolled into view), Enter follows it, "]" and "[" jump
// to the next and previous heading, Backspace or Alt+Left goes back and
// Alt+Right forward. Headings() lists the table of contents for a side panel,
// and GoTo jumps to a heading by its anchor.
//
// Stability: experimental. Its API may change in any minor release.
package markdownview

import (
	"regexp"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/markdown"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/viewport"
)

// Heading is one heading of the document.
type Heading struct {
	// Level is 1 for "#" to 6 for "######".
	Level int
	// Text is the heading as plain text.
	Text string
	// Anchor is the name an in-page link uses, as GitHub makes it: lower case,
	// spaces as hyphens, punctuation dropped; a repeated heading gets "-1", "-2".
	Anchor string
	// Line is the row of the rendered document the heading is drawn on.
	Line int
}

// Link is one link of the document.
type Link struct {
	Text, URL string
	// Line is the row of the rendered document the link text is drawn on.
	Line int
}

// LinkMsg is delivered (via the Cmd Update returns) when Enter follows a link
// that does not point inside the document.
type LinkMsg struct{ Text, URL string }

// Model is a Markdown reader. Build one with New.
type Model struct {
	Theme theme.Theme
	// Options are passed to markdown.RenderWith, for a Lexer that colours code.
	Options markdown.Options
	// KeyMap holds the keys of the reader's own actions. New fills it with
	// DefaultKeyMap. Scrolling uses the viewport's keys.
	KeyMap KeyMap
	// Mouse, when true, makes Update pass mouse events to the viewport: the
	// wheel scrolls, and a press and drag inside Bounds selects text (y copies
	// it). Off (the default) ignores the mouse.
	Mouse bool
	// Bounds is the screen rectangle where the app draws the reader; the app
	// sets it each frame it moves.
	Bounds hittest.Rect

	src      string
	vp       viewport.Model
	headings []Heading
	links    []Link
	active   int // index into links, or -1
	back     []int
	forward  []int
}

// KeyMap names the keys of the reader's actions.
type KeyMap struct {
	NextLink    keymap.Binding // move to the next link
	PrevLink    keymap.Binding // move to the previous link
	Follow      keymap.Binding // follow the active link
	NextHeading keymap.Binding // scroll to the next heading
	PrevHeading keymap.Binding // scroll to the previous heading
	Back        keymap.Binding // go back to where the last jump left from
	Forward     keymap.Binding // undo a Back
}

// DefaultKeyMap returns Tab, Shift+Tab, Enter, "]", "[", Backspace and Alt+Left,
// and Alt+Right.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		NextLink:    keymap.NewBinding("next link", "tab"),
		PrevLink:    keymap.NewBinding("previous link", "shift+tab"),
		Follow:      keymap.NewBinding("follow link", "enter"),
		NextHeading: keymap.NewBinding("next heading", "]"),
		PrevHeading: keymap.NewBinding("previous heading", "["),
		Back:        keymap.NewBinding("back", "backspace", "alt+left"),
		Forward:     keymap.NewBinding("forward", "alt+right"),
	}
}

func (km KeyMap) list() []keymap.Binding {
	return []keymap.Binding{km.NextLink, km.PrevLink, km.Follow, km.NextHeading, km.PrevHeading, km.Back, km.Forward}
}

func (m Model) keys() KeyMap {
	for _, b := range m.KeyMap.list() {
		if len(b.Keys) > 0 {
			return m.KeyMap
		}
	}
	return DefaultKeyMap()
}

// Bindings returns the reader's actions followed by the viewport's scroll keys,
// for help text.
func (m Model) Bindings() []keymap.Binding { return append(m.keys().list(), m.vp.Bindings()...) }

// New returns a reader for md in a window of width by height cells.
func New(md string, width, height int) Model {
	m := Model{Theme: theme.DarkTheme(), KeyMap: DefaultKeyMap(), src: md, active: -1}
	m.vp = viewport.New(width, height)
	return m.render()
}

// SetSource replaces the document and scrolls to its top, forgetting the history.
func (m *Model) SetSource(md string) {
	m.src, m.back, m.forward, m.active = md, nil, nil, -1
	*m = m.render()
	m.vp.GotoTop()
}

// SetSize changes the window and renders the document again for the new width,
// keeping the heading at the top of the window in place when it can.
func (m *Model) SetSize(width, height int) {
	top := m.currentHeading()
	m.vp.Width, m.vp.Height = width, height
	*m = m.render()
	if top >= 0 && top < len(m.headings) {
		m.vp.SetYOffset(m.headings[top].Line)
	}
}

// Headings returns the document's headings in order: its table of contents.
func (m Model) Headings() []Heading { return append([]Heading(nil), m.headings...) }

// Links returns the document's links in order.
func (m Model) Links() []Link { return append([]Link(nil), m.links...) }

// Active returns the link Tab has reached, and false when none has been.
func (m Model) Active() (Link, bool) {
	if m.active < 0 || m.active >= len(m.links) {
		return Link{}, false
	}
	return m.links[m.active], true
}

// YOffset is the first visible row of the rendered document.
func (m Model) YOffset() int { return m.vp.YOffset() }

// Width is the window's width in cells.
func (m Model) Width() int { return m.vp.Width }

// Height is the window's height in cells, the status row included.
func (m Model) Height() int { return m.vp.Height }

var (
	atx     = regexp.MustCompile(`^ {0,3}(#{1,6})[ \t]+(.*?)(?:[ \t]+#+)?[ \t]*$`)
	setext  = regexp.MustCompile(`^ {0,3}(=+|-+)[ \t]*$`)
	link    = regexp.MustCompile(`(^|[^!\\])\[([^\]\n]+)\]\(\s*<?([^)\s>]+)>?[^)]*\)`)
	inlines = strings.NewReplacer("`", "", "*", "", "_", " ", "\\", "")
)

// outline reads the headings and links from the source, skipping fenced code.
func outline(md string) (hs []Heading, ls []Link) {
	lines := strings.Split(strings.ReplaceAll(md, "\r\n", "\n"), "\n")
	fence := ""
	seen := map[string]int{}
	add := func(level int, text string) {
		plain := strings.TrimSpace(link.ReplaceAllString(text, "$1$2"))
		plain = strings.TrimSpace(inlines.Replace(plain))
		a := slug(plain)
		if n := seen[a]; n > 0 {
			seen[a]++
			a += "-" + itoa(n)
		} else {
			seen[a] = 1
		}
		hs = append(hs, Heading{Level: level, Text: plain, Anchor: a})
	}
	for i, l := range lines {
		t := strings.TrimLeft(l, " ")
		if fence != "" {
			if strings.HasPrefix(t, fence) {
				fence = ""
			}
			continue
		}
		if strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~") {
			fence = t[:3]
			continue
		}
		if mm := atx.FindStringSubmatch(l); mm != nil {
			add(len(mm[1]), mm[2])
		} else if mm := setext.FindStringSubmatch(l); mm != nil && i > 0 && isText(lines[i-1]) {
			level := 1
			if mm[1][0] == '-' {
				level = 2
			}
			add(level, strings.TrimSpace(lines[i-1]))
		}
		for _, mm := range link.FindAllStringSubmatch(l, -1) {
			ls = append(ls, Link{Text: strings.TrimSpace(inlines.Replace(mm[2])), URL: mm[3]})
		}
	}
	return hs, ls
}

// isText reports whether line could be the text a setext underline sets: not
// blank, not a heading, and not the start of a list, a quote or a table row.
func isText(line string) bool {
	t := strings.TrimSpace(line)
	return t != "" && !atx.MatchString(line) && !strings.ContainsRune("-*+>|", rune(t[0]))
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

// slug is GitHub's heading anchor: lower case, spaces to hyphens, everything
// but letters, digits, hyphens and underscores dropped.
func slug(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r == ' ':
			b.WriteByte('-')
		case r == '-' || r == '_' || (r >= '0' && r <= '9') || r >= 0x80 || (r >= 'a' && r <= 'z'):
			b.WriteRune(r)
		}
	}
	return b.String()
}

// render draws the document at the window's width, and finds the row of each
// heading and link in the result.
func (m Model) render() Model {
	out := markdown.RenderWith(m.src, max(m.vp.Width, 1), m.Theme, m.Options)
	m.vp.SetContent(out)
	rows := strings.Split(out, "\n")
	plain := make([]string, len(rows))
	for i, r := range rows {
		plain[i] = ansi.StripANSI(r)
	}
	hs, ls := outline(m.src)
	at := 0
	for i := range hs {
		hs[i].Line = find(plain, hs[i].Text, &at)
	}
	at = 0
	for i := range ls {
		ls[i].Line = find(plain, ls[i].Text, &at)
	}
	m.headings, m.links = hs, ls
	if m.active >= len(ls) {
		m.active = -1
	}
	return m
}

// find returns the first row at or after *from that holds text, and moves *from
// there; 0 when no row does.
func find(rows []string, text string, from *int) int {
	if text == "" {
		return *from
	}
	for i := *from; i < len(rows); i++ {
		if strings.Contains(rows[i], text) {
			*from = i
			return i
		}
	}
	return *from
}

// currentHeading is the index of the last heading at or above the top row, or -1.
func (m Model) currentHeading() int {
	top, cur := m.vp.YOffset(), -1
	for i, h := range m.headings {
		if h.Line <= top {
			cur = i
		}
	}
	return cur
}

// jump scrolls to row, remembering where it left from for Back.
func (m *Model) jump(row int) {
	if m.vp.YOffset() != row {
		m.back, m.forward = append(m.back, m.vp.YOffset()), nil
	}
	m.vp.SetYOffset(row)
}

// GoTo scrolls to the heading with the anchor, as an in-page link "#anchor" does,
// and reports whether there is one. Back returns to where it left from.
func (m *Model) GoTo(anchor string) bool {
	anchor = strings.TrimPrefix(anchor, "#")
	for _, h := range m.headings {
		if h.Anchor == anchor {
			m.jump(h.Line)
			return true
		}
	}
	return false
}

// Update handles the reader's keys, then the viewport's scroll keys and wheel.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	if _, ok := msg.(tui.MouseEvent); ok {
		if !m.Mouse {
			return m, nil
		}
		m.vp.Mouse, m.vp.Bounds = true, m.Bounds
	}
	k, isKey := msg.(tui.Key)
	if !isKey {
		var cmd tui.Cmd
		m.vp, cmd = m.vp.Update(msg)
		return m, cmd
	}
	km := m.keys()
	switch {
	case keymap.Matches(k, km.NextLink) && len(m.links) > 0:
		m.active = (m.active + 1) % len(m.links)
		m.reveal()
	case keymap.Matches(k, km.PrevLink) && len(m.links) > 0:
		if m.active <= 0 {
			m.active = len(m.links) - 1
		} else {
			m.active--
		}
		m.reveal()
	case keymap.Matches(k, km.Follow):
		l, ok := m.Active()
		if !ok {
			break
		}
		if strings.HasPrefix(l.URL, "#") {
			m.GoTo(l.URL)
			break
		}
		msg := LinkMsg{Text: l.Text, URL: l.URL}
		return m, func() tui.Msg { return msg }
	case keymap.Matches(k, km.NextHeading):
		top := m.vp.YOffset()
		for _, h := range m.headings {
			if h.Line > top {
				m.jump(h.Line)
				break
			}
		}
	case keymap.Matches(k, km.PrevHeading):
		top := m.vp.YOffset()
		for i := len(m.headings) - 1; i >= 0; i-- {
			if m.headings[i].Line < top {
				m.jump(m.headings[i].Line)
				break
			}
		}
	case keymap.Matches(k, km.Back):
		if n := len(m.back); n > 0 {
			m.forward = append(m.forward, m.vp.YOffset())
			m.vp.SetYOffset(m.back[n-1])
			m.back = m.back[:n-1]
		}
	case keymap.Matches(k, km.Forward):
		if n := len(m.forward); n > 0 {
			m.back = append(m.back, m.vp.YOffset())
			m.vp.SetYOffset(m.forward[n-1])
			m.forward = m.forward[:n-1]
		}
	default:
		var cmd tui.Cmd
		m.vp, cmd = m.vp.Update(msg)
		return m, cmd
	}
	return m, nil
}

// reveal scrolls the active link into view, with a row of context above it.
func (m *Model) reveal() {
	l, ok := m.Active()
	if !ok {
		return
	}
	top, h := m.vp.YOffset(), m.bodyHeight()
	if l.Line < top || l.Line >= top+h {
		m.vp.SetYOffset(max(l.Line-1, 0))
	}
}

// bodyHeight is the rows left for the text: all but the status row.
func (m Model) bodyHeight() int {
	if len(m.links) > 0 {
		return max(m.vp.Height-1, 1)
	}
	return m.vp.Height
}

// View draws the document, and under it, when it has links, a status row that
// names the active link and where it goes.
func (m Model) View() string {
	body := m.vp
	body.Height = m.bodyHeight()
	out := body.View()
	if len(m.links) == 0 {
		return out
	}
	return out + "\n" + m.statusRow()
}

// statusRow names the active link and where it goes, in the focus colour, one
// window wide.
func (m Model) statusRow() string {
	status := ""
	if l, ok := m.Active(); ok {
		status = ansi.Truncate(ansi.Sanitize(l.Text)+" -> "+ansi.Sanitize(l.URL), max(m.vp.Width, 1))
	}
	return m.Theme.ResolvedStates().Focus.Render(status + strings.Repeat(" ", max(m.vp.Width-ansi.Width(status), 0)))
}

// Compile-time proof that Model satisfies tui.Component[Model].
var _ tui.Component[Model] = Model{}

// Linearize reads the document in words, as markdown.Model does.
func (m Model) Linearize() string { return markdown.New(m.src).Linearize() }

// LayoutNode adapts the reader to a layout.Node: the text fills the space it is
// given and, when the document has links, the status row sits under it. The
// Model is not changed.
func (m Model) LayoutNode() layout.Node {
	text := layout.FillWeight(m.vp.LayoutNode(), 1)
	if len(m.links) == 0 {
		return layout.Column(0, text)
	}
	return layout.Column(0, text, layout.FlexChild{Node: layout.Block(m.statusRow())})
}

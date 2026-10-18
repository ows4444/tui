// Command mdstudio is a Markdown editor with a live preview: the text on the
// left, the rendered document on the right, and a list of the headings
// (the outline) beside them. The preview follows the section you are editing.
//
// Mouse: click in a pane to work in it, drag in the editor to select, click a
// heading in the outline to jump there, scroll the wheel in the editor or the
// preview, click a toolbar button (H heading, B bold, I italic, ` code, Link,
// List, Save). Keyboard: Ctrl+W moves between the editor, the preview and the
// outline; in the editor type as usual, Ctrl+B makes the selection bold; in the
// preview Tab picks a link, Enter follows it and [ and ] jump between headings;
// in the outline Up and Down choose and Enter jumps; Ctrl+S saves, F1 shows
// help, Ctrl+Q quits (asking first if changes would be lost), Ctrl+C quits at
// once. On a narrow terminal one pane shows at a time.
//
//	go run ./examples/mdstudio [file.md]
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/examples/internal/ui"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/markdownview"
	"github.com/ows4444/tui/textarea"
	"github.com/ows4444/tui/textinput"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

const (
	outlineW = 22
	threeAt  = 100 // narrowest terminal with the outline beside the editor and preview
	twoAt    = 60  // narrowest terminal with the editor and preview side by side
	chrome   = 3   // toolbar, status, hints
)

type focus int

const (
	inEditor focus = iota
	inPreview
	inOutline
	numFocus
)

var focusNames = [...]string{"editor", "preview", "outline"}

type mode int

const (
	modeEdit mode = iota
	modeSaveAs
	modeDiscard
)

type model struct {
	th    ui.Theming // the theme the Program chose; see SetTheme
	st    store
	path  string
	saved string

	ed textarea.Model
	pv markdownview.Model
	in textinput.Model

	hs      []heading
	outSel  int
	syncIdx int // the heading the preview was last scrolled to
	focus   focus
	mode    mode
	msg     string
	help    bool
	hover   string
	drag    bool

	width, height int
}

func newModel(st store, path string) (model, error) {
	text := sampleDoc
	if path != "" {
		t, err := st.read(path)
		if err != nil {
			return model{}, err
		}
		text = t
	}
	m := model{st: st, path: path, saved: text, syncIdx: -2}
	m.ed = textarea.New()
	m.ed.Mouse, m.ed.AutoIndent, m.ed.SoftWrap, m.ed.ShowLineNumbers = true, true, true, true
	m.ed.Highlighter = highlighter(m.theme())
	m.ed.SetValue(text)
	m.ed.SetCursor(0)
	m.ed.Focus()
	m.in = textinput.New()
	m.in.Width = 40
	m.pv = markdownview.New(text, 40, 10)
	m.pv.Mouse = true
	m.hs = outline(text)
	return m.resize().sync(true), nil
}

func (m model) Init() tui.Cmd { return m.ed.Focus() }

func (m model) dirty() bool { return m.ed.Value() != m.saved }

// ---- geometry --------------------------------------------------------------

func (m model) size() layout.Size {
	if m.width == 0 {
		return layout.Size{W: 80, H: 24}
	}
	return layout.Size{W: m.width, H: m.height}
}

func (m model) bodyH() int { return max(1, m.size().H-chrome) }

// panes are the rectangles of the panes on screen. A narrow terminal shows
// only the one with focus.
func (m model) panes() map[focus]layout.Rect {
	w, h := m.size().W, m.bodyH()
	switch {
	case w >= threeAt:
		ew := (w - outlineW - 2) / 2
		return map[focus]layout.Rect{
			inOutline: {X: 0, Y: 1, W: outlineW, H: h},
			inEditor:  {X: outlineW + 1, Y: 1, W: ew, H: h},
			inPreview: {X: outlineW + 2 + ew, Y: 1, W: w - outlineW - 2 - ew, H: h},
		}
	case w >= twoAt:
		ew := (w - 1) / 2
		return map[focus]layout.Rect{
			inEditor:  {X: 0, Y: 1, W: ew, H: h},
			inPreview: {X: ew + 1, Y: 1, W: w - 1 - ew, H: h},
		}
	}
	return map[focus]layout.Rect{m.focus: {X: 0, Y: 1, W: w, H: h}}
}

func (m model) gutter() int { return len(strconv.Itoa(max(99, m.ed.LineCount()))) + 1 }

// resize fits the editor and the preview to their panes.
func (m model) resize() model {
	ps := m.panes()
	if r, ok := ps[inEditor]; ok {
		m.ed.Width = max(8, r.W-m.gutter())
		m.ed.Height = r.H
		m.ed.Bounds = r
	}
	if r, ok := ps[inPreview]; ok {
		m.pv.SetSize(max(8, r.W), r.H)
		m.pv.Bounds = r
	}
	return m
}

// ---- keeping the three views in step ---------------------------------------

// anchors are the preview anchors of the source headings, in order, or nil
// when they cannot be matched up.
func (m model) anchors() []string {
	ph := m.pv.Headings()
	out := make([]string, len(m.hs))
	if len(ph) == len(m.hs) {
		for i := range ph {
			out[i] = ph[i].Anchor
		}
		return out
	}
	j := 0
	for i, h := range m.hs {
		want := plain(h.text)
		for ; j < len(ph); j++ {
			if ph[j].Text == want {
				out[i] = ph[j].Anchor
				j++
				break
			}
		}
		if out[i] == "" {
			return nil
		}
	}
	return out
}

// plain strips the inline markup a heading may carry.
func plain(s string) string {
	return strings.NewReplacer("**", "", "__", "", "*", "", "_", "", "`", "").Replace(s)
}

// sync scrolls the preview to the section the cursor is in. force does it even
// if the section is the one shown already (the source just changed).
func (m model) sync(force bool) model {
	idx := current(m.hs, m.ed.Line())
	if idx == m.syncIdx && !force {
		return m
	}
	m.syncIdx = idx
	if idx < 0 {
		return m
	}
	if a := m.anchors(); a != nil {
		m.pv.GoTo(a[idx])
	}
	m.outSel = idx
	return m
}

// edited refreshes the outline and preview after the text changed.
func (m model) edited() model {
	v := m.ed.Value()
	m.hs = outline(v)
	m.pv.SetSource(v)
	m = m.resize()
	if m.outSel >= len(m.hs) {
		m.outSel = max(0, len(m.hs)-1)
	}
	return m.sync(true)
}

// afterEditor runs after the editor handled a message: if the text changed,
// refresh; either way follow the cursor.
func (m model) afterEditor(before string) model {
	if m.ed.Value() != before {
		m.msg = ""
		return m.edited()
	}
	return m.sync(false)
}

func (m model) jumpTo(h int) model {
	if h < 0 || h >= len(m.hs) {
		return m
	}
	m.ed.SetCursorPosition(m.hs[h].line, 0)
	m.outSel = h
	m = m.setFocus(inEditor)
	return m.sync(true)
}

func (m model) setFocus(f focus) model {
	m.focus = f
	if f == inEditor {
		m.ed.Focus()
	} else {
		m.ed.Blur()
	}
	return m.resize()
}

// ---- update ----------------------------------------------------------------

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m.resize(), nil
	case markdownview.LinkMsg:
		m.msg = "Link: " + msg.URL + " (not opened here)"
		return m, nil
	case tui.Key:
		return m.key(msg)
	case tui.MouseEvent:
		return m.mouse(msg)
	}
	before := m.ed.Value()
	var cmd tui.Cmd
	m.ed, cmd = m.ed.Update(msg) // pastes and blink ticks
	return m.afterEditor(before), cmd
}

func (m model) key(k tui.Key) (tui.Model, tui.Cmd) {
	if k.Type == tui.KeyCtrlC {
		return m, tui.Quit()
	}
	switch m.mode {
	case modeDiscard:
		return m.discardKey(k)
	case modeSaveAs:
		return m.saveAsKey(k)
	}
	if k.Type == tui.KeyF1 {
		m.help = !m.help
		return m, nil
	}
	if m.help {
		m.help = k.Type != tui.KeyEsc && k.Type != tui.KeyEnter
		return m, nil
	}
	if k.Type == tui.KeyCtrl {
		if next, cmd, ok := m.ctrlKey(k.Code); ok {
			return next, cmd
		}
	}
	switch m.focus {
	case inPreview:
		return m.previewKey(k)
	case inOutline:
		return m.outlineKey(k), nil
	}
	before := m.ed.Value()
	var cmd tui.Cmd
	m.ed, cmd = m.ed.Update(k)
	return m.afterEditor(before), cmd
}

func (m model) ctrlKey(code rune) (model, tui.Cmd, bool) {
	switch code {
	case 'w':
		return m.setFocus((m.focus + 1) % numFocus), nil, true
	case 's':
		return m.saveCmd(), nil, true
	case 'q':
		next, cmd := m.quit()
		return next, cmd, true
	case 'b':
		if m.focus == inEditor {
			return m.wrap("**"), nil, true
		}
	}
	return m, nil, false
}

func (m model) quit() (model, tui.Cmd) {
	if m.dirty() {
		m.mode = modeDiscard
		return m, nil
	}
	return m, tui.Quit()
}

func (m model) discardKey(k tui.Key) (tui.Model, tui.Cmd) {
	switch {
	case k.Type == tui.KeyRunes && (k.Text == "y" || k.Text == "Y"):
		return m, tui.Quit()
	case k.Type == tui.KeyEsc, k.Type == tui.KeyRunes && (k.Text == "n" || k.Text == "N"):
		m.mode = modeEdit
	}
	return m, nil
}

func (m model) previewKey(k tui.Key) (tui.Model, tui.Cmd) {
	if k.Type == tui.KeyEsc {
		return m.setFocus(inEditor), nil
	}
	var cmd tui.Cmd
	m.pv, cmd = m.pv.Update(k)
	return m, cmd
}

func (m model) outlineKey(k tui.Key) model {
	n := len(m.hs)
	switch k.Type {
	case tui.KeyUp:
		m.outSel--
	case tui.KeyDown:
		m.outSel++
	case tui.KeyEnter:
		return m.jumpTo(m.outSel)
	case tui.KeyEsc:
		return m.setFocus(inEditor)
	case tui.KeyRunes:
		switch k.Text {
		case "k":
			m.outSel--
		case "j":
			m.outSel++
		}
	}
	m.outSel = max(0, min(m.outSel, n-1))
	return m
}

// ---- editing helpers -------------------------------------------------------

// wrap puts marker round the selection, or a pair of them round the cursor.
func (m model) wrap(marker string) model {
	before := m.ed.Value()
	if sel := m.ed.SelectedText(); sel != "" {
		m.ed.InsertString(marker + sel + marker)
	} else {
		m.ed.InsertString(marker + marker)
		m.ed.SetCursor(m.ed.Cursor() - len([]rune(marker)))
	}
	return m.setFocus(inEditor).afterEditor(before)
}

// linePrefix puts prefix at the start of the cursor's line.
func (m model) linePrefix(prefix string) model {
	before := m.ed.Value()
	line, col := m.ed.Line(), m.ed.Column()
	m.ed.SetCursorPosition(line, 0)
	m.ed.InsertString(prefix)
	m.ed.SetCursorPosition(line, col+len([]rune(prefix)))
	return m.setFocus(inEditor).afterEditor(before)
}

func (m model) link() model {
	before := m.ed.Value()
	sel := m.ed.SelectedText()
	if sel == "" {
		sel = "text"
	}
	m.ed.InsertString("[" + sel + "](https://)")
	return m.setFocus(inEditor).afterEditor(before)
}

var toolbar = []struct{ id, label string }{
	{"tb-h", "H"}, {"tb-b", "B"}, {"tb-i", "I"}, {"tb-code", "`"}, {"tb-link", "Link"}, {"tb-list", "List"}, {"tb-save", "Save"},
}

func (m model) tool(id string) (model, tui.Cmd) {
	switch id {
	case "tb-h":
		return m.linePrefix("## "), nil
	case "tb-b":
		return m.wrap("**"), nil
	case "tb-i":
		return m.wrap("*"), nil
	case "tb-code":
		return m.wrap("`"), nil
	case "tb-link":
		return m.link(), nil
	case "tb-list":
		return m.linePrefix("- "), nil
	case "tb-pane":
		return m.setFocus((m.focus + 1) % numFocus), nil
	}
	return m.saveCmd(), nil
}

// ---- saving ----------------------------------------------------------------

func (m model) saveCmd() model {
	if m.path == "" {
		m.mode = modeSaveAs
		m.in.Prompt = "Save as ▸ "
		m.in.SetValue("")
		m.in.Focus()
		m.ed.Blur()
		return m
	}
	return m.save(m.path)
}

func (m model) save(path string) model {
	text := m.ed.Value()
	if err := m.st.write(path, text); err != nil {
		m.msg = "Cannot save: " + err.Error()
		return m
	}
	m.path, m.saved, m.msg = path, text, "Saved "+path
	return m
}

func (m model) saveAsKey(k tui.Key) (tui.Model, tui.Cmd) {
	switch k.Type {
	case tui.KeyEsc:
		m.mode = modeEdit
		m.in.Blur()
		return m.setFocus(m.focus), nil
	case tui.KeyEnter:
		m.mode = modeEdit
		m.in.Blur()
		if p := strings.TrimSpace(m.in.Value()); p != "" {
			m = m.save(p)
		}
		return m.setFocus(m.focus), nil
	}
	var cmd tui.Cmd
	m.in, cmd = m.in.Update(k)
	return m, cmd
}

// ---- mouse -----------------------------------------------------------------

func inside(r layout.Rect, x, y int) bool { return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H }

func (m model) toolHit(x, y int) string {
	ids := []string{"tb-pane"}
	for _, t := range toolbar {
		ids = append(ids, t.id)
	}
	for _, id := range ids {
		if r, ok := layout.RectOf(m.screen(), m.size(), id); ok && inside(r, x, y) {
			return id
		}
	}
	return ""
}

func (m model) paneAt(x, y int) (focus, layout.Rect, bool) {
	for f, r := range m.panes() {
		if inside(r, x, y) {
			return f, r, true
		}
	}
	return 0, layout.Rect{}, false
}

func (m model) mouse(ev tui.MouseEvent) (tui.Model, tui.Cmd) {
	if m.mode != modeEdit || m.help {
		return m, nil
	}
	if ev.Action == tui.MouseActionMotion && !m.drag {
		m.hover = m.toolHit(ev.X, ev.Y)
		return m, nil
	}
	if ev.Action == tui.MouseActionRelease {
		m.drag = false
	}
	if id := m.toolHit(ev.X, ev.Y); id != "" && ev.Button == tui.MouseButtonLeft && ev.Action == tui.MouseActionPress {
		return m.tool(id)
	}
	f, r, ok := m.paneAt(ev.X, ev.Y)
	if m.drag {
		f, r, ok = inEditor, m.panes()[inEditor], true // a drag that began in the editor stays there
	}
	if !ok {
		return m, nil
	}
	press := ev.Button == tui.MouseButtonLeft && ev.Action == tui.MouseActionPress
	if press {
		m = m.setFocus(f)
		m.drag = f == inEditor
	}
	switch f {
	case inEditor:
		before := m.ed.Value()
		m.ed.Bounds = r
		var cmd tui.Cmd
		m.ed, cmd = m.ed.Update(ev)
		return m.afterEditor(before), cmd
	case inPreview:
		m.pv.Bounds = r
		var cmd tui.Cmd
		m.pv, cmd = m.pv.Update(ev)
		return m, cmd
	}
	return m.outlineMouse(ev, r, press), nil
}

func (m model) outlineMouse(ev tui.MouseEvent, r layout.Rect, press bool) model {
	switch ev.Button {
	case tui.MouseButtonWheelDown:
		m.outSel = min(m.outSel+1, len(m.hs)-1)
	case tui.MouseButtonWheelUp:
		m.outSel = max(m.outSel-1, 0)
	}
	if press {
		if i := m.outTop() + ev.Y - r.Y; i >= 0 && i < len(m.hs) {
			return m.jumpTo(i)
		}
	}
	return m
}

// outTop is the first heading shown in the outline.
func (m model) outTop() int {
	h := m.bodyH()
	return max(0, min(m.outSel-h+1, max(0, len(m.hs)-h)))
}

// ---- view ------------------------------------------------------------------

func pad(s string, w int) string {
	if n := ansi.Width(s); n < w {
		return s + strings.Repeat(" ", w-n)
	}
	return s
}

// column cuts or pads the lines of a pane to exactly w by h.
func column(lines []string, w, h int) []string {
	out := make([]string, h)
	for i := range out {
		l := ""
		if i < len(lines) {
			l = ansi.Truncate(lines[i], w)
		}
		out[i] = pad(l, w)
	}
	return out
}

func (m model) outlineLines(w, h int, th theme.Theme) []string {
	if len(m.hs) == 0 {
		return column([]string{ansi.NewStyle().Faint().Render(" No headings yet")}, w, h)
	}
	var lines []string
	cur := current(m.hs, m.ed.Line())
	for i := m.outTop(); i < len(m.hs) && len(lines) < h; i++ {
		x := m.hs[i]
		line := strings.Repeat("  ", x.level-1) + plain(x.text)
		st := ansi.NewStyle()
		mark := " "
		switch {
		case i == m.outSel && m.focus == inOutline:
			st, mark = ansi.NewStyle().Bold().Foreground(th.TextInverse).Background(th.Primary), "›"
		case i == cur:
			st, mark = ansi.NewStyle().Bold().Background(th.Selection), "•"
		}
		lines = append(lines, st.Render(pad(ansi.Truncate(mark+line, w), w)))
	}
	return column(lines, w, h)
}

func (m model) divider(f focus, th theme.Theme) string {
	st := ansi.NewStyle().Foreground(th.Muted)
	if m.focus == f {
		st = ansi.NewStyle().Foreground(th.Primary)
	}
	return st.Render("│")
}

func (m model) body(th theme.Theme) []string {
	ps, h := m.panes(), m.bodyH()
	cols := map[focus][]string{}
	if r, ok := ps[inOutline]; ok {
		cols[inOutline] = m.outlineLines(r.W, h, th)
	}
	if r, ok := ps[inEditor]; ok {
		cols[inEditor] = column(strings.Split(m.ed.View(), "\n"), r.W, h)
	}
	if r, ok := ps[inPreview]; ok {
		cols[inPreview] = column(strings.Split(m.pv.View(), "\n"), r.W, h)
	}
	order := []focus{inOutline, inEditor, inPreview}
	lines := make([]string, h)
	for y := range lines {
		var b strings.Builder
		first := true
		for _, f := range order {
			c, ok := cols[f]
			if !ok {
				continue
			}
			if !first {
				b.WriteString(m.divider(f, th))
			}
			first = false
			b.WriteString(c[y])
		}
		lines[y] = b.String()
	}
	return lines
}

func (m model) toolbarNode(th theme.Theme) layout.Node {
	w := m.size().W
	name := m.path
	if name == "" {
		name = "untitled.md"
	}
	if m.dirty() {
		name += " ●"
	}
	title := ansi.NewStyle().Bold().Foreground(th.Primary).Render("mdstudio") + " " + ansi.NewStyle().Bold().Render(name)
	var btns []layout.FlexChild
	add := func(id, label string) {
		st := ansi.NewStyle().Foreground(th.Muted)
		if m.hover == id {
			st = ansi.NewStyle().Bold().Foreground(th.Primary).Underline()
		}
		btns = append(btns, layout.FlexChild{Node: layout.Named(id, layout.Block(st.Render("["+label+"]")))})
	}
	if w < twoAt {
		add("tb-pane", focusNames[m.focus])
	}
	for _, t := range toolbar {
		add(t.id, t.label)
	}
	return layout.Fixed(layout.Row(0,
		layout.FlexChild{Node: layout.Block(ansi.Truncate(title, max(1, w-34))), Grow: 1},
		layout.FlexChild{Node: layout.Row(0, btns...)},
	), layout.Size{W: w, H: 1})
}

func (m model) statusLine(th theme.Theme) string {
	w := m.size().W
	switch m.mode {
	case modeDiscard:
		return ansi.NewStyle().Bold().Foreground(th.Warning).Render("Discard unsaved changes? ") + "[y] yes  [n] no"
	case modeSaveAs:
		return ansi.Truncate(m.in.View(), w)
	}
	left := fmt.Sprintf(" %s  Ln %d, Col %d  %d words  %d headings", focusNames[m.focus], m.ed.Line()+1, m.ed.Column()+1, words(m.ed.Value()), len(m.hs))
	if m.msg != "" {
		left += "   " + m.msg
	}
	return ansi.NewStyle().Background(th.Surface).Render(pad(ansi.Truncate(left, w), w))
}

func (m model) helpLines(th theme.Theme) []string {
	key := func(k, a string) string {
		return " " + ansi.NewStyle().Bold().Foreground(th.Primary).Render(pad(k, 14)) + a
	}
	return []string{
		key("Ctrl+W", "editor, preview, outline in turn"), key("Ctrl+B", "bold the selection"),
		key("Ctrl+S", "save"), key("Ctrl+Q", "quit, asking first"), key("F1", "close this help"),
		"", ansi.NewStyle().Bold().Render(" Preview"), key("Tab Enter", "pick a link, follow it"),
		key("[ ]", "previous and next heading"), key("Esc", "back to the editor"),
		"", ansi.NewStyle().Bold().Render(" Outline"), key("Up Down", "choose a heading"), key("Enter", "jump to it"),
	}
}

func (m model) screen() layout.Node {
	th := m.theme()
	w := m.size().W
	lines := m.body(th)
	if m.help {
		lines = column(m.helpLines(th), w, m.bodyH())
	}
	hints := widgets.Footer(w, widgets.Hint{Key: "ctrl+w", Action: "pane"}, widgets.Hint{Key: "ctrl+s", Action: "save"},
		widgets.Hint{Key: "ctrl+b", Action: "bold"}, widgets.Hint{Key: "f1", Action: "help"}, widgets.Hint{Key: "ctrl+q", Action: "quit"})
	var body []layout.FlexChild
	for _, l := range lines {
		body = append(body, layout.FlexChild{Node: layout.Block(l)})
	}
	return layout.Column(0,
		layout.FlexChild{Node: m.toolbarNode(th)},
		layout.FlexChild{Node: layout.Column(0, body...)},
		layout.FlexChild{Node: layout.Block(m.statusLine(th))},
		layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render(ansi.Truncate(hints, w)))},
	)
}

func (m model) View() string { return layout.DrawTight(m.screen(), m.size()) }

func main() {
	path := ""
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	m, err := newModel(diskStore{}, path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "mdstudio:", err)
		os.Exit(1)
	}
	if _, err := ui.NewProgram(m, tui.WithAltScreen(true), tui.WithMouse(tui.MouseAllMotion)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

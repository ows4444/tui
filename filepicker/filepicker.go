// Package filepicker is a real filesystem browse-and-select widget —
// InkUI has no direct equivalent, but it follows the single-choice-list
// shape of picker.Model, browsing one directory at a time.
package filepicker

import (
	"github.com/ows4444/tui/ansi"
	"path/filepath"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/internal/fsutil"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/theme"
)

// Model is a keyboard-navigable file browser rooted at Dir. Enter on a
// directory (or Right) descends into it; Left/Backspace goes back to the
// parent. Enter on a file confirms it via SelectedMsg. Extensions, when
// non-empty, restricts which files are listed (directories are always
// listed, for navigation); DirsOnly excludes files entirely and makes
// Enter on a directory confirm it instead of descending into it.
type Model struct {
	Dir        string
	Extensions []string // e.g. []string{".go", ".md"}; empty means no filter
	DirsOnly   bool
	Theme      theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens
	// Height is the number of entries View shows at once; zero shows all of
	// them. With Height set, View renders (and sanitises) only that window,
	// which keeps the cursor entry in view, so a directory of 10,000 entries
	// costs O(Height) per frame, not O(entries).
	Height int

	// Raw, when true, draws file and directory names unchanged. By default
	// each name is sanitised (ansi.Sanitize): a file name may hold escape
	// sequences, and drawing them would let a file in the listing act on the
	// terminal.
	Raw bool

	// KeyMap holds the keys Update reacts to. New sets DefaultKeyMap; a
	// Model built as a struct literal with a zero KeyMap uses DefaultKeyMap.
	KeyMap KeyMap

	// Mouse turns on mouse handling in Update (off by default). Bounds is the
	// screen rectangle where the app draws the list, set by the app; events
	// outside it are ignored. WheelStep is the entries one wheel notch
	// scrolls (0 means 3); the wheel needs Height set, since otherwise every
	// entry is shown. A left press on an entry moves the cursor to it.
	Mouse     bool
	Bounds    hittest.Rect
	WheelStep int

	cursor  int
	offset  int // first entry shown when Height is set; kept by Update
	entries []fsutil.Entry
}

// KeyMap is the set of keys a file picker reacts to.
type KeyMap struct {
	Up   keymap.Binding
	Down keymap.Binding
	// Select descends into the directory under the cursor, or confirms a
	// file (or, in DirsOnly mode, a directory).
	Select keymap.Binding
	// Open descends into the directory under the cursor.
	Open keymap.Binding
	// Back goes to the parent directory.
	Back keymap.Binding
}

// DefaultKeyMap returns the default keys: up/down, enter to select, right to
// open, left or backspace to go back.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:     keymap.NewBinding("up", "up"),
		Down:   keymap.NewBinding("down", "down"),
		Select: keymap.NewBinding("select", "enter"),
		Open:   keymap.NewBinding("open", "right"),
		Back:   keymap.NewBinding("parent", "left", "backspace"),
	}
}

func (k KeyMap) all() []keymap.Binding {
	return []keymap.Binding{k.Up, k.Down, k.Select, k.Open, k.Back}
}

// keys is the KeyMap in force: KeyMap, or the defaults when it is unset.
func (m Model) keys() KeyMap {
	for _, b := range m.KeyMap.all() {
		if len(b.Keys) > 0 {
			return m.KeyMap
		}
	}
	return DefaultKeyMap()
}

// Bindings returns the active key bindings, with descriptions, for help text.
func (m Model) Bindings() []keymap.Binding { return m.keys().all() }

// New builds a Model rooted at dir, listing its entries immediately.
func New(dir string) Model {
	m := Model{Dir: dir, Theme: theme.DarkTheme(), KeyMap: DefaultKeyMap()}
	m.reload()
	return m
}

func (m *Model) reload() {
	all, err := fsutil.ListDir(m.Dir)
	if err != nil {
		m.entries = nil
		m.cursor = 0
		return
	}

	entries := make([]fsutil.Entry, 0, len(all))
	for _, e := range all {
		if !e.IsDir {
			if m.DirsOnly || !fsutil.HasExt(e.Name, m.Extensions) {
				continue
			}
		}
		entries = append(entries, e)
	}
	m.entries = entries
	if len(entries) == 0 {
		m.cursor, m.offset = 0, 0
		return
	}
	m.cursor = clamp(m.cursor, 0, len(entries)-1)
	m.reveal()
}

// window is the half-open range of entries View shows: all of them when
// Height is unset or they fit, otherwise Height entries that contain the
// cursor, scrolling from the kept offset only as far as needed.
func (m Model) window() (start, end int) {
	h := m.Height
	if h <= 0 || h >= len(m.entries) {
		return 0, len(m.entries)
	}
	start = clamp(m.offset, 0, len(m.entries)-h)
	if m.cursor < start {
		start = m.cursor
	} else if m.cursor >= start+h {
		start = m.cursor - h + 1
	}
	return start, start + h
}

func (m *Model) reveal() { m.offset, _ = m.window() }

// Cursor returns the index of the entry currently under the cursor.
func (m Model) Cursor() int { return m.cursor }

// Entries returns the currently listed entries of Dir.
func (m Model) Entries() []fsutil.Entry { return m.entries }

// SelectedMsg is delivered (via the Cmd Update returns) when a file is
// confirmed with Enter, or a directory is confirmed with Enter in
// DirsOnly mode.
type SelectedMsg struct {
	Path string
}

// Update, with the default KeyMap, moves the cursor on Up/Down, descends into a directory on Enter
// or Right, goes to the parent directory on Left or Backspace, and
// confirms a file (or, in DirsOnly mode, a directory) on Enter, returning
// a Cmd that delivers SelectedMsg. With Mouse on, it also handles mouse
// events inside Bounds. Any other Msg is a no-op.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	m, cmd := m.step(msg)
	m.reveal()
	return m, cmd
}

func (m Model) step(msg tui.Msg) (Model, tui.Cmd) {
	if ev, ok := msg.(tui.MouseEvent); ok {
		return m.mouse(ev), nil
	}
	km := m.keys()
	switch {
	case keymap.Matches(msg, km.Up):
		if m.cursor > 0 {
			m.cursor--
		}
	case keymap.Matches(msg, km.Down):
		if m.cursor < len(m.entries)-1 {
			m.cursor++
		}
	case keymap.Matches(msg, km.Open):
		m.descend()
	case keymap.Matches(msg, km.Back):
		m.ascend()
	case keymap.Matches(msg, km.Select):
		if len(m.entries) == 0 {
			return m, nil
		}
		entry := m.entries[m.cursor]
		if entry.IsDir && !m.DirsOnly {
			m.descend()
			return m, nil
		}
		path := filepath.Join(m.Dir, entry.Name)
		return m, func() tui.Msg { return SelectedMsg{Path: path} }
	}
	return m, nil
}

func (m Model) mouse(ev tui.MouseEvent) Model {
	if !m.Mouse || !m.Bounds.Contains(ev.X, ev.Y) || ev.Action != tui.MouseActionPress {
		return m
	}
	step := m.WheelStep
	if step <= 0 {
		step = 3
	}
	switch ev.Button {
	case tui.MouseButtonWheelUp, tui.MouseButtonWheelDown:
		h := m.Height
		if h <= 0 || h >= len(m.entries) {
			return m
		}
		start, _ := m.window()
		if ev.Button == tui.MouseButtonWheelUp {
			step = -step
		}
		start = clamp(start+step, 0, len(m.entries)-h)
		m.offset = start
		m.cursor = clamp(m.cursor, start, start+h-1)
	case tui.MouseButtonLeft:
		start, end := m.window()
		_, ly := m.Bounds.Local(ev.X, ev.Y)
		if start+ly < end {
			m.cursor = start + ly
		}
	}
	return m
}

func (m *Model) descend() {
	if len(m.entries) == 0 {
		return
	}
	entry := m.entries[m.cursor]
	if !entry.IsDir {
		return
	}
	m.Dir = filepath.Join(m.Dir, entry.Name)
	m.cursor = 0
	m.reload()
}

func (m *Model) ascend() {
	parent := filepath.Dir(m.Dir)
	if parent == m.Dir {
		return
	}
	m.Dir = parent
	m.cursor = 0
	m.reload()
}

// View renders the current directory's entries, directories marked with
// a trailing "/", with the entry under the cursor highlighted.
func (m Model) View() string {
	start, end := m.window()
	return m.render(start, end)
}

// render draws entries [start, end).
func (m Model) render(start, end int) string {
	if start >= end {
		return ""
	}

	cursorStyle := m.themed().ResolvedStates().Selected.Bold()

	var b strings.Builder
	for i := start; i < end; i++ {
		entry := m.entries[i]
		prefix := "  "
		label := ansi.Clean(m.Raw, entry.Name)
		if entry.IsDir {
			label += "/"
		}
		if i == m.cursor {
			prefix = "> "
			label = cursorStyle.Render(label)
		}
		b.WriteString(prefix + label)
		if i < end-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// Compile-time proof that Model satisfies tui.Component[Model] — see
// tui.Component's doc comment for what this contract means and why.
var _ tui.Component[Model] = Model{}

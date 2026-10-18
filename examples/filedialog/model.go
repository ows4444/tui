package main

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/filepicker"
	"github.com/ows4444/tui/layout"
)

// filters are the kinds of file the dialog can be narrowed to.
var filters = []struct {
	name string
	exts []string
}{
	{"All files", nil},
	{"Go source", []string{".go"}},
	{"Text", []string{".txt", ".md"}},
	{"Images", []string{".png", ".jpg", ".jpeg", ".gif"}},
}

const chromeRows = 6 // title, path, filter row, status, buttons, hints

type model struct {
	fp         filepicker.Model
	save       bool
	showHidden bool
	filter     int
	name       []rune // the file name typed in Save As
	onName     bool   // the name field has the keys
	confirm    string // path waiting for an overwrite answer
	result     string // the chosen path, set when the dialog ends
	help       bool
	hover      string
	msg        string
	lastClick  string         // row id of the last single click, for click-again-to-open
	loc        *time.Location // time zone for dates; nil is local (tests use UTC)
	root       string         // when set, the path bar starts at this folder (tests use it to hide the temporary path)

	width, height int
}

func newModel(dir string, save bool) model {
	abs, err := filepath.Abs(dir)
	if err != nil {
		abs = dir
	}
	fp := filepicker.New(abs)
	fp.TypeAhead = true
	m := model{fp: fp, save: save}
	m.onName = save
	return m.fix(1)
}

func (m model) Init() tui.Cmd { return nil }

func (m model) size() layout.Size {
	if m.width == 0 {
		return layout.Size{W: 90, H: 28}
	}
	return layout.Size{W: m.width, H: m.height}
}

func (m model) rows() int { return max(1, m.size().H-chromeRows) }

func hidden(name string) bool { return strings.HasPrefix(name, ".") }

// visible lists the indexes of the entries the list shows.
func (m model) visible() []int {
	var out []int
	for i, e := range m.fp.Entries() {
		if m.showHidden || !hidden(e.Name) {
			out = append(out, i)
		}
	}
	return out
}

// fix moves the picker's cursor off a dotfile that is hidden, in direction d
// (1 down, -1 up); with nothing visible in that direction it goes the other way.
func (m model) fix(d int) model {
	es := m.fp.Entries()
	if len(es) == 0 || m.showHidden || !hidden(es[m.fp.Cursor()].Name) {
		return m
	}
	for _, dir := range []int{d, -d} {
		fp := m.fp
		for range len(es) {
			before := fp.Cursor()
			fp, _ = fp.Update(arrow(dir))
			if fp.Cursor() == before {
				break
			}
			if !hidden(es[fp.Cursor()].Name) {
				m.fp = fp
				return m
			}
		}
	}
	return m
}

func arrow(d int) tui.Key {
	if d < 0 {
		return tui.Key{Type: tui.KeyUp}
	}
	return tui.Key{Type: tui.KeyDown}
}

// moveTo puts the cursor on entry index i by stepping the picker.
func (m model) moveTo(i int) model {
	for n := 0; m.fp.Cursor() != i && n < len(m.fp.Entries()); n++ {
		d := 1
		if i < m.fp.Cursor() {
			d = -1
		}
		m.fp, _ = m.fp.Update(arrow(d))
	}
	return m
}

func (m model) current() (name string, isDir, ok bool) {
	es := m.fp.Entries()
	if len(es) == 0 || (!m.showHidden && hidden(es[m.fp.Cursor()].Name)) {
		return "", false, false
	}
	e := es[m.fp.Cursor()]
	return e.Name, e.IsDir, true
}

func (m model) path(name string) string { return filepath.Join(m.fp.Dir, name) }

// goTo shows another folder.
func (m model) goTo(dir string) model {
	m.fp.Dir = dir
	m.fp = m.fp.Reload()
	for m.fp.Cursor() > 0 { // the picker has no Home: step to the top
		m.fp, _ = m.fp.Update(arrow(-1))
	}
	m.lastClick = ""
	return m.fix(1)
}

func (m model) setFilter(i int) model {
	m.filter = ((i % len(filters)) + len(filters)) % len(filters)
	m.fp.Extensions = filters[m.filter].exts
	m.fp = m.fp.Reload()
	return m.fix(1)
}

func (m model) toggleHidden() model {
	m.showHidden = !m.showHidden
	return m.fix(1)
}

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.fp.Height = m.rows()
	case tui.Key:
		return m.key(msg)
	case tui.MouseEvent:
		return m.mouse(msg)
	}
	return m, nil
}

// ---- finishing -------------------------------------------------------------

// choose ends the dialog with path, after asking about an overwrite.
func (m model) choose(path string) (model, tui.Cmd) {
	if m.save {
		if _, err := os.Stat(path); err == nil && m.confirm == "" {
			m.confirm = path
			return m, nil
		}
	}
	m.confirm = ""
	m.result = path
	return m, tui.Quit()
}

// activate opens the folder under the cursor, or chooses the file.
func (m model) activate() (model, tui.Cmd) {
	name, isDir, ok := m.current()
	if !ok {
		return m, nil
	}
	if isDir {
		return m.goTo(m.path(name)), nil
	}
	if m.save {
		m.name = []rune(name)
		return m.choose(m.path(name))
	}
	return m.choose(m.path(name))
}

// accept is the Open or Save button.
func (m model) accept() (model, tui.Cmd) {
	if m.save {
		name := strings.TrimSpace(string(m.name))
		if name == "" || strings.ContainsAny(name, `/\`) {
			m.msg = "Type a file name without slashes"
			return m, nil
		}
		return m.choose(m.path(name))
	}
	name, isDir, ok := m.current()
	if !ok || isDir {
		m.msg = "Select a file to open"
		return m, nil
	}
	return m.choose(m.path(name))
}

func (m model) setMode(save bool) model {
	m.save = save
	m.onName = save
	m.msg = ""
	return m
}

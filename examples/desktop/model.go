package main

import (
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/examples/internal/ui"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/motion"
	"github.com/ows4444/tui/window"
)

const (
	chromeRows = 4 // title, task bar, status, hints
	maxLog     = 200
)

// app describes one window: its id, its title and where it starts, as fractions
// of the desktop so it fits any size.
type app struct {
	id, title  string
	x, y, w, h int // percent of the desktop
}

var apps = []app{
	{"clock", "Clock", 3, 5, 30, 40},
	{"counter", "Counter", 38, 10, 30, 45},
	{"notes", "Notes", 6, 38, 46, 50},
	{"log", "Event log", 52, 52, 44, 40},
}

type modal struct {
	title, text string
	yes         string // what the yes button does: "quit", "close:<id>"
}

type tickMsg struct{}

type model struct {
	th      ui.Theming // the theme the Program chose; see SetTheme
	desk    window.Model
	hidden  []window.Window // hidden to the task bar, in the order they were hidden
	closed  []window.Window // closed, F3 brings them back
	counter int
	notes   []rune
	dirty   bool
	logs    []string
	logOff  int
	secs    int
	moving  bool
	modal   *modal
	help    bool
	hover   string
	msg     string
	manual  bool // the caller sends the ticks

	width, height int
}

func newModel() model {
	m := model{}
	m.desk = window.New()
	m.desk.Mouse = true
	m.desk.Step = 2
	m.notes = []rune("Type here. Drag the window by its title bar.")
	return m.log("started")
}

func (m model) Init() tui.Cmd { return m.tick() }

func (m model) tick() tui.Cmd {
	if m.manual {
		return nil
	}
	return tui.FromCtx(motion.After(time.Second, func(time.Time) tui.Msg { return tickMsg{} }))
}

func (m model) size() layout.Size {
	if m.width == 0 {
		return layout.Size{W: 100, H: 30}
	}
	return layout.Size{W: m.width, H: m.height}
}

func (m model) deskRect() hittest.Rect {
	s := m.size()
	return hittest.Rect{X: 0, Y: 1, W: s.W, H: max(1, s.H-chromeRows)}
}

func (m model) clock() string {
	t := 9*3600 + m.secs
	return fmt.Sprintf("%02d:%02d:%02d", t/3600%24, t/60%60, t%60)
}

func (m model) log(s string) model {
	m.logs = append(append([]string(nil), m.logs...), m.clock()+"  "+s)
	if len(m.logs) > maxLog {
		m.logs = m.logs[len(m.logs)-maxLog:]
	}
	m.logOff = 0
	return m
}

func titleOf(id string) string {
	for _, a := range apps {
		if a.id == id {
			return a.title
		}
	}
	return id
}

// place is the starting rectangle of an app on a desktop of the given size.
func place(a app, d hittest.Rect) hittest.Rect {
	return hittest.Rect{X: d.W * a.x / 100, Y: d.H * a.y / 100, W: max(10, d.W*a.w/100), H: max(4, d.H*a.h/100)}
}

func (m model) open(a app) model {
	w := window.Window{ID: a.id, Title: a.title, Rect: place(a, m.deskRect()), MinW: 14, MinH: 4}
	m.desk.Add(w)
	return m
}

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		first := m.width == 0
		m.width, m.height = msg.Width, msg.Height
		m.desk.Bounds = m.deskRect()
		if first {
			for _, a := range apps {
				m = m.open(a)
			}
			m.desk.Raise("notes")
		}
		for _, w := range m.desk.Windows {
			m.desk.SetRect(w.ID, w.Rect)
		}
	case tickMsg:
		m.secs++
		return m, m.tick()
	case window.ChangedMsg:
		return m.log(fmt.Sprintf("%s moved to %d,%d size %dx%d", titleOf(msg.ID), msg.Rect.X, msg.Rect.Y, msg.Rect.W, msg.Rect.H)), nil
	case window.FocusedMsg:
		return m.log(titleOf(msg.ID) + " has focus"), nil
	case window.ClosedMsg:
		return m.afterClose(msg.ID), nil
	case tui.Key:
		return m.key(msg)
	case tui.MouseEvent:
		return m.mouse(msg)
	}
	return m, nil
}

// afterClose files a window the user closed with its close box, so F3 can bring
// it back.
func (m model) afterClose(id string) model {
	for _, a := range apps {
		if a.id == id {
			m.closed = append(append([]window.Window(nil), m.closed...), window.Window{ID: a.id, Title: a.title, Rect: place(a, m.deskRect()), MinW: 14, MinH: 4})
		}
	}
	return m.log(titleOf(id) + " closed")
}

func (m model) closeWindow(id string) model {
	w, ok := m.desk.Window(id)
	if !ok {
		return m
	}
	m.desk.Remove(id)
	m.closed = append(append([]window.Window(nil), m.closed...), w)
	if id == "notes" {
		m.dirty = false
	}
	return m.log(titleOf(id) + " closed")
}

// askClose closes the window, after asking if it holds text that would be lost.
func (m model) askClose(id string) model {
	if id == "notes" && m.dirty {
		m.modal = &modal{title: "Close Notes?", text: "The text you typed will be lost.", yes: "close:notes"}
		return m
	}
	return m.closeWindow(id)
}

func (m model) reopen() model {
	if len(m.closed) == 0 {
		m.msg = "No closed window to bring back"
		return m
	}
	w := m.closed[0]
	m.closed = slices.Clone(m.closed[1:])
	m.desk.Add(w)
	return m.log(w.Title + " opened")
}

func (m model) hide(id string) model {
	w, ok := m.desk.Window(id)
	if !ok {
		return m
	}
	m.desk.Remove(id)
	m.hidden = append(append([]window.Window(nil), m.hidden...), w)
	return m.log(w.Title + " hidden")
}

func (m model) show(id string) model {
	for i, w := range m.hidden {
		if w.ID == id {
			m.hidden = slices.Delete(slices.Clone(m.hidden), i, i+1)
			m.desk.Add(w)
			return m.log(w.Title + " shown")
		}
	}
	return m
}

func itoa(n int) string { return strconv.Itoa(n) }

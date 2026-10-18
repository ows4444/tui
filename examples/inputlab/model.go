package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/examples/internal/ui"
	"github.com/ows4444/tui/layout"
)

const (
	maxLog     = 500
	chromeRows = 3 // top bar, status, hints
	detailW    = 40
)

type entry struct {
	n     int
	kind  kind
	text  string
	count int // events joined into this line
	gap   time.Duration
}

type model struct {
	th       ui.Theming // the theme the Program chose; see SetTheme
	log      []entry
	total    int
	counts   [numKinds]int
	shown    [numKinds]bool
	coalesce bool // join runs of mouse motion into one line
	paused   bool
	frozen   int // entries visible while paused
	back     int // rows scrolled back from the newest, while paused
	help     bool

	lastKey, lastMouse, lastPaste string
	focus                         string
	scheme                        string
	caps                          *tui.Capabilities
	last                          time.Time
	now                           func() time.Time

	width, height int
}

func newModel() model {
	return model{shown: [numKinds]bool{true, true, true, true}, coalesce: true, focus: "unknown", scheme: "unknown", now: time.Now}
}

func (m model) Init() tui.Cmd { return nil }

func (m model) size() layout.Size {
	if m.width == 0 {
		return layout.Size{W: 100, H: 30}
	}
	return layout.Size{W: m.width, H: m.height}
}

func (m model) logRows() int { return max(1, m.size().H-chromeRows) }

// record adds a line to the log, or joins it to the last one when it is
// motion and joining is on.
func (m model) record(k kind, text string, join bool) model {
	m.total++
	m.counts[k]++
	at := m.now()
	var gap time.Duration
	if !m.last.IsZero() {
		gap = at.Sub(m.last)
	}
	m.last = at
	log := m.log
	if n := len(log); join && n > 0 && log[n-1].kind == k && log[n-1].count > 0 && strings.HasPrefix(log[n-1].text, "motion") && strings.HasPrefix(text, "motion") {
		log = append([]entry(nil), log...)
		log[n-1].text, log[n-1].count = text, log[n-1].count+1
		m.log = log
		return m
	}
	log = append(log[:len(log):len(log)], entry{n: m.total, kind: k, text: text, count: 1, gap: gap})
	if len(log) > maxLog {
		log = log[len(log)-maxLog:]
		m.frozen = max(0, m.frozen-1)
	}
	m.log = log
	return m
}

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m.record(kOther, fmt.Sprintf("resize  %dx%d", msg.Width, msg.Height), false), nil
	case tui.Key:
		return m.key(msg)
	case tui.KeyRepeatMsg:
		m.lastKey = keyText("repeat", msg.Key)
		return m.record(kKey, m.lastKey, false), nil
	case tui.KeyReleaseMsg:
		m.lastKey = keyText("release", msg.Key)
		return m.record(kKey, m.lastKey, false), nil
	case tui.MouseEvent:
		return m.mouse(msg), nil
	case tui.PasteEvent:
		m.lastPaste = pasteText(msg)
		return m.record(kPaste, m.lastPaste, false), nil
	case tui.FocusEvent:
		m.focus = map[bool]string{true: "in", false: "out"}[msg.Focused]
		return m.record(kOther, "focus   "+m.focus, false), nil
	case tui.ColorSchemeEvent:
		m.scheme = map[bool]string{true: "dark", false: "light"}[msg.Dark]
		return m.record(kOther, "scheme  "+m.scheme, false), nil
	case tui.CapabilitiesMsg:
		c := msg.Capabilities
		m.caps = &c
		return m.record(kOther, "caps    probe finished", false), nil
	case tui.InputErrorMsg:
		return m.record(kOther, "error   "+msg.Err.Error(), false), nil
	}
	return m.record(kOther, fmt.Sprintf("other   %T", msg), false), nil
}

func (m model) mouse(ev tui.MouseEvent) model {
	if ev.Action == tui.MouseActionPress && ev.Button == tui.MouseButtonLeft {
		if id := m.hitButton(ev.X, ev.Y); id != "" {
			m = m.record(kMouse, mouseText(ev), false)
			return m.click(id)
		}
	}
	text := mouseText(ev)
	m.lastMouse = text
	join := m.coalesce && ev.Action == tui.MouseActionMotion
	return m.record(kMouse, text, join)
}

func (m model) click(id string) model {
	switch {
	case id == "pause":
		return m.togglePause()
	case id == "clear":
		return m.clear()
	case id == "coalesce":
		m.coalesce = !m.coalesce
	case len(id) > 5 && id[:5] == "kind-":
		n, _ := strconv.Atoi(id[5:])
		m.shown[n] = !m.shown[n]
	}
	return m
}

func (m model) togglePause() model {
	m.paused = !m.paused
	m.frozen, m.back = len(m.log), 0
	return m
}

func (m model) clear() model {
	m.log, m.frozen, m.back = nil, 0, 0
	m.total, m.counts = 0, [numKinds]int{}
	return m
}

// key logs the key, then does what a reserved key asks.
func (m model) key(k tui.Key) (tui.Model, tui.Cmd) {
	m.lastKey = keyText("press", k)
	m = m.record(kKey, m.lastKey, false)
	if m.help && k.Type != tui.KeyF1 {
		return m, nil
	}
	switch {
	case k.Type == tui.KeyCtrlC:
		return m, tui.Quit()
	case k.Type == tui.KeyCtrl && k.Code == 'p':
		return m.togglePause(), nil
	case k.Type == tui.KeyCtrl && k.Code == 'l':
		return m.clear(), nil
	case k.Type == tui.KeyF1:
		m.help = !m.help
	case k.Type == tui.KeyF2, k.Type == tui.KeyF3, k.Type == tui.KeyF4, k.Type == tui.KeyF5:
		n := map[tui.KeyType]kind{tui.KeyF2: kKey, tui.KeyF3: kMouse, tui.KeyF4: kPaste, tui.KeyF5: kOther}[k.Type]
		m.shown[n] = !m.shown[n]
	case k.Type == tui.KeyF6:
		m.coalesce = !m.coalesce
	case k.Type == tui.KeyPgUp, k.Type == tui.KeyPgDown:
		return m.page(k.Type == tui.KeyPgUp), nil
	}
	return m, nil
}

// page scrolls the log; scrolling back pauses it.
func (m model) page(up bool) model {
	if !m.paused {
		m = m.togglePause()
	}
	step := max(1, m.logRows()-1)
	if !up {
		step = -step
	}
	m.back = max(0, min(max(0, len(m.visible(true))-m.logRows()), m.back+step))
	return m
}

// visible is the log lines that pass the filters; while paused and frozen it
// stops at the point where the pause began.
func (m model) visible(frozen bool) []entry {
	src := m.log
	if frozen && m.paused {
		src = src[:min(m.frozen, len(src))]
	}
	var out []entry
	for _, e := range src {
		if m.shown[e.kind] {
			out = append(out, e)
		}
	}
	return out
}

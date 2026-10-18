package main

import (
	"strconv"
	"strings"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/clockview"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/motion"
)

type tab int

const (
	tStopwatch tab = iota
	tTimer
	tClock
	numTabs
)

var tabNames = [numTabs]string{"Stopwatch", "Timer", "Clock"}

const (
	tickEvery  = 50 * time.Millisecond
	chromeRows = 3 // tabs, status, hints
)

var presets = []int{1, 5, 10, 25} // timer lengths in minutes

type zone struct {
	name string
	clk  clockview.Model
}

type tickMsg struct{ seq int }

type model struct {
	tab   tab
	now   func() time.Time
	w     watch
	seq   int // tells a stale stopwatch tick from the live one
	lapAt int // laps scrolled off the top

	timer   clockview.Model
	minutes int
	started bool // the timer has been started since it was last reset
	zones   []zone
	boot    []tui.Cmd

	help  bool
	hover string
	msg   string

	width, height int
}

func newModel() model { return newModelAt(time.Now) }

func newModelAt(now func() time.Time) model {
	m := model{now: now, minutes: 5}
	m.timer = m.newTimer()
	for _, z := range []struct {
		name string
		loc  *time.Location
	}{{"Here", nil}, {"UTC", time.UTC}, {"New York", time.FixedZone("EST", -5*3600)}, {"Tokyo", time.FixedZone("JST", 9*3600)}} {
		loc := z.loc
		c := clockview.New(clockview.ModeClock)
		c.Layout = "15:04:05"
		c.Now = func() time.Time {
			if loc == nil {
				return now()
			}
			return now().In(loc)
		}
		m.boot = append(m.boot, c.Start())
		m.zones = append(m.zones, zone{z.name, c})
	}
	return m
}

func (m model) newTimer() clockview.Model {
	t := clockview.New(clockview.ModeTimer)
	t.Duration = time.Duration(m.minutes) * time.Minute
	return t
}

func (m model) Init() tui.Cmd { return tui.Batch(m.boot...) }

func (m model) size() layout.Size {
	if m.width == 0 {
		return layout.Size{W: 80, H: 24}
	}
	return layout.Size{W: m.width, H: m.height}
}

func (m model) tick() tui.Cmd {
	seq := m.seq
	return tui.FromCtx(motion.After(tickEvery, func(time.Time) tui.Msg { return tickMsg{seq} }))
}

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tickMsg:
		if msg.seq != m.seq || !m.w.running {
			return m, nil
		}
		return m, m.tick()
	case tui.Key:
		return m.key(msg)
	case tui.MouseEvent:
		return m.mouse(msg)
	}
	return m.clocks(msg)
}

// clocks hands a message to the timer and every clock, which tell their own
// ticks from anyone else's.
func (m model) clocks(msg tui.Msg) (tui.Model, tui.Cmd) {
	var cmds []tui.Cmd
	var cmd tui.Cmd
	m.timer, cmd = m.timer.Update(msg)
	cmds = append(cmds, cmd)
	zs := append([]zone(nil), m.zones...)
	for i := range zs {
		zs[i].clk, cmd = zs[i].clk.Update(msg)
		cmds = append(cmds, cmd)
	}
	m.zones = zs
	return m, tui.Batch(cmds...)
}

// ---- actions ---------------------------------------------------------------

func (m model) toggleWatch() (model, tui.Cmd) {
	if m.w.running {
		m.w = m.w.stop(m.now())
		m.seq++
		return m, nil
	}
	m.w = m.w.start(m.now())
	m.seq++
	return m, m.tick()
}

func (m model) lap() model {
	if m.w.elapsed(m.now()) <= 0 {
		m.msg = "Start the stopwatch first"
		return m
	}
	m.w = m.w.lap(m.now())
	m.lapAt = 0
	return m
}

func (m model) resetWatch() model {
	m.w = m.w.reset()
	m.seq++
	m.lapAt = 0
	return m
}

// remaining reads what is left on the timer from its display.
func (m model) remaining() time.Duration {
	var h, mi, s int
	parts := strings.Split(m.timer.View(), ":")
	if len(parts) != 3 {
		return 0
	}
	h, _ = strconv.Atoi(parts[0])
	mi, _ = strconv.Atoi(parts[1])
	s, _ = strconv.Atoi(parts[2])
	return time.Duration(h*3600+mi*60+s) * time.Second
}

func (m model) timerDone() bool { return m.started && !m.timer.Running() && m.remaining() == 0 }

func (m model) toggleTimer() (model, tui.Cmd) {
	switch {
	case m.timer.Running():
		m.timer.Stop()
		return m, nil
	case m.timerDone():
		m = m.resetTimer()
	}
	m.started = true
	cmd := m.timer.Start() // Start changes m.timer, so it runs before m is returned
	return m, cmd
}

func (m model) resetTimer() model {
	m.timer.Stop()
	m.timer = m.newTimer()
	m.started = false
	return m
}

// setMinutes changes the timer length; it stops and resets the timer.
func (m model) setMinutes(n int) model {
	m.minutes = max(1, min(99, n))
	return m.resetTimer()
}

func (m model) nextPreset() model {
	for _, p := range presets {
		if p > m.minutes {
			return m.setMinutes(p)
		}
	}
	return m.setMinutes(presets[0])
}

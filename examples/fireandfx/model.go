package main

import (
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/motion"
	"github.com/ows4444/tui/motion/fx"
)

type scene int

const (
	sFire scene = iota
	sPlasma
	sType
	numScenes
)

var sceneNames = [numScenes]string{"Fire", "Plasma", "Typewriter"}

// frameTime is the time between frames; a transition lasts transitionTime.
const (
	frameTime      = 40 * time.Millisecond
	transitionTime = 600 * time.Millisecond
	chromeRows     = 3 // tabs, status, hints
)

type effect struct {
	name string
	fx   fx.Effect
}

var effects = []effect{
	{"Dissolve", fx.Dissolve{}},
	{"Fade", fx.Fade{}},
	{"Slide", fx.Slide{From: fx.FromRight}},
	{"Sweep", fx.Sweep{From: fx.FromLeft, Soft: 6}},
}

type tickMsg struct{}

type model struct {
	scene  scene
	fire   fire
	sim    int // plasma time, in steps
	chars  int // characters typed so far
	wait   int // frames the typewriter still waits
	frame  int // frames since the program started
	since  int // the frame the current scene was chosen on
	fx     int // index into effects
	speed  int // steps per frame, 1 to 4
	wind   int
	paused bool
	still  bool // reduced motion: no tick loop, finished frames
	pref   motion.Preference

	mx, my int // pointer position inside the body, for the plasma ripple
	hover  string
	help   bool
	msg    string

	width, height int
}

func newModel() model {
	pref := motion.Detect()
	// since starts far back so the first scene shows at once; only a change of
	// scene runs a transition.
	return model{speed: 1, pref: pref, still: pref.Reduced(), mx: -100, my: -100, since: -1 << 20}
}

func (m model) Init() tui.Cmd {
	if m.still {
		return nil
	}
	return m.tick()
}

func (m model) tick() tui.Cmd {
	return tui.FromCtx(motion.After(frameTime, func(time.Time) tui.Msg { return tickMsg{} }))
}

func (m model) size() layout.Size {
	if m.width == 0 {
		return layout.Size{W: 100, H: 30}
	}
	return layout.Size{W: m.width, H: m.height}
}

func (m model) bodySize() (w, h int) { return m.size().W, max(1, m.size().H-chromeRows) }

// progress is how far the transition into the current scene has got, 0 to 1.
func (m model) progress() float64 {
	return fx.Progress(time.Duration(m.frame-m.since)*frameTime, transitionTime, motion.OutCubic, m.pref)
}

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
		w, h := m.bodySize()
		if m.fire.w != w || m.fire.h != h {
			on := m.fire.on || m.fire.w == 0
			m.fire = newFire(w, h)
			m.fire.on = on
		}
		if m.still && m.chars == 0 {
			m.chars = scriptLen()
		}
	case tickMsg:
		m.frame++
		if !m.paused {
			for range m.speed {
				m = m.step()
			}
		}
		return m, m.tick()
	case tui.Key:
		return m.key(msg)
	case tui.MouseEvent:
		return m.mouse(msg)
	}
	return m, nil
}

// step moves the current scene one step on.
func (m model) step() model {
	switch m.scene {
	case sFire:
		m.fire = m.fire.step(m.wind)
	case sPlasma:
		m.sim++
	case sType:
		switch {
		case m.wait > 0:
			m.wait--
		case m.chars < scriptLen():
			m.wait = pause(m.chars)
			m.chars++
		}
	}
	return m
}

func (m model) setScene(s scene) model {
	if s == m.scene {
		return m
	}
	m.scene, m.since = s, m.frame
	if s == sType && m.still {
		m.chars = scriptLen()
	}
	return m
}

func (m model) restart() model {
	m.since = m.frame
	switch m.scene {
	case sFire:
		on := m.fire.on
		m.fire = newFire(m.fire.w, m.fire.h)
		m.fire.on = on
	case sPlasma:
		m.sim = 0
	case sType:
		m.chars, m.wait = 0, 0
		if m.still {
			m.chars = scriptLen()
		}
	}
	return m
}

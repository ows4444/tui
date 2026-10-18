package main

import (
	"strconv"
	"strings"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/motion"
	"github.com/ows4444/tui/spinner"
)

func itoa(n int) string { return strconv.Itoa(n) }

const tickEvery = 100 * time.Millisecond

type tickMsg struct{}

type model struct {
	b       build
	verbose bool
	still   bool // reduced motion: no spinner frames
	frames  []string
	help    bool
	width   int
	manual  bool // the caller sends the ticks; no timer runs
}

func newModel(jobs int, fail string) model {
	return model{b: newBuild(jobs, fail), verbose: true, still: motion.Detect().Reduced(), frames: spinner.Frames()}
}

func (m model) Init() tui.Cmd { return m.tick() }

func (m model) tick() tui.Cmd {
	if m.manual {
		return nil
	}
	return tui.FromCtx(motion.After(tickEvery, func(time.Time) tui.Msg { return tickMsg{} }))
}

// line is the scrollback line for a finished step.
func (m model) line(i int) string {
	s, j := m.b.steps[i], m.b.jobs[i]
	switch j.status {
	case done:
		return "✓ " + pad(s.name, 16) + seconds(j.ended-j.started)
	case failed:
		return "✗ " + pad(s.name, 16) + seconds(j.ended-j.started) + "  failed"
	}
	return "· " + s.name
}

func pad(s string, w int) string {
	if len(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-len(s))
}

// commits are the scrollback lines for the steps that just ended, a failed
// step's output after its line.
func (m model) commits(ended []int) []tui.Cmd {
	var cmds []tui.Cmd
	for _, i := range ended {
		cmds = append(cmds, tui.Println(m.line(i)))
		if m.b.jobs[i].status == failed {
			for _, l := range m.b.jobs[i].log {
				cmds = append(cmds, tui.Println("    │ "+l))
			}
		}
	}
	return cmds
}

func (m model) summary() string {
	b := m.b
	took := seconds(b.tick)
	switch {
	case b.ok():
		return "Build succeeded in " + took + ": " + itoa(len(b.steps)) + " steps"
	case b.count(cancelled) > 0:
		return "Build cancelled after " + took + ": " + itoa(b.count(done)) + " steps finished, " + itoa(b.count(cancelled)) + " stopped"
	}
	return "Build failed after " + took + ": " + itoa(b.count(failed)) + " failed, " + itoa(b.count(skipped)) + " skipped, " + itoa(b.count(done)) + " ok"
}

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width = msg.Width
	case tickMsg:
		if m.b.over {
			return m, nil
		}
		var ended []int
		m.b, ended = m.b.advance()
		cmds := m.commits(ended)
		if m.b.over {
			// the summary goes to scrollback and the program ends
			cmds = append(cmds, m.printSkipped()...)
			cmds = append(cmds, tui.Println(m.summary()), tui.Quit())
			return m, tui.Sequence(cmds...)
		}
		if t := m.tick(); t != nil {
			cmds = append(cmds, t)
		}
		if len(cmds) == 0 {
			return m, nil
		}
		return m, tui.Sequence(cmds...)
	case tui.Key:
		return m.key(msg)
	}
	return m, nil
}

// printSkipped lists the steps that never ran, once, when the build ends.
func (m model) printSkipped() []tui.Cmd {
	var cmds []tui.Cmd
	for i, j := range m.b.jobs {
		switch j.status {
		case skipped:
			cmds = append(cmds, tui.Println("- "+pad(m.b.steps[i].name, 16)+"skipped: "+j.reason+" did not finish"))
		case cancelled:
			cmds = append(cmds, tui.Println("- "+pad(m.b.steps[i].name, 16)+"cancelled"))
		}
	}
	return cmds
}

func (m model) key(k tui.Key) (tui.Model, tui.Cmd) {
	if m.help && k.Type != tui.KeyCtrlC {
		m.help = k.Type != tui.KeyEsc && k.Type != tui.KeyEnter && !(k.Type == tui.KeyRunes && (k.Text == "?" || k.Text == "q"))
		return m, nil
	}
	switch {
	case k.Type == tui.KeyCtrlC:
		if m.b.over {
			return m, tui.Quit()
		}
		m.b = m.b.cancel()
		cmds := append(m.printSkipped(), tui.Println(m.summary()), tui.Quit())
		return m, tui.Sequence(cmds...)
	case k.Type == tui.KeyEnter && m.b.over:
		return m, tui.Quit()
	case k.Type == tui.KeyRunes:
		return m.runeKey(k.Text)
	}
	return m, nil
}

func (m model) runeKey(r string) (tui.Model, tui.Cmd) {
	switch r {
	case "q":
		if m.b.over {
			return m, tui.Quit()
		}
	case "?":
		m.help = true
	case "v":
		m.verbose = !m.verbose
	case "+", "=":
		m.b.max = min(len(m.b.steps), m.b.max+1)
		m.b = m.b.startReady()
	case "-":
		m.b.max = max(1, m.b.max-1)
	}
	return m, nil
}

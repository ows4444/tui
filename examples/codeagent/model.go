package main

import (
	"strconv"
	"strings"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/motion"
)

func itoa(n int) string { return strconv.Itoa(n) }

type blockKind int

const (
	bUser blockKind = iota
	bAgent
	bTool
	bNote
)

type status int

const (
	pending status = iota
	running
	done
	failed
	denied
)

// block is one item of the transcript.
type block struct {
	kind   blockKind
	text   string // what was said; for an agent block, all of it
	shown  int    // runes of an agent block revealed so far
	t      tool
	st     status
	ticks  int
	open   bool // the card shows all its output and diff
	auto   bool // it ran without asking
	result []string
}

type phase int

const (
	idle phase = iota
	thinking
	saying
	toolRunning
	waiting // a call needs approval
)

const (
	tickEvery  = 40 * time.Millisecond
	thinkTicks = 8
	toolTicks  = 10
	revealRate = 3 // runes per tick
	chromeRows = 4 // title, status, prompt, hints
)

type tickMsg struct{}

type model struct {
	blocks  []block
	plan    []planStep
	pc      int // next plan step to start
	phase   phase
	wait    int
	w       world
	always  map[string]bool
	tokens  int
	prompt  []rune
	cur     int
	hist    []string
	histPos int // index into hist while browsing, or len(hist)
	draft   string
	off     int // lines scrolled up from the newest
	still   bool
	manual  bool // the caller sends the ticks
	help    bool
	hover   string
	msg     string

	width, height int
}

func newModel() model {
	m := model{always: map[string]bool{}, still: motion.Detect().Reduced()}
	return m.note("Ask for something: \"fix the failing test\", \"delete the build directory\" or \"explain the layout package\". /help lists the commands.")
}

func (m model) Init() tui.Cmd { return nil }

func (m model) size() layout.Size {
	if m.width == 0 {
		return layout.Size{W: 100, H: 30}
	}
	return layout.Size{W: m.width, H: m.height}
}

func (m model) busy() bool { return m.phase != idle && m.phase != waiting }

func (m model) tick() tui.Cmd {
	if m.manual {
		return nil
	}
	return tui.FromCtx(motion.After(tickEvery, func(time.Time) tui.Msg { return tickMsg{} }))
}

func (m model) add(b block) model {
	m.blocks = append(append([]block(nil), m.blocks...), b)
	m.off = 0
	return m
}

func (m model) edit(i int, f func(*block)) model {
	bs := append([]block(nil), m.blocks...)
	f(&bs[i])
	m.blocks = bs
	return m
}

func (m model) note(s string) model { return m.add(block{kind: bNote, text: s}) }

func (m model) pendingIdx() int {
	for i := len(m.blocks) - 1; i >= 0; i-- {
		if b := m.blocks[i]; b.kind == bTool && b.st == pending {
			return i
		}
	}
	return -1
}

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tickMsg:
		m = m.step()
		if m.busy() {
			return m, m.tick()
		}
		return m, nil
	case tui.PasteEvent:
		return m.paste(msg), nil
	case tui.Key:
		return m.key(msg)
	case tui.MouseEvent:
		return m.mouse(msg)
	}
	return m, nil
}

// ---- the script runner -----------------------------------------------------

func (m model) send(text string) (model, tui.Cmd) {
	m = m.add(block{kind: bUser, text: text})
	m.tokens += len(text)/4 + 1
	m.plan, m.pc, m.phase, m.wait = planFor(text), 0, thinking, thinkTicks
	if m.still {
		m = m.settle()
		return m, nil
	}
	return m, m.tick()
}

// settle runs the script without waiting, up to the next approval or the end.
func (m model) settle() model {
	for n := 0; m.busy() && n < 10000; n++ {
		m = m.step()
	}
	return m
}

// step moves the script one tick on.
func (m model) step() model {
	switch m.phase {
	case thinking:
		if m.wait--; m.wait <= 0 {
			return m.next()
		}
	case saying:
		i := len(m.blocks) - 1
		m = m.edit(i, func(b *block) { b.shown = min(len([]rune(b.text)), b.shown+revealRate) })
		if m.blocks[i].shown >= len([]rune(m.blocks[i].text)) {
			return m.next()
		}
	case toolRunning:
		i := m.runningIdx()
		m = m.edit(i, func(b *block) { b.ticks++ })
		if m.blocks[i].ticks >= toolTicks {
			return m.finishTool(i)
		}
	}
	return m
}

func (m model) runningIdx() int {
	for i := len(m.blocks) - 1; i >= 0; i-- {
		if b := m.blocks[i]; b.kind == bTool && b.st == running {
			return i
		}
	}
	return -1
}

// next starts the next plan step, or goes idle when there is none.
func (m model) next() model {
	if m.pc >= len(m.plan) {
		m.phase = idle
		return m
	}
	s := m.plan[m.pc]
	m.pc++
	if s.kind == sSay {
		text := s.say(m.w)
		m.tokens += len(text) / 4
		m = m.add(block{kind: bAgent, text: text})
		if m.still {
			m = m.edit(len(m.blocks)-1, func(b *block) { b.shown = len([]rune(b.text)) })
			return m.next()
		}
		m.phase = saying
		return m
	}
	b := block{kind: bTool, t: s.tool, st: pending}
	if s.tool.risk == low || m.always[s.tool.name] {
		b.st, b.auto = running, true
	}
	m = m.add(b)
	if b.st == pending {
		m.phase = waiting
		return m
	}
	m.phase = toolRunning
	if m.still {
		return m.finishTool(len(m.blocks) - 1)
	}
	return m
}

func (m model) finishTool(i int) model {
	t := m.blocks[i].t
	if t.effect != nil {
		m.w = t.effect(m.w)
	}
	st := done
	if t.fails != nil && t.fails(m.w) {
		st = failed
	}
	res := t.result(m.w)
	m = m.edit(i, func(b *block) { b.st, b.result = st, res })
	return m.next()
}

// decide answers the waiting call.
func (m model) decide(approve, always bool) (model, tui.Cmd) {
	i := m.pendingIdx()
	if i < 0 {
		return m, nil
	}
	name := m.blocks[i].t.name
	if !approve {
		m = m.edit(i, func(b *block) { b.st = denied })
		m.plan, m.phase = nil, idle
		m = m.add(block{kind: bAgent, text: "Understood, I won't run " + name + ". Nothing was changed.", shown: 1 << 20})
		return m, nil
	}
	if always && m.blocks[i].t.risk == high {
		always = false
		m = m.note("High-risk calls always ask; approved this once")
	}
	if always {
		al := map[string]bool{name: true}
		for k := range m.always {
			al[k] = true
		}
		m.always = al
		m = m.note("From now on " + name + " runs without asking (/reset undoes this)")
	}
	m = m.edit(i, func(b *block) { b.st = running })
	m.phase = toolRunning
	if m.still {
		m = m.finishTool(i).settle()
		return m, nil
	}
	return m, m.tick()
}

// interrupt stops whatever the agent is doing.
func (m model) interrupt() model {
	switch m.phase {
	case idle:
		return m
	case waiting:
		if i := m.pendingIdx(); i >= 0 {
			m = m.edit(i, func(b *block) { b.st = denied })
		}
	case toolRunning:
		if i := m.runningIdx(); i >= 0 {
			m = m.edit(i, func(b *block) { b.st, b.result = failed, []string{"interrupted"} })
		}
	case saying:
		i := len(m.blocks) - 1
		m = m.edit(i, func(b *block) { b.text = string([]rune(b.text)[:b.shown]) })
	}
	m.plan, m.phase = nil, idle
	return m.note("Stopped")
}

func (m model) toggleDiff() model {
	i := m.pendingIdx()
	if i < 0 {
		for j := len(m.blocks) - 1; j >= 0; j-- {
			if m.blocks[j].kind == bTool {
				i = j
				break
			}
		}
	}
	if i < 0 {
		return m
	}
	return m.edit(i, func(b *block) { b.open = !b.open })
}

func (m model) command(line string) (model, tui.Cmd) {
	switch strings.Fields(line)[0] {
	case "/help":
		m.help = true
	case "/clear":
		m.blocks, m.off = nil, 0
	case "/history":
		if len(m.hist) == 0 {
			return m.note("No earlier prompts"), nil
		}
		return m.note("Earlier prompts: " + strings.Join(m.hist, "  ·  ")), nil
	case "/allow":
		var names []string
		for k := range m.always {
			names = append(names, k)
		}
		if len(names) == 0 {
			return m.note("Nothing is always allowed; every call that changes things asks"), nil
		}
		return m.note("Always allowed: " + strings.Join(names, ", ")), nil
	case "/reset":
		m.w, m.always = world{}, map[string]bool{}
		return m.note("Project and permissions put back"), nil
	case "/quit":
		return m, tui.Quit()
	default:
		return m.note("Unknown command " + strings.Fields(line)[0] + "; /help lists them"), nil
	}
	return m, nil
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// Command avatar is a gallery for the avatar package: a wall of names, each
// drawn as its avatar, at a size you can change. Every avatar watches the
// mouse pointer, and the selected one blinks whenever a key is pressed.
//
//	mouse       the avatars look at the pointer
//	click       the avatar under the pointer pulls a face, and is selected
//	r           the selected avatar pulls a face
//	space       blink (so does every other key)
//	←/→ or h/l  previous / next name
//	+/- or ↑/↓  larger / smaller avatars
//	b           cycle the background: none, squircle, circle, square
//	e           ease every avatar into the next expression: happy, sad, mad, ...
//	i           idle: off, on hover (only the avatar under the pointer), always
//	c           pin the hue: name's own, 30, 90, 140, 250, 320
//	t           pin the tone: name's own, pastel, pale, mid, deep, bright, ink
//	s           pin the silhouette: name's own, round, organic, ...
//	p           pin traits: none, big eyes, small wide-set eyes, big square bodies
//	a           ASCII glyphs on / off
//	g           draw real images in place of cells, on a terminal that can:
//	            kitty graphics, else iTerm2's inline images, else Sixel;
//	            cells otherwise. The status line names the one in use.
//
// Images are drawn through imageview, which tries those three protocols in
// that order; its package comment says what each costs. Sixel is the slow
// one: every changed picture is reduced to 256 colours before it is sent.
// A terminal that reports none of them keeps the cells, and g says so.
//
//	q, esc      quit
package main

import (
	"fmt"
	"math"
	"os"
	"strings"
	"sync"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/avatar"
	"github.com/ows4444/tui/imageview"
	"github.com/ows4444/tui/theme"
)

// names is the wall. Any string draws an avatar; these are ordinary handles.
var names = []string{
	"ada", "linus", "grace", "ken", "margaret", "dennis", "barbara", "alan",
	"hedy", "edsger", "radia", "donald", "frances", "bjarne", "katherine", "guido",
	"sophie", "tim", "annie", "rob", "joan", "brian", "karen", "niklaus",
	"evelyn", "john", "jean", "claude", "mary", "vint", "lynn", "whitfield",
	"shafi", "leslie", "adele", "butler", "anita", "andrew", "carol", "martin",
}

// sizes are the avatar sizes, in cells, that + and - step through. A cell is
// about twice as tall as it is wide, so each is square on screen.
var sizes = [][2]int{{4, 2}, {6, 3}, {8, 4}, {12, 6}, {16, 8}, {24, 12}}

var backgrounds = []struct {
	bg   avatar.Background
	name string
}{
	{avatar.BackgroundNone, "none"},
	{avatar.BackgroundSquircle, "squircle"},
	{avatar.BackgroundCircle, "circle"},
	{avatar.BackgroundSquare, "square"},
}

// presets are the trait pins p steps through.
var presets = []struct {
	name string
	pins map[avatar.Trait]float64
}{
	{"own", nil},
	{"big eyes", map[avatar.Trait]float64{avatar.TraitEyeSize: 1, avatar.TraitEyeRoundness: 0}},
	{"wide-set", map[avatar.Trait]float64{avatar.TraitEyeSize: 0, avatar.TraitEyeSeparation: 1}},
	{"square", map[avatar.Trait]float64{avatar.TraitBodySize: 1, avatar.TraitBodySquareness: 1}},
}

// hues are the hues c steps through; 0 is each name's own.
var hues = []float64{0, 30, 90, 140, 250, 320}

const (
	tileGap    = 2 // columns between tiles
	minTile    = 9 // a tile is at least this wide, so a name fits under it
	chromeRows = 4 // title, blank, status, help
	lookSteps  = 3 // steps each way the eyes turn through in image mode
	wallTop    = 2 // the row the wall starts on, under the title
	defaultW   = 80
	defaultH   = 24
)

type model struct {
	selected      int
	size          int // index into sizes
	bg            int // index into backgrounds
	ascii         bool
	expression    avatar.Expression
	hue           int // index into hues
	tone          avatar.Tone
	silhouette    avatar.Silhouette
	preset        int // index into presets
	width, height int
	// wall holds one avatar per name, built once, so each keeps its cached
	// View and its own animation from frame to frame.
	wall []avatar.Model
	// images holds one image view per name, built once, so each keeps its
	// encoded picture until the avatar's PNG changes.
	images []imageview.Model
	// kitty, inline and sixel are what the terminal said it can draw, and
	// graphics whether g has asked for images.
	kitty, inline, sixel, graphics bool
	idle                           int // index into idleModes
	// mouseX and mouseY are the pointer's cell, once it has moved.
	mouseX, mouseY int
	mouse          bool
}

func initialModel() model {
	m := model{size: 2, wall: make([]avatar.Model, len(names)), images: make([]imageview.Model, len(names))}
	for i, name := range names {
		m.wall[i] = avatar.New(name)
		m.images[i] = imageview.New(nil, 0, 0, "")
		m.images[i].ID = uint32(i + 1) // #nosec G115 -- a few dozen names
	}
	return m
}

func (m model) Init() tui.Cmd { return nil }

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tui.CapabilitiesMsg:
		m.kitty, m.inline, m.sixel = msg.Capabilities.KittyGraphics, msg.Capabilities.InlineImages, msg.Capabilities.Sixel
		return m, nil
	case tui.MouseEvent:
		m.mouseX, m.mouseY, m.mouse = msg.X, msg.Y, true
		if msg.Action == tui.MouseActionPress && msg.Button == tui.MouseButtonLeft {
			if i, ok := m.tileAt(msg.X, msg.Y); ok {
				m.selected = i
				return m, m.wall[i].React()
			}
		}
		return m, m.hover()
	case tui.Key:
		switch msg.Type {
		case tui.KeyCtrlC, tui.KeyEsc:
			return m, tui.Quit()
		case tui.KeyLeft:
			m.move(-1)
		case tui.KeyRight:
			m.move(1)
		case tui.KeyUp:
			m.resize(1)
		case tui.KeyDown:
			m.resize(-1)
		case tui.KeyRunes:
			switch msg.Text {
			case "q":
				return m, tui.Quit()
			case "h":
				m.move(-1)
			case "l":
				m.move(1)
			case "+", "=":
				m.resize(1)
			case "-":
				m.resize(-1)
			case "b":
				m.bg = (m.bg + 1) % len(backgrounds)
			case "g":
				m.graphics = !m.graphics
			case "a":
				m.ascii = !m.ascii
			case "e":
				// Every avatar eases into the next expression.
				m.expression = (m.expression + 1) % (avatar.ExpressionSick + 1)
				var cmds []tui.Cmd
				for i := range m.wall {
					cmds = append(cmds, m.wall[i].SetExpression(m.expression))
				}
				return m, tui.Batch(cmds...)
			case "c":
				m.hue = (m.hue + 1) % len(hues)
			case "t":
				m.tone = (m.tone + 1) % (avatar.ToneInk + 1)
			case "s":
				m.silhouette = (m.silhouette + 1) % (avatar.SilhouetteTriangle + 1)
			case "p":
				m.preset = (m.preset + 1) % len(presets)
			case "r":
				return m, m.wall[m.selected].React()
			case "i":
				return m, m.toggleIdle()
			}
		}
		// Whatever the key did, the selected avatar blinks at it.
		return m, m.wall[m.selected].Blink()
	}
	// Every avatar sees every other Msg; a tick moves only the one that
	// scheduled it.
	var cmds []tui.Cmd
	for i := range m.wall {
		var cmd tui.Cmd
		if m.wall[i], cmd = m.wall[i].Update(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return m, batch(cmds)
}

// drawing names how the wall is drawn: as images through a graphics protocol
// when g asked for them and the terminal has one, and as cells otherwise.
func (m model) drawing() string {
	switch {
	case m.graphics && m.kitty:
		return "kitty"
	case m.graphics && m.inline:
		return "inline"
	case m.graphics && m.sixel:
		return "sixel"
	case m.graphics:
		return "no images"
	case m.ascii:
		return "ascii"
	}
	return "blocks"
}

// picture returns avatar a of name i as rows of cells: its View, or, when
// the wall is drawn as images, an image view of its PNG that fills the same
// cells.
func (m model) picture(i int, a avatar.Model) string {
	if !m.pictures() {
		return a.View()
	}
	img := m.images[i]
	// A Sixel image is scaled to its cells at imageview's cell size, so it
	// is drawn at exactly that many pixels; with kitty graphics and inline
	// images the terminal scales the PNG itself, and gets a sharper one. A
	// Model from New returns the same slice for the same picture, so the
	// image is encoded again only when the avatar changes.
	px := imageview.DefaultCellWidth * a.Width
	if m.kitty || m.inline {
		px = min(256, 12*a.Width)
	}
	img.PNG = a.PNG(px)
	img.Width, img.Height, img.Alt = a.Width, a.Height, a.Linearize()
	img.Kitty, img.Inline, img.Sixel = m.kitty, m.inline, m.sixel
	return img.View()
}

// pictures reports whether the wall is drawn as images: g asked for them and
// the terminal has a way to draw one.
func (m model) pictures() bool { return m.graphics && (m.kitty || m.inline || m.sixel) }

// tile is where one name's avatar goes on the page.
type tile struct{ name, x, y int }

// draw returns the picture of each tile. Every avatar has a Model and an
// image view of its own, so with parallel set they are drawn at once, one
// goroutine each: a wall of images is redrawn on every core.
func (m model) draw(tiles []tile, parallel bool) []string {
	out := make([]string, len(tiles))
	one := func(k int) { out[k] = m.picture(tiles[k].name, m.avatar(tiles[k].name, tiles[k].x, tiles[k].y)) }
	if !parallel || len(tiles) < 2 {
		for k := range tiles {
			one(k)
		}
		return out
	}
	var wg sync.WaitGroup
	for k := range tiles {
		wg.Add(1)
		go func() {
			defer wg.Done()
			one(k)
		}()
	}
	wg.Wait()
	return out
}

// tileAt returns the index of the name whose tile, avatar and label, holds
// cell (x, y) on the page that is showing.
func (m model) tileAt(x, y int) (int, bool) {
	cols, rows, tileW := m.grid()
	tileH := sizes[m.size][1] + 2
	c, r := x/(tileW+tileGap), (y-wallTop)/tileH
	if x < 0 || y < wallTop || c >= cols || r >= rows || x%(tileW+tileGap) >= tileW || (y-wallTop)%tileH == tileH-1 {
		return 0, false
	}
	perPage := cols * rows
	i := m.selected/perPage*perPage + r*cols + c
	return i, i < len(names)
}

// The idle modes i steps through.
const (
	idleOff = iota
	idleHover
	idleAlways
)

var idleModes = [...]string{idleOff: "still", idleHover: "hover", idleAlways: "idle"}

// toggleIdle steps to the next idle mode: every avatar at rest, only the one
// under the pointer idling, or all of them.
func (m *model) toggleIdle() tui.Cmd {
	m.idle = (m.idle + 1) % len(idleModes)
	var cmds []tui.Cmd
	for i := range m.wall {
		if m.idle == idleAlways {
			cmds = append(cmds, m.wall[i].StartIdle())
		} else {
			m.wall[i].StopIdle()
		}
	}
	return batch(append(cmds, m.hover()))
}

// hover, in the hover mode, tells every avatar whether the pointer is over
// its tile: the one it has reached starts to idle and the one it left stops.
func (m *model) hover() tui.Cmd {
	if m.idle != idleHover {
		return nil
	}
	at, ok := -1, false
	if m.mouse {
		if at, ok = m.tileAt(m.mouseX, m.mouseY); !ok {
			at = -1
		}
	}
	var cmds []tui.Cmd
	for i := range m.wall {
		cmds = append(cmds, m.wall[i].Hover(i == at))
	}
	return batch(cmds)
}

// batch returns the one Cmd of cmds that is not nil when there is exactly
// one, the usual case of a single avatar's next tick, and else a Batch.
func batch(cmds []tui.Cmd) tui.Cmd {
	var only tui.Cmd
	n := 0
	for _, c := range cmds {
		if c != nil {
			only = c
			n++
		}
	}
	if n == 1 {
		return only
	}
	return tui.Batch(cmds...)
}

func (m *model) move(d int) { m.selected = (m.selected + d + len(names)) % len(names) }

func (m *model) resize(d int) { m.size = min(max(m.size+d, 0), len(sizes)-1) }

func (m model) theme() theme.Theme {
	if m.ascii {
		return theme.DarkTheme().ASCII()
	}
	return theme.DarkTheme()
}

// avatar returns the avatar for names[i], whose tile's top-left cell is at
// (x, y), looking at the pointer.
func (m model) avatar(i, x, y int) avatar.Model {
	a := m.wall[i].SetTheme(m.theme())
	a.Width, a.Height = sizes[m.size][0], sizes[m.size][1]
	a.Background = backgrounds[m.bg].bg
	a.Hue, a.Tone, a.Silhouette = hues[m.hue], m.tone, m.silhouette
	a.Pins = presets[m.preset].pins
	if m.mouse {
		a.LookAt(m.mouseX-(x+a.Width/2), m.mouseY-(y+a.Height/2))
	}
	if m.pictures() {
		// An image is costly to redraw, and every pointer move turns every
		// avatar's eyes a little: turn them in steps, so an avatar is drawn
		// again only when its eyes have somewhere new to be.
		a.LookX, a.LookY = math.Round(a.LookX*lookSteps)/lookSteps, math.Round(a.LookY*lookSteps)/lookSteps
	}
	return a
}

// grid returns how many tiles fit across and down, and a tile's width. At
// least one tile is always drawn, clipped by the terminal if it must be.
func (m model) grid() (cols, rows, tileW int) {
	w, h := m.width, m.height
	if w <= 0 || h <= 0 {
		w, h = defaultW, defaultH
	}
	tileW = max(sizes[m.size][0], minTile)
	cols = max((w+tileGap)/(tileW+tileGap), 1)
	tileH := sizes[m.size][1] + 2 // the name, and a blank row below it
	rows = max((h-chromeRows)/tileH, 1)
	return cols, rows, tileW
}

var (
	titleStyle    = ansi.NewStyle().Bold().Foreground(ansi.BrightCyan)
	selectedStyle = ansi.NewStyle().Bold().Reverse()
	faintStyle    = ansi.NewStyle().Faint()
)

// centre pads s, which is w cells wide, to width cells.
func centre(s string, w, width int) string {
	left := max((width-w)/2, 0)
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", max(width-w-left, 0))
}

func hex(c ansi.RGB) string { return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B) }

func (m model) View() string {
	w := m.width
	if w <= 0 {
		w = defaultW
	}
	cols, rows, tileW := m.grid()
	perPage := cols * rows
	first := m.selected / perPage * perPage
	aw, ah := sizes[m.size][0], sizes[m.size][1]

	// Lay the page out, draw every tile's picture, then put them in place.
	var tiles []tile
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			if i := first + r*cols + c; i < len(names) {
				// The wall starts on row wallTop; a tile's avatar is
				// centred in it.
				tiles = append(tiles, tile{i, c*(tileW+tileGap) + max((tileW-aw)/2, 0), wallTop + r*(ah+2)})
			}
		}
	}
	drawn := m.draw(tiles, true)

	var b strings.Builder
	b.WriteString(titleStyle.Render(ansi.Truncate("Avatars: every name draws its own", w)))
	b.WriteString("\n\n")
	for r := 0; r < rows; r++ {
		lines := make([]string, ah+1)
		for c := 0; c < cols; c++ {
			i := first + r*cols + c
			if i >= len(names) {
				break
			}
			gap := ""
			if c > 0 {
				gap = strings.Repeat(" ", tileGap)
			}
			for y, row := range strings.Split(drawn[i-first], "\n") {
				lines[y] += gap + centre(row, aw, tileW)
			}
			label := ansi.Truncate(names[i], tileW)
			lw := ansi.Width(label)
			if i == m.selected {
				label = selectedStyle.Render(label)
			}
			lines[ah] += gap + centre(label, lw, tileW)
		}
		if lines[0] == "" {
			break
		}
		for i := range lines {
			b.WriteString(ansi.Truncate(lines[i], w))
			b.WriteByte('\n')
		}
		b.WriteByte('\n')
	}

	sel := m.avatar(m.selected, 0, 0)
	body, eyes, _ := sel.Colors()
	glyphs := m.drawing()
	status := fmt.Sprintf("%s: %s, body %s, eyes %s  [%dx%d, %s, %s, %s, %s, %s]",
		names[m.selected], sel.Shape(), hex(body), hex(eyes), aw, ah, backgrounds[m.bg].name, glyphs, m.expression, presets[m.preset].name, idleModes[m.idle])
	b.WriteString(ansi.Truncate(status, w))
	b.WriteByte('\n')
	b.WriteString(faintStyle.Render(ansi.Truncate("click or r react  ←/→ name  +/- size  g images  b bg  e expression  i idle  c hue  t tone  s shape  p pins  a ascii  q quit", w)))
	return b.String()
}

func main() {
	if _, err := tui.NewProgram(initialModel(), tui.WithMouse(tui.MouseAllMotion), tui.WithCapabilityProbe(0)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// Command faces is a gallery for the faces package. Faces rest on their
// first frame; click one to play its animation once.
//
//	click       play that face (the sprite, or any face on the wall)
//	←/→ or h/l  previous / next face (page, in the wall)
//	space       single view: loop on/off · wall: play the whole page
//	w           toggle the wall
//	s           small / large faces
//	q, esc      quit
//
// With NO_ANIMATION set nothing plays; every face stays on its first frame.
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/faces"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/motion"
	"github.com/ows4444/tui/theme"
)

const (
	maxWallCols = 4 // columns when the terminal is wide enough
	wallRows    = 2
	wallGap     = 2
)

// playMsg advances one face on the wall. gen ties it to the click that
// started it, so restarting a face mid-loop drops the old ticker instead of
// running two at once.
type playMsg struct{ face, gen int }

type play struct{ frame, gen int }

type model struct {
	player faces.Model
	wall   bool
	page   int
	plays  map[int]play // wall faces currently animating, by catalog index
	serial int          // last gen handed out
	still  bool         // NO_ANIMATION: never play anything
	width  int          // terminal width from the last ResizeMsg; 0 until known
	t      theme.Theme
}

func initialModel() model {
	m := model{t: theme.DarkTheme(), still: motion.Detect().Reduced(), plays: map[int]play{}}
	m.player = faces.New()
	m.player.Theme = m.t
	m.player.ShowLabel = true
	return m
}

func (m model) Init() tui.Cmd { return nil }

// cell is the size of one sprite in terminal cells.
func (m model) cell() (w, h int) { return m.player.Size.Cells() }

// cols is how many faces fit across the wall: as many as the terminal
// holds, up to maxWallCols, and at least one.
func (m model) cols() int {
	if m.width <= 0 {
		return maxWallCols
	}
	w, _ := m.cell()
	return min(maxWallCols, max(1, (m.width+wallGap)/(w+wallGap)))
}

func (m model) perPage() int { return m.cols() * wallRows }

func (m model) pages() int { return (faces.Count() + m.perPage() - 1) / m.perPage() }

func isRune(k tui.Key, r rune) bool {
	return k.Type == tui.KeyRunes && k.Text == string(r)
}

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		// Keep the same faces in view when the page size changes.
		first := m.page * m.perPage()
		m.width = msg.Width
		m.page = min(first/m.perPage(), m.pages()-1)
		m.plays = map[int]play{}
		return m, nil

	case playMsg:
		return m.advance(msg)

	case tui.MouseEvent:
		if msg.Action == tui.MouseActionPress && msg.Button == tui.MouseButtonLeft {
			return m.click(msg.X, msg.Y)
		}
		return m, nil

	case tui.Key:
		switch {
		case msg.Type == tui.KeyCtrlC, msg.Type == tui.KeyEsc, isRune(msg, 'q'):
			return m, tui.Quit()
		case msg.Type == tui.KeyLeft, isRune(msg, 'h'):
			return m.step(-1), nil
		case msg.Type == tui.KeyRight, isRune(msg, 'l'):
			return m.step(1), nil
		case msg.Type == tui.KeySpace:
			return m.space()
		case isRune(msg, 'w'):
			return m.toggleWall(), nil
		case isRune(msg, 's'):
			return m.toggleSize(), nil
		}
	}

	var cmd tui.Cmd
	m.player, cmd = m.player.Update(msg)
	return m, cmd
}

func (m model) step(d int) model {
	m.plays = map[int]play{}
	if m.wall {
		m.page = ((m.page+d)%m.pages() + m.pages()) % m.pages()
		return m
	}
	if d < 0 {
		m.player.Prev()
	} else {
		m.player.Next()
	}
	m.player.Stop() // a new face rests until it is clicked
	return m
}

func (m model) toggleWall() model {
	m.wall = !m.wall
	m.plays = map[int]play{}
	m.player.Stop()
	if m.wall {
		// Open on the page holding the face being viewed.
		m.page = m.player.Index() / m.perPage()
	} else {
		m.player.Set(m.page * m.perPage())
	}
	return m
}

// toggleSize switches between small and large faces, keeping the same
// faces in view (the wall's page size changes with the sprite width).
func (m model) toggleSize() model {
	first := m.page * m.perPage()
	m.player.Size = faces.Large - m.player.Size
	m.plays = map[int]play{}
	m.player.Stop()
	m.page = min(first/m.perPage(), m.pages()-1)
	return m
}

func (m model) space() (tui.Model, tui.Cmd) {
	if m.still {
		return m, nil
	}
	if !m.wall {
		if m.player.Running() {
			m.player.Stop()
			return m, nil
		}
		return m, m.player.Start()
	}
	var cmds []tui.Cmd
	for _, i := range m.visible() {
		var c tui.Cmd
		m, c = m.startPlay(i)
		cmds = append(cmds, c)
	}
	return m, tui.Batch(cmds...)
}

// visible lists the catalog indexes on the current wall page.
func (m model) visible() []int {
	var out []int
	for i := m.page * m.perPage(); i < (m.page+1)*m.perPage() && i < faces.Count(); i++ {
		out = append(out, i)
	}
	return out
}

// withPlays returns m with its own copy of the plays map, so a model value
// that was handed out earlier never sees a later change.
func (m model) withPlays() model {
	next := make(map[int]play, len(m.plays)+1)
	for k, v := range m.plays {
		next[k] = v
	}
	m.plays = next
	return m
}

func (m model) startPlay(i int) (model, tui.Cmd) {
	m = m.withPlays()
	m.serial++
	m.plays[i] = play{frame: 0, gen: m.serial}
	gen := m.serial
	return m, tui.FromCtx(motion.After(faces.Get(i).Interval, func(time.Time) tui.Msg { return playMsg{face: i, gen: gen} }))
}

func (m model) advance(msg playMsg) (tui.Model, tui.Cmd) {
	p, ok := m.plays[msg.face]
	if !ok || p.gen != msg.gen {
		return m, nil // restarted or cleared since this tick was scheduled
	}
	m = m.withPlays()
	p.frame++
	if p.frame >= len(faces.Get(msg.face).Frames(m.player.Size)) {
		delete(m.plays, msg.face) // one loop done: back to rest
		return m, nil
	}
	m.plays[msg.face] = p
	gen := p.gen
	return m, tui.FromCtx(motion.After(faces.Get(msg.face).Interval, func(time.Time) tui.Msg { return playMsg{face: msg.face, gen: gen} }))
}

// spriteName is the layout name of the single view's sprite.
const spriteName = "sprite"

// faceName is the layout name of catalog face i on the wall.
func faceName(i int) string { return fmt.Sprintf("face-%d", i) }

// shown lists the catalog indexes of the faces on screen: the single face, or
// the current wall page.
func (m model) shown() []int {
	if m.wall {
		return m.visible()
	}
	return []int{m.player.Index()}
}

// regionOf is where face i's sprite is drawn, read from the layout at the
// screen's natural size. It covers the sprite rows only: not the label row
// under it, the gap between columns or the blank row between rows.
func (m model) regionOf(i int) (hittest.Rect, bool) {
	root := m.screen()
	name := faceName(i)
	if !m.wall {
		name = spriteName
	}
	r, ok := layout.RectOf(root, root.Measure(layout.Unconstrained()), name)
	if !ok {
		return hittest.Rect{}, false
	}
	w, h := m.cell()
	r.W, r.H = min(r.W, w), min(r.H, h)
	return hittest.Rect(r), true
}

// regions maps each face on screen to its clickable area.
func (m model) regions() hittest.Map[int] {
	var regions hittest.Map[int]
	for _, i := range m.shown() {
		if r, ok := m.regionOf(i); ok {
			regions = regions.Add(i, r)
		}
	}
	return regions
}

// faceAt returns the catalog index of the face drawn at (x, y), in screen
// cells, and false if that cell is not on a sprite.
func (m model) faceAt(x, y int) (int, bool) {
	h, ok := m.regions().At(x, y)
	return h.ID, ok
}

// click plays whatever face is under the cursor at (x, y), in screen cells.
func (m model) click(x, y int) (tui.Model, tui.Cmd) {
	if m.still {
		return m, nil
	}
	i, ok := m.faceAt(x, y)
	switch {
	case !ok:
		return m, nil
	case !m.wall:
		return m, m.player.PlayOnce()
	}
	return m.startPlay(i)
}

var faint = ansi.NewStyle().Faint()

// fit returns the longest of the candidate help lines that fits the terminal
// width, cut to it when even the shortest does not. Before the first
// ResizeMsg the width is unknown and the longest is used whole.
func (m model) fit(candidates ...string) string {
	if m.width <= 0 {
		return candidates[0]
	}
	for _, c := range candidates {
		if ansi.Width(c) <= m.width {
			return c
		}
	}
	return ansi.Truncate(candidates[len(candidates)-1], m.width)
}

// screen is the gallery as a layout.Node: the title line, then the single face
// or the wall, then the help line, one blank row apart. Click regions are read
// from the sprites named here (see regionOf), so they cannot drift from it.
func (m model) screen() layout.Node {
	child := func(n layout.Node) layout.FlexChild { return layout.FlexChild{Node: n} }
	title := ansi.NewStyle().Bold().Foreground(m.t.Primary).Render("faces")
	if m.wall {
		help := faint.Render(m.fit("click a face to play · ←/→ page · s size · w single · space play page · q quit", "click play · ←/→ page · s w space q"))
		heading := title + faint.Render(fmt.Sprintf("  wall %d/%d", m.page+1, m.pages()))
		return layout.Column(1, child(layout.Block(heading)), child(m.wallNode()), child(layout.Block(help)))
	}
	help := faint.Render(m.fit("click a face to play · ←/→ change · s size · w wall · space loop · q quit", "click play · ←/→ s w space q"))
	n := fmt.Sprintf("  #%d of %d", m.player.Index()+1, faces.Count())
	return layout.Column(1, child(layout.Block(title+faint.Render(n))), child(layout.Named(spriteName, m.player.LayoutNode())), child(layout.Block(help)))
}

func (m model) View() string { return layout.Draw(m.screen(), layout.Unconstrained()) }

// wallView is the wall drawn on its own.
func (m model) wallView() string { return layout.Draw(m.wallNode(), layout.Unconstrained()) }

// wallNode is the wall: wallRows rows of faces, wallGap columns apart, one
// blank row between the rows.
func (m model) wallNode() layout.Node {
	sprite := ansi.NewStyle().Foreground(m.t.Primary)
	sz := m.player.Size
	cw, _ := m.cell()
	var rows []layout.FlexChild
	for r := 0; r < wallRows; r++ {
		var cells []layout.FlexChild
		for c := 0; c < m.cols(); c++ {
			i := m.page*m.perPage() + r*m.cols() + c
			cell := strings.Repeat(" ", cw) + "\n"
			if i < faces.Count() {
				f := faces.Get(i)
				frames := f.Frames(sz)
				frame := frames[m.plays[i].frame%len(frames)] // rests on frame 0
				label := fmt.Sprintf("#%d %s", i+1, f.Name)
				cell = sprite.Render(frame) + "\n" + faint.Render(label) + strings.Repeat(" ", max(0, cw-len(label)))
			}
			var node layout.Node = layout.Block(cell)
			if i < faces.Count() {
				node = layout.Named(faceName(i), node)
			}
			cells = append(cells, layout.FlexChild{Node: node, CrossAlign: layout.CrossStart})
		}
		rows = append(rows, layout.FlexChild{Node: layout.Row(wallGap, cells...), CrossAlign: layout.CrossStart})
	}
	return layout.Column(1, rows...)
}

func main() {
	if _, err := tui.NewProgram(initialModel(), tui.WithMouse(tui.MouseClick)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

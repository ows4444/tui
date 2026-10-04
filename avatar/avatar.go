// Package avatar draws a deterministic avatar for a name: a soft body in
// one of ten silhouettes with two capsule eyes, in a colour the name chose.
// The same name always draws the same avatar, so whoever a user learns in one
// list is who they recognise in the next. Nothing is stored or fetched.
//
// The avatar stands for somebody. For a catalog of ready-made animated
// characters, picked by index or name to show a mood or a status, use
// package faces.
//
// View draws the figure in terminal cells, PNG returns it as an image for a
// terminal that shows images, and SVG returns it as markup. The
// mapping from a name to its figure and colours is frozen: the tests pin it
// against the reference vectors in testdata.
//
// Hue, Tone and Silhouette pin the colour or the shape, for an app with a
// house style, while the name still decides everything else.
//
// The figure can react. Expression sets a pose the eyes hold (SetExpression
// eases into it), React pulls a face for a moment and lets it go, Blink closes and reopens the eyes once
// and StartIdle keeps the avatar breathing, blinking and glancing aside, all
// three driven by Update, and LookAt turns the eyes toward a target such as
// the mouse pointer.
//
// A name is put in Unicode Normalization Form C, trimmed and lowercased
// before it is hashed, so "Alain" and " alain " are one avatar, and so are a
// name written with combining marks ("e" followed by U+0301) and its
// precomposed spelling. Set Raw to hash the name as written.
//
// Stability: experimental. Its API may change in any minor release.
package avatar

import (
	"strings"
	"sync"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/motion"
	"github.com/ows4444/tui/theme"
)

// DefaultWidth and DefaultHeight are the size, in cells, of a Model from New.
// A cell is about twice as tall as it is wide, so a width of twice the height
// is square on screen.
const (
	DefaultWidth  = 8
	DefaultHeight = 4
)

// Background is the plate drawn behind the body.
type Background int

// The backgrounds. BackgroundNone, the zero value, leaves the cells around
// the body untouched.
const (
	BackgroundNone Background = iota
	BackgroundSquircle
	BackgroundCircle
	BackgroundSquare
)

// Model is the avatar for Name, drawn as a Width by Height cell area.
type Model struct {
	// Name is who the avatar stands for: a username, an email, an id. It is
	// trimmed and lowercased before hashing unless Raw is set.
	Name string
	// Width and Height are the size in terminal cells. The figure is square
	// and centred, so the shorter side (counting a row as two columns)
	// decides how large it is drawn.
	Width, Height int
	// Background is the plate behind the body.
	Background Background
	// Raw hashes Name as written. Set it for case-sensitive ids, which
	// normalization would otherwise collide.
	Raw bool
	// Hue pins the colour's hue, an angle in degrees in the OKLCh colour
	// space: about 30 is red-orange, 90 yellow, 140 green, 250 blue and 320
	// magenta. Zero lets the name choose; use 360 for the hue at 0. The name
	// still chooses the tone and the figure.
	Hue float64
	// Tone pins how light and saturated the body is. The zero value,
	// ToneAuto, lets the name choose.
	Tone Tone
	// Silhouette pins the shape. The zero value, SilhouetteAuto, lets the
	// name choose; the name still sizes and places it.
	Silhouette Silhouette
	// Expression is the pose the eyes hold. The zero value, ExpressionNone,
	// is the figure at rest. Assigning it changes the pose at once;
	// SetExpression eases into it.
	Expression Expression
	// LookX and LookY turn the eyes: each runs from -1 to 1, where LookX 1
	// is fully right and LookY 1 fully down, and zero is the name's own
	// resting gaze. LookAt sets both from where a target is.
	LookX, LookY float64
	// Motion is the reduced-motion preference. The zero value is
	// motion.Normal. With motion.Reduced, Blink, StartIdle and React play
	// nothing.
	Motion motion.Preference
	// Theme supplies the glyph set: under an ASCII one the avatar is drawn
	// with its Shades. The colours come from Name, not from the theme.
	Theme theme.Theme

	blink  int  // frame of the blink being played; 0 is at rest
	idling bool // the idle loop is running
	beat   int  // beats of the idle loop played so far
	// owner is the token of the animation in progress; its ticks carry it,
	// and a tick with another token is ignored.
	owner *int
	// reaction is the expression React is holding, or ExpressionNone, and
	// reacts counts the reactions so far.
	reaction Expression
	reacts   int
	// from is the pose the eyes are easing away from and tween the frame of
	// that ease; 0 is at rest on the pose shown. poseOwner is the token the
	// ease's ticks and a reaction's release carry.
	from      Expression
	tween     int
	poseOwner *int
	// cache holds the last View; nil on a struct literal.
	cache *viewCache
}

// viewCache memoises the View of a Model, so a frame that shows the same
// avatar does not lay it out and rasterise it again. It is keyed on every
// field the output depends on, so changing one invalidates it. Copies of a
// Model share the cache, which holds one View: give each avatar on screen its
// own Model from New. A Model built as a struct literal has no cache and
// draws each time.
type viewCache struct {
	mu   sync.Mutex
	key  viewKey
	view string
	ok   bool
}

type viewKey struct {
	name          string
	raw           bool
	width, height int
	bg            Background
	hue           float64
	tone          Tone
	silhouette    Silhouette
	lookX, lookY  float64
	zoom          float64
	expression    Expression
	from          Expression
	tween         int
	blink         int
	ascii         bool
	shades        string
}

// drawHook, when non-nil, is called each time View draws the figure. Tests
// use it to prove the cache; it is nil in production.
var drawHook func()

// New returns a Model for name at DefaultWidth by DefaultHeight cells, using
// theme.DarkTheme(). Its View is cached until a field it depends on changes.
func New(name string) Model {
	return Model{Name: name, Width: DefaultWidth, Height: DefaultHeight, Theme: theme.DarkTheme(), cache: &viewCache{}}
}

func (m Model) traits() traits {
	t := newTraits(m.Name, m.Raw)
	t.fixed = m.pins()
	return t
}

// restPalette is the avatar's own colours, before any expression tints them.
func (m Model) restPalette(t traits) palette {
	hue, pinned := m.hue()
	if !pinned {
		hue = t.num("hue", 0, 360)
	}
	return paletteFor(hue, t.at("tone"))
}

// palette is the colours the avatar is drawn in now: its own, tinted by the
// expression showing if that one tints.
func (m Model) palette(t traits) palette {
	rest := m.restPalette(t)
	wear := func(pose pose) palette {
		if pose.tint == nil {
			return rest
		}
		return rest.tinted(pose.tint, pose.heat)
	}
	from, to, at := m.poses()
	p := wear(to)
	if at < 1 && (from.tint != nil || to.tint != nil) {
		// Part of the way from one pose's colours to the other's.
		a := wear(from)
		p.head, p.eye = mixRGB(a.head, p.head, at), mixRGB(a.eye, p.eye, at)
	}
	return p
}

// Shape returns the name of the avatar's silhouette: round, organic, boxy,
// capsule, nub, cloud, droplet, hexagon, sun or triangle. It is the pinned
// Silhouette when one is set.
func (m Model) Shape() string {
	return shapes[pickShape(m.traits().at("shape"))].name
}

// Colors returns the colours the avatar is drawn in now: the body, the eyes,
// and the plate drawn when Background is set. An expression that tints, such
// as ExpressionMad, changes the body and may change the eyes. The eyes
// contrast with the body at 4.5:1 or better, whatever Hue and Tone are pinned
// and whatever expression shows.
func (m Model) Colors() (body, eyes, background ansi.RGB) {
	p := m.palette(m.traits())
	return p.head, p.eye, p.bg
}

// plate returns the background outline, or nil for BackgroundNone.
func (m Model) plate() path {
	switch m.Background {
	case BackgroundSquircle:
		return superellipse(50, 50, 50, 50, 6, 0)
	case BackgroundCircle:
		return superellipse(50, 50, 50, 50, 2, 0)
	case BackgroundSquare:
		var p path
		p.add('M', 0, 0)
		p.add('H', 100)
		p.add('V', 100)
		p.add('H', 0)
		p.add('Z')
		return p
	}
	return nil
}

// SVG returns the avatar as SVG markup on a 100 by 100 view box, with no
// width or height, so the page sizes it. Hue, Tone and Silhouette are part
// of the figure and are drawn. It is the figure at rest:
// Expression, a reaction, LookX, LookY, a blink in progress and the idle
// loop do not change it.
func (m Model) SVG() string {
	t := m.traits()
	return svg(layoutFigure(t), m.restPalette(t), m.plate())
}

func svg(f figure, p palette, plate path) string {
	var b strings.Builder
	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100">`)
	if plate != nil {
		b.WriteString(`<path d="` + plate.String() + `" fill="` + hex(p.bg) + `"/>`)
	}
	b.WriteString(`<g fill="` + hex(p.head) + `">`)
	for _, c := range f.petals {
		b.WriteString(`<circle cx="` + num(round2(c.cx)) + `" cy="` + num(round2(c.cy)) + `" r="` + num(round2(c.r)) + `"/>`)
	}
	for _, e := range f.extra {
		b.WriteString(`<path d="` + e.String() + `"/>`)
	}
	b.WriteString(`<path d="` + f.core.String() + `"/></g><g fill="` + hex(p.eye) + `">`)
	for _, e := range f.eyes {
		b.WriteString(`<path d="` + e.String() + `"/>`)
	}
	b.WriteString(`</g></svg>`)
	return b.String()
}

// View renders Height rows, each exactly Width cells wide. Each cell is a
// two by two block of pixels drawn with a quadrant block character, in two
// colours: the body is the character's ink, and the eyes and the plate are
// the cell's background. Without colour the body still reads as a solid
// silhouette with the eyes cut out of it. In a small avatar the eyes are drawn
// larger than their true size so they stay legible, and each takes at least
// one pixel except during a blink. Under an ASCII glyph set a cell is one of
// the theme's Shades, darker the more of it the body covers. Width <= 0 or
// Height <= 0 renders "".
func (m Model) View() string {
	if m.Width <= 0 || m.Height <= 0 {
		return ""
	}
	glyphs := m.Theme.GlyphSet()
	c := m.cache
	if c == nil {
		return m.draw(glyphs)
	}
	lookX, zoom := m.moved()
	key := viewKey{
		name: m.Name, raw: m.Raw, width: m.Width, height: m.Height, bg: m.Background,
		hue: m.Hue, tone: m.Tone, silhouette: m.Silhouette,
		lookX: unit(unit(m.LookX) + lookX), lookY: unit(m.LookY), zoom: zoom,
		expression: m.shown(), from: m.easing(), tween: m.tween, blink: m.blink,
		ascii: glyphs.ASCII(), shades: glyphs.Shades,
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.ok || c.key != key {
		c.key, c.view, c.ok = key, m.draw(glyphs), true
	}
	return c.view
}

// draw lays the figure out, rasterises it and renders the cells.
func (m Model) draw(glyphs theme.Glyphs) string {
	if drawHook != nil {
		drawHook()
	}
	t := m.traits()
	open := blinkOpen[m.blink%len(blinkOpen)]
	lookX, zoom := m.moved()
	f := layoutFigure(t)
	s := newScene(f, m.plate())
	from, to, at := m.poses()
	s.setEyes(f.posed(from, to, at, unit(unit(m.LookX)+lookX), unit(m.LookY), open, s.eyeBoost(f, m.Width, m.Height)))
	s.zoom = zoom
	g := s.rasterize(m.Width, m.Height, open == 1)
	return g.render(m.palette(t), glyphs)
}

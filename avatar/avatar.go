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
// Hue, Tone, Silhouette, Accessory and Eyes pin the colour or the look, for an
// app with a house style, while the name still decides everything else.
//
// The figure can react. Expression sets a pose the eyes hold (SetExpression
// eases into it), React pulls a face for a moment and lets it go, Blink closes and reopens the eyes once
// and StartIdle keeps the avatar breathing, blinking and glancing aside, all
// three driven by Update, and LookAt turns the eyes toward a target such as
// the mouse pointer. Blink, StartIdle, StopIdle, Hover, LookAt, React and
// SetExpression change the Model they are called on: call them on the one your
// own model keeps (m.avatar.Blink()), never on a copy that is then thrown away.
//
// A name is put in Unicode Normalization Form C, trimmed and lowercased
// before it is hashed, so "Alain" and " alain " are one avatar, and so are a
// name written with combining marks ("e" followed by U+0301) and its
// precomposed spelling. Only the first 256 bytes of a name count, so a very
// long one costs no more than a short one. Set Raw to hash the name as
// written (still cut to 256 bytes). The rules read their data from tables
// frozen at one Unicode version (see package internal/normname), not from the
// standard library's, so a new Go release cannot change anyone's avatar.
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
	// magenta; 0 and 360 are the same hue. Nil lets the name choose. The name
	// still chooses the tone and the figure.
	Hue *float64
	// Tone pins how light and saturated the body is. The zero value,
	// ToneAuto, lets the name choose.
	Tone Tone
	// Silhouette pins the shape. The zero value, SilhouetteAuto, lets the
	// name choose; the name still sizes and places it.
	Silhouette Silhouette
	// Accessory pins what the body wears on top, and Eyes how the eyes are
	// drawn. Their zero values let the name choose.
	Accessory Accessory
	Eyes      EyeStyle
	// traitPins fixes single traits of the figure for the tests, each at a
	// position from 0 to 1; it is not part of the API.
	traitPins map[traitID]float64
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
	// wobble is the frame of the held pose's own motion, its tremble or its
	// rock; 0 is still. holdLeft counts the frames a moving reaction has
	// left before it is released.
	wobble   int
	holdLeft int
	// cache holds the last View; nil on a struct literal.
	cache *viewCache
}

// viewCache memoises the Views and the PNGs of a Model, so a frame that
// shows the same avatar does not lay it out and rasterise it again. Copies of
// a Model share the cache. A Model built as a struct literal has no cache and
// draws each time.
type viewCache struct {
	mu    sync.Mutex
	views frames[string]
	pngs  frames[[]byte]
}

// frameMemory is how many pictures of each kind a Model keeps.
const frameMemory = 24

// frames is a small store of pictures by what they were drawn from.
type frames[V any] struct {
	at    map[viewKey]V
	order []viewKey
}

func (f *frames[V]) get(k viewKey) (V, bool) {
	v, ok := f.at[k]
	return v, ok
}

func (f *frames[V]) put(k viewKey, v V) {
	if f.at == nil {
		f.at = map[viewKey]V{}
	}
	if len(f.order) == frameMemory {
		delete(f.at, f.order[0])
		f.order = append(f.order[:0], f.order[1:]...)
	}
	f.at[k] = v
	f.order = append(f.order, k)
}

type viewKey struct {
	name          string
	raw           bool
	width, height int
	bg            Background
	hue           float64
	tone          Tone
	silhouette    Silhouette
	pins          string
	lookX, lookY  float64
	zoom          float64
	expression    Expression
	from          Expression
	tween         int
	wobble        int
	blink         int
	ascii         bool
	shades        string
	page          ansi.RGB // the page colour the body is held apart from
}

// drawHook, when non-nil, is called each time View draws the figure. Tests
// use it to prove the cache; it is nil in production.
var drawHook func()

// New returns a Model for name at DefaultWidth by DefaultHeight cells, using
// theme.DarkTheme(). The pictures drawn lately are remembered for every Model
// alike, so a frame shown before is not drawn again, even by a new Model made
// for it.
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
	return paletteFor(hue, t.at("tone"), m.ground())
}

// ground is what the body is drawn on: the plate if Background is set, else the
// page, which is the theme's background (or its inverse text colour, or a dark
// page when it names neither).
func (m Model) ground() ground {
	return ground{page: m.page(), plated: m.Background != BackgroundNone}
}

// page is the colour of the page behind the avatar, from the theme.
func (m Model) page() oklch { return oklchOf(m.pageRGB()) }

func (m Model) pageRGB() ansi.RGB {
	for _, c := range []ansi.Color{m.Theme.Background, m.Theme.TextInverse} {
		if rgb, ok := rgbOf(c); ok {
			return rgb
		}
	}
	return darkSurface.rgb()
}

// rgbOf resolves a theme colour to RGB; a nil or unknown colour is not ok.
func rgbOf(c ansi.Color) (ansi.RGB, bool) {
	switch v := c.(type) {
	case ansi.RGB:
		return v, true
	case ansi.Color256:
		return v.RGB(), true
	case ansi.BasicColor:
		return v.RGB(), true
	}
	return ansi.RGB{}, false
}

// palette is the colours the avatar is drawn in now: its own, tinted by the
// expression showing if that one tints.
func (m Model) palette(t traits) palette {
	rest := m.restPalette(t)
	wear := func(pose pose) palette {
		if pose.tint == nil {
			return rest
		}
		return rest.tinted(pose.tint, pose.heat, m.ground())
	}
	from, to, at := m.poses()
	p := wear(to)
	if at < 1 && (from.tint != nil || to.tint != nil) {
		// Part of the way from one pose's colours to the other's.
		a := wear(from)
		head := ensureContrast(oklchOf(mixRGB(a.head, p.head, at)), m.ground().ref(rest.bg), bodyFloor)
		p.head, p.eye = head.rgb(), eyeFor(head, oklchOf(rest.eye).h).rgb()
	}
	return p
}

// Shape returns the avatar's silhouette, never SilhouetteAuto: the pinned
// Silhouette when one is set, else the one the name chose. Its String is the
// name, such as "round".
func (m Model) Shape() Silhouette {
	return SilhouetteRound + Silhouette(pickShape(m.traits().at("shape")))
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
// width or height, so the page sizes it. Hue, Tone, Silhouette, Accessory and Eyes are
// part of the figure and are drawn, and so are the pose it holds, with its
// eyebrows, mouth and tint, how it lifts or sinks, and LookX and LookY. It is a
// still: the middle of an ease, a tremble or a rock, a blink in progress and the
// idle loop are not drawn.
func (m Model) SVG() string {
	still := m
	still.tween, still.wobble, still.blink, still.idling = 0, 0, 0, false
	t := still.traits()
	f := layoutFigure(t)
	pose := still.shown().pose()
	f = f.posed(pose, pose, 1, unit(still.LookX), unit(still.LookY), 0, 1, 1)
	return svg(f, still.palette(t), still.plate(), pose.lift)
}

func svg(f figure, p palette, plate path, lift float64) string {
	var b strings.Builder
	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100">`)
	if plate != nil {
		b.WriteString(`<path d="` + plate.String() + `" fill="` + hex(p.bg) + `"/>`)
	}
	if lift != 0 {
		b.WriteString(`<g transform="translate(0 ` + num(round2(lift)) + `)">`)
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
	for _, mk := range f.marks {
		b.WriteString(`<path d="` + mk.p.String() + `"/>`)
	}
	b.WriteString(`</g>`)
	if lift != 0 {
		b.WriteString(`</g>`)
	}
	b.WriteString(`</svg>`)
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
	key := m.key(m.Width, m.Height)
	key.ascii, key.shades = glyphs.ASCII(), glyphs.Shades
	c.mu.Lock()
	defer c.mu.Unlock()
	view, ok := c.views.get(key)
	if !ok {
		view = m.draw(glyphs)
		c.views.put(key, view)
	}
	return view
}

// key is everything a drawing of the avatar at width by height depends on,
// apart from the glyph set.
func (m Model) key(width, height int) viewKey {
	lookX, zoom := m.moved()
	// A tremble has two pictures, one for each side, however many frames it
	// has run for: frames on the same side share a key.
	wobble := m.wobble
	if wobble > 0 && m.tween == 0 && !m.shown().pose().rock {
		wobble = 2 - wobble%2
	}
	hue, pinned := m.hue()
	if !pinned {
		hue = -1
	}
	return viewKey{
		name: m.Name, raw: m.Raw, width: width, height: height, bg: m.Background,
		hue: hue, tone: m.Tone, silhouette: m.Silhouette, pins: m.pinKey(),
		lookX: unit(unit(m.LookX) + lookX), lookY: unit(m.LookY), zoom: zoom,
		expression: m.shown(), from: m.easing(), tween: m.tween, wobble: wobble, blink: m.blink,
		page: m.pageRGB(),
	}
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
	// A tremble is at least half a pixel wide, or it would not show.
	shift := m.tremble(to, f, 0.5*s.pixelWidth(m.Width, m.Height))
	posed := f.posed(from, to, at, unit(unit(m.LookX)+lookX), unit(m.LookY), shift, open, s.eyeBoost(f, m.Width, m.Height))
	s.setEyes(posed)
	s.setMarks(posed.marks, s.pixelWidth(m.Width, m.Height))
	s.zoom = zoom
	// In cells a pixel is twice as tall as it is wide.
	s.setLift(lift(from, to, at), 2*s.pixelWidth(m.Width, m.Height))
	g := s.rasterize(m.Width, m.Height, open == 1)
	return g.render(m.palette(t), glyphs)
}

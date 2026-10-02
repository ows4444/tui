package faces

import (
	"strings"
	"time"
)

// Face is one character: a name and a looping animation, drawn at either
// Size. Every frame of a face at one Size is the same block of cells.
type Face struct {
	Name     string
	Anim     string        // what it does: blink, sleep, glitch...
	Interval time.Duration // the pace the loop reads best at
	frames   [2][]string
}

// Frames returns the loop at the given size: 4, 8 or 12 Braille frames,
// each Cells() wide and tall.
func (f Face) Frames(sz Size) []string { return f.frames[sz] }

type spec struct {
	name string
	head *head
	anim anim
	look look
}

func s(name string, h *head, a anim, e eyeStyle, m mouthStyle) spec {
	return spec{name, h, a, look{e, m}}
}

// The 50 characters: 14 head shapes crossed with 22 animations, eye styles
// and mouth styles. Animation lengths are 4, 8 or 12 frames.
var specs = []spec{
	s("Pip", headRound, animBlink, styleDot, mouthSoft),
	s("Drizzle", headRound, animCry, styleTall, mouthSoft),
	s("Nimbus", headRound, animLook, styleRound, mouthSoft),
	s("Sunny", headRound, animLaugh, styleDot, mouthSoft),

	s("Meh", headSquare, animBored, styleDot, mouthSoft),
	s("Quill", headSquare, animTalk, styleOval, mouthSoft),
	s("Fable", headSquare, animSurprise, styleRing, mouthSoft),

	s("Wink", headSquircle, animWink, styleDot, mouthSoft),
	s("Smirk", headSquircle, animSmug, styleOval, mouthSoft),
	s("Boss", headSquircle, animEvil, styleRound, mouthBoxy),

	s("Cupid", headOctagon, animLove, styleStar, mouthSoft),
	s("Scowl", headOctagon, animEvil, styleTall, mouthSoft),
	s("Rumble", headOctagon, animAngry, styleRing, mouthSoft),
	s("Tango", headOctagon, animBounce, styleSquare, mouthSoft),

	s("Yawn", headDither, animYawn, styleTall, mouthSoft),
	s("Bubbles", headDither, animLaugh, styleRound, mouthSoft),
	s("Nap", headDither, animSleep, styleDot, mouthSoft),

	s("Dizzy", headGhost, animDizzy, styleSquare, mouthSoft),
	s("Dot", headGhost, animBlink, styleRound, mouthSoft),
	s("Static", headGhost, animGlitch, styleBlock, mouthSoft),
	s("Hush", headGhost, animThink, styleDot, mouthSoft),

	s("Jitter", headDashed, animNervous, styleOval, mouthSoft),
	s("Ghost", headDashed, animLook, styleDot, mouthSoft),
	s("Drift", headDashed, animBored, styleOval, mouthSoft),

	s("Shade", headShield, animCool, styleBlock, mouthSoft),
	s("Vibe", headShield, animCool, styleDot, mouthBoxy),
	s("Echo", headShield, animTalk, styleBlock, mouthBoxy),

	s("Bolt", headRobot, animScan, styleBlock, mouthBoxy),
	s("Chip", headRobot, animGlitch, styleBlock, mouthBoxy),
	s("Radar", headRobot, animScan, styleOval, mouthSoft),
	s("Sonar", headRobot, animScan, styleDot, mouthBoxy),

	s("Whisk", headCat, animTalk, styleRound, mouthCat),
	s("Hop", headCat, animBounce, styleDot, mouthCat),
	s("Luna", headCat, animSleep, styleDot, mouthCat),
	s("Sprout", headCat, animWink, styleRound, mouthCat),

	s("Zed", headAlien, animLook, styleRing, mouthSoft),
	s("Quasar", headAlien, animThink, styleRing, mouthSoft),
	s("Blob", headAlien, animNervous, styleRound, mouthSoft),
	s("Glorp", headAlien, animDizzy, styleRing, mouthSoft),

	s("Mochi", headBear, animSleep, styleDot, mouthSoft),
	s("Brr", headBear, animShiver, styleRound, mouthCat),
	s("Honey", headBear, animLove, styleDot, mouthSoft),
	s("Bumble", headBear, animBounce, styleRound, mouthSoft),

	s("Grumble", headSpiky, animAngry, styleRound, mouthSoft),
	s("Spike", headSpiky, animEvil, styleDot, mouthSoft),
	s("Zap", headSpiky, animShiver, styleBlock, mouthSoft),

	s("Ponder", headCRT, animThink, styleRound, mouthBoxy),
	s("Pixel", headCRT, animGlitch, styleDot, mouthBoxy),
	s("Signal", headCRT, animLook, styleBlock, mouthBoxy),
	s("Buzz", headCRT, animDizzy, styleSquare, mouthBoxy),
}

var catalog = build()

func build() []Face {
	out := make([]Face, len(specs))
	for i, sp := range specs {
		poses := sp.anim.frames(sp.look)
		f := Face{Name: sp.name, Anim: sp.anim.name, Interval: sp.anim.interval}
		for _, sz := range []Size{Small, Large} {
			for _, p := range poses {
				f.frames[sz] = append(f.frames[sz], sp.head.render(sz, sp.look, p))
			}
		}
		out[i] = f
	}
	return out
}

// All returns every face, in catalog order. The slice is a copy.
func All() []Face { return append([]Face(nil), catalog...) }

// Count is the number of faces in the catalog.
func Count() int { return len(catalog) }

// Get returns face i, wrapping around in both directions.
func Get(i int) Face {
	n := len(catalog)
	return catalog[((i%n)+n)%n]
}

// ByName finds a face by name, ignoring case.
func ByName(name string) (Face, bool) {
	for _, f := range catalog {
		if strings.EqualFold(f.Name, name) {
			return f, true
		}
	}
	return Face{}, false
}

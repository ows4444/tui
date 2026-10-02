// This file adds the two-pass measure/render protocol from decision #10.
// It is purely additive: Box, Join*, Grid, FlexRow, GridFlex and Overlay
// keep their string-in/string-out signatures and output. A Node lets a
// parent ask a child how big it wants to be (Measure, top-down constraints,
// bottom-up preferred size) before handing it a final Size to fill exactly
// (Render). Block adapts any pre-rendered string, so existing widgets
// compose immediately.
package layout

import (
	"strings"

	"github.com/ows4444/tui/ansi"
)

// Unbounded, used as a Constraints Max, means "no upper limit on this
// axis". It is a large finite value rather than a special case so bounds
// arithmetic never needs a branch.
const Unbounded = 1 << 30

// Size is a width and height in terminal cells.
type Size struct{ W, H int }

// Constraints bounds the Size a Node may report from Measure. A Max of
// Unbounded leaves that axis open. Constraints are normalized before use:
// a negative bound counts as 0 and a Max below its Min is raised to it, so
// any Constraints value is safe to pass. Note the zero value bounds both
// axes to 0; use Unconstrained or set MaxW/MaxH (to Unbounded) to leave an
// axis open.
type Constraints struct{ MinW, MaxW, MinH, MaxH int }

// Loose returns constraints allowing anything from 0x0 up to max.
func Loose(max Size) Constraints {
	return Constraints{MaxW: max.W, MaxH: max.H}
}

// Tight returns constraints that allow exactly size on both axes.
func Tight(size Size) Constraints {
	return Constraints{MinW: size.W, MaxW: size.W, MinH: size.H, MaxH: size.H}
}

// Unconstrained returns constraints with no bounds at all.
func Unconstrained() Constraints {
	return Constraints{MaxW: Unbounded, MaxH: Unbounded}
}

// Constrain clamps s into the constraints' range on both axes.
func (c Constraints) Constrain(s Size) Size {
	return Size{W: clampAxis(s.W, c.MinW, c.MaxW), H: clampAxis(s.H, c.MinH, c.MaxH)}
}

func clampAxis(v, min, max int) int {
	if min < 0 {
		min = 0
	}
	if max < min {
		max = min
	}
	if v > max {
		v = max
	}
	if v < min {
		v = min
	}
	return v
}

// Node is a layout participant. Measure returns the size the node prefers
// under c (always within c). Render returns a string of exactly s.W columns
// by s.H rows (ansi.Width-measured); s is the final size the parent
// allotted, which need not equal what Measure returned. Measure must be
// side-effect free: parents may call it several times before Render.
type Node interface {
	Measure(c Constraints) Size
	Render(s Size) string
}

// Draw runs both passes on root: it measures under c, then renders at the
// measured size. The result is exactly that size, or "" if either
// dimension is 0. Within one Draw, Measure is called at most once per node
// per distinct Constraints for the built-in container nodes' children.
func Draw(root Node, c Constraints) string {
	a := getArena()
	root = memoized(root, a)
	out := root.Render(root.Measure(c))
	putArena(a)
	return out
}

// DrawTight is Draw under Tight(size): the root is rendered at exactly size,
// so Fill children absorb all remaining main-axis space. Draw with Loose
// renders at the measured size and is unchanged.
func DrawTight(root Node, size Size) string {
	return Draw(root, Tight(size))
}

// Block adapts a pre-rendered (possibly multi-line, possibly styled)
// string to a Node. Measure reports the string's natural size (widest
// line by display width, by line count; "" is 0x0). Render pads or clips
// it to the allotted size: lines are truncated by display width without
// splitting a rune or an escape sequence, and short lines and missing rows
// are filled with spaces.
func Block(s string) Node { return block(s) }

type block string

func (b block) Measure(c Constraints) Size {
	if b == "" {
		return c.Constrain(Size{})
	}
	w, n := 0, 0
	for rest, more := string(b), true; more; {
		var l string
		l, rest, more = nextLine(rest)
		if lw := ansi.Width(l); lw > w {
			w = lw
		}
		n++
	}
	return c.Constrain(Size{W: w, H: n})
}

func (b block) Render(s Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	rest, more := string(b), b != ""
	var out strings.Builder
	out.Grow(s.H*(s.W+1) + len(b))
	var carry pen // style open at the start of the current line
	for i := 0; i < s.H; i++ {
		if i > 0 {
			out.WriteByte('\n')
		}
		w := 0
		if more {
			var l string
			full := ""
			l, rest, more = nextLine(rest)
			full = l
			hasEsc := strings.IndexByte(full, 0x1b) >= 0
			if hasEsc {
				l = ansi.Truncate(l, s.W)
				w = ansi.Width(l)
			} else if w = ansi.Width(l); w > s.W {
				// Plain text: Truncate is the identity when it fits, so
				// measure first and only cut when the line is too wide.
				l = ansi.Truncate(l, s.W)
				w = ansi.Width(l)
			}
			// Each line is self-contained: re-open the style carried in
			// from earlier lines and close whatever is still open at its
			// end, so it never bleeds into a sibling's cells.
			end := carry
			if carry.active() {
				out.WriteString(carry.sgr)
				out.WriteString(carry.link)
			}
			out.WriteString(l)
			if hasEsc || carry.active() {
				end.scan(l)
				if end.link != "" {
					out.WriteString("\x1b]8;;\x1b\\")
				}
				if end.sgr != "" {
					out.WriteString("\x1b[0m")
				}
				carry.scan(full)
			}
		}
		writeSpaces(&out, s.W-w)
	}
	return out.String()
}

// nextLine splits off the first line of s without allocating: it returns the
// line, the remainder after the newline, and whether a remainder exists (a
// trailing "\n" yields a final empty line, matching strings.Split).
func nextLine(s string) (line, rest string, more bool) {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i], s[i+1:], true
	}
	return s, "", false
}

// flexSpec describes one item for solveFlex, in cells along the main axis.
type flexSpec struct {
	basis  int // preferred size before growing or shrinking
	grow   int // weight for sharing positive free space (0 = none)
	shrink int // weight, scaled by basis, for absorbing overflow (0 = none)
	min    int // lower bound
	max    int // upper bound; Unbounded for none
}

// solveFlex sizes items along one axis to fill avail cells. Each size
// starts at the item's basis clamped to [min, max]. Positive free space is
// shared among items with grow > 0 by weight; overflow is removed from
// items with shrink > 0 by weight*basis. An item that hits its bound is
// frozen at it and the rest re-share, so the loop runs at most len(items)
// times. All arithmetic is integer, with cumulative rounding so shares sum
// to exactly the amount distributed: the result sums to avail whenever the
// items' bounds allow it, and otherwise stops at the nearest reachable
// total (every size stays within [min, max]).
func solveFlex(avail int, items []flexSpec) []int { return solveFlexIn(nil, avail, items) }

// solveFlexIn is solveFlex with its scratch (and the result) taken from a.
func solveFlexIn(a *arena, avail int, items []flexSpec) []int {
	if avail < 0 {
		avail = 0
	}
	n := len(items)
	buf := intsFrom(a, 5*n)
	size, lo, hi := buf[:n:n], buf[n:2*n:2*n], buf[2*n:3*n:3*n]
	weight, frozen := buf[3*n:4*n:4*n], buf[4*n:5*n:5*n]
	total := 0
	for i, it := range items {
		lo[i] = clampAxis(it.min, 0, Unbounded)
		hi[i] = clampAxis(it.max, lo[i], Unbounded)
		size[i] = clampAxis(it.basis, lo[i], hi[i])
		total += size[i]
	}

	for iter := 0; iter <= n; iter++ {
		diff := avail - total
		if diff == 0 {
			break
		}
		sum := 0
		for i, it := range items {
			weight[i] = 0
			if frozen[i] != 0 {
				continue
			}
			w := 0
			switch {
			case diff > 0 && it.grow > 0 && size[i] < hi[i]:
				w = it.grow
			case diff < 0 && it.shrink > 0 && size[i] > lo[i]:
				w = it.shrink * max(size[i], 1)
			}
			weight[i] = w
			sum += w
		}
		if sum == 0 {
			break
		}
		amount := diff
		if amount < 0 {
			amount = -amount
		}
		cum, prev := 0, 0
		for i := range items {
			if weight[i] == 0 {
				continue
			}
			cum += weight[i]
			upTo := int(int64(amount) * int64(cum) / int64(sum))
			share := upTo - prev
			prev = upTo
			if diff < 0 {
				share = -share
			}
			next := clampAxis(size[i]+share, lo[i], hi[i])
			total += next - size[i]
			if next != size[i]+share {
				frozen[i] = 1
			}
			size[i] = next
		}
	}
	return size
}

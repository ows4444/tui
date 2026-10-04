package avatar

import (
	"bytes"
	"image/png"
	"math"
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
)

// bodyRows returns the mean row of the opaque pixels of m's PNG, their
// count, and the first and last rows that hold any.
func bodyRows(t *testing.T, m Model, size int) (mean float64, n, top, bottom int) {
	t.Helper()
	img, err := png.Decode(bytes.NewReader(m.PNG(size)))
	if err != nil {
		t.Fatal(err)
	}
	top, bottom = size, -1
	sum := 0
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a > 0x8000 {
				sum += y
				n++
				top, bottom = min(top, y), max(bottom, y)
			}
		}
	}
	return float64(sum) / float64(n), n, top, bottom
}

// A light pose lifts the body in the image and a heavy one sinks it; with no
// expression the image is the avatar's own.
func TestPosesLiftAndSinkTheBody(t *testing.T) {
	for _, name := range []string{"ada", "linus", "grace", "ken", "dennis", "alain00"} {
		rest := New(name)
		at, area, _, _ := bodyRows(t, rest, 200)
		none := New(name)
		none.Expression = ExpressionNone
		if !bytes.Equal(none.PNG(64), rest.PNG(64)) || none.View() != rest.View() {
			t.Errorf("%q: no expression changed the picture", name)
		}
		for e, up := range map[Expression]bool{ExpressionHappy: true, ExpressionLove: true, ExpressionSad: false, ExpressionSick: false} {
			m := New(name)
			m.Expression = e
			got, n, top, bottom := bodyRows(t, m, 200)
			if up && got >= at || !up && got <= at {
				t.Errorf("%q %v: the body's mean row went from %.2f to %.2f", name, e, at, got)
			}
			// Nothing is cut off: the body has the area it had at rest
			// and does not reach past the image's edge.
			if math.Abs(float64(n-area)) > 0.01*float64(area) {
				t.Errorf("%q %v: the body covers %d pixels, %d at rest", name, e, n, area)
			}
			if top < 0 || bottom > 199 {
				t.Errorf("%q %v: the body spans rows %d to %d", name, e, top, bottom)
			}
		}
	}
}

// A lift never carries the body past the edge of the drawn square, with or
// without a plate, and is whole pixels in cells.
func TestLiftStaysInsideTheSquare(t *testing.T) {
	for s := SilhouetteRound; s <= SilhouetteTriangle; s++ {
		for _, bg := range []Background{BackgroundNone, BackgroundSquare} {
			m := New("ada")
			m.Silhouette, m.Background = s, bg
			f := layoutFigure(m.traits())
			sc := newScene(f, m.plate())
			_, cy, side := sc.bounds()
			_, minY, _, maxY := sc.extent()
			for _, want := range []float64{-50, -2.2, 0, 2.6, 50} {
				sc.setLift(want, 0)
				if minY+sc.lift < cy-side/2-1e-9 || maxY+sc.lift > cy+side/2+1e-9 {
					t.Errorf("%v bg %d: a lift of %v carried the body out of the square", s, bg, want)
				}
				if want == 0 && sc.lift != 0 {
					t.Errorf("%v: no lift moved the body", s)
				}
			}
			// With pixels 4 units tall a lift of 2.6 is no whole pixel, and
			// with pixels 1 unit tall it is at most 2 of them.
			if sc.setLift(2.6, 4); sc.lift != 0 {
				t.Errorf("%v: a lift of 2.6 with 4-unit pixels became %v", s, sc.lift)
			}
			if sc.setLift(2.6, 1); sc.lift != math.Trunc(sc.lift) || sc.lift > 2 {
				t.Errorf("%v: a lift of 2.6 with 1-unit pixels became %v", s, sc.lift)
			}
		}
	}
}

// In a few cells a lift is under a pixel and draws nothing different from
// the same eyes with no lift; in many cells it moves the body a whole row
// of pixels.
func TestLiftInCells(t *testing.T) {
	// Over a plate the body has room to move; without one it is fitted to
	// the tile and has only the air around it.
	big := New("ada")
	big.Width, big.Height, big.Expression, big.Background = 120, 60, ExpressionSad, BackgroundSquare
	flat := big
	flat.cache = nil
	sunk := strings.Split(ansi.StripANSI(big.View()), "\n")
	// The same pose drawn with its lift removed.
	saved := poses[ExpressionSad]
	p := saved
	p.lift = 0
	poses[ExpressionSad] = p
	level := strings.Split(ansi.StripANSI(flat.View()), "\n")
	poses[ExpressionSad] = saved
	if strings.Join(sunk, "\n") == strings.Join(level, "\n") {
		t.Error("at 120x60 the sad pose's lift did not move the body")
	}
	small := New("ada")
	small.Width, small.Height, small.Expression = 16, 8, ExpressionSad
	smallFlat := small
	smallFlat.cache = nil
	withLift := small.View()
	poses[ExpressionSad] = p
	without := smallFlat.View()
	poses[ExpressionSad] = saved
	if withLift != without {
		t.Error("at 16x8 a lift under a pixel changed the picture")
	}
}

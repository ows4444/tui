package faces

import "strings"

// Everything below is art for the face itself. Eyes, brows, mouths and
// tears are CUT OUT of the solid head; effects are dots drawn beside it.
// Each is drawn twice, once per Size, since a 1-dot detail at 12x12 needs
// a 3-dot equivalent at 20x20 rather than a scaled copy.

// eyeStyle is the shape of an eye when open.
type eyeStyle int

const (
	styleDot eyeStyle = iota
	styleRound
	styleRing
	styleTall
	styleBlock
	styleDiamond
	styleStar
	styleOval
	styleSquare
)

var eyeArts = [2][]art{
	Small: {
		styleDot:     {"#", "#"},
		styleRound:   {"##", "##"},
		styleRing:    {"###", "#.#", "###"},
		styleTall:    {"#", "#", "#"},
		styleBlock:   {"###", "###"},
		styleDiamond: {".#.", "###", ".#."},
		styleStar:    {"#.#", ".#.", "#.#"},
		styleOval:    {"###", "###", ".#."},
		styleSquare:  {"##", "##", "##"},
	},
	Large: {
		styleDot:     {"##", "##", "##"},
		styleRound:   {".##.", "####", "####", ".##."},
		styleRing:    {".##.", "#..#", "#..#", ".##."},
		styleTall:    {"##", "##", "##", "##", "##"},
		styleBlock:   {"####", "####", "####"},
		styleDiamond: {"..#..", ".###.", "#####", ".###.", "..#.."},
		styleStar:    {"#...#", ".#.#.", "..#..", ".#.#.", "#...#"},
		styleOval:    {".###.", "#####", "#####", ".###."},
		styleSquare:  {"###", "###", "###", "###"},
	},
}

// eyeState is what one eye is doing.
type eyeState int

const (
	eyeOpen eyeState = iota
	eyeHalf
	eyeShut
	eyeWide
	eyeHappy
	eyeSpin0 // four rotations of a ring with a gap: dizzy
	eyeSpin1
	eyeSpin2
	eyeSpin3
	eyeSquint // '>' on the left eye, '<' on the right
	eyeGlitch
	eyeShadesHalf // sunglasses sliding down (drawn across both eyes)
	eyeShades
)

var (
	happyArt = [2]art{Small: {".#.", "#.#"}, Large: {"..#..", ".#.#.", "#...#"}}
	spinArt  = [2][4]art{
		Small: {{".##", "#.#", "###"}, {"##.", "#.#", "###"}, {"###", "#.#", "##."}, {"###", "#.#", ".##"}},
		Large: {
			{"..###", ".#..#", "#...#", "#..#.", ".###."},
			{"###..", "#..#.", "#...#", ".#..#", ".###."},
			{".###.", ".#..#", "#...#", "#..#.", "###.."},
			{".###.", "#..#.", "#...#", ".#..#", "..###"},
		},
	}
	squintArt = [2]art{Small: {"#.", ".#", "#."}, Large: {"#..", ".#.", "..#", ".#.", "#.."}}
)

// eyeShape returns the art for one eye in a state, and how many dots below
// the top of the eye band it sits. left says which side of the face it is
// on, for the directional states.
func eyeShape(sz Size, st eyeStyle, es eyeState, left bool) (art, int) {
	g := sz.geom()
	base := eyeArts[sz][st]
	y0 := (g.eyeMax - len(base)) / 2
	w := base.width()
	switch es {
	case eyeHalf:
		if len(base) > 1 {
			return base[len(base)/2:], y0 + len(base)/2
		}
		return base, y0
	case eyeShut:
		return art{strings.Repeat("#", w)}, y0 + len(base)/2
	case eyeWide:
		wide := append(art{base[0]}, base...)
		return append(wide, base[len(base)-1]), y0 - 1
	case eyeHappy:
		a := happyArt[sz]
		return a, y0 + (len(base)-len(a))/2
	case eyeSpin0, eyeSpin1, eyeSpin2, eyeSpin3:
		a := spinArt[sz][es-eyeSpin0]
		return a, (g.eyeMax - len(a)) / 2
	case eyeSquint:
		a := squintArt[sz]
		if !left {
			a = a.mirror()
		}
		return a, (g.eyeMax - len(a)) / 2
	case eyeGlitch:
		out := make(art, len(base))
		for j, row := range base {
			b := []byte(strings.Repeat(".", len(row)))
			for i := range b {
				if (i+j)%2 == 0 {
					b[i] = '#'
				}
			}
			out[j] = string(b)
		}
		return out, y0
	}
	return base, y0
}

// browKind is an eyebrow. Angry and sad are drawn for the left eye and
// mirrored for the right.
type browKind int

const (
	browNone browKind = iota
	browFlat
	browUp
	browAngry
	browSad
)

var browArts = [2]map[browKind]art{
	Small: {
		browFlat:  {"###"},
		browUp:    {".#.", "#.#"},
		browAngry: {"#..", ".##"},
		browSad:   {"..#", "##."},
	},
	Large: {
		browFlat:  {"#####"},
		browUp:    {"..#..", ".#.#.", "#...#"},
		browAngry: {"##...", ".###.", "...##"},
		browSad:   {"...##", ".###.", "##..."},
	},
}

// mouthKind is what the mouth is doing.
type mouthKind int

const (
	mIdle mouthKind = iota
	mSmile
	mGrin
	mOh
	mBig
	mFlat
	mFrown
	mWave
	mZig
	mZag
	mNoise
)

// mouthStyle picks a family of mouth shapes.
type mouthStyle int

const (
	mouthSoft mouthStyle = iota
	mouthBoxy
	mouthCat
)

// mouths[size][style][kind]; a missing kind falls back to the soft set.
var mouths = [2][3]map[mouthKind]art{
	Small: {
		mouthSoft: {
			mIdle: {"##"}, mSmile: {"#..#", ".##."}, mGrin: {"#....#", ".####."},
			mOh: {"##", "##"}, mBig: {"####", "####"}, mFlat: {"####"}, mFrown: {".##.", "#..#"},
			mWave: {"..#.#.", "#.#.#."}, mZig: {"#.#.#.", "######"}, mZag: {"#.#.#.", ".#.#.#"},
			mNoise: {"#.##.#", "##.##."},
		},
		mouthBoxy: {
			mIdle: {"####"}, mSmile: {"#..#", "####"}, mGrin: {"#....#", "######"},
			mOh: {"####", "#..#"}, mFrown: {"####", "#..#"},
		},
		mouthCat: {
			mIdle: {"#.##.#", ".#..#."}, mSmile: {"#..##..#", ".##..##."}, mGrin: {"#..##..#", ".######."},
		},
	},
	Large: {
		mouthSoft: {
			mIdle: {"#......#", ".######."}, mSmile: {"#......#", ".#....#.", "..####.."},
			mGrin: {"#........#", "##......##", ".########."},
			mOh:   {".##.", "####", "####", ".##."}, mBig: {".######.", "########", "########", ".######."},
			mFlat: {"########"}, mFrown: {"..####..", ".#....#.", "#......#"},
			mWave: {".#.#.#.#.#", "#.#.#.#.#."}, mZig: {"#.#.#.#.#.", "##########"}, mZag: {"#.#.#.#.#.", ".#.#.#.#.#"},
			mNoise: {"#.##.#.##.", "##.##.#.##"},
		},
		mouthBoxy: {
			mIdle: {"########"}, mSmile: {"#......#", "########"}, mGrin: {"#........#", "##########"},
			mOh: {"######", "#....#", "#....#", "######"}, mFrown: {"########", "#......#"},
			mBig: {"########", "########", "########", "########"},
		},
		mouthCat: {
			mIdle: {"#..##..#", ".##..##."}, mSmile: {"#...##...#", ".###..###."}, mGrin: {"#...##...#", ".##########."},
		},
	},
}

func mouthArt(sz Size, ms mouthStyle, k mouthKind) art {
	if a, ok := mouths[sz][ms][k]; ok {
		return a
	}
	return mouths[sz][mouthSoft][k]
}

var (
	blushArt = [2]art{Small: {"##"}, Large: {"###"}}
)

// fxKind is an effect drawn beside the head as filled dots.
type fxKind int

const (
	fxZ fxKind = iota
	fxZBig
	fxBang
	fxHeart
	fxSpark
	fxSweat
	fxSteam
	fxDot
	fxBubbleS
	fxBubbleM
	fxBubbleL
	fxFlake
)

var fxArts = [2][]art{
	Small: {
		fxZ:       {"###", ".#.", "###"},
		fxZBig:    {"#####", "...#.", "..#..", ".#...", "#####"},
		fxBang:    {"#", "#", ".", "#"},
		fxHeart:   {".#.#.", "#####", ".###.", "..#.."},
		fxSpark:   {".#.", "###", ".#."},
		fxSweat:   {"#", "#"},
		fxSteam:   {".#.#", "#.#."},
		fxDot:     {"#"},
		fxBubbleS: {"#"},
		fxBubbleM: {".#.", "#.#", ".#."},
		fxBubbleL: {".###.", "#...#", "#...#", "#...#", ".###."},
		fxFlake:   {"#.#", ".#.", "#.#"},
	},
	Large: {
		fxZ:       {"#####", "...#.", "..#..", ".#...", "#####"},
		fxZBig:    {"######", "...##.", "..##..", ".##...", "##....", "######"},
		fxBang:    {"##", "##", "##", "..", "##"},
		fxHeart:   {".#..#.", "######", "######", ".####.", "..##.."},
		fxSpark:   {"..#..", "..#..", "#####", "..#..", "..#.."},
		fxSweat:   {".#", "##", "##"},
		fxSteam:   {"..#.#", "#.#.#"},
		fxDot:     {"##", "##"},
		fxBubbleS: {"##", "##"},
		fxBubbleM: {".##.", "#..#", "#..#", ".##."},
		fxBubbleL: {".####.", "#....#", "#....#", "#....#", "#....#", ".####."},
		fxFlake:   {"#...#", ".#.#.", "..#..", ".#.#.", "#...#"},
	},
}

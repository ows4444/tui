package layout

// Junction glyphs. Two box-drawing characters in one cell make a third that has
// the lines of both: "┐" (a line going left and down) with "┌" (right and down)
// is "┬". The tables below give each line character the sides it touches and the
// family it belongs to; a family is thin, heavy, double or ASCII, since the
// Unicode block has junctions for same-weight lines only.

const (
	jUp = 1 << iota
	jDown
	jLeft
	jRight
)

type family uint8

const (
	famThin family = iota + 1
	famHeavy
	famDouble
	famASCII
)

type lineGlyph struct {
	sides uint8
	fam   family
}

var (
	glyphSides = map[rune]lineGlyph{}
	// glyphFor is the character for a set of sides in a family; an entry is
	// missing for a set no character draws (a single stub is allowed to be absent).
	glyphFor = map[family]map[uint8]rune{}
)

func init() {
	define := func(fam family, glyphs string, sides []uint8) {
		if glyphFor[fam] == nil {
			glyphFor[fam] = map[uint8]rune{}
		}
		for i, r := range []rune(glyphs) {
			glyphSides[r] = lineGlyph{sides[i], fam}
			glyphFor[fam][sides[i]] = r
		}
	}
	ud, lr := uint8(jUp|jDown), uint8(jLeft|jRight)
	dr, dl := uint8(jDown|jRight), uint8(jDown|jLeft)
	ur, ul := uint8(jUp|jRight), uint8(jUp|jLeft)
	udr, udl := uint8(jUp|jDown|jRight), uint8(jUp|jDown|jLeft)
	dlr, ulr := uint8(jDown|jLeft|jRight), uint8(jUp|jLeft|jRight)
	all := uint8(jUp | jDown | jLeft | jRight)
	order := []uint8{ud, lr, dr, dl, ur, ul, udr, udl, dlr, ulr, all}
	define(famThin, "│─┌┐└┘├┤┬┴┼", order)
	define(famHeavy, "┃━┏┓┗┛┣┫┳┻╋", order)
	define(famDouble, "║═╔╗╚╝╠╣╦╩╬", order)
	// Characters that draw the same sides as a thin or heavy one but are not what
	// a junction is made of: arcs and dashes. They merge as their solid kind.
	for r, g := range map[rune]lineGlyph{
		'╭': {dr, famThin}, '╮': {dl, famThin}, '╰': {ur, famThin}, '╯': {ul, famThin},
		'┄': {lr, famThin}, '┈': {lr, famThin}, '╌': {lr, famThin},
		'┆': {ud, famThin}, '┊': {ud, famThin}, '╎': {ud, famThin},
		'┅': {lr, famHeavy}, '┉': {lr, famHeavy}, '╍': {lr, famHeavy},
		'┇': {ud, famHeavy}, '┋': {ud, famHeavy}, '╏': {ud, famHeavy},
	} {
		glyphSides[r] = g
	}
	// ASCII borders: '+' is a corner or a junction of any shape.
	glyphSides['|'], glyphSides['-'], glyphSides['+'] = lineGlyph{ud, famASCII}, lineGlyph{lr, famASCII}, lineGlyph{all, famASCII}
	glyphFor[famASCII] = map[uint8]rune{ud: '|', lr: '-', dr: '+', dl: '+', ur: '+', ul: '+', udr: '+', udl: '+', dlr: '+', ulr: '+', all: '+'}
}

// mergeGlyph returns the character that has the lines of both a and b, and false
// when it cannot: one is not a line character, they are of different families, or
// no character draws the combination.
func mergeGlyph(a, b rune) (rune, bool) {
	ga, oka := glyphSides[a]
	gb, okb := glyphSides[b]
	if !oka || !okb || ga.fam != gb.fam {
		return 0, false
	}
	r, ok := glyphFor[ga.fam][ga.sides|gb.sides]
	return r, ok
}

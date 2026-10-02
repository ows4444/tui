package ansi

import "sort"

//go:generate go run ../internal/tools/gengrapheme -o graphemebreak_tables.go

// gbClass is a Grapheme_Cluster_Break property value (UAX #29).
type gbClass uint8

const (
	gbOther gbClass = iota
	gbCR
	gbLF
	gbControl
	gbExtend
	gbZWJ
	gbRegionalIndicator
	gbPrepend
	gbSpacingMark
	gbL
	gbV
	gbT
	gbLV
	gbLVT
)

// incbClass is an Indic_Conjunct_Break property value (UAX #29 rule GB9c).
type incbClass uint8

const (
	incbNone incbClass = iota
	incbConsonant
	incbLinker
	incbExtend
)

// gbRange is an inclusive rune range with its Grapheme_Cluster_Break value.
type gbRange struct {
	lo, hi rune
	class  gbClass
}

// runeRange is an inclusive rune range.
type runeRange struct{ lo, hi rune }

// incbRange is an inclusive rune range with its Indic_Conjunct_Break value.
type incbRange struct {
	lo, hi rune
	class  incbClass
}

// graphemeBreakOf returns r's Grapheme_Cluster_Break value, gbOther if it has
// none.
func graphemeBreakOf(r rune) gbClass {
	i := sort.Search(len(generatedGraphemeBreak), func(i int) bool { return generatedGraphemeBreak[i].hi >= r })
	if i < len(generatedGraphemeBreak) && generatedGraphemeBreak[i].lo <= r {
		return generatedGraphemeBreak[i].class
	}
	return gbOther
}

// isExtPict reports whether r is Extended_Pictographic.
func isExtPict(r rune) bool {
	i := sort.Search(len(generatedExtPict), func(i int) bool { return generatedExtPict[i].hi >= r })
	return i < len(generatedExtPict) && generatedExtPict[i].lo <= r
}

// incbOf returns r's Indic_Conjunct_Break value, incbNone if it has none.
func incbOf(r rune) incbClass {
	i := sort.Search(len(generatedIndicConjunctBreak), func(i int) bool { return generatedIndicConjunctBreak[i].hi >= r })
	if i < len(generatedIndicConjunctBreak) && generatedIndicConjunctBreak[i].lo <= r {
		return generatedIndicConjunctBreak[i].class
	}
	return incbNone
}

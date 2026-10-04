package avatar

import "testing"

// Names are trimmed and lowercased by the Unicode default rules.
func TestNormalizeTrimsAndLowercases(t *testing.T) {
	for in, want := range map[string]string{
		"  Alain  ":            "alain",
		"\ufeffA\u00a0":        "a",
		"\u2028A\u2029":        "a",
		"\u0085A":              "\u0085a", // NEL is not trimmed
		"\u0130":               "i\u0307",
		"\u039f\u03a3":         "\u03bf\u03c2", // a final sigma
		"\u03a3":               "\u03c3",
		"\u039f\u03a3\u0394":   "\u03bf\u03c3\u03b4",
		"\u039f\u03a3.":        "\u03bf\u03c2.",
		"\u039f.\u03a3\u0301A": "\u03bf.\u03c3\u0301a",
	} {
		if got := normalize(in); got != want {
			t.Errorf("normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

// A lone surrogate cannot be in a Go string; invalid bytes hash as U+FFFD,
// and a rune outside the BMP counts as two in the length.
func TestSeedStateCountsUTF16Units(t *testing.T) {
	if seedState("\xff") != seedState("\uFFFD") {
		t.Error("an invalid byte does not hash as U+FFFD")
	}
	if seedState("\U0001F98A") == feedString(1779033703^1, "\U0001F98A") {
		t.Error("an astral rune was counted as one unit")
	}
}

// A pinned trait is clamped into [0, 1) rather than trusted.
func TestPinnedTraitsAreClamped(t *testing.T) {
	tr := newTraits("alain", false)
	tr.fixed = map[string]float64{"low": -3, "high": 7, "mid": 0.5}
	for key, want := range map[string]float64{"low": 0, "high": 0.999999, "mid": 0.5} {
		if got := tr.at(key); got != want {
			t.Errorf("at(%q) = %v, want %v", key, got, want)
		}
	}
	if got := pickShape(1); got != shapeTriangle {
		t.Errorf("pickShape(1) = %v", got)
	}
}

// Every path command flattens to points on its outline.
func TestFlattenWalksEveryCommand(t *testing.T) {
	var p path
	p.add('M', 0, 0)
	p.add('H', 10)
	p.add('V', 10)
	p.add('L', 5, 12)
	p.add('Q', 0, 12, 0, 10)
	p.add('C', -2, 8, -2, 2, 0, 0)
	p.add('Z')
	pts := p.flatten()
	if want := 4 + quadSteps + cubicSteps; len(pts) != want {
		t.Fatalf("%d points, want %d", len(pts), want)
	}
	if pts[1] != (point{10, 0}) || pts[2] != (point{10, 10}) || pts[3] != (point{5, 12}) {
		t.Errorf("straight commands flattened to %v", pts[:4])
	}
	if last := pts[len(pts)-1]; last != (point{0, 0}) {
		t.Errorf("the outline ends at %v", last)
	}
	r := newRegion(p)
	if !r.contains(5, 5) || r.contains(20, 5) || r.contains(5, 30) {
		t.Error("contains is wrong for the flattened outline")
	}
	if got := p.String(); got != "M0 0H10V10L5 12Q0 12 0 10C-2 8 -2 2 0 0Z" {
		t.Errorf("String = %s", got)
	}
}

// When no lightness step reaches the floor, the colour falls back to black
// or white, whichever contrasts more.
func TestEnsureContrastFallsBackToBlackOrWhite(t *testing.T) {
	if got := ensureContrast(oklch{0.5, 0.1, 30}, oklch{0.5, 0, 0}, 50); got != (oklch{0, 0, 30}) && got != (oklch{1, 0, 30}) {
		t.Errorf("fallback = %+v", got)
	}
	if got := ensureContrast(oklch{0.9, 0, 0}, oklch{0.95, 0, 0}, 50); got != (oklch{0, 0, 0}) {
		t.Errorf("fallback against a light colour = %+v", got)
	}
	if got := ensureContrast(oklch{0.1, 0, 0}, oklch{0.05, 0, 0}, 50); got != (oklch{1, 0, 0}) {
		t.Errorf("fallback against a dark colour = %+v", got)
	}
}

// A cell holding three layers keeps the two that cover the most pixels.
func TestCellKeepsTwoLayers(t *testing.T) {
	g := grid{w: 1, h: 1, px: []layer{layerBody, layerBody, layerEye, layerPlate}}
	c := g.cellAt(0, 0)
	if c.ink != layerBody || c.paper != layerEye || c.mask != 0b1011 {
		t.Errorf("cell = %+v", c)
	}
	g.px = []layer{layerNone, layerEye, layerNone, layerNone}
	if c := g.cellAt(0, 0); c.ink != layerEye || c.paper != layerNone || c.mask != 0b0010 {
		t.Errorf("an eye over nothing = %+v", c)
	}
	g.px = []layer{layerPlate, layerEye, layerPlate, layerPlate}
	if c := g.cellAt(0, 0); c.ink != layerPlate || c.paper != layerEye || c.mask != 0b1101 {
		t.Errorf("an eye over the plate = %+v", c)
	}
}

package avatar

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"testing"
)

// vectors is testdata/vectors.json, the reference vectors that freeze the
// mapping from a name to its avatar. Nothing in it was produced by this
// package, and it is never regenerated: a mismatch means somebody's avatar
// changed.
type vectors struct {
	Hash []struct {
		Seed       string
		Normalized string
		// State is recorded as a signed 32-bit integer.
		State   int64
		Streams map[string]float64
	}
	Palette []struct {
		Hue, Tone float64
		Hex       struct{ Bg, Head, Eye string }
	}
	Cases []struct {
		Seed          string
		ShapeOverride *float64 `json:"shape_override"`
		Shape         string
		BodyPath      string
		EyePaths      []string
		Extra         []string
		Petals        int
		Palette       struct{ Bg, Head, Eye string }
	}
	Histogram map[string]int
	Markup    map[string]string
	Hashes    [][3]string
}

func loadVectors(t *testing.T) vectors {
	t.Helper()
	raw, err := os.ReadFile("testdata/vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var v vectors
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	return v
}

// nfcOnly lists the vector seeds this package cannot match because they are
// not in NFC and the standard library has no normalizer. The test asserts
// they differ, so that adding a normalizer shows up here.
var nfcOnly = map[string]bool{
	"cafe\u0301": true,
}

func TestHashMatchesVectors(t *testing.T) {
	v := loadVectors(t)
	if len(v.Hash) < 30 {
		t.Fatalf("only %d hash vectors", len(v.Hash))
	}
	skipped := 0
	for _, h := range v.Hash {
		got := normalize(h.Seed)
		if nfcOnly[h.Seed] {
			skipped++
			if got == h.Normalized {
				t.Errorf("%q now normalizes as the vectors expect; remove it from nfcOnly", h.Seed)
			}
			continue
		}
		if got != h.Normalized {
			t.Errorf("normalize(%q) = %q, want %q", h.Seed, got, h.Normalized)
			continue
		}
		state := seedState(got)
		if state != uint32(h.State) {
			t.Errorf("seedState(%q) = %d, want %d", h.Seed, state, uint32(h.State))
		}
		for key, want := range h.Streams {
			if got := stream(state, key); got != want {
				t.Errorf("stream(%q, %q) = %v, want %v", h.Seed, key, got, want)
			}
		}
	}
	if skipped != len(nfcOnly) {
		t.Errorf("%d of %d nfcOnly seeds are in the vectors", skipped, len(nfcOnly))
	}
}

func TestPaletteMatchesVectors(t *testing.T) {
	v := loadVectors(t)
	if len(v.Palette) < 100 {
		t.Fatalf("only %d palette vectors", len(v.Palette))
	}
	for _, p := range v.Palette {
		got := paletteFor(p.Hue, p.Tone)
		if hex(got.bg) != p.Hex.Bg || hex(got.head) != p.Hex.Head || hex(got.eye) != p.Hex.Eye {
			t.Errorf("hue %v tone %v = %s %s %s, want %s %s %s", p.Hue, p.Tone,
				hex(got.bg), hex(got.head), hex(got.eye), p.Hex.Bg, p.Hex.Head, p.Hex.Eye)
		}
	}
}

func TestGeometryMatchesVectors(t *testing.T) {
	v := loadVectors(t)
	seen := map[string]int{}
	for _, c := range v.Cases {
		tr := newTraits(c.Seed, false)
		if c.ShapeOverride != nil {
			tr.fixed = map[string]float64{"shape": *c.ShapeOverride}
		}
		f := layoutFigure(tr)
		name := fmt.Sprintf("%s(%s)", c.Seed, c.Shape)
		if got := shapes[f.kind].name; got != c.Shape {
			t.Errorf("%s: shape %s", name, got)
			continue
		}
		seen[c.Shape]++
		if got := f.core.String(); got != c.BodyPath {
			t.Errorf("%s: body\n got %s\nwant %s", name, got, c.BodyPath)
		}
		for i, want := range c.EyePaths {
			if got := f.eyes[i].String(); got != want {
				t.Errorf("%s: eye %d\n got %s\nwant %s", name, i, got, want)
			}
		}
		if len(f.extra) != len(c.Extra) {
			t.Errorf("%s: %d extra outlines, want %d", name, len(f.extra), len(c.Extra))
		} else {
			for i, want := range c.Extra {
				if got := f.extra[i].String(); got != want {
					t.Errorf("%s: extra %d\n got %s\nwant %s", name, i, got, want)
				}
			}
		}
		if len(f.petals) != c.Petals {
			t.Errorf("%s: %d petals, want %d", name, len(f.petals), c.Petals)
		}
		p := paletteFor(tr.num("hue", 0, 360), tr.at("tone"))
		if hex(p.bg) != c.Palette.Bg || hex(p.head) != c.Palette.Head || hex(p.eye) != c.Palette.Eye {
			t.Errorf("%s: palette %s %s %s, want %+v", name, hex(p.bg), hex(p.head), hex(p.eye), c.Palette)
		}
	}
	for _, s := range shapes {
		if seen[s.name] < 2 {
			t.Errorf("only %d cases cover %s", seen[s.name], s.name)
		}
	}
}

// The vectors record the markup of each silhouette, forced by
// pinning the shape trait, and of three whole avatars.
func TestMarkupMatchesVectors(t *testing.T) {
	v := loadVectors(t)
	pinned := map[string]float64{
		"round": 0.11, "organic": 0.35, "boxy": 0.54, "capsule": 0.65, "nub": 0.745,
		"cloud": 0.825, "droplet": 0.888, "hexagon": 0.933, "sun": 0.965, "triangle": 0.99,
	}
	for shape, at := range pinned {
		tr := newTraits("alain", false)
		tr.fixed = map[string]float64{"shape": at}
		got := svg(layoutFigure(tr), paletteFor(tr.num("hue", 0, 360), tr.at("tone")), nil)
		if want := v.Markup["shape:"+shape]; got != want {
			t.Errorf("shape:%s\n got %s\nwant %s", shape, got, want)
		}
	}
	squircle := New("alain")
	squircle.Background = BackgroundSquircle
	for key, m := range map[string]Model{"plain": New("alain"), "backdrop": squircle, "astral": New("\U0001F98A")} {
		if got, want := m.SVG(), v.Markup[key]; got != want {
			t.Errorf("%s\n got %s\nwant %s", key, got, want)
		}
	}
}

// The vectors also record a digest of the markup of a thousand names
// and of twelve of them under each background and with normalization off.
func TestMarkupDigestsMatchVectors(t *testing.T) {
	v := loadVectors(t)
	if len(v.Hashes) < 1000 {
		t.Fatalf("only %d digests", len(v.Hashes))
	}
	labels := map[string]func(*Model){
		"":                func(*Model) {},
		"bg:none":         func(m *Model) { m.Background = BackgroundNone },
		"bg:square":       func(m *Model) { m.Background = BackgroundSquare },
		"bg:circle":       func(m *Model) { m.Background = BackgroundCircle },
		"bg:squircle":     func(m *Model) { m.Background = BackgroundSquircle },
		"normalize:false": func(m *Model) { m.Raw = true },
	}
	bad := 0
	for _, h := range v.Hashes {
		seed, label, want := h[0], h[1], h[2]
		set, ok := labels[label]
		if !ok {
			t.Fatalf("unknown label %q", label)
		}
		m := New(seed)
		set(&m)
		sum := sha256.Sum256([]byte(m.SVG()))
		if got := fmt.Sprintf("%x", sum[:8]); got != want {
			if bad++; bad <= 5 {
				t.Errorf("%q %s: digest %s, want %s\n%s", seed, label, got, want, m.SVG())
			}
		}
	}
	if bad > 0 {
		t.Errorf("%d of %d digests differ", bad, len(v.Hashes))
	}
}

// The vectors count the silhouettes of 20,000 names; the bands must give the
// same counts.
func TestShapeHistogramMatchesVectors(t *testing.T) {
	v := loadVectors(t)
	got := map[string]int{}
	for i := 0; i < 20000; i++ {
		got[New(fmt.Sprintf("histogram-%d", i)).Shape()]++
	}
	for name, want := range v.Histogram {
		if got[name] != want {
			t.Errorf("%s: %d, want %d", name, got[name], want)
		}
	}
	if len(got) != len(shapes) || len(v.Histogram) != len(shapes) {
		t.Errorf("%d shapes drawn, %d in the fixture, want %d", len(got), len(v.Histogram), len(shapes))
	}
}

func TestRound2RoundsHalfUp(t *testing.T) {
	for in, want := range map[float64]float64{1.005: 1, 2.5: 2.5, -0.004: -0, -0.005: -0, -0.006: -0.01, 0.125: 0.13, -0.125: -0.12} {
		if got := round2(in); got != want {
			t.Errorf("round2(%v) = %v, want %v", in, got, want)
		}
	}
	if got := num(math.Copysign(0, -1)); got != "0" {
		t.Errorf("num(-0) = %q", got)
	}
}

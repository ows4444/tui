package tuitest

import (
	"encoding/xml"
	"io"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
)

type paint struct{ view string }

func (paint) Init() tui.Cmd                         { return nil }
func (p paint) Update(tui.Msg) (tui.Model, tui.Cmd) { return p, nil }
func (p paint) View() string                        { return p.view }

// wellFormed fails the test unless s is a well-formed XML document.
func wellFormed(t *testing.T, s string) {
	t.Helper()
	d := xml.NewDecoder(strings.NewReader(s))
	for {
		if _, err := d.Token(); err != nil {
			if err != io.EOF {
				t.Fatalf("not well-formed XML: %v\n%s", err, s)
			}
			return
		}
	}
}

func TestSVGDrawsTextColoursAndAttributes(t *testing.T) {
	view := ansi.NewStyle().Foreground(ansi.Red).Bold().Render("hot") + " " +
		ansi.NewStyle().Background(ansi.RGB{R: 1, G: 2, B: 3}).Italic().Render("bg") + "\nplain"
	s := New(paint{view}, 10, 2)
	defer s.Close()
	svg := s.SVG(SVGOptions{})
	wellFormed(t, svg)
	for _, want := range []string{
		`width="112" height="67"`,
		`fill="#cd3131" font-weight="bold">hot</text>`, // red, bold
		`fill="#010203"`,                               // the background rect
		`font-style="italic">bg</text>`,
		`>plain</text>`,
	} {
		if !strings.Contains(svg, want) {
			t.Errorf("SVG lacks %q:\n%s", want, svg)
		}
	}
	if strings.Contains(svg, "<circle") {
		t.Error("a title bar was drawn without a Title")
	}
}

func TestSVGEscapesTextAndHandlesWideCells(t *testing.T) {
	s := New(paint{`<a href="x">&'</a>` + "\n世界ab"}, 20, 2)
	defer s.Close()
	svg := s.SVG(SVGOptions{})
	wellFormed(t, svg)
	if !strings.Contains(svg, "&lt;ahref=&quot;x&quot;&gt;&amp;&apos;&lt;/a&gt;") {
		t.Errorf("text not escaped:\n%s", svg)
	}
	if !strings.Contains(svg, "世界ab") || strings.Contains(svg, "\x00") {
		t.Errorf("wide characters wrong:\n%s", svg)
	}
}

func TestSVGOptions(t *testing.T) {
	fg, bg := ansi.RGB{R: 0xaa, G: 0xbb, B: 0xcc}, ansi.RGB{R: 0x10, G: 0x20, B: 0x30}
	pal := defaultPalette
	pal[ansi.Red] = ansi.RGB{R: 0xff, G: 0, B: 0}
	s := New(paint{ansi.NewStyle().Foreground(ansi.Red).Render("x") + " " +
		ansi.NewStyle().Reverse().Render("r")}, 6, 1)
	defer s.Close()
	svg := s.SVG(SVGOptions{Title: "demo <1>", Foreground: &fg, Background: &bg, Palette: &pal, FontSize: 10, FontFamily: "Fira Code", Cursor: true})
	wellFormed(t, svg)
	for _, want := range []string{
		`fill="#102030"`, // the background
		`fill="#ff0000"`, // the custom palette's red
		`demo &lt;1&gt;`, // the title, escaped
		`<circle`,        // the title bar
		`font-size="10"`, // size
		`font-family="Fira Code"`,
		`fill="#aabbcc"`, // the foreground; reverse swaps it into a rect
	} {
		if !strings.Contains(svg, want) {
			t.Errorf("SVG lacks %q:\n%s", want, svg)
		}
	}
}

func TestSVGResolves256Colours(t *testing.T) {
	pal := &defaultPalette
	def := ansi.RGB{R: 9, G: 9, B: 9}
	cases := map[ansi.Color]ansi.RGB{
		nil:                        def,
		ansi.Color256(1):           pal[1],
		ansi.Color256(16):          {R: 0, G: 0, B: 0},
		ansi.Color256(196):         {R: 255, G: 0, B: 0},
		ansi.Color256(232):         {R: 8, G: 8, B: 8},
		ansi.Color256(255):         {R: 238, G: 238, B: 238},
		ansi.RGB{R: 1, G: 2, B: 3}: {R: 1, G: 2, B: 3},
	}
	for in, want := range cases {
		if got := colorOrDefault(in, def, pal); got != want {
			t.Errorf("colorOrDefault(%v) = %v, want %v", in, got, want)
		}
	}
}

func TestSVGTrimRows(t *testing.T) {
	s := New(paint{"one\ntwo"}, 10, 6)
	defer s.Close()
	full, trimmed := s.SVG(SVGOptions{}), s.SVG(SVGOptions{TrimRows: true})
	wellFormed(t, trimmed)
	if !strings.Contains(full, `height="146"`) || !strings.Contains(trimmed, `height="67"`) {
		t.Errorf("heights: full %q trimmed %q", full[:90], trimmed[:90])
	}
	blank := New(paint{""}, 10, 3)
	defer blank.Close()
	if !strings.Contains(blank.SVG(SVGOptions{TrimRows: true}), `height="48"`) {
		t.Error("an empty screen is not kept at one row")
	}
}

func TestSVGOfMatchesASession(t *testing.T) {
	view := ansi.NewStyle().Foreground(ansi.Red).Bold().Render("hi") + " there\nsecond"
	s := New(paint{view: view}, 20, 4)
	defer s.Close()
	want := s.SVG(SVGOptions{TrimRows: true})
	got := SVGOf(view, 20, 4, SVGOptions{TrimRows: true})
	wellFormed(t, got)
	if got != want {
		t.Fatalf("SVGOf differs from Session.SVG\n got  %s\n want %s", got, want)
	}
}

func TestSVGOfClampsItsSize(t *testing.T) {
	wellFormed(t, SVGOf("x", 0, -3, SVGOptions{}))
}

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// token is one colour a theme file can set.
type token struct {
	key, label string
	get        func(theme.Theme) ansi.Color
}

var tokens = []token{
	{"primary", "Primary", func(t theme.Theme) ansi.Color { return t.Primary }},
	{"secondary", "Secondary", func(t theme.Theme) ansi.Color { return t.Secondary }},
	{"success", "Success", func(t theme.Theme) ansi.Color { return t.Success }},
	{"warning", "Warning", func(t theme.Theme) ansi.Color { return t.Warning }},
	{"error", "Error", func(t theme.Theme) ansi.Color { return t.Error }},
	{"info", "Info", func(t theme.Theme) ansi.Color { return t.Info }},
	{"muted", "Muted", func(t theme.Theme) ansi.Color { return t.Muted }},
	{"text", "Text", func(t theme.Theme) ansi.Color { return t.Text }},
	{"text_inverse", "Text inverse", func(t theme.Theme) ansi.Color { return t.TextInverse }},
	{"border_color", "Border", func(t theme.Theme) ansi.Color { return t.BorderColor }},
	{"focus", "Focus", func(t theme.Theme) ansi.Color { return t.Focus }},
	{"selection", "Selection", func(t theme.Theme) ansi.Color { return t.Selection }},
	{"background", "Background", func(t theme.Theme) ansi.Color { return t.Background }},
	{"surface", "Surface", func(t theme.Theme) ansi.Color { return t.Surface }},
	{"overlay", "Overlay", func(t theme.Theme) ansi.Color { return t.Overlay }},
}

// borders and glyph sets a file can name; "" leaves the base theme's own.
var (
	borderNames = []string{"", "normal", "rounded", "ascii", "double", "thick"}
	glyphNames  = []string{"", "unicode", "ascii", "nerd"}
)

// toRGB resolves any colour to RGB. A nil colour is black.
func toRGB(c ansi.Color) ansi.RGB {
	switch v := c.(type) {
	case ansi.RGB:
		return v
	case ansi.Color256:
		return v.RGB()
	case ansi.BasicColor:
		return v.RGB()
	}
	return ansi.RGB{}
}

func hex(c ansi.RGB) string { return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B) }

// parseHex reads "#rrggbb", "rrggbb" or "#rgb".
func parseHex(s string) (ansi.RGB, bool) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) == 3 {
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	}
	if len(s) != 6 {
		return ansi.RGB{}, false
	}
	var r, g, b uint8
	if _, err := fmt.Sscanf(s, "%02x%02x%02x", &r, &g, &b); err != nil {
		return ansi.RGB{}, false
	}
	return ansi.RGB{R: r, G: g, B: b}, true
}

// spec is what the studio edits: a base theme and what is changed on it.
type spec struct {
	base   string
	over   map[string]string // token key to "#rrggbb"
	border int               // index into borderNames
	glyphs int               // index into glyphNames
}

// document is the theme file for s: the base first, then the changed colours
// in token order, then the border and the glyph set.
func (s spec) document() []byte {
	var b bytes.Buffer
	b.WriteString("{\n")
	line := func(k, v string) {
		kb, _ := json.Marshal(k)
		vb, _ := json.Marshal(v)
		fmt.Fprintf(&b, "  %s: %s,\n", kb, vb)
	}
	line("base", s.base)
	for _, t := range tokens {
		if v, ok := s.over[t.key]; ok {
			line(t.key, v)
		}
	}
	if s.border > 0 {
		line("border", borderNames[s.border])
	}
	if s.glyphs > 0 {
		line("glyphs", glyphNames[s.glyphs])
	}
	out := bytes.TrimSuffix(b.Bytes(), []byte(",\n"))
	return append(out, []byte("\n}\n")...)
}

// build makes the theme by reading the same document the export writes, so
// the preview and the file cannot differ.
func (s spec) build() (theme.Theme, error) {
	return theme.Parse(bytes.NewReader(s.document()))
}

func (s spec) changes() int {
	n := len(s.over)
	if s.border > 0 {
		n++
	}
	if s.glyphs > 0 {
		n++
	}
	return n
}

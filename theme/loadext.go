package theme

import (
	"fmt"
	"slices"
	"strings"

	"github.com/ows4444/tui/ansi"
)

type fileSpacing struct {
	XS *int `json:"xs"`
	S  *int `json:"s"`
	M  *int `json:"m"`
	L  *int `json:"l"`
}

// fileStyle is the JSON form of an ansi.Style.
type fileStyle struct {
	FG        string `json:"fg"`
	BG        string `json:"bg"`
	Bold      bool   `json:"bold"`
	Faint     bool   `json:"faint"`
	Italic    bool   `json:"italic"`
	Underline bool   `json:"underline"`
	Reverse   bool   `json:"reverse"`
	Strike    bool   `json:"strike"`
}

type fileStates struct {
	Focus    *fileStyle `json:"focus"`
	Hover    *fileStyle `json:"hover"`
	Disabled *fileStyle `json:"disabled"`
	Selected *fileStyle `json:"selected"`
}

type fileTypography struct {
	H1       *fileStyle `json:"h1"`
	H2       *fileStyle `json:"h2"`
	Emphasis *fileStyle `json:"emphasis"`
	Strong   *fileStyle `json:"strong"`
	Code     *fileStyle `json:"code"`
	Link     *fileStyle `json:"link"`
}

// fileTokens is the JSON form of Tokens.
type fileTokens struct {
	Text        string `json:"text"`
	Muted       string `json:"muted"`
	Accent      string `json:"accent"`
	Focus       string `json:"focus"`
	Selection   string `json:"selection"`
	Border      string `json:"border"`
	Background  string `json:"background"`
	Surface     string `json:"surface"`
	Success     string `json:"success"`
	Warning     string `json:"warning"`
	Error       string `json:"error"`
	Info        string `json:"info"`
	Overlay     string `json:"overlay"`
	TextInverse string `json:"text_inverse"`
}

func (s *fileStyle) style(path string) (ansi.Style, error) {
	st := ansi.NewStyle()
	if s.FG != "" {
		c, err := parseHex(s.FG)
		if err != nil {
			return st, fmt.Errorf("theme: %s.fg: %w", path, err)
		}
		st = st.Foreground(c)
	}
	if s.BG != "" {
		c, err := parseHex(s.BG)
		if err != nil {
			return st, fmt.Errorf("theme: %s.bg: %w", path, err)
		}
		st = st.Background(c)
	}
	for _, f := range []struct {
		on bool
		fn func(ansi.Style) ansi.Style
	}{
		{s.Bold, ansi.Style.Bold}, {s.Faint, ansi.Style.Faint}, {s.Italic, ansi.Style.Italic},
		{s.Underline, ansi.Style.Underline}, {s.Reverse, ansi.Style.Reverse}, {s.Strike, ansi.Style.Strikethrough},
	} {
		if f.on {
			st = f.fn(st)
		}
	}
	return st, nil
}

// applyExtras applies the parts of a theme file that are not colour roles.
func applyExtras(t *Theme, f fileTheme) error {
	switch strings.ToLower(f.Glyphs) {
	case "":
	case "unicode":
		t.Glyphs = UnicodeGlyphSet()
	case "ascii":
		t.Glyphs = ASCIIGlyphSet()
	case "nerd":
		t.Glyphs = NerdGlyphSet()
	default:
		return fmt.Errorf("theme: glyphs: %q is not unicode, ascii or nerd", f.Glyphs)
	}
	if sp := f.Spacing; sp != nil {
		for _, e := range []struct {
			name string
			v    *int
			dst  *int
		}{{"xs", sp.XS, &t.Spacing.XS}, {"s", sp.S, &t.Spacing.S}, {"m", sp.M, &t.Spacing.M}, {"l", sp.L, &t.Spacing.L}} {
			if e.v == nil {
				continue
			}
			if *e.v < 0 {
				return fmt.Errorf("theme: spacing.%s: %d is negative", e.name, *e.v)
			}
			*e.dst = *e.v
		}
	}
	set := func(path string, fs *fileStyle, dst *ansi.Style) error {
		if fs == nil {
			return nil
		}
		st, err := fs.style(path)
		if err != nil {
			return err
		}
		*dst = st
		return nil
	}
	if s := f.States; s != nil {
		for _, e := range []struct {
			name string
			fs   *fileStyle
			dst  *ansi.Style
		}{{"states.focus", s.Focus, &t.States.Focus}, {"states.hover", s.Hover, &t.States.Hover},
			{"states.disabled", s.Disabled, &t.States.Disabled}, {"states.selected", s.Selected, &t.States.Selected}} {
			if err := set(e.name, e.fs, e.dst); err != nil {
				return err
			}
		}
	}
	if ty := f.Typography; ty != nil {
		for _, e := range []struct {
			name string
			fs   *fileStyle
			dst  *ansi.Style
		}{{"typography.h1", ty.H1, &t.Typography.H1}, {"typography.h2", ty.H2, &t.Typography.H2},
			{"typography.emphasis", ty.Emphasis, &t.Typography.Emphasis}, {"typography.strong", ty.Strong, &t.Typography.Strong},
			{"typography.code", ty.Code, &t.Typography.Code}, {"typography.link", ty.Link, &t.Typography.Link}} {
			if err := set(e.name, e.fs, e.dst); err != nil {
				return err
			}
		}
	}
	known := Components()
	names := make([]string, 0, len(f.Components))
	for n := range f.Components {
		names = append(names, n)
	}
	slices.Sort(names)
	for _, n := range names {
		if !slices.Contains(known, n) {
			return fmt.Errorf("theme: components: %q is not a component (one of %s)", n, strings.Join(known, ", "))
		}
		tok, err := f.Components[n].tokens("components." + n)
		if err != nil {
			return err
		}
		*t = t.WithTokens(n, tok)
	}
	return nil
}

func (ft fileTokens) tokens(path string) (Tokens, error) {
	var tok Tokens
	for _, e := range []struct {
		name, v string
		dst     *ansi.Color
	}{
		{"text", ft.Text, &tok.Text}, {"muted", ft.Muted, &tok.Muted}, {"accent", ft.Accent, &tok.Accent},
		{"focus", ft.Focus, &tok.Focus}, {"selection", ft.Selection, &tok.Selection}, {"border", ft.Border, &tok.Border},
		{"background", ft.Background, &tok.Background}, {"surface", ft.Surface, &tok.Surface},
		{"success", ft.Success, &tok.Success}, {"warning", ft.Warning, &tok.Warning}, {"error", ft.Error, &tok.Error},
		{"info", ft.Info, &tok.Info}, {"overlay", ft.Overlay, &tok.Overlay}, {"text_inverse", ft.TextInverse, &tok.TextInverse},
	} {
		if e.v == "" {
			continue
		}
		c, err := parseHex(e.v)
		if err != nil {
			return tok, fmt.Errorf("theme: %s.%s: %w", path, e.name, err)
		}
		*e.dst = c
	}
	return tok, nil
}

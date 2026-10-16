package widgets

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// maxTraceDepth stops a chain that wraps itself.
const maxTraceDepth = 32

// ErrorTrace renders err and every error it wraps as a tree: what each layer
// added to the message, and its Go type, one level under the next, so a long
// "open a: read b: permission denied" reads as a path to the cause. An
// errors.Join (or any error with Unwrap() []error) shows its members as
// siblings. Text is sanitised, since an error message often holds input the
// program did not write. A nil err renders as "".
func ErrorTrace(err error) string { return ErrorTraceWith(err, theme.Theme{}) }

// ErrorTraceWith is ErrorTrace with the error mark and colour taken from t and
// the tree connectors from its glyph set. ErrorTrace is ErrorTraceWith on the
// default theme.
func ErrorTraceWith(err error, t theme.Theme) string {
	if err == nil {
		return ""
	}
	g := t.GlyphSet()
	typeStyle := ansi.NewStyle().Faint()
	label := func(e error, own string) string {
		typ := typeStyle.Render("(" + fmt.Sprintf("%T", e) + ")")
		if own == "" {
			return typ
		}
		return own + " " + typ
	}
	var build func(e error, depth int) (string, []TreeNode)
	build = func(e error, depth int) (string, []TreeNode) {
		own, kids := traceParts(e)
		out := label(e, ansi.Sanitize(own))
		if depth >= maxTraceDepth {
			return out, nil
		}
		var nodes []TreeNode
		for _, k := range kids {
			l, c := build(k, depth+1)
			nodes = append(nodes, TreeNode{Label: l, Children: c})
		}
		return out, nodes
	}
	rootLabel, children := build(err, 0)
	root := ansi.NewStyle().Foreground(t.Error).Bold().Render(g.Cross + " " + rootLabel)
	if len(children) == 0 {
		return root
	}
	return root + "\n" + TreeWith(children, t)
}

// traceParts splits err into the text it adds itself and the errors it wraps.
func traceParts(err error) (own string, wrapped []error) {
	switch u := err.(type) {
	case interface{ Unwrap() []error }:
		kids := u.Unwrap()
		return fmt.Sprintf("%d errors", len(kids)), kids
	case interface{ Unwrap() error }:
		inner := u.Unwrap()
		if inner == nil {
			return err.Error(), nil
		}
		msg, in := err.Error(), inner.Error()
		switch {
		case msg == in:
			return "", []error{inner}
		case strings.HasSuffix(msg, in):
			return strings.TrimRight(strings.TrimSuffix(msg, in), ": "), []error{inner}
		}
		return msg, []error{inner}
	}
	return err.Error(), nil
}

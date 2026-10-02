// Package cellbuf is a retained grid of terminal cells: the fast path for a
// widget that wants to draw straight into cells instead of building a styled
// string that the renderer parses back every frame (spec S04). The View()
// string stays the primary way to draw; a Buffer is additive and converts to
// and from styled strings (Parse and Buffer.String), so the two compose.
//
// A Buffer is a width x height grid. Each Cell holds one grapheme cluster, its
// column width (1, or 2 for a wide cluster) and a StyleID that indexes the
// Buffer's style table. A wide cluster occupies a head cell followed by a
// continuation cell (Width 0, empty Cluster). No operation ever leaves half of
// a wide cluster in the grid: writing over one half blanks the other, and a
// wide cluster that does not fit in the last column is replaced by a blank
// instead of being split.
//
// Sub returns a clipped view that shares the same cells and styles, so a
// layout can hand each child its own rectangle to draw into.
//
// Parsing follows the cell renderer's rules (the same SGR subset, OSC 8
// hyperlinks, tabs expanded to spaces, grapheme clusters measured by an [ansi.Measurer]); a string the
// grid cannot represent (control characters other than tab, other escapes,
// unknown SGR codes) is rejected with an error wrapping [ErrUnsupported].
//
// Buffers are not safe for concurrent use. The package uses only the standard
// library and this module's ansi package.
package cellbuf

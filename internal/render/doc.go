// Package render is the cell-diffing renderer behind the root package's
// default renderer. Cells.Frame turns a view, either lines of styled text or a
// GridSource drawn directly, into a grid of cells, compares it with the
// previous frame and emits only the escape sequences and text needed to repaint
// the cells that changed. It performs no terminal I/O itself: callers write the
// bytes it returns.
package render

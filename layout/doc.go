// Package layout composes pre-rendered blocks and measured nodes into
// terminal layouts.
//
// A Node is a two-pass measure/render protocol: it reports a preferred Size
// under Constraints (Measure) and is then handed a final Size to fill exactly
// (Render). Block adapts a pre-rendered string. Row, Column, RowJustify and
// ColumnJustify size FlexChild values along the main axis with Basis,
// BasisLen (Pct and Fr), Grow, Shrink, Min and Max, place them on the cross
// axis with CrossAlign, and space them with Justify. Fill and FillWeight take
// leftover space; Scroll windows a tall child; BoxNode, GridNode, MinSize,
// Responsive and Overlay compose nodes; Text is a wrapping text node. Draw
// renders at the measured size (Loose); DrawTight renders at exactly a given
// size so Fill children absorb all remaining space. A node that also implements
// CellNode draws straight into a CellSurface (DrawTo, with cellbuf.Layout
// adapting a cellbuf.Buffer) without building a string.
//
// Box, Overlay and JoinHorizontalJustify work on pre-rendered strings. The
// string join helpers that once sat beside them (JoinHorizontal, JoinVertical,
// FlexRow, Grid, GridFlex) were removed in v1.0; docs/migrating-to-v1.md maps
// each to its node.
package layout

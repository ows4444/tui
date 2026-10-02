# Migrating to v1

Each identifier removed before v1, and what to use instead. The `- BREAKING:`
entries under Removed in [CHANGELOG.md](../CHANGELOG.md) list the same
removals. Deprecated identifiers that still exist, such as `layout.Box.Render`,
are listed under Deprecated there.

## Program options (package tui)

`WithInput`, `WithOutput` and `WithErrOutput` now take any `io.Reader` or
`io.Writer`, file or not, so the separate options for non-file streams are
gone. Option defaults are in [program.md](program.md).

### WithInputReader

Use `WithInput`. Given a reader that is not an `*os.File`, `WithInput` behaves
as `WithInputReader` did: `Run` needs no terminal and doesn't switch raw mode,
and the Program quits when the reader returns `io.EOF`
(`TestWithInputPlainReaderNeedsNoTerminal`). Given an `*os.File`, `WithInput`
reads it as a terminal, and `Run` fails with a "not a terminal" error if it
isn't one (`TestWithInputFileIsReadAsATerminal`). To read a file as a plain
stream instead, wrap it so it is no longer an `*os.File`, as the `WithInput`
godoc shows (`struct{ io.Reader }{f}`).

A reader that also implements `io.Closer` is closed when the Program quits.
`WithInputCloser` names a different closer to use instead.

### WithOutputWriter

Use `WithOutput`. A writer that is not an `*os.File`, such as a
`bytes.Buffer`, has no terminal size: the Program assumes 80x24 and never
queries or resizes it. A `ResizeMsg` can still deliver another size.

### WithErrWriter

Use `WithErrOutput`, which sets where `Eprintln` writes (default
`os.Stderr`). Any `io.Writer` receives the text as it is.

### WithLineRenderer

Use `WithCellRenderer(false)`. The cell renderer is the default, and
`WithCellRenderer(false)` selects the line renderer. See
[rendering.md](rendering.md#cell-renderer-and-line-renderer) for how the two
differ.

## String layout helpers (package layout)

The string join and grid helpers are replaced by Nodes. Wrap each
pre-rendered string in `layout.Block` to make it a Node, then compose Nodes
with `Row`, `Column` and `GridNode`. `ExampleCrossAlign` and `ExampleGridNode`
in `layout/example_test.go` show the replacements in use, and
[layout.md](layout.md) covers sizing with `FlexChild`.

`Box`, `Overlay` and `JoinHorizontalJustify` still work on strings and are not
removed.

### JoinHorizontal

Use `layout.Row(gap, children...)`, with one `FlexChild` per block.

### JoinHorizontalAlign

Use `layout.Row` and set `CrossAlign` on each `FlexChild`: `CrossStart`,
`CrossCenter` or `CrossEnd`. The zero value, `CrossStretch`, stretches the
child to the row's full height.

### JoinVertical

Use `layout.Column(gap, children...)`, with one `FlexChild` per block.

### JoinVerticalAlign

Use `layout.Column` and set `CrossAlign` on each `FlexChild`, as for
`JoinHorizontalAlign`. In a Column the cross axis is horizontal, so
`CrossStart` keeps a child at its own width, flush left
(`ExampleColumn_crossStart`).

### FlexRow

Use `layout.Row`. Sizing that was set per item now goes in each child's
`FlexChild` fields (`Basis`, `BasisLen`, `Grow`, `Shrink`, `Min`, `Max`).

### FlexItem

Use `layout.FlexChild`. The zero `FlexChild` is sized to its content and
neither grows nor shrinks; `layout.Fill(n)` makes a child take the space left
over.

### Grid

Use `layout.GridNode(tracks, gap, cells...)`. It takes one `Track` per column
and fills cells row by row. `GridNodeGaps` gives rows and columns different
gaps.

### GridFlex

Use `layout.GridNode`. A `Track` with `Grow > 0` shares the grid's leftover
width by weight, on top of the column's content width.

### ColSpec

Use `layout.Track`. The zero `Track` sizes the column to its widest cell,
`Size > 0` fixes the width, and `Grow > 0` shares leftover width.

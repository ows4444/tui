# Changelog

User-visible changes to `github.com/ows4444/tui`. The format loosely follows
[Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/). There is no
tagged release yet, so everything is under Unreleased.

A removed or changed exported identifier needs a `- BREAKING:` entry under
Unreleased that names it; see [CONTRIBUTING.md](CONTRIBUTING.md#changelog).

## [Unreleased]

### Deprecated

- The chart functions in `widgets` (`BarChart`, `Gauge`, `HeatMap`,
  `LineChart`, `Sparkline`, `SparklineWith` and the `BarItem` alias) moved to
  `widgets/chart`; the `widgets` versions forward to them.
- `layout.Box.Render`: use `layout.BoxNode(box, layout.Block(content))`, or
  `BoxNode` with any Node child.
- `ansi.SetClusterWidth`: use an `ansi.Measurer` for anything tied to one
  terminal. A Program no longer calls it.

### Removed

- BREAKING: `WithInputReader`, `WithOutputWriter` and `WithErrWriter` are
  removed. `WithInput`, `WithOutput` and `WithErrOutput` take any reader or
  writer: a non-file reader given to `WithInput` needs no terminal and quits
  the Program at EOF, as `WithInputReader` did.
- BREAKING: `WithLineRenderer` is removed. `WithCellRenderer(false)` selects
  the line renderer.
- BREAKING: `layout.JoinHorizontal`, `JoinHorizontalAlign`, `JoinVertical`,
  `JoinVerticalAlign`, `FlexRow`, `FlexItem`, `GridFlex`, `Grid` and `ColSpec`
  are removed. Use the Node constructors: `JoinHorizontal` and `FlexRow`
  become `layout.Row`, `JoinVertical` becomes `layout.Column`, the `*Align`
  variants set `FlexChild.CrossAlign` on each child, `GridFlex` and `Grid` become
  `layout.GridNode`, `ColSpec` becomes `layout.Track`, and `FlexItem` becomes
  `layout.FlexChild`.

### Fixed

- `SuspendMsg.Err` reports a failure to re-enter raw mode after a `Suspend`
  whose function succeeded. It was nil before, with the terminal left in its
  normal mode.
- With `WithRecover(true)`, a panic in a `Tick` or `FromCtx` Cmd, a
  `Sequence` step or an `Every` callback now makes Run return a `*PanicError`.
  Before, these restored the terminal and re-panicked.

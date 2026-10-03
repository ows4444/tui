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

- `Go` behaves as `FromCtx` does: `Sequence` waits for a `Go` Cmd before
  starting the next one, and `RunCmd` runs it and returns its Msg. Before,
  `Sequence` moved on at once and `RunCmd` returned an internal value.
- `imageview`: a Model from `New` encodes its image once and reuses the
  result until the image, size, mode, id or cell size changes. It used to
  decode and encode on every `View`. Bytes of `PNG` overwritten in place are
  not noticed; assign a new slice.
- `tuitest.Session.Keys` sends every key name `Key.String` produces
  (function keys, `insert`, modified keys such as `ctrl+left` and
  `ctrl+shift+a`, media keys) as that key. Before, names it did not know were
  typed one character at a time. An argument that is exactly a key name can
  no longer be typed as text with `Keys`.
- `Run` cancels `Context` and releases callers blocked in `Send` when it
  returns before the loop starts (the input is not a terminal, or raw mode
  cannot be entered). A `Run` called while the first is still running no
  longer reads the running model; it returns the model given to `NewProgram`.
- `SuspendMsg.Err` reports a failure to re-enter raw mode after a `Suspend`
  whose function succeeded. It was nil before, with the terminal left in its
  normal mode.
- With `WithRecover(true)`, a panic in a `Tick` or `FromCtx` Cmd, a
  `Sequence` step or an `Every` callback now makes Run return a `*PanicError`.
  Before, these restored the terminal and re-panicked.

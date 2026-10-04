# Changelog

User-visible changes to `github.com/ows4444/tui`. The format loosely follows
[Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/). There is no
tagged release yet, so everything is under Unreleased.

A removed or changed exported identifier needs a `- BREAKING:` entry under
Unreleased that names it; see [CONTRIBUTING.md](CONTRIBUTING.md#changelog).

## [Unreleased]

### Added

- `faces.For(name)` and `faces.IndexFor(name)` pick a catalog face for any
  string, the same one every time, so a name can have a character of its
  own. `faces.Model.Color` draws the face in a colour other than the theme's
  primary, and `faces.ColorFor(name)` derives one from a name.
- `avatar`: a new experimental package that draws a deterministic avatar
  for a name: one of ten silhouettes with two eyes, in a colour the name
  chose. `Model.View` draws it in terminal cells and `Model.SVG` returns it
  as markup. Names are NFC-normalised, trimmed and lowercased before
  hashing, so the spellings of a name a reader cannot tell apart are one
  avatar.
  `Model.Blink` plays one blink, driven by `Model.Update`, and
  `Model.LookAt` turns the eyes toward a target such as the mouse pointer.
  A Model from `New` caches its `View` until a field it depends on changes.
  `Model.Expression` sets a pose the eyes hold: happy, sad, mad, surprised,
  wink, sleepy or thinking. `Model.StartIdle` keeps the avatar breathing,
  blinking and glancing aside, at a rhythm drawn from its name, until
  `Model.StopIdle`. `Model.Hue`, `Model.Tone` and `Model.Silhouette` pin the
  colour or the shape while the name decides the rest. `Model.PNG` returns
  the avatar as a PNG, for `imageview` on a terminal that shows images.
  `Model.React` pulls a face for `ReactHold` and lets it go, for a click.
- `tui.WriteClipboard(text)` copies text: the Program writes the OSC 52
  sequence to its own output and keeps the text. `tui.PasteCopied()` delivers
  that text to `Update` as a `PasteEvent`.
- `filepicker.Model.Err` reports why the current directory could not be
  listed, and `View` shows a line saying so. An unreadable directory used to
  look like an empty one.
- `filepicker.Model.Reload` lists the directory again. `Extensions` and
  `DirsOnly` set after `New` had no effect on the starting directory; set
  them and call `Reload`.

### Changed

- BREAKING: `textinput.Model.ClipboardWrite` and
  `textarea.Model.ClipboardWrite` no longer default to `os.Stdout` when nil.
  With the field nil, Copy and Cut return `tui.WriteClipboard`, so the OSC 52
  sequence goes to the Program's output, and Paste returns `tui.PasteCopied`,
  so the pasted text arrives one `Update` later as a `PasteEvent`. A Model
  whose `Update` is called outside a Program copies and pastes only if the
  caller runs the returned Cmd.
- The text the paste key inserts is no longer shared by every text input in
  the process. It belongs to the Program, so two Programs in one process (two
  SSH sessions) cannot paste each other's copies.
- Cancelling the context given to `WithContext` now ends `Run`: it restores
  the terminal and returns the context's error. Before, only
  `Program.Context` was cancelled and `Run` kept going until `Quit`.
- `Run` returns an error wrapping the write error when a write to the output
  fails. Before, write errors were ignored and the Program kept rendering.

### Deprecated

- `textinput.Model.ClipboardWrite` and `textarea.Model.ClipboardWrite`: use
  `tui.WithOutput` to direct the Program's output. While set, the field still
  receives the OSC 52 sequence and Paste inserts only what that Model copied.
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

- `tuitest.Replay` says why it fails on a recording it cannot reproduce: one
  in which a tick or resize was handled between two keys that arrived in one
  input read. Before, the failure was only a frame diff.
- `colorpicker` passes a `PasteEvent` to its hex field while that field has
  focus. It forwarded only keys, so a paste never reached the field.
- `Run` waits for its resize watcher before returning, so nothing asks the
  `Terminal` for its size, or reads the output file's descriptor, after `Run`
  has returned and the caller has closed the output.
- `filepicker` treats a symlink to a directory as a directory: Enter descends
  into it instead of selecting it as a file.
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

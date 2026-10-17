# Changelog

User-visible changes to `github.com/ows4444/tui`. The format loosely follows
[Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/). Before v1.0
a minor release may remove or change exported identifiers; each one is a
`- BREAKING:` entry.

A removed or changed exported identifier needs a `- BREAKING:` entry under
Unreleased that names it; see [CONTRIBUTING.md](CONTRIBUTING.md#changelog).

## [Unreleased]

### Added

- `treeview.FromValue` and `ValueNode`: show any Go value as a tree: struct fields,
  sorted map entries, slice elements, pointers followed with cycles marked, `time.Time`
  and values with an `Error` or `String` method by their text; `ValueOptions` bounds
  the depth and the items per collection. Text from the value is cleaned of escapes.
- `markdownview` (experimental): a Markdown document you read in a window. It scrolls,
  lists its headings (`Headings`, for a table of contents) and links, moves through the
  links with Tab, follows in-page links (`#section`) with a Back and Forward history,
  jumps by heading with `]` and `[`, and reports other links with a `LinkMsg`.
  `viewport` gains `YOffset` and `SetYOffset`.
- `imageview` takes more than PNG and works without a graphics protocol: `New` accepts a
  JPEG or GIF (its first frame) and keeps its PNG form; `FromImage`, `EncodePNG`
  and `ToPNG` convert; `Frames` decodes an animated GIF into whole-picture frames with
  their delays; and where the terminal has no graphics protocol the image is drawn in
  half blocks, two pixels to a cell (`HalfBlocks`), unless `PlaceholderOnly` is set or
  the theme's glyph set is ASCII.
- Two experimental packages: `rangeslider` (two thumbs on one track, Space or `[` and
  `]` to pick one, the low never passing the high, mouse drag) and `timepicker`
  (hours, minutes, optional seconds and AM or PM as fields; arrows, typed digits,
  wheel). `theme.ComponentRangeSlider` and `ComponentTimePicker` name them for
  theme tokens.
- `textinput` history and suggestions: `Model.History` with Up and Down (and Ctrl+R to
  search it), and `Model.Suggestion` or `SuggestFromHistory` for grey text after the
  cursor that Right, End or Ctrl+E takes (`KeyMap.HistoryPrev`, `HistoryNext`,
  `HistorySearch`, `AcceptSuggestion`). With no history and no suggestion nothing changes.
- Double and triple click: the Program counts presses of the same button on the same
  cell within 400 ms into `MouseEvent.Clicks` (`WithMultiClick` sets or turns off the
  interval). In `textinput` a double click selects the word and a triple click the
  line; in `textarea` a double click selects the word and a triple click the line.
- `datatable` multi-row marking: with `Model.Multi` on, Space marks the row under the
  cursor and Ctrl+A marks all (`KeyMap.Mark`, `MarkAll`); a marked row shows a check
  in the gutter. `Marked`, `IsMarked`, `Mark` and `ClearMarks` read and set the marks,
  which belong to source rows and stay with their rows through sort, filter and row edits.
- `datatable.Column` and `Model.Columns`: per-column alignment (numbers flush right),
  minimum and maximum width (a cell wider than the maximum ends in an ellipsis), a
  `Format` that changes what is shown and not the cell, and a `Less` for the sort key.
  View, the layout node, `Linearize` and `DrawCells` follow them.
- `prompt.Progress` and `prompt.Track`: a bar with the elapsed time, rate and time
  left (from `widgets.ProgressStats`) while a function or a loop runs, with
  Ctrl+C cancelling its context; the counterpart of `prompt.Spin` for work with a
  known size.
- `layout.Anchor(corner, dx, dy, node)`: pin a `Stack` child to a corner or the middle of
  the Stack, following its size (a badge top right, a status bottom left); and
  `layout.Aspect(w, h, node)`, which sizes a node to a ratio in cells.
- `tui.SetPointerShape` and `ViewState.Pointer`: change the mouse pointer's shape
  (OSC 22: a hand over a button, an I-beam over a text field) on terminals that
  support it; the default is put back on every exit path.
- `layout.Box.Shadow(colour)` and `ShadowGlyph(glyph, colour)`: a drop shadow, one
  column down the right side and one row along the bottom, which adds that much to
  the box's size.
- `autocomplete.Source` and `Model.Source`: suggestions found for the typed text,
  run on `tui.Tasks` so newer text cancels the older run, like the command
  palette's; `autocomplete.PathSource()` completes file system paths.
- Four Program options: `WithEnvironment` (the program's own `TERM`, `COLORTERM`,
  `NO_COLOR`, `ACCESSIBLE` and reduced-motion variables, read through the new
  `Program.Getenv` and `motion.DetectEnv`, so a host serving several sessions can
  give each its own), `WithFilter` (sees every message on its way to the model,
  `QuitMsg` included, and may replace or drop it), `WithoutRenderer` (the event
  loop with no frames) and `WithWindowSize` (the starting size when the terminal
  reports none).
- Text overflow and justification: `ansi.TruncateStart` and `TruncateMiddle` keep the
  end, or both ends, of a line that is too wide; `ansi.Justify` widens the gaps
  between words; `layout.WithOverflow(OverflowStart|OverflowMiddle|OverflowEnd, glyph)`
  and `layout.WithJustify()` use them in a `Text` node.
- Seven more border presets in `layout` (`BlockBorder`, `OuterHalfBlockBorder`,
  `InnerHalfBlockBorder`, `DashedBorder`, `HeavyDashedBorder`, `MarkdownBorder`,
  `HiddenBorder`, the last taking a border's space and drawing none of it), and
  `Box.BorderColors(top, right, bottom, left)` to colour the sides separately.
- The rest of the standard easing set in `motion`: Quart, Quint, Sine, Expo, Circ,
  Back, Elastic and Bounce, each in, out and in-out (31 in all, with the first
  eight), and `motion.ByName` and `EaseNames` so a config file can name one.
- `ansi.RGB` colour math: `HSL` and `FromHSL`, `Complement`, `Luminance` (WCAG relative
  luminance) and `Readable` (black or white, whichever reads better on the colour).
- Three more form fields: `form.FieldConfirm` (a Yes/No choice answered with Y and
  N; `Bool` reads it), `form.FieldNote` (a read-only line between fields, no
  focus, no value) and `form.FieldFile` (a path with a file list that
  `KeyMap.Browse`, Ctrl+O, opens), plus the `form.FileExists` validator.
- `layout.Span(node, cols, rows)`: a grid cell that covers several columns or
  rows, so a dashboard with one wide panel over two narrow ones is one
  `GridNode`. Cells still fill row by row, skipping what a span covers; a span
  wider than the auto-sized columns it covers widens them. Rendering,
  `DrawCells`, `Rects` and `NamedAt` all follow it.
- Screenshots of a running program: `tui.WithScreenshot(key, save)` hands `save` a
  `tui.Screenshot` (the rows as plain `Text`, as `ANSI`, and the size) when the
  key is pressed, and `tui.ScreenshotCmd(wrap)` delivers one to `Update`.
  `tuitest.SVGOf(shot.ANSI, w, h, opts)` draws it as an SVG image.
- Markdown coverage: task lists (`- [ ]`, `- [x]`), reference links
  (`[text][label]`, `[text][]`, `[label]` with `[label]: url`), character
  references (`&amp;`, `&#8212;`; ones that decode to a control character stay
  as typed) and lists nested up to six levels instead of two. Reference links
  are resolved in `Render`, `RenderWith` and `Model`, not in a `Stream`.
- Text area editing: `textarea.Model.Highlighter` colours the text from a function
  of one line (View, DrawCells, soft wrap, scrolling and the selection all follow
  it); `AutoIndent` copies the leading whitespace to a new line; `Find`,
  `FindNext`, `FindPrev` select matches ignoring case and `ReplaceAll` replaces
  them in one undo step.
- `appshell.Stack`: a screen stack. A screen pushes another over itself with
  `PushCmd` and the pushed one pops itself with `PopCmd(result)`, which delivers
  the result to the screens below; `Replace`, `ViewAt` and `Depth` complete it.
  Keys and mouse go to the top screen, everything else to all of them.
- `commandpalette.Source` and `Model.Sources`: commands found for what was typed,
  not from a fixed list. Each keystroke runs every Source on `tui.Tasks`, so a
  newer query cancels the older one, and the answers join the dropdown when the
  `tui.TaskDoneMsg` comes back through `Update`. `Model.Tasks` is optional and
  `Model.Name` keeps two palettes apart.
- Lazy trees: `treeview.Node.Load` makes a node a branch whose children are
  fetched the first time it opens (`Update` returns a Cmd that runs it, and the
  `LoadedMsg` it delivers goes back through `Update`); `treeview.LazyDirectoryTree`
  lists one directory at a time. `Model.Notify` makes opening and closing a
  branch deliver `ExpandedMsg` and `CollapsedMsg`; `Model.Name` keeps two lazy
  trees' messages apart.
- `datatable.Source` and `datatable.NewSource`: a table over rows read on demand
  (`Rows() int`, `Row(i) []string`), so a database cursor or a million-row log
  loads only the rows in view. Sort, filter and editing are off for it;
  `SetSource` swaps the source and `SetRows` goes back to in-memory rows.
- Key bindings from a file: `keymap.Parse` reads `{"action": ["key", ...]}` JSON
  (every key name is checked, with `keymap.ValidKey`), and `keymap.Apply` swaps
  those keys into a list of bindings by their description, returning the names
  that matched nothing. `tui keys check <file.json>` lints such a file.
- `widgets.ProgressStats` and `widgets.ProgressBarStats`: elapsed time, rate (over
  the last ten seconds) and ETA for a progress bar, from the counts the caller
  feeds in with `Update`. It takes the time as an argument and never reads the
  clock. `ProgressBarStats` draws the bar with the stats line under it.
- `tuitest.Session.Find`, `ClickText` and `CursorOn`: a test finds text on the
  screen and clicks it or checks the cursor is on it, without knowing the
  column and row the layout drew it at.
- Styled text as data in three overlays: `dialog.Model.MessageText`,
  `notificationcenter.Notification.Text` and `hovercard.Model.ContentText` take an
  `ansi.Text`. When set it is shown instead of the string field, wrapped at
  `Width` with each span keeping its style (a notification's unstyled spans take
  the variant's colour), and `Linearize` and `LayoutNode` follow it. The string
  fields work as before. `ansi.Text` gains `WrapString` and `WithDefault`.
- `ansi.Span` and `ansi.Text`: styled text as data. A `Text` is a list of spans
  that can be measured (`Width`, `Height`), cut (`Slice`, `Truncate`), wrapped at
  spaces (`Wrap`, which cuts a word wider than the line) and rendered (`String`),
  with each piece keeping its style and nothing parsed back out of an escape
  sequence. Text is sanitised when built, so a span cannot carry control codes.
  `cellbuf.Buffer.SetText` writes one straight into the cell grid, and
  `layout.RichText(t, wrap)` lays it out as a `Node`.
- `layout.Box.BorderGradient(from, to)`: the border blends cell by cell around the
  box, from one colour at the top-left corner to another half way round and back,
  so the loop has no seam. It changes no dimensions and takes precedence over
  `BorderColor`.
- `hexview` (experimental): bytes as a hex dump with an offset column, hex in
  groups, an ASCII gutter, a cursor, a selection copied as hex (`v` then `y`),
  scrolling with `Height`, and mouse click, drag and wheel with `Mouse` on.
- Seven more components in the Widget gallery, 48 in all: hex view, tooltip,
  hover card, popover, menu bar, notification center and file picker.

### Changed

- The data table's `LayoutNode` honours `FrozenCols`: when the columns do not fit
  the allotted width it scrolls them, keeping the frozen ones, as `View` does.
  Before, it shrank every column with an ellipsis. A table with `FrozenCols` of 0
  is drawn as before.
- `chart.LineChart` draws on a `chart.Canvas`, as `Plot` does. Its output is the
  same: the chart tests and every example pass unchanged.

## [1.0.0-rc.1] - 2026-10-17

Release candidate for v1.0.0, condition 7 of `docs/v1.md`. It has no breaking
change since 0.2.0; everything in it is added, opt-in or internal. It holds a
fuzzy filter, form groups and typed values, data table filter, sort, cell mode and
editing, text-area line numbers, `MaxHeight` and `LineInfo`, a multi-series plot
and a public canvas, a colour toolkit, background tasks by key, `Map`, opt-in
colour-scheme and in-band resize reports, themes from JSON and four light
themes, `tui prompt` for shell scripts, a key-hint footer, a static tree, a
calendar and two more charts, a session screenshot as SVG and a gallery of 41
components, and text selection in the viewport. A breaking change found in the
week it is out would be a reason to cut an rc.2.

### Changed

- Documented which renderer is the renderer: the cell renderer (a tenth of the
  bytes for a changing frame, 202 against 2,004 on an 80x24 view, for about a
  quarter more CPU). `WithCellRenderer(false)` stays, for comparing and
  debugging, since the line renderer is also the cell renderer's fallback. A
  test keeps the byte figure from regressing.
- The cell renderer reads a drawn frame 40% faster: a text-area keystroke through
  `DrawCells` costs 64 µs instead of 107 µs. `internal/render` remembers the
  style of the cell before and takes a printable ASCII cell without scanning
  it; the cost was a map lookup and a rune slice for every one of the 1,920
  cells of an 80x24 frame. The profile had pointed at the `write` system call,
  which was the sampler's attribution, not the cost: with a no-op writer the
  time was the same.
- `internal/celldraw.StyleID` remembers each style's conversion (up to 512), so
  every `DrawCells` widget stops rendering and parsing a probe string per style
  per frame.

### Added

- `examples/asyncload` now runs its fetch with `Tasks` and has an `r` key that
  reloads, cancelling the load still in flight.
- `widgets.Calendar` and `LinearizeCalendar`: a month as a 20-column grid with a
  title, a chosen first weekday, today in the theme's selected style and days
  styled by day of the month, for printed output (`datepicker` is the interactive
  one).
- `chart.BreakdownBar` and `chart.Histogram`, each with a `Linearize...`: a
  stacked one-row bar with a legend (segments differ by colour and by shade), and
  a count of values into equal-width bins drawn as bars.
- Community files: `SECURITY.md` (private reports through GitHub) and
  `CODE_OF_CONDUCT.md`.
- `Tasks`, `NewTasks` and `TaskDoneMsg`: background work by key. `Tasks.Start`
  cancels the earlier run of the same key, so a search-as-you-type needs no
  hand-written stale-result check; `Cancel`, `CancelAll` and `Running`.
- `WithColorSchemeReports` (DECSET 2031) and `ColorSchemeEvent`: a terminal that
  switches between dark and light while the program runs is followed. With
  `WithTheme(theme.Auto{...})` the Program swaps the theme itself.
- `WithInBandResize` (DECSET 2048) and `InBandResizeEvent`: the terminal reports
  its size through the input stream, so a resize is seen where no window-size
  signal arrives (over SSH, in some multiplexers). Both options are off by
  default, so no output changes unless asked for.
- `theme.Parse`, `theme.Preset` and `theme.PresetNames`: a theme from JSON, built
  on a named preset, and `tui theme check file.json` / `tui theme list` to lint
  one from the command line.
- `tui prompt input|password|confirm|select|multiselect|spin`: the `prompt`
  package from a shell script. The question is drawn on standard error, the
  answer printed on standard output; `confirm` answers with its exit code.
- `widgets.Tree` and `TreeWith`: a static tree with connector lines, for printed
  output (the interactive one is `treeview`).
- `layout.Ratio(n, d)`: a flex child sized to an exact fraction of the space after
  gaps (`Pct(33)` is not a third); combine it with `Min` and `Max`.
- Data table: `SetCell`, `InsertRow` and `RemoveRow` edit the source rows on a
  copy and keep a sort or filter in force.
- Text area: `MaxHeight` (grows with its content up to a cap), `LineInfo` (the
  cursor's line, column and wrapped row) and `LineLabel` (your own text in the
  left column in place of the line number).
- `widgets.ErrorTrace` and `ErrorTraceWith`: a wrapped or joined error as a tree of
  what each layer added and its Go type, sanitised.
- `widgets.HintsFromBindings` and `widgets.Footer`: a one-line key-hint bar built
  from a focused widget's `Bindings()`, cut to the width with a "…" when it does
  not fit.
- `Map` and `MapAs[T]`: wrap the messages of a child's Cmd, through `Batch`,
  `Sequence`, `FromCtx`, `Tick` and `Every`, so a parent tags or converts them
  without a hand-written envelope Cmd.
- Four light themes: `CatppuccinLatteTheme`, `OneLightTheme`, `RosePineDawnTheme`
  and `GitHubLightTheme`. Text and Muted pass AA; the accents keep each palette's
  published values and some are paler than 4.5:1 (`Theme.Check` lists them).
- `tuitest.Session.SVG` and `tuitest.SVGOptions`: the screen as a standalone SVG
  image, with colours, attributes, an optional title bar and cursor, and
  `TrimRows` to cut the blank rows below what was drawn.
- A Widget gallery page on the site: one picture of each of 41 components (31 at
  first; a dialog, toast, context menu, command palette, stopwatch, viewport, log
  view, form, drawer and help screen were added), each drawn by a small real
  program and checked against a golden file in `internal/tools/widgetshots`.
- `DrawCells` on `textinput`: a 60-column field draws in 3.1 µs and 1 allocation
  against 9.3 µs and 93 through `View` and parse. `passwordinput`, `maskedinput`
  and `inputgroup` define their own, through `View`, so a mask or a prefix is not
  bypassed (a test checks `passwordinput` never draws its secret).
- Viewport text selection: with `Mouse` on, press and drag across the content,
  `y` (the `Copy` key) copies the selection through `tui.WriteClipboard`;
  `SelectedText`, `SelectAll` and `ClearSelection`. It follows soft wrap,
  scrolls when dragged past the top or bottom, is drawn in `SelectionStyle`
  (reverse video by default) over any search highlight, and `DrawCells` draws the
  same cells. Set `Bounds` to where the viewport is drawn.
- `datatable.Model.CellMode`, `CellStyle`, `FrozenCols` and `SelectedMsg.Col`: a
  cursor that is one cell (Left and Right move it, only that cell is
  highlighted, Enter reports its column), a hook that styles any cell from its
  row, column and value, and leading columns that stay in view while the rest
  scroll sideways. All are off by default. `DrawCells` draws a table that uses
  them through `View`.
- `chart.Canvas`, `NewCanvas`: a public grid of braille dots (2 x 4 per
  cell) with `Set`, `Line`, `Rect`, `Circle`, `Clear`, `Rows` and `String`, in
  any colour per shape. `Plot` now draws on one.
- `chart.Series.Kind` with `Line` (the zero value), `Scatter` and `Bars`: a
  series drawn as joined lines, lone points, or columns that rise from the
  bottom of the plot or from zero.
- `form.FieldMultiSelect` and `form.FieldTextArea`, `form.Selected`,
  `Field.Description`, `Field.Height` and `KeyMap.SubmitInArea`: a list with
  any number of options ticked (value: the ticked options joined by ","), a
  multi-line text box (Enter adds a line; Ctrl+S submits from inside it), and a
  line of help under any field. Both draw as several rows, and the mouse,
  layout node and accessible output follow them.
- `tui.ViewState` and `tui.ViewStater`: an optional `ViewState()` method beside
  `View() string` that states the alternate screen, the mouse mode and the
  window title for the frame, as `EnableMouse`, `EnterAltScreen`, `ExitAltScreen`
  and `SetWindowTitle` do from `Update`. A nil field has no opinion; a set one
  is applied before the frame when it differs from the terminal's state. The
  Cmds stay.
- `DrawCells` on `picker`, `multiselect`, `treeview`, `spinner`, `tabs`, `menu`
  and `filepicker`: each draws
  its rows straight into the cell grid, as `viewport`, `datatable` and
  `textarea` already did, instead of building a string for the frame to parse.
  A 24-row list costs 47 µs and 131 allocations through `DrawCells` against
  172 µs and 1,100 through `View` and parse. A test per package checks the cells
  against `View`'s. `View` is unchanged.
- Tests that lift ten library packages (`passwordinput`, `maskedinput`,
  `taginput`, `numberinput`, `spinner`, `wizard`, `loadingbar`, `motion`,
  `keymap`, `layout`, `emailinput`, `commandpalette`, `toast`, `helpscreen`) past 90% on their own tests, and a note in
  `CONTRIBUTING.md` on why `go test -cover` and the CI floor differ.
- `docs/tutorial-2.md` and `examples/tutorial2`: a second tutorial that turns
  the task list into a searchable one with `picker.Filter`, with a test.
- Seven cookbook recipes: a one-line prompt, a filtering list, a form in steps,
  a sorted and filtered table, line numbers in a text area, and a multi-series
  chart; and examples `datatable.ExampleModel_filter` and
  `textarea.ExampleModel_ShowLineNumbers` behind them.
- `prompt`, an experimental package that asks one question and returns the
  answer without a `Model` of your own: `Input`, `Password`, `Confirm`,
  `Select` (typing narrows), `MultiSelect` and `Spin`. They run an inline
  program, return `ErrCanceled` on Esc or Ctrl+C, and take an optional
  `prompt.Config` (`Default`, `Placeholder`, `Validate`, `NoFilter`, `Program`).
- `chart.Plot`, `chart.Series`, `chart.PlotOptions` and `chart.LinearizePlot`:
  several series drawn as braille lines on a shared y axis with labelled ticks,
  an x axis with optional labels and a legend. A fixed `Min` and `Max` or the
  data's range; NaN and infinite values leave a gap; ASCII glyph sets draw
  with `+`, `-` and `|`.
- `textarea.Model.ShowLineNumbers` and `LineNumberStyle`: a right-aligned
  line-number column left of the text, blank on soft-wrapped continuation rows.
  The column is extra width, so `Width` still counts the text only; the cursor
  cell, mouse clicks, `DrawCells` and `LayoutNode` account for it. Off by default.
- `textarea.Model.Line`, `Column`, `LineCount`, `SetCursorPosition` and
  `InsertString`: where the cursor is as a line and column, move it there, and
  insert text at it as one undo step.
- `datatable.Model.Filter`: "/" opens a filter line that keeps the rows whose
  cells fuzzy-match the typed text; Enter closes the line and keeps the filter,
  Esc clears it. `Query`, `SetQuery` and `Filtering` read and set it. Off by
  default.
- `datatable.Model.SortBy` and `ClearSort`, and `KeyMap.ClearSort` ("S"): sort
  from code, and put the rows back in the order they were given. `SourceRow`
  and `SelectedMsg.SourceRow` map a shown row to the row given to `New`.
- `form.SelectOf`, `form.Option` and `form.Value`: a select whose options carry
  a value of any type, read back with `Value[T]`; `form.Bool` and `form.Int`
  read a checkbox and a number field. The string values in `Values` and
  `SubmittedMsg` are unchanged.
- `form.Field.OptionsFunc`: a select's options computed from the other fields'
  values, asked after every change; the chosen option stays while still offered.
- `form.NewGroups`, `form.Group` and the `form.KeyMap` fields `NextGroup` and
  `PrevGroup`: a form of titled groups, one showing at a time under its title
  and "step n of N". Enter checks the group and moves on, the last group's
  Enter submits every field, Tab stays inside the group, and `Group.Hide`
  drops a group (and its values) while the values entered say it does not
  apply. `Model.CurrentGroup` and `Model.GroupCount` report where it is.
  `form.New` is unchanged and shows no header.
- `fuzzy`, an experimental package that matches a typed query against text
  (`Score`, `Find`) and marks the matched runes (`Highlight`). It ignores case,
  prefers consecutive runes and word starts, and ranks best first.
- `picker.Model.Filter`: type to narrow the list to the fuzzy matches, best
  first, matched runes underlined; Backspace edits and Esc clears the query.
  `Query` and `SetQuery` read and set it. Off by default, so existing lists do
  not change. `fuzzy.Split` cuts a text into matched and unmatched runs.
- `multiselect.Model.Filter` and `treeview.Model.Filter`, the same type-to-narrow
  filter as `picker`. In `multiselect` checked items stay checked while hidden.
  In `treeview` a match shows with its ancestors, opened for the query without
  changing the expand state, and the tree keeps its order.
- `autocomplete.Model.Fuzzy`: suggest every suggestion the text fuzzy-matches,
  best first, instead of only those it is a prefix of. Off by default.
- `ansi.RGB.Mix`, `Lighten` and `Darken`, and `ansi.Blend`: move a colour
  toward another, white or black by a fraction, and spread n colours along a
  list of stops. A translucent colour is a `Mix` with the background it sits on.

### Changed

- `datatable.Model.SetRows` applies the sort and filter that are active to the
  new rows; before, a sorted table showed its sort arrow over rows in the given
  order.
- `commandpalette` lists matching commands best first instead of in the order
  given, using `fuzzy`. The set of commands that match is unchanged.

## [0.2.0] - 2026-10-15

The last v0 release before v1.0: every identifier deprecated in 0.1.0 is
removed, so no exported identifier carries `Deprecated:`. Each removal is a
`- BREAKING:` entry under Removed, with its replacement; `docs/migrating-to-v1.md`
has one section per identifier.

### Added

- Three small complete programs as examples, each with tests: `examples/jsonview`
  (search a JSON document as a tree), `examples/diskuse` (what takes the space in
  a directory) and `examples/newproject` (a project skeleton from five prompts).
  `docs/built-with.md` lists programs built with tui, and `.github` has issue and
  pull-request templates and a link to Discussions.
- `tui.SetProgress`, `ProgressState` and `ViewState.Progress`: the terminal's
  progress indicator (OSC 9;4), as a bar at a percent, an error or paused bar, a
  busy indicator, or none. The Program removes it on every exit path.
- `cmd/tui`: `go run github.com/ows4444/tui/cmd/tui new myapp` writes a new
  program (`go.mod`, `main.go`, a `tuitest` test, a README) that builds and
  passes its own test; `-module` sets the module path. It adds no dependency
  version: `go mod tidy` resolves the latest.
- `docs/tutorial.md` and `examples/tutorial`: a first tutorial from an empty
  directory to a small task list with a text field, and a test of it with
  `tuitest`. The program and its test also compile against v0.1.0.
- `button`, a new experimental package: a pressable button with five
  variants and three sizes. Enter or Space presses it when focused; with
  `Mouse` on, so does a left click that goes down and comes up inside it.
  It delivers `button.PressedMsg` carrying the button's `ID`. Its states
  (focused, held down, hovered, disabled, loading) are told apart without
  colour. Focus is shown by angle brackets, `< Save >`, as in every control
  of this family: `buttongroup`, `radiogroup`, `checkbox`, `toggle`, `slider`
  and `rating`. `theme.ComponentButton` names it for token overrides. With
  `Toggle` set it is a toggle button: a press turns it on or off, a dot
  before the label marks it on, and `PressedMsg.On` carries its state.
- `buttongroup`, a new experimental package: a row of buttons that take one
  place in the focus order. Left, Right, Home and End move its cursor,
  skipping disabled buttons and wrapping. `ModeActions` is a set of plain
  buttons; `ModeSingle` and `ModeMultiple` are toggle buttons of which one or
  several are on, read with `On` and reported with `ChangedMsg`. `Required`
  keeps a single choice from being left empty.
- `radiogroup`, a new experimental package: a set of options of which
  exactly one is chosen. The arrow keys move between the options and choose
  as they go; Space or Enter chooses the one under the cursor. Vertical or
  on one row, with disabled options skipped. It delivers
  `radiogroup.ChangedMsg`.
- `slider`, a new experimental package: a value in a range, moved along a
  track by the arrow keys, Page Up and Page Down, Home and End, and by a
  press or a drag with the pointer. `Step` rounds the value and `Max` stays
  reachable. It delivers `slider.ChangedMsg`.
- `rating`, a new experimental package: a score out of a small maximum, as a
  row of filled and empty marks. Left and Right, a digit, or a click set it;
  a click on the mark it is at clears it. It delivers `rating.ChangedMsg`.
- `theme.ComponentRadioGroup`, `ComponentSlider` and `ComponentRating` name
  the three for token overrides.
- `checkbox`, a new experimental package: one box with a label that Space or
  a click checks and unchecks, with an `Indeterminate` state drawn `[-]`. It
  delivers `checkbox.ChangedMsg`. `widgets.Checkbox` still draws one without
  state.
- `toggle`, a new experimental package: an on/off switch with a label. Space
  or Enter flips it, Left turns it off and Right turns it on. It delivers
  `toggle.ChangedMsg`. `widgets.Toggle` still draws one without state.
- `theme.ComponentCheckbox` and `ComponentToggle` name the two for token
  overrides.
- `otpinput`, a new experimental package: a short code typed one character
  to a cell, such as a one-time password. A typed character fills a cell and
  moves on, Backspace steps back, and a paste fills as many cells as it has
  characters. `Group` draws a dash between groups of cells, `Accept` chooses
  the characters, `Mask` hides them. It delivers `otpinput.ChangedMsg` with
  `Complete`. `theme.ComponentOTPInput` names it for token overrides.
- `inputgroup`, a new experimental package: a `textinput.Model` with fixed
  text before and after it. The suffix keeps its column when a `Width` is
  set, `CursorCell` is reported past the prefix, and `FullValue` joins the
  three.
- `pagination`, a new experimental package: a row of page numbers with the
  current one in brackets and far pages folded into an ellipsis. Left,
  Right, Home, End or a click on a number change the page. It delivers
  `pagination.ChangedMsg`. `widgets.Pagination` still draws "Page X of Y".
- `breadcrumb`, a new experimental package: a trail of places with a cursor
  moved by Left and Right; Enter, Space or a click chooses a place and
  delivers `breadcrumb.ChosenMsg`. `Max` folds a long trail around an
  ellipsis. `widgets.Breadcrumb` still draws one without state.
- `theme.ComponentPagination` and `ComponentBreadcrumb` name the two for
  token overrides.
- `transferlist`, a new experimental package: two lists side by side. Up and
  Down move a cursor, Left and Right change list, and Enter, Space or a
  click moves an item across; Ctrl+A moves a whole list. It delivers
  `transferlist.ChangedMsg`.
- `carousel`, a new experimental package: one slide of several at a time
  with a row of dots under it. Left, Right, Home, End or a click change the
  slide; `Loop` wraps at the ends. It delivers `carousel.ChangedMsg`.
- `theme.ComponentTransferList` and `ComponentCarousel` name the two for
  token overrides.
- `tooltip`, a new experimental overlay: a short hint drawn under its
  `Target` while the pointer is over it. It needs `tui.MouseAllMotion`;
  `Show` and `Hide` serve keyboard focus. `widgets.TooltipOverlay` still
  draws one without state.
- `hovercard`, a new experimental overlay: a titled card drawn under its
  `Target`, which stays open while the pointer is on the target or on the
  card. Esc or the pointer leaving closes it and delivers
  `hovercard.ClosedMsg`.
- `backdrop`, a new experimental overlay: `Render` returns a frame dimmed,
  to put behind a dialog or a drawer. A press on it delivers
  `backdrop.PressedMsg`.
- `theme.ComponentTooltip`, `ComponentHoverCard` and `ComponentBackdrop`
  name the three for token overrides.
- `widgets.Empty` draws the placeholder for a view with nothing to show: a
  title, a description and a hint, centred.
- `widgets.Timeline` draws `[]widgets.TimelineItem` one under another with a
  marker for each status and a rail between them.
- `examples/gallery`, seven pages that show `otpinput`, `inputgroup`,
  `transferlist`, `carousel`, `widgets.Timeline`, `widgets.Empty`, and
  `tooltip`, `hovercard` and a dialog over a `backdrop`, with a `breadcrumb`
  and a `pagination` row as its own navigation.
- `examples/controls`, an export panel that uses `button`, `buttongroup`,
  `radiogroup`, `slider`, `rating`, `checkbox` and `toggle` together, by
  keyboard and by mouse.
- `tuitest.Session.WaitForText(text, timeout)` waits until a row of the
  screen contains `text` and reports whether it did. It is for output that
  arrives on its own time, after a command or a tick, which `Keys` and `Send`
  do not wait for.

### Changed

- The minimum Go version is 1.26, up from 1.25. tui supports the two newest
  Go releases, now 1.26 and 1.27, and CI tests on both.

- `layout.BoxNode` measures a `Width`, a `Height` and a title set on its
  box. It asked only for the child's size, so a box with a `Width` or a title
  came out narrower than `Box.Render` drew it, and one with a `Height` lost
  its bottom border. Given the size it asks for, `BoxNode` now draws what
  `Box.Render` draws; a layout that allots less still wins. A box with any
  of the three set may take a different amount of room than before.
- `commandpalette` is promoted from experimental to stable-ish, as
  `docs/v1.md` decided once its deprecated `Init` was removed. ACLine is the
  program outside this repository that uses it.

### Removed

- BREAKING: `autocomplete.Model.Init` and `commandpalette.Model.Init` are
  removed. Use `Focus`, which returns the same Cmd. They were the only two
  components with an `Init`; a component has none.
- BREAKING: `layout.Box.Render` is removed from the exported API. Use
  `layout.BoxNode(box, layout.Block(content))`, or `BoxNode` with any Node
  child. `BoxNode` draws what `Render` drew.
- BREAKING: `ansi.SetClusterWidth` is removed. Grapheme-cluster width is on
  by default and `TUI_NO_CLUSTERS` set to a non-empty value turns it off for
  the process, read once when the first width is measured. For one terminal use
  an `ansi.Measurer`: `ansi.ClusterMeasurer(false)` counts each rune on its
  own, and `Program.Measurer` and `ResizeMsg.Measurer` carry the probed answer.
- BREAKING: `textinput.Model.ClipboardWrite` and
  `textarea.Model.ClipboardWrite` are removed. Copy and Cut always return
  `tui.WriteClipboard`, and Paste returns `tui.PasteCopied`, so the Program
  writes the OSC 52 sequence to its output and holds the copied text. To direct
  that output elsewhere use `tui.WithOutput`. A Model no longer holds its own
  copy buffer: a second text field in the same Program pastes what the first
  copied.
- BREAKING: the chart functions in `widgets` are removed: `BarChart`, `Gauge`,
  `HeatMap`, `LineChart`, `Sparkline`, `SparklineWith` and the `BarItem`
  alias. Use the functions of the same names in `widgets/chart`.

### Fixed

- `clockview`: a Model ignores another Model's ticks. Two of them in one
  program each took the other's tick for their own, so a stopwatch ran fast
  and the number of pending ticks doubled every interval.
- `spinner`, `skeleton`, `loadingbar` and `faces`: a Model ignores another
  Model's ticks, as `clockview` now does. Two spinners started separately in
  one program each took the other's tick, so both ran fast and the number of
  pending ticks doubled every interval. Models that share a `motion.Clock`
  were not affected.
- The `asyncload`, `buildlog`, `chat` and `inlinespinners` examples animate.
  Each started its spinner, or its typed-out reply, inside an `Init` with a
  value receiver, so the model the Program kept was never running and the
  animation stayed on its first frame. `Start`'s doc comments now say where
  to call it.
- `layout.BoxNode` draws its whole frame when the child has no room. Around
  an empty child it drew a blank row where the bottom border belongs, did not
  stretch to the width it was given, and, when only as wide as its frame, left
  the bottom border above the last row.
- `layout.BoxNode` measures the box's margin. It left the margin out, so a
  box with one was drawn larger than it measured and lost its right and
  bottom borders to the clip. A box with a margin now takes that much more
  room in a layout.
- `layout.BoxNode` measures only the border sides the box draws. It counted
  two rows and two columns for any border, so a box with sides switched off
  by `BorderSides` was padded out with blank rows and columns. Such a box now
  takes less room in a layout.
- A terminal resize in a program's first moments is no longer lost on Linux,
  macOS and the BSDs. The program began listening for the window-size signal
  on a goroutine of its own, and a resize that came before that goroutine ran
  found no listener, so the program stayed drawn at its old size until the
  next resize.
- `tuitest`: `Keys`, `Send` and the other input methods return only after the
  program has exited when the model quit in response. On a slow terminal they
  returned while it was still leaving the screen, so `Done` was false straight
  after.
- `numberinput` and `emailinput` reject the space bar. Their filters looked
  only at `KeyRunes` keys, and the space bar is a `KeySpace` key, so a space
  went into a digits-only field and into an email address.
- `tuitest.Session.Keys` waits until the model has received the keys it
  sent. It counted any `Update`, so one caused by a tick or a command's
  result while a key was still on its way was taken for the key, and `Keys`
  could return before the model had seen it; on a loaded machine tests then
  read a stale screen. `Paste`, `Resize` and `Send` wait for their own
  message in the same way.

## [0.1.0] - 2026-10-08

The first tagged release: everything written before a tag existed.

### Added

- `toolapproval`: a prompt can show fewer options and name them itself.
  `Model.Choices` lists which options are shown and in what order (empty
  shows all three), `Model.Labels` replaces an option's text, and
  `Model.Body` is drawn between the header and the options, for the command
  and where it runs. `KeyMap.Approve`, `KeyMap.Deny` and `KeyMap.Always`
  resolve the prompt at once with that option; they have no keys until a
  caller binds them. A prompt that sets none of these behaves as before.
- `imageview` draws through iTerm2's inline-image protocol (OSC 1337) when
  `Model.Inline` is set: it sends the PNG as it is, and is preferred over
  Sixel. `imageview.Inline` builds the sequence, and
  `tui.Capabilities.InlineImages` reports a terminal known to draw it
  (iTerm2, WezTerm), inferred from its XTVERSION reply.
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
  wink, sleepy, thinking, smug, unsure, scared, love, shy or sick. `Model.StartIdle` keeps the avatar breathing,
  blinking and glancing aside, at a rhythm drawn from its name, until
  `Model.StopIdle`; `Model.Hover` runs it only while the pointer is over the
  avatar. `Model.Hue`, `Model.Tone` and `Model.Silhouette` pin the
  colour or the shape, and `Model.Pins` any single trait such as the size of
  the eyes or the number of a sun's petals, while the name decides the rest. `Model.PNG` returns
  the avatar as a PNG, for `imageview` on a terminal that shows images, and
  the same slice until the avatar changes.
  `Model.React` pulls a face for `ReactHold` and lets it go, for a click;
  it eases in and out, as does `Model.SetExpression`. Eased into, the mad,
  scared and sick poses tremble and the thinking pose rocks. A pose also
  lifts or sinks the body a little; in cells that shows only at large sizes.
  `ExpressionMad`, `ExpressionLove`, `ExpressionShy` and `ExpressionSick`
  tint the body; `Colors` reports the tinted colours.
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

- `tuitest`: a session larger than 80x24 drew clipped frames. The Program
  under test kept the 80x24 size of a plain writer whatever size `New` or
  `Resize` was given, so only the model was told the new size. The session
  now gives the Program a terminal of its own size, and `Resize` changes
  that size too. A test that passes its own `tui.WithTerminal` keeps
  control of the size.
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

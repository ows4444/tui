# Widgets

Widgets come in two shapes. The functions in
[`widgets`](https://pkg.go.dev/github.com/ows4444/tui/widgets) and
[`widgets/chart`](https://pkg.go.dev/github.com/ows4444/tui/widgets/chart) are
stateless: each takes its data (and usually a `theme.Theme`) and returns a
styled string to place in a view. Anything with state, such as a cursor, a
scroll offset or an open/closed flag, is a component in its own top-level
package, with a value-typed `Model`, a `New` constructor, and `Update` and
`View` methods.

```go
fmt.Println(ansi.StripANSI(widgets.Box("Status", "all good", theme.DarkTheme(), 20)))
// Output:
// ┌──────────────────┐
// │                  │
// │ Status           │
// │ all good         │
// │                  │
// └──────────────────┘
```

From `ExampleBox` in `widgets/example_test.go`.

## The widget contract

A component is embedded in your own model. You hold its `Model` in a field,
pass messages to its `Update`, keep the Model it returns, and draw its `View` or
`LayoutNode`. It has no `Init` (`autocomplete` and `commandpalette` still
export a deprecated one; use their `Focus`). The root package names these shapes, and
packages assert them at compile time with lines such as
`var _ tui.Component[Model] = Model{}`:

| Contract | Shape | Asserted by |
| --- | --- | --- |
| `tui.Component[T]` | `Update(tui.Msg) (T, tui.Cmd)`, `View() string` | accordion, appshell, autocomplete, avatar, breadcrumb, button, buttongroup, carousel, checkbox, clipboard, clockview, colorpicker, commandpalette, confirm, datatable, datepicker, dropdown, emailinput, errorretry, faces, filepicker, form, hexview, inputgroup, loadingbar, logview, markdownview, maskedinput, menu, multiselect, numberinput, otpinput, pagination, passwordinput, picker, radiogroup, rangeslider, rating, scrollbar, skeleton, slider, spinner, splitpane, streamtext, tabs, taginput, termpane, textarea, textinput, timepicker, toggle, toolapproval, transferlist, treeview, viewport, virtuallist, window |
| `tui.Overlay[T]` | `Open() bool`, `Update`, `Render(base string) string`. It composites onto a drawn frame. `Show` and `Hide` are pointer methods outside the interface | backdrop, contextmenu, dialog, drawer, helpscreen, hovercard, menubar, popover, toast, tooltip |
| `tui.ThemeSetter[T]` | `SetTheme(theme.Theme) T` | every component except clockview, logview, markdown, viewport and wizard; see [Theming](theming.md) |
| `tui.Linearizer` | `Linearize() string`, plain text for accessible output | every component has the method; most assert it. See [Accessibility](accessibility.md) |
| `tui.CursorProvider` | `CursorCell() (x, y int, ok bool)` | inputgroup, textarea, textinput |
| `tui.CellDrawer` | `DrawCells(buf *cellbuf.Buffer, r cellbuf.Rect)`, draws into the cell grid with no View string to parse | avatar, form |

Every component `Model` also has `LayoutNode()`, a test enforces it, and most
have `Tokens()` and `WithTokens()`. Some packages implement a contract's
methods without the assertion: `scrollbar` and `splitpane` have
`Update`/`View`, and `notificationcenter` has `Render(base string)` with no
`Update`. `wizard.Model` and `markdown.Model` take the theme as an argument to
`View` and `LayoutNode`. For layout see [Layout](layout.md). For keys, focus
and the `KeyMap` convention see [Input](input.md).

## Component packages

"Keys" marks packages with a `KeyMap` field, `DefaultKeyMap()` and
`Bindings()`. "Experimental" packages say so in their package comment, and
their API may change in any minor release. The example column names a program
in `examples/` that imports the package.

| Package | Purpose (from the package comment) | Keys | Example program |
| --- | --- | --- | --- |
| [accordion](../accordion) | A list of collapsible sections | yes | `examples/settings` |
| [appshell](../appshell) | Header, full-width input, scrollable content and optional key-hints footer, composed from existing widgets. Experimental | | |
| [autocomplete](../autocomplete) | A text input with a filtered suggestion dropdown | yes | `examples/signup` |
| [backdrop](../backdrop) | An overlay that dims a finished frame, to put behind a dialog or a drawer; a press on it delivers `PressedMsg`. Experimental | | `tui demo` |
| [avatar](../avatar) | A deterministic avatar for a name: one of ten silhouettes with two eyes, in a colour the name chose. For ready-made animated characters, see faces. Experimental | | `tui demo` |
| [breadcrumb](../breadcrumb) | A trail, ` Home / Docs / Guide `, with a cursor moved by Left and Right; Enter or a click delivers `ChosenMsg`. `Max` folds a long trail around an ellipsis. Experimental | | `tui demo` |
| [button](../button) | A pressable button with variants and sizes; Enter, Space or a click delivers `PressedMsg`. With `Toggle` set, a toggle button that is on or off. Experimental | | `tui demo` |
| [buttongroup](../buttongroup) | A row of buttons with one cursor moved by Left and Right: actions, or toggle buttons of which one or several are on. Experimental | | `tui demo` |
| [carousel](../carousel) | One slide of several at a time, with a row of dots under it, ` ○ ● ○ `; Left, Right, Home, End or a click change the slide and deliver `ChangedMsg`. Experimental | | `tui demo` |
| [checkbox](../checkbox) | One box with a label, drawn `[x] Label`; Space or a click checks and unchecks it, and `Indeterminate` draws `[-]`. For a list of them see multiselect. Experimental | | `tui demo` |
| [clipboard](../clipboard) | A "copy to clipboard" button that writes OSC 52. Experimental | | |
| [clockview](../clockview) | A wall-clock, stopwatch or countdown timer |  `tui demo` |
| [colorpicker](../colorpicker) | A palette-swatch and hex-input colour picker | yes  `tui demo` |
| [commandpalette](../commandpalette) | A text input with a fuzzy-filtered dropdown of Commands ("Ctrl+K" style) | yes | |
| [confirm](../confirm) | A yes/no prompt | yes | `examples/signup` |
| [contextmenu](../contextmenu) | A popup menu opened at an anchor point | yes  `tui demo` |
| [datatable](../datatable) | `widgets.Table` plus row navigation | yes | `examples/csvedit`, `examples/jsonx` |
| [datepicker](../datepicker) | A keyboard-navigable calendar on `time.Time` | yes  `tui demo` |
| [dialog](../dialog) | A modal box with title and message, composited over the screen, dismissed with Enter/Esc | yes | `examples/dashboard` |
| [dropdown](../dropdown) | A closed one-line control that shows the chosen option and opens a list of the others: Enter opens, arrows move, typing jumps, Enter chooses, Esc closes. Experimental | | |
| [drawer](../drawer) | An overlay anchored to an edge of the base view | yes  `examples/workspace` |
| [emailinput](../emailinput) | A `textinput.Model` wrapper that rejects whitespace |  `tui demo` |
| [errorretry](../errorretry) | An error with retry (Enter or `r`, up to `MaxRetries`) and dismiss (Esc). Experimental | yes | |
| [faces](../faces) | A catalog of 50 ready-made animated Braille characters and a widget that plays them. For a picture derived from a name, see avatar. Experimental | | |
| [filepicker](../filepicker) | A filesystem browser, one directory at a time | yes  `tui demo` |
| [form](../form) | A validating column of labelled fields (text, select, checkbox, multi-select, text area) with Submit, optionally in titled groups shown one at a time | yes | `examples/login`, `examples/signup` |
| [helpscreen](../helpscreen) | A full-screen key-binding help overlay | | |
| [hovercard](../hovercard) | An overlay with a title and a body, drawn under its target while the pointer is on the target or the card; Esc closes it. Experimental | | `tui demo` |
| [imageview](../imageview) | A PNG drawn with the kitty graphics protocol, iTerm2 inline images or Sixel, or a text placeholder. Experimental | | `tui demo` |
| [loadingbar](../loadingbar) | An indeterminate progress animation | | `examples/dashboard` |
| [inputgroup](../inputgroup) | A `textinput.Model` with fixed text before and after it, such as `https://` and `.com`; the suffix keeps its column and the cursor cell is reported past the prefix. Experimental | yes | `tui demo` |
| [logview](../logview) | An append-only scrolling log | | `examples/logtail` |
| [markdown](../markdown) | A CommonMark subset rendered as styled, width-aware text | | `examples/chat`, `examples/codeagent` |
| [markdownview](../markdownview) | A Markdown document read in a window: it scrolls, jumps between headings and links, follows in-page links with a back and forward history, and reports other links as a `LinkMsg`. Experimental | yes | `examples/docsreader`, `examples/mdstudio` |
| [maskedinput](../maskedinput) | A `textinput.Model` wrapper that masks each character with a configurable rune |  `tui demo` |
| [menu](../menu) | Nested-navigation list on top of `picker.Model` | yes  `tui demo` |
| [menubar](../menubar) | A horizontal bar of titled dropdown menus | yes  `tui demo` |
| [motion/fx](../motion/fx) | Effects for a region of cells that appears or goes away: `Fade`, `Dissolve`, `Slide` and `Sweep`, as a function of progress; `Wrap` applies one to a Model. Experimental | | `examples/fireandfx` |
| [multiselect](../multiselect) | A multi-choice list: Space toggles, Enter confirms | yes | `examples/todo` |
| [notificationcenter](../notificationcenter) | A panel showing every queued notification at once. Experimental | | |
| [numberinput](../numberinput) | A `textinput.Model` wrapper that accepts digits and one leading `-` |  `tui demo` |
| [otpinput](../otpinput) | A short code typed one character to a cell, `[4][8][2] — [1][ ][ ]`; a paste fills the cells, and `ChangedMsg` says when it is complete. Experimental | | `tui demo` |
| [pagination](../pagination) | A row of page numbers, ` 1  …  9 [10] 11  …  20 `; Left, Right, Home, End or a click change the page and deliver `ChangedMsg`. Experimental | | `tui demo` |
| [passwordinput](../passwordinput) | A `textinput.Model` wrapper that masks the value | | `examples/hello` |
| [picker](../picker) | A single-choice list (InkUI's "Select") | yes | `examples/signup`, `examples/routerapp`, `examples/signup` |
| [popover](../popover) | An overlay anchored near a point | yes  `examples/workspace` |
| [radiogroup](../radiogroup) | A set of options of which exactly one is chosen, drawn as `( ) Label` rows or on one row; the arrow keys move and choose, and a change delivers `ChangedMsg`. Experimental | | `tui demo` |
| [rangeslider](../rangeslider) | A range chosen from a span with two thumbs on a track; Space switches thumbs, Left and Right step, Page Up and Page Down jump. Experimental | yes | |
| [hexview](../hexview) | Bytes as a hex dump: offset, hex in groups, an ASCII gutter, a cursor, a selection copied as hex (`y`), and the mouse. Experimental | | `examples/` none yet; see `hexview` `Example` |
| [rating](../rating) | A score out of `Max` as filled and empty marks; Left and Right, a digit, or a click set it. Experimental | | `tui demo` |
| [scrollbar](../scrollbar) | A track and thumb showing how much content is visible and where | yes  `examples/workspace` |
| [skeleton](../skeleton) | A loading placeholder block |  `tui demo` |
| [slider](../slider) | A value in a range on a track with a thumb; arrows, Page Up and Page Down, Home and End move it, and the pointer can press or drag. Experimental | | `tui demo` |
| [spinner](../spinner) | An animated loading indicator, with 18 named animations (`Presets`, `Lookup`, `WithPreset`) and ASCII fallbacks | | `examples/weather`, `examples/buildstatus` |
| [splitpane](../splitpane) | Two `layout.Node`s with a divider moved by keyboard or mouse | yes  `examples/workspace` |
| [streamtext](../streamtext) | Text revealed a few characters at a time (`New` streams, `NewTypewriter` types). Experimental | | `examples/chat`, `examples/codeagent` |
| [tabs](../tabs) | A horizontal tab bar | yes | `examples/settings`, `examples/jsonx` |
| [taginput](../taginput) | A text input plus a list of committed tags drawn as `widgets.Tag` chips | yes  `tui demo` |
| [termpane](../termpane) | A program run on a pseudo-terminal and drawn in a pane: a shell or a full-screen tool inside the app, with keys, paste, mouse, resize and scrollback. darwin and linux only. Experimental | | |
| [textarea](../textarea) | A multi-line text input | yes | `examples/hello`, `examples/codeagent` |
| [textinput](../textinput) | A single-line text input | yes | `examples/hello`, `examples/cursorfield`, `examples/signup` |
| [timepicker](../timepicker) | A time of day in fields (hours, minutes, seconds, AM or PM), moved with Left and Right and changed with Up, Down or digits, on `time.Time`. Experimental | yes | `examples/planner` |
| [toast](../toast) | A transient notification in a screen corner that closes after `Duration` | | `examples/dashboard` |
| [toggle](../toggle) | An on/off switch with a label, drawn `(○  )` and `(  ●)`; Space or Enter flips it, Left and Right set it. Experimental | | `tui demo` |
| [toolapproval](../toolapproval) | A gate-before-execution prompt for an agent tool call. Experimental | yes | `examples/codeagent` |
| [tooltip](../tooltip) | An overlay with a short hint, drawn under its target while the pointer is over it. Experimental | | `tui demo` |
| [transferlist](../transferlist) | Two lists side by side; Enter, Space or a click moves the item under the cursor to the other list and delivers `ChangedMsg`. Experimental | | `tui demo` |
| [treeview](../treeview) | A hierarchical expandable tree | yes | `examples/jsonx` |
| [tuislog](../tuislog) | A `log/slog` handler that writes one styled line per record, above the live region of a running Program or to any function. Experimental | | |
| [viewport](../viewport) | A scrollable window onto content taller than it | yes | `examples/pager` |
| [virtuallist](../virtuallist) | A scrolling window onto a large uniform-height list that never builds off-screen rows | yes  `examples/workspace` |
| [window](../window) | A desktop of overlapping windows moved by the title bar, resized by the edges, raised by a press and closed by the close box, with the same actions on the keyboard. Experimental | yes | |
| [wizard](../wizard) | Step navigation for multi-step flows, drawn with `widgets.Stepper` | | `examples/signup` |

Every package listed has an `Example` in its `example_test.go`
(`TestEveryPackageHasExample` in `exampleevery_test.go` requires one per
package).

## The widgets package

Everything in `widgets` is a pure function of its arguments that returns a
string. It needs no `Msg` handling and never imports `tui` or a component
package: it sits below the components. Use `layout.Block(s)` (or
`widgets.Node(s)`, the same adapter) to place the result in a layout.

| Kind | Functions |
| --- | --- |
| Containers and labels | `Alert`, `Badge`, `Banner`, `Box`, `Card`, `Empty`, `InfoBox`, `Panel`, `Tag`, `Tooltip`, `TooltipOverlay`, `Center`, `Spacer` |
| Separators and headings | `Divider`, `DividerLabel`, `DividerWith`, `DividerLabelWith`, `Header`, `HeaderWithAccessory`, `Breadcrumb` |
| Progress and status | `ProgressBar`, `ProgressCircle`, `MultiProgress`, `StatusIndicator`, `Stepper`, `Timeline`, `Pagination`, `PaginationDots` |
| Controls drawn as text | `Checkbox`, `Toggle`, `FormField`, `Form` |
| Lists and tables | `List`, `ListWith`, `Tree`, `TreeWith`, `Calendar`, `KeyValue`, `Table`, `TableCells`, `TableRows`, `TableRowsWith` |
| Key hints | `KeyHint`, `KeyHints`, `HintsFromKeymap`, `HintsFromBindings`, `Footer` |
| Text formatting | `CodeBlock`, `CodeBlockLang`, `DiffView`, `Gradient`, `BigText`, `BigTextGradient`, `Link` |
| Chat and agent helpers | `ChatMessage`, `ChatMessageRaw`, `TokenCounter`, `UsageMonitor`, `CompactCount` |
| Error isolation | `ErrorBoundary` renders a fallback if the render function panics; `ErrorTrace` and `ErrorTraceWith` show a wrapped error and its causes as a tree |

`widgets/chart` holds the data plots: `Sparkline`, `SparklineWith`,
`BarChart`, `BreakdownBar`, `Histogram`, `LineChart`, `Plot`, `HeatMap` and
`Gauge`, each with a `Linearize...` function for accessible output.
`examples/chat` and `examples/dashboard` use `widgets/chart`.

`examples/hello`, `examples/signup` and
`examples/kanban` build whole screens from `widgets` functions and a
`layout.Row`.

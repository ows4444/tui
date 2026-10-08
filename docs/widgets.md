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
| `tui.Component[T]` | `Update(tui.Msg) (T, tui.Cmd)`, `View() string` | accordion, appshell, autocomplete, avatar, button, clipboard, clockview, colorpicker, commandpalette, confirm, datatable, datepicker, emailinput, errorretry, faces, filepicker, form, loadingbar, logview, maskedinput, menu, multiselect, numberinput, passwordinput, picker, skeleton, spinner, streamtext, tabs, taginput, textarea, textinput, toolapproval, treeview, viewport, virtuallist |
| `tui.Overlay[T]` | `Open() bool`, `Update`, `Render(base string) string`. It composites onto a drawn frame. `Show` and `Hide` are pointer methods outside the interface | contextmenu, dialog, drawer, helpscreen, menubar, popover, toast |
| `tui.ThemeSetter[T]` | `SetTheme(theme.Theme) T` | every component except clockview, logview, markdown, viewport and wizard; see [Theming](theming.md) |
| `tui.Linearizer` | `Linearize() string`, plain text for accessible output | every component has the method; most assert it. See [Accessibility](accessibility.md) |
| `tui.CursorProvider` | `CursorCell() (x, y int, ok bool)` | textarea, textinput |

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
| [autocomplete](../autocomplete) | A text input with a filtered suggestion dropdown | yes | `examples/form` |
| [avatar](../avatar) | A deterministic avatar for a name: one of ten silhouettes with two eyes, in a colour the name chose. For ready-made animated characters, see faces. Experimental | | `examples/avatar` |
| [button](../button) | A pressable button with variants and sizes; Enter, Space or a click delivers `PressedMsg`. Experimental | | |
| [clipboard](../clipboard) | A "copy to clipboard" button that writes OSC 52. Experimental | | |
| [clockview](../clockview) | A wall-clock, stopwatch or countdown timer |  `examples/timers` |
| [colorpicker](../colorpicker) | A palette-swatch and hex-input colour picker | yes  `examples/pickers` |
| [commandpalette](../commandpalette) | A text input with a fuzzy-filtered dropdown of Commands ("Ctrl+K" style). Experimental | yes | |
| [confirm](../confirm) | A yes/no prompt | yes | `examples/form` |
| [contextmenu](../contextmenu) | A popup menu opened at an anchor point | yes  `examples/menus` |
| [datatable](../datatable) | `widgets.Table` plus row navigation | yes | `examples/table`, `examples/inspector` |
| [datepicker](../datepicker) | A keyboard-navigable calendar on `time.Time` | yes  `examples/pickers` |
| [dialog](../dialog) | A modal box with title and message, composited over the screen, dismissed with Enter/Esc | yes | `examples/dashboard` |
| [drawer](../drawer) | An overlay anchored to an edge of the base view | yes  `examples/panes` |
| [emailinput](../emailinput) | A `textinput.Model` wrapper that rejects whitespace |  `examples/inputs` |
| [errorretry](../errorretry) | An error with retry (Enter or `r`, up to `MaxRetries`) and dismiss (Esc). Experimental | yes | |
| [faces](../faces) | A catalog of 50 ready-made animated Braille characters and a widget that plays them. For a picture derived from a name, see avatar. Experimental | | `examples/faces` |
| [filepicker](../filepicker) | A filesystem browser, one directory at a time | yes  `examples/pickers` |
| [form](../form) | A validating column of labelled fields with Submit | yes | `examples/login`, `examples/signup` |
| [helpscreen](../helpscreen) | A full-screen key-binding help overlay | | |
| [imageview](../imageview) | A PNG drawn with the kitty graphics protocol, iTerm2 inline images or Sixel, or a text placeholder. Experimental | | `examples/avatar` |
| [loadingbar](../loadingbar) | An indeterminate progress animation | | `examples/dashboard` |
| [logview](../logview) | An append-only scrolling log | | `examples/procstream` |
| [markdown](../markdown) | A CommonMark subset rendered as styled, width-aware text | | `examples/chat`, `examples/agentshell` |
| [maskedinput](../maskedinput) | A `textinput.Model` wrapper that masks each character with a configurable rune |  `examples/inputs` |
| [menu](../menu) | Nested-navigation list on top of `picker.Model` | yes  `examples/menus` |
| [menubar](../menubar) | A horizontal bar of titled dropdown menus | yes  `examples/menus` |
| [multiselect](../multiselect) | A multi-choice list: Space toggles, Enter confirms | yes | `examples/list` |
| [notificationcenter](../notificationcenter) | A panel showing every queued notification at once. Experimental | | |
| [numberinput](../numberinput) | A `textinput.Model` wrapper that accepts digits and one leading `-` |  `examples/inputs` |
| [passwordinput](../passwordinput) | A `textinput.Model` wrapper that masks the value | | `examples/focus` |
| [picker](../picker) | A single-choice list (InkUI's "Select") | yes | `examples/loginflow`, `examples/router`, `examples/setupflow` |
| [popover](../popover) | An overlay anchored near a point | yes  `examples/panes` |
| [scrollbar](../scrollbar) | A track and thumb showing how much content is visible and where | yes  `examples/panes` |
| [skeleton](../skeleton) | A loading placeholder block |  `examples/timers` |
| [spinner](../spinner) | An animated loading indicator | | `examples/asyncload`, `examples/buildlog`, `examples/inlinespinners` |
| [splitpane](../splitpane) | Two `layout.Node`s with a divider moved by keyboard or mouse | yes  `examples/panes` |
| [streamtext](../streamtext) | Text revealed a few characters at a time (`New` streams, `NewTypewriter` types). Experimental | | `examples/chat`, `examples/agentshell` |
| [tabs](../tabs) | A horizontal tab bar | yes | `examples/settings`, `examples/inspector` |
| [taginput](../taginput) | A text input plus a list of committed tags drawn as `widgets.Tag` chips | yes  `examples/inputs` |
| [textarea](../textarea) | A multi-line text input | yes | `examples/focus`, `examples/agentshell` |
| [textinput](../textinput) | A single-line text input | yes | `examples/focus`, `examples/cursorfield`, `examples/form` |
| [toast](../toast) | A transient notification in a screen corner that closes after `Duration` | | `examples/dashboard` |
| [toolapproval](../toolapproval) | A gate-before-execution prompt for an agent tool call. Experimental | yes | `examples/agentshell` |
| [treeview](../treeview) | A hierarchical expandable tree | yes | `examples/inspector` |
| [viewport](../viewport) | A scrollable window onto content taller than it | yes | `examples/pager` |
| [virtuallist](../virtuallist) | A scrolling window onto a large uniform-height list that never builds off-screen rows | yes  `examples/panes` |
| [wizard](../wizard) | Step navigation for multi-step flows, drawn with `widgets.Stepper` | | `examples/form` |

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
| Containers and labels | `Alert`, `Badge`, `Banner`, `Box`, `Card`, `InfoBox`, `Panel`, `Tag`, `Tooltip`, `TooltipOverlay`, `Center`, `Spacer` |
| Separators and headings | `Divider`, `DividerLabel`, `DividerWith`, `DividerLabelWith`, `Header`, `HeaderWithAccessory`, `Breadcrumb` |
| Progress and status | `ProgressBar`, `ProgressCircle`, `MultiProgress`, `StatusIndicator`, `Stepper`, `Pagination`, `PaginationDots` |
| Controls drawn as text | `Checkbox`, `Toggle`, `FormField`, `Form` |
| Lists and tables | `List`, `ListWith`, `KeyValue`, `Table`, `TableRows`, `TableRowsWith` |
| Key hints | `KeyHint`, `KeyHints`, `HintsFromKeymap` |
| Text formatting | `CodeBlock`, `CodeBlockLang`, `DiffView`, `Gradient`, `BigText`, `BigTextGradient`, `Link` |
| Chat and agent helpers | `ChatMessage`, `ChatMessageRaw`, `TokenCounter`, `UsageMonitor`, `CompactCount` |
| Error isolation | `ErrorBoundary` renders a fallback if the render function panics |

`widgets/chart` holds the data plots: `Sparkline`, `SparklineWith`,
`BarChart`, `LineChart`, `HeatMap` and `Gauge`, each with a `Linearize...`
function for accessible output. The functions of the same names in `widgets`
are deprecated wrappers for the chart versions. `examples/chat` and
`examples/dashboard` use `widgets/chart`.

`examples/welcomescreen`, `examples/splashscreen`, `examples/loginflow` and
`examples/setupflow` build whole screens from `widgets` functions and a
`layout.Row`.

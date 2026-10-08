# tui

A terminal UI framework for Go built on the Elm Architecture (Model / Update /
View). It uses only the standard library: `go.mod` has no `require` directive,
and a test in `internal/archtest` fails if any package imports outside the
standard library and this module.

An application implements `tui.Model`, hands it to `tui.NewProgram` with the
options it wants, and calls `Run`. The Program reads keys, mouse events and
resizes, calls `Update` on each Msg, runs the Cmds `Update` returns, and redraws
`View` with a cell diff so only changed cells are written.

Guides, a cookbook and screens of every example are at
**[tui.nizaami.com](https://tui.nizaami.com)**; the API reference is on
[pkg.go.dev](https://pkg.go.dev/github.com/ows4444/tui).

## Install

Requires Go 1.26 or later.

```console
$ go get github.com/ows4444/tui
```

This resolves to the latest tagged release. Before v1.0 a release may change
exported identifiers; [CHANGELOG.md](CHANGELOG.md) lists each one.

## Usage

A Model is any value with `Init`, `Update` and `View`. This is
`ExampleNewProgram` from [example_test.go](example_test.go); it reads keys from a
string and discards the output, so it runs without a terminal:

```go
type counter struct{ n int }

func (c counter) Init() tui.Cmd { return nil }

func (c counter) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	if k, ok := msg.(tui.Key); ok && k.Type == tui.KeyRunes {
		switch k.Text {
		case "+":
			c.n++
		case "q":
			return c, tui.Quit()
		}
	}
	return c, nil
}

func (c counter) View() string { return fmt.Sprintf("count: %d", c.n) }

func ExampleNewProgram() {
	p := tui.NewProgram(counter{},
		tui.WithInput(strings.NewReader("++q")),
		tui.WithOutput(io.Discard),
	)
	final, err := p.Run()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(final.View())
	// Output: count: 2
}
```

Against a real terminal, leave out `WithInput` and `WithOutput`: they default
to `os.Stdin` and `os.Stdout`. [examples/counter](examples/counter/main.go) does
exactly that:

```console
$ go run ./examples/counter
```

Every directory under [examples/](examples) is a runnable program, for instance
`./examples/dashboard`, `./examples/form`, `./examples/signup`,
`./examples/agentshell` and `./examples/probe`.

## Packages

API reference is the godoc of each package. The layering and stability levels
below come from the package comment in [doc.go](doc.go); a package may import
only the layers above it, and `go test ./internal/archtest` fails on an import
that points up.

Stability levels:

- **Core**: the root package `tui`, the primitives, codecs and `tuitest`.
- **Stable-ish**: `cellbuf`, the render helpers and most components.
- **Experimental**: packages whose own package comment says
  `Stability: experimental.` `go run ./internal/tools/doccheck` fails if that
  list and the list in `doc.go` disagree.

### Runtime

| Package | Purpose | Stability |
|---|---|---|
| `tui` | The Program and its event loop; owns terminal I/O through `term` and `internal/termio`. Re-exports 14 `input` types as aliases (`Key`, `MouseEvent`, `PasteEvent`, ...), each as stable as the type it names | Core |
| `cellbuf` | A retained grid of terminal cells, for widgets that draw straight into cells | Stable-ish |

### Primitives

| Package | Purpose | Stability |
|---|---|---|
| `ansi` | Styles, colour, escape encoding, width, graphemes, wrapping | Core |
| `layout` | A Node tree with flex, grid and overlay layout | Core |
| `theme` | Colours, glyph sets and states as data | Core |
| `term` | Raw mode and terminal size | Core |
| `motion` | Springs, transitions and the reduced-motion preference | Core |

### Codecs and ports

| Package | Purpose | Stability |
|---|---|---|
| `input` | Decodes terminal bytes into keys, mouse, paste and focus events | Core |
| `keymap` | A registry of key bindings that help widgets read | Core |
| `hittest` | Named click regions: which region a mouse event landed on | Core |

### Renderers and test harness

| Package | Purpose | Stability |
|---|---|---|
| `widgets` | Stateless render helpers: Badge, Box, CodeBlock, ChatMessage, DiffView, ProgressBar, ... | Stable-ish |
| `widgets/chart` | Sparkline, BarChart, LineChart, HeatMap, Gauge | Stable-ish |
| `markdown` | A CommonMark subset rendered to styled, width-aware text | Stable-ish |
| `focus` | A focus Ring over components, moved with Tab and Shift+Tab | Stable-ish |
| `tuitest` | A headless harness that runs a model against a virtual screen | Core |

### Components

One package per stateful widget, each a value-typed `Model` configured through
exported fields. Every widget has a `LayoutNode()` adapter, and
every widget implements `Linearize` for accessible mode; tests in
`internal/archtest` and `internal/tools/doccheck` fail when one is missing.

| Package | Purpose | Stability |
|---|---|---|
| `textinput` | Single-line text input | Stable-ish |
| `textarea` | Multi-line text input | Stable-ish |
| `passwordinput` | `textinput` that masks every character | Stable-ish |
| `maskedinput` | `textinput` that masks with a configurable rune | Stable-ish |
| `emailinput` | `textinput` that rejects whitespace | Stable-ish |
| `numberinput` | `textinput` restricted to digits and a leading `-` | Stable-ish |
| `taginput` | A list of short tags entered through a text input | Stable-ish |
| `autocomplete` | Text input with a filtered suggestion dropdown | Stable-ish |
| `form` | A column of labelled, validated single-line fields | Stable-ish |
| `confirm` | Yes/no prompt | Stable-ish |
| `picker` | Single-choice list | Stable-ish |
| `multiselect` | Multi-choice list | Stable-ish |
| `menu` | Nested-navigation list built on `picker` | Stable-ish |
| `menubar` | Horizontal bar of titled dropdown menus | Stable-ish |
| `contextmenu` | Popup menu opened at an anchor point | Stable-ish |
| `datatable` | `widgets.Table` plus row navigation | Stable-ish |
| `treeview` | Hierarchical expandable tree | Stable-ish |
| `filepicker` | Filesystem browse-and-select | Stable-ish |
| `datepicker` | Keyboard-navigable calendar on `time.Time` | Stable-ish |
| `colorpicker` | Palette swatches plus hex input | Stable-ish |
| `virtuallist` | Scrollable window onto a large uniform-height list | Stable-ish |
| `viewport` | Scrollable window onto content taller than it | Stable-ish |
| `logview` | Append-only scrolling log | Stable-ish |
| `scrollbar` | Track and thumb showing the visible part of some content | Stable-ish |
| `splitpane` | Two panes with a draggable divider | Stable-ish |
| `tabs` | Horizontal tab bar | Stable-ish |
| `accordion` | List of collapsible sections | Stable-ish |
| `wizard` | Step navigation for multi-step flows | Stable-ish |
| `dialog` | Modal box composited over the screen | Stable-ish |
| `drawer` | Overlay anchored to an edge of the screen | Stable-ish |
| `popover` | Overlay anchored near a point | Stable-ish |
| `toast` | Transient auto-dismissing notification | Stable-ish |
| `helpscreen` | Full-screen key-binding help overlay | Stable-ish |
| `spinner` | Animated loading indicator | Stable-ish |
| `loadingbar` | Indeterminate progress animation | Stable-ish |
| `skeleton` | Loading placeholder block | Stable-ish |
| `clockview` | Wall clock, stopwatch or countdown timer | Stable-ish |
| `appshell` | Header, input, scrollable content and key-hints footer composed from existing widgets | Experimental |
| `streamtext` | Text revealed a few characters at a time | Experimental |
| `toolapproval` | Gate-before-execution prompt for an agent tool call | Experimental |
| `notificationcenter` | Panel showing every queued notification at once | Experimental |
| `commandpalette` | Text input with a fuzzy-filtered list of Commands | Experimental |
| `errorretry` | An error with retry and dismiss keys | Experimental |
| `faces` | 50 ready-made animated Braille characters and a widget that plays them; for a picture derived from a name, see `avatar` | Experimental |
| `imageview` | PNG through the kitty graphics protocol, iTerm2 inline images or Sixel, with a text placeholder otherwise | Experimental |
| `clipboard` | A "copy to clipboard" button that writes OSC 52 | Experimental |
| `avatar` | A deterministic avatar drawn from a name; for ready-made animated characters, see `faces` | Experimental |
| `button` | A pressable button, or a toggle button: Enter, Space or a click reports a press | Experimental |
| `buttongroup` | A row of buttons with one cursor: a set of actions, or a choice of one or of several | Experimental |
| `radiogroup` | A set of options of which exactly one is chosen; the arrow keys move and choose | Experimental |
| `slider` | A value in a range, set by moving a thumb along a track with keys or the pointer | Experimental |
| `rating` | A score out of a small maximum, as a row of filled and empty marks | Experimental |
| `checkbox` | One box that is checked, unchecked or partly checked, with a label | Experimental |
| `toggle` | An on/off switch with a label | Experimental |
| `otpinput` | A short code typed one character to a cell, such as a one-time password | Experimental |
| `inputgroup` | A text field with fixed text before and after it | Experimental |
| `pagination` | A row of page numbers with the current one marked and far pages folded | Experimental |
| `breadcrumb` | A trail of places that can be walked back along and chosen from | Experimental |
| `transferlist` | Two lists side by side with items that move between them | Experimental |
| `carousel` | One slide of several at a time, with a row of dots that says which | Experimental |
| `tooltip` | A short hint shown under the thing the pointer is over | Experimental |
| `hovercard` | A card of details shown while the pointer is over its target or the card | Experimental |
| `backdrop` | Dims a finished frame behind a dialog or a drawer | Experimental |

Packages under `internal/` are not public API.

## Platforms and limitations

- **Platforms.** Linux, macOS (darwin), Windows, FreeBSD, OpenBSD, NetBSD and
  DragonFly BSD. Any other GOOS, including solaris and illumos, is unsupported:
  the build fails on purpose with an import named
  `tui_unsupported_platform_this_GOOS_is_not_supported_see_README`
  ([platform_unsupported.go](platform_unsupported.go)). CI runs the tests on
  Linux, macOS, Windows and FreeBSD; OpenBSD, NetBSD and DragonFly BSD are only
  cross-compiled (`TestBuildsOnSupportedPlatforms`). `WithSuspendOnCtrlZ` does
  nothing on Windows, where Ctrl+Z stays an ordinary key.
- **Colour.** Without `WithColorProfile`, `NewProgram` detects the depth from
  the output and the environment (`ansi.DetectColorProfileFor`), in this order:
  `NO_COLOR` (non-empty) turns colour off; `CLICOLOR_FORCE` (non-empty, not
  `0`) colours output that is not a terminal; otherwise non-terminal output or
  `CLICOLOR=0` turns colour off; `TERM=dumb` turns colour off;
  `COLORTERM=truecolor` or `24bit`, `WT_SESSION`, and some `TERM_PROGRAM`
  values give 24-bit colour (`Apple_Terminal` gives 256); then the `TERM` name
  decides (an unrecognised name gets 16 colours; an empty `TERM` gets none,
  except 16 on Windows). A truecolor terminal that sets none of these (some SSH sessions,
  tmux with `TERM=screen`) is detected at a lower depth; pass
  `WithColorProfile(ansi.TrueColor)` to override. Passing `WithColorProfile`
  also overrides `NO_COLOR`.
- **Unicode.** Width, truncation and trimming treat an extended grapheme cluster
  (a ZWJ emoji sequence, a flag, a letter with combining marks) as one unit,
  following UAX #29. Set `TUI_NO_CLUSTERS=1` for a terminal that draws the
  parts separately. The width, grapheme and bidi tables are from Unicode
  17.0.0. Right-to-left text is reordered for display only with
  `WithBidi(true)`; it is off by default.
- **Accessibility.** `WithAccessible(true)` switches to append-only, unstyled
  output and renders the root model's `Linearize` instead of `View` when it has
  one. `WithAccessibleAuto` turns it on for `ACCESSIBLE=1` or `TERM=dumb`, and
  `TERM=dumb` alone also turns it on unless `WithAltScreen` or `WithAccessible`
  was given. VoiceOver, NVDA and Orca are unverified: no screen-reader run is
  recorded for this repository.

## Terminal probe results

[examples/probe](examples/probe/main.go) reports what a terminal supports
(colour depth, focus reporting, OSC 11 background detection) and leaves a
`probe:` line in the scrollback. No probe run is recorded for any terminal yet,
so every row is unverified. The middle column is what `ansi/profile.go` detects
from the environment alone, not a measured result.

| Terminal | Detected colour depth | Probe result |
|---|---|---|
| Windows Terminal (`WT_SESSION`) | 24-bit | unverified |
| iTerm2 (`TERM_PROGRAM=iTerm.app`) | 24-bit | unverified |
| WezTerm (`TERM_PROGRAM=WezTerm`) | 24-bit | unverified |
| VS Code (`TERM_PROGRAM=vscode`) | 24-bit | unverified |
| Ghostty (`TERM_PROGRAM=ghostty`) | 24-bit | unverified |
| Hyper (`TERM_PROGRAM=Hyper`) | 24-bit | unverified |
| kitty (`TERM=xterm-kitty`) | 24-bit | unverified |
| Alacritty (`TERM=alacritty`) | 24-bit | unverified |
| Apple Terminal (`TERM_PROGRAM=Apple_Terminal`) | 256 colours | unverified |

## Documentation

- [docs/](docs/README.md): topic guides, architecture and testing.
- [CONTRIBUTING.md](CONTRIBUTING.md): conventions and the checks CI runs.
- [CHANGELOG.md](CHANGELOG.md): user-visible changes.

## License

MIT; see [LICENSE](LICENSE).

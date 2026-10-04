# Theming

A [`theme.Theme`](https://pkg.go.dev/github.com/ows4444/tui/theme#Theme) is a
plain, comparable value: colour roles (`Primary`, `Text`, `Muted`, `Focus`,
`Selection`, `BorderColor`, `Background`, `Surface`, `Overlay`, ...), a `Border`
style, glyphs, spacing, state and typography styles. There is no global
theme. Themed code takes a `Theme` as an argument or holds one in a field, so
switching themes is an assignment in `Update`. The preset constructors
(`theme.DarkTheme()`, `theme.LightTheme()`, ...) each return a fresh copy, so
no caller can change a preset for everyone else.

To follow the terminal's background, ask for it with
`tui.WithBackgroundDetection` and pass every message to `theme.Detect`:

```go
func (a app) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	if th, ok := theme.Detect(msg, theme.DarkTheme()); ok {
		a.theme = th
	}
	return a, nil
}
```

```go
var m tui.Model = app{theme: theme.DarkTheme()}

// A light terminal answers with a near-white background.
m, _ = m.Update(tui.BackgroundColorEvent{R: 250, G: 250, B: 250})
fmt.Println("light background -> Text is Light's:", m.(app).theme.Text == theme.LightTheme().Text)

// A dark one answers dark.
m, _ = m.Update(tui.BackgroundColorEvent{R: 10, G: 10, B: 10})
fmt.Println("dark background  -> Text is Dark's:", m.(app).theme.Text == theme.DarkTheme().Text)

// Silence keeps the fallback and tells the app to stop waiting.
m, _ = m.Update(tui.BackgroundUnknownMsg{})
fmt.Println("no answer        -> Text is Dark's:", m.(app).theme.Text == theme.DarkTheme().Text)

// Unrelated messages change nothing.
m, _ = m.Update(tui.Key{Type: tui.KeyEnter})
fmt.Println("other message    -> unchanged:", m.(app).theme.Text == theme.DarkTheme().Text)
// Output:
// light background -> Text is Light's: true
// dark background  -> Text is Dark's: true
// no answer        -> Text is Dark's: true
// other message    -> unchanged: true
```

From `ExampleDetect_switchTheme` and its `app` model in
`theme/example_test.go`. `theme.ForBackground` makes the Light or Dark choice
from the background's perceived luminance.

## How a theme reaches widgets

There are three paths, depending on the kind of widget.

**Stateless helpers** in `widgets` and `widgets/chart` take the theme as an
argument, for example `widgets.Box(title, content, t, width)`. `wizard.Model`
and `markdown.Model` work the same way: their `View` and `LayoutNode` take a
`theme.Theme`.

**Component Models** that are themed have a `Theme theme.Theme` field, which
`New` sets to `theme.DarkTheme()`, and a `SetTheme(theme.Theme) Model` method.
That method makes them a
[`tui.ThemeSetter[Model]`](https://pkg.go.dev/github.com/ows4444/tui#ThemeSetter).
Each package asserts it at compile time with
`var _ tui.ThemeSetter[Model] = Model{}`. Some widgets, such as
`textinput`, have no `Theme` field and copy styles out of the theme instead.
Their exported style fields can be overridden after `SetTheme`. Your root model
forwards the theme to each widget:

```go
// restyle rebuilds the theme from the design settings and hands it to every
// themed widget.
func (m *model) restyle() {
	m.t = m.design.theme()
	m.form = m.form.SetTheme(m.t)
	m.spin = m.spin.SetTheme(m.t)
}
```

From `examples/login/main.go`.

**The Program** can own the theme. With `tui.WithTheme(auto)`, a root model
that implements [`tui.Themeable`](https://pkg.go.dev/github.com/ows4444/tui#Themeable)
(`SetTheme(theme.Theme) tui.Model`) gets `auto.Dark` before `Init`, so before the
first frame. The Program then queries the terminal background (OSC 11, 300ms
timeout unless `WithBackgroundDetection` set one). If the terminal answers, it
calls `SetTheme` again with `auto.Light` or `auto.Dark` to match. A terminal
that does not answer keeps `auto.Dark`. The theme is not delivered again if it
equals the one the model already has. The `BackgroundColorEvent` or
`BackgroundUnknownMsg` still reaches `Update`. Without `WithTheme`, nothing is
delivered and nothing is queried (`theme_program_test.go`).

`theme.DefaultAuto()` is `Auto{Dark: DarkTheme(), Light: LightTheme()}`.
`theme.Pair(preset)` uses `preset` on dark backgrounds and `preset.LightTwin()`
on light ones.

The Program looks for `Themeable` along the model's `Unwrap` chain (see
`tui.Unwrapper`). A wrapper must also implement `tui.Rewrapper`, or a theme set
through it is not applied. `appshell.Router` passes its theme to each child that
is a `ThemeSetter`, or a `Themeable` whose `SetTheme` returns the child's type.

No program in `examples/` uses `WithTheme` yet, and the repository has no
Example for it.

## Presets

`DarkTheme` (the default), `LightTheme`, `CatppuccinTheme`, `DraculaTheme`,
`EverforestDarkTheme`, `GitHubDarkTheme`, `GruvboxTheme`, `HighContrastTheme`,
`MonokaiTheme`, `NightOwlTheme`, `NordTheme`, `OneDarkTheme`, `RosePineTheme`,
`SolarizedDarkTheme`, `SolarizedLightTheme`, `TokyoNightTheme`.
`examples/login` has a design panel where you can try each preset in dark or
light mode, with different borders, glyph sets, colour depths, border colours,
accent tokens and padding.

## Overriding colours for one widget

Component tokens change some colour roles for one kind of widget, or one
instance, and leave the rest of the theme alone. A
[`theme.Tokens`](https://pkg.go.dev/github.com/ows4444/tui/theme#Tokens) names
only the roles that differ. A nil field inherits.

- `t.WithTokens(theme.ComponentTabs, tok)` returns a copy of the theme that
  overrides every `tabs` widget it reaches. Use the `theme.Component*`
  constants: a misspelt name overrides nothing, silently. `theme.Components()`
  lists them.
- `m.WithTokens(tok)` on a widget Model overrides that instance only.

When a widget renders, it calls `Theme.Resolve(component, instanceTokens)`. The
registered component tokens are applied first, then the instance tokens on top.
`m.Tokens()` reports what the widget will use.

```go
if a := d.pick[setAccent]; a > 0 {
	// Component tokens override the accent of these widgets only; the
	// rest of the theme keeps its roles.
	c := roles[a-1].of(t)
	tok := theme.Tokens{Accent: c, Focus: c}
	t = t.WithTokens(theme.ComponentForm, tok).WithTokens(theme.ComponentSpinner, tok)
}
```

From `design.theme` in `examples/login/design.go`.

## States, typography and spacing

Each is a field on `Theme`. A zero field means "derive it", so read them
through the resolving methods:

- `ResolvedStates()`: `Focus` in `t.Focus`, `Hover` in `t.Primary`,
  `Disabled` in `t.Muted`, `Selected` on a `t.Selection` background in `t.Text`.
- `ResolvedTypography()`: H1 bold underlined `Primary`, H2 bold `Primary`,
  Emphasis italic, Strong bold, Code in `Secondary`, Link underlined `Info`.
- `ResolvedSpacing()`: defaults `XS: 1, S: 1, M: 2, L: 3` cells.

A style or size the theme sets explicitly is kept.

## Glyphs

`Theme.Glyphs` holds the non-letter symbols widgets draw: progress fill, tree
and accordion markers, spinner frames, status dots, check and cross, bullets,
rules, cursors, heat and spark ramps, the password mask, the ellipsis. Empty
fields fall back to `theme.UnicodeGlyphSet()`, so a theme that never sets
`Glyphs` draws Unicode. The other sets are `ASCIIGlyphSet()` and
`NerdGlyphSet()`. Every ASCII glyph is as wide as its Unicode counterpart except
`Arrow` (`->` against `→`).

`theme.DetectGlyphs()` picks a set from the environment. It returns ASCII when
`TERM` is `dumb`, or when the first non-empty of `LC_ALL`, `LC_CTYPE`, `LANG`
is `C`, `POSIX` or a charset other than UTF-8. Otherwise it returns Nerd when
`TUI_NERD_FONT` is exactly `1`, and Unicode in every other case. It does not run
automatically, so assign the result to `t.Glyphs` yourself. `t.ASCII()` returns
the theme with the ASCII glyphs and `layout.ASCIIBorder()`.

Code that reads glyphs from the theme:

- Component packages: accordion, avatar, commandpalette, contextmenu, datatable,
  errorretry, faces, loadingbar, markdown, maskedinput, passwordinput,
  scrollbar, skeleton, spinner, splitpane, streamtext, toast, treeview.
- layout: the `Text` ellipsis.
- widgets files: alert, bigtext, chatmessage, codeblock, codeblock_lang,
  divider, infobox, list, pagination, panel, progressbar, progresscircle,
  status, stepper, table, toggle.
- widgets/chart: barchart, gauge, heatmap, linechart, sparkline.

`TestDocsListEveryGlyphReaderAndLexer` (`internal/tools/doccheck`) fails if a
package or widgets file that names `Glyphs` is missing from this list.

## Colour depth, borders and contrast

- `t.ForProfile(p)` downgrades every colour to what an `ansi.Profile` can show.
  Under `ansi.NoColor` all colours become nil and `Border` is kept. The Program
  also downgrades its own output; see [Capabilities](capabilities.md).
- `t.Plain()` clears `Border`, so bordered widgets render without box drawing.
  Pair it with reduced motion; see [Accessibility](accessibility.md).
- `t.LightTwin()` derives a light-background theme. Accents are darkened until
  they reach 4.5:1 against the new background. A theme that is already light is
  returned unchanged.
- `t.Check(min)` lists text and accent roles whose contrast against
  `Background` is below `min`. If `Background` is unset, it measures against
  `TextInverse`. `t.CheckOn(bg, min)` measures against another colour. Use them
  in your CI to vet a custom theme.
- `theme.NewPalette()` starts from xterm's ANSI 0-15 defaults. `Observe` records
  the terminal's `tui.PaletteColorEvent` answers (sent under
  `WithBackgroundDetection`), so `Palette.Check` measures named colours as the
  terminal shows them.

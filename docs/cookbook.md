# Cookbook

Short recipes. Each is copied from an `Example` function, which `go test` runs
and checks against its `// Output:` comment, so the code compiles and the output
is real. The source is named under each recipe. In `layout/example_test.go`,
`show` is a helper that prints each line with trailing spaces trimmed.

Recipes on topic pages:
[follow the terminal's light or dark background](theming.md),
[click regions from a layout](input.md#mouse-and-hit-testing),
[focus across mixed widgets](input.md#focus-between-widgets),
[a widget inside a sized layout](layout.md#putting-a-widget-in-a-layout).

## Header and footer that stay put around a long body

Wrap the body in `layout.Fill`. A plain `Grow: 1` child that measures taller
than the screen pushes the footer out of view instead (`Example_growTrap`).

```go
lines := make([]string, 100)
for i := range lines {
	lines[i] = fmt.Sprintf("log %02d", i)
}
ui := layout.Column(0,
	layout.FlexChild{Node: layout.Block("== header ==")},
	layout.Fill(layout.Block(strings.Join(lines, "\n"))),
	layout.FlexChild{Node: layout.Block("== footer ==")},
)
show(layout.Draw(ui, layout.Constraints{MinW: 12, MaxW: 12, MinH: 5, MaxH: 5}))
// Output:
// == header ==
// log 00
// log 01
// log 02
// == footer ==
```

Source: `ExampleFill`, `layout/example_test.go`.

## Columns that line up across rows

A zero `Track` sizes the column to its widest cell, `Grow` takes the leftover
width, and `Size` fixes it.

```go
g := layout.GridNode(
	[]layout.Track{{}, {Grow: 1}, {Size: 3}},
	1,
	layout.Block("id"), layout.Block("name"), layout.Block("ok"),
	layout.Block("7"), layout.Block("widget"), layout.Block("no"),
)
// MaxH must be set: a zero Max means "at most 0".
show(layout.Draw(g, layout.Constraints{MinW: 20, MaxW: 20, MaxH: layout.Unbounded}))
// Output:
// id name          ok
//
// 7  widget        no
```

Source: `ExampleGridNode`, `layout/example_test.go`. For different column and
row gaps, see `ExampleGridNodeGaps`.

## A different screen for narrow terminals

```go
ui := layout.Responsive([]layout.Break{{MaxW: 20}}, layout.Block("menu"), layout.Block("home | search | settings"))
fmt.Println(layout.Draw(ui, layout.Loose(layout.Size{W: 20, H: 3})))
fmt.Println(layout.Draw(ui, layout.Loose(layout.Size{W: 40, H: 3})))
// Output:
// menu
// home | search | settings
```

Source: `ExampleResponsive`, `layout/example_test.go`. The first node whose
`Break` admits the available width and height is used. A node with no `Break`
is the fallback for larger sizes.

## Tab order that follows the screen

Number the fields in the order your model keeps them, label them with
`layout.Named`, and let `focus.LayoutOrder` sort them by position.

```go
cell := func(name string) layout.FlexChild {
	return layout.FlexChild{Node: layout.Named(name, layout.Block(name)), Basis: 1}
}
form := layout.Row(2,
	layout.FlexChild{Node: layout.Column(0, cell("name"), cell("email")), Basis: 8},
	layout.FlexChild{Node: layout.Column(0, cell("city"), cell("submit")), Basis: 8},
)
names := []string{"name", "email", "city", "submit"}
size := layout.Size{W: 20, H: 2}

ring := focus.New(len(names)).WithOrder(focus.LayoutOrder(form, size, names...))
for range names {
	fmt.Print(names[ring.Current()], " ")
	ring = ring.Next()
}
fmt.Println()
// Output: name city email submit
```

Source: `ExampleLayoutOrder`, `focus/order_test.go`.

## Scroll text in a fixed window

```go
m := viewport.New(10, 3)
m.SetContent(strings.Join([]string{"one", "two", "three", "four", "five"}, "\n"))
fmt.Println(m.View())
m.LineDown(2)
fmt.Println("--")
fmt.Println(m.View())
// Output:
// one
// two
// three
// --
// three
// four
// five
```

Source: `Example`, `viewport/example_test.go`.

## A selectable table

```go
m := datatable.New([]string{"name", "qty"}, [][]string{{"apple", "3"}, {"pear", "5"}, {"plum", "8"}})
m.Height = 3
m, _ = m.Update(tui.Key{Type: tui.KeyDown})
for _, line := range strings.Split(ansi.StripANSI(m.View()), "\n") {
	fmt.Println(strings.TrimRight(line, " "))
}
fmt.Println("cursor row:", m.Cursor())
// Output:
// name   qty
//   ──────────
//   apple  3
// > pear   5
//   plum   8
// cursor row: 1
```

Source: `Example`, `datatable/example_test.go`. `examples/table` is the full
program, confirming a row with Enter via `datatable.SelectedMsg`.

## A dialog and its dismissal message

`dialog.New` returns an open dialog. Enter closes it and returns a Cmd whose
message tells your `Update` it was dismissed.

```go
d := dialog.New("Saved", "Your changes were saved.")
fmt.Println(d.Open())
d, cmd := d.Update(tui.Key{Type: tui.KeyEnter})
fmt.Println(d.Open())
fmt.Printf("%T\n", cmd())
d.Show()
fmt.Println(d.Open())
// Output:
// true
// false
// dialog.DismissedMsg
// true
```

Source: `Example`, `dialog/example_test.go`. To draw it over your screen, see
[overlays](layout.md#overlays-on-a-drawn-screen).

## Resize a split pane from the keyboard

```go
m := splitpane.New(layout.Block("files"), layout.Block("preview"))
m.SetTotal(21)
m.Min1, m.Min2 = 4, 4
m, _ = m.Update(tui.Key{Type: tui.KeyRight, Mod: input.ModCtrl})
first, second := m.Sizes()
fmt.Println(first, second)
// Output: 11 9
```

Source: `Example`, `splitpane/example_test.go`. By default, Ctrl+Right or
Ctrl+Down grows the first pane, Ctrl+Left or Ctrl+Up shrinks it, and Ctrl+E
centres the divider.

## Detect conflicting key bindings

```go
var r keymap.Registry
r.Add(keymap.NewBinding("quit", "q"))
conflicts, _ := r.Add(keymap.NewBinding("close", "q"))
fmt.Println(len(conflicts))
for _, h := range r.Hints("") {
	fmt.Println(h.Key, h.Desc)
}
// Output:
// 1
// q quit
// q close
```

Source: `Example`, `keymap/example_test.go`. The second binding is still
registered. `Add` reports the conflict and returns a non-nil error.

## Ask one question without writing a Model

`prompt` runs a small inline program, asks, and returns the answer. Esc or
Ctrl+C returns `prompt.ErrCanceled`.

```go
name, err := prompt.Input("Your name:")
ok, err := prompt.Confirm("Deploy?", false)
port, err := prompt.Int("Port:")                        // a number, or the question stays open
timeout, err := prompt.Duration("Timeout:")                // 90s, 1h30m
file, err := prompt.Path("Config file:", true)            // ~ is expanded; it has to exist
v, err := prompt.Parse("Level:", parseLevel)               // your own type from the typed text
i, err := prompt.Select("Region:", []string{"eu", "us", "ap"}) // typing narrows
```

Here the keys come from a string and the output is discarded, as a test would
do; the answer is the same one a person types.

```go
keys := tui.WithInput(strings.NewReader("ap\r"))
out := tui.WithOutput(io.Discard)

i, _ := prompt.Select("Region:", []string{"eu", "us", "ap"},
	prompt.Config{Program: []tui.ProgramOption{keys, out}})
fmt.Println(i)
// Output: 2
```

Source: `ExampleSelect`, `prompt/example_test.go`.

## Narrow a list as you type

Set `Filter` on a `picker`, `multiselect` or `treeview`. Typed keys that no
binding uses go to a query line; matches stay, best first, with the matched
letters underlined. Esc clears the query.

```go
m := picker.NewStrings("Go", "Rust", "Zig", "TypeScript")
m.Filter = true
for _, r := range "rs" {
	m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: string(r), Code: r})
}
fmt.Println(m.Query(), m.Highlighted().Label)
// Output: rs Rust
```

Source: `ExampleModel_filter`, `picker/example_test.go`. The matching is the
`fuzzy` package, which you can use on its own: `fuzzy.Find(query, n, text)`.

## A form in steps, with a step that can be skipped

`form.NewGroups` shows one group at a time. Enter checks the group and moves
on, and a group's `Hide` is asked with every value entered so far.

```go
m := form.NewGroups(
	form.Group{Title: "Account", Fields: []form.Field{
		{Name: "email", Label: "Email", Value: "ada@example.com", Validators: []form.Validator{form.Required()}},
	}},
	form.Group{Title: "Company", Fields: []form.Field{
		{Name: "company", Label: "Company", Validators: []form.Validator{form.Required()}},
	}, Hide: func(v map[string]string) bool { return v["email"] != "" }},
	form.Group{Title: "Done", Fields: []form.Field{
		{Name: "terms", Label: "Accept terms", Kind: form.FieldCheckbox},
	}},
)
m.Focus()
m, _ = m.Update(tui.Key{Type: tui.KeyEnter}) // Account is valid; Company is hidden
_, cmd := m.Update(tui.Key{Type: tui.KeyEnter}) // the last group submits
fmt.Println(cmd().(form.SubmittedMsg).Values)
// Output: map[email:ada@example.com terms:false]
```

Source: `ExampleNewGroups`, `form/example_test.go`. A `SelectOf[T]` field carries a
typed value, read back with `form.Value[T](m, name)`; `Field.OptionsFunc`
computes a select's options from the other fields.

## Sort and filter a table

`datatable` sorts by a column (`s` on the current column, or `SortBy` from
code), puts the rows back with `ClearSort` (`S`), and with `Filter` on, `/`
opens a filter line. `SourceRow` maps a shown row to the row you gave it.

```go
m := datatable.New(
	[]string{"Name", "City"},
	[][]string{{"Carol", "Lahore"}, {"alice", "Tokyo"}, {"Bob", "Lahore"}},
)
m.Filter = true
m.SortBy(0, false)
m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: "/"})
for _, r := range "lah" {
	m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: string(r)})
}
for i, row := range m.Rows {
	fmt.Println(row[0], "was row", m.SourceRow(i))
}
// Output:
// Bob was row 2
// Carol was row 0
```

Source: `ExampleModel_filter`, `datatable/example_test.go`.

## Line numbers and a cursor position in a text area

```go
m := textarea.New()
m.ShowLineNumbers = true
m.SetValue("first\nsecond")
m.SetCursorPosition(0, 5)
m.InsertString("!")
fmt.Println(m.Line(), m.Column(), m.LineCount())
fmt.Println(ansi.StripANSI(m.View()))
// Output:
// 0 6 2
// 1 first!
// 2 second
```

Source: `ExampleModel_ShowLineNumbers`, `textarea/example_test.go`. The number
column is extra width: `Width` still counts the text only.

`MaxHeight` makes the area grow with its text and scroll past that many rows
(`Height` still wins when set), `LineInfo` gives the cursor's line, column and
wrapped row for a status bar, and `LineLabel` replaces the number with your own
text per line ("> " on the cursor line, an error mark).

## Several series on one chart

`chart.Plot` draws lines on shared, labelled axes with a legend. Series are
told apart by colour, so give each a `Name` and set `Legend`.

```go
s := []chart.Series{
	{Name: "requests", Values: []float64{10, 40, 30, 80, 60, 100}},
}
out := chart.Plot(s, chart.PlotOptions{
	Width: 16, Height: 5, YTicks: 3,
	XLabels: []string{"00:00", "12:00"}, Legend: true,
}, theme.DarkTheme())
// 100─              ⡠⠊
//    │        ⡰⠑⠢⣀⢀⠔⠁
//  55─       ⡰⠁   ⠁
//    │  ⡠⠒⠤⢄⡰⠁
//  10─⡠⠊
//    ─────────────────
//     00:00      12:00
//     ● requests
```

Source: `ExamplePlot`, `widgets/chart/plot_test.go`. `chart.LinearizePlot` gives
the same data as text for accessible output.

## Ask a question in a shell script

The `tui` command wraps the `prompt` package for scripts. The question is drawn
on standard error and the answer is the only thing on standard output, so a
command substitution captures just that.

```sh
tui=github.com/ows4444/tui/cmd/tui
name=$(go run $tui prompt input -default guest "Your name:")
region=$(go run $tui prompt select "Region" eu us ap)
go run $tui prompt confirm "Deploy to $region?" || exit 1
go run $tui prompt spin "Building" -- go build ./...
```

`confirm` answers with its exit code (0 for yes, 1 for no), `multiselect` prints
one choice per line, and `spin` runs a command under a spinner, shows its
output only if it fails, and exits with its code. Esc or Ctrl+C exits with 130
and prints nothing. Source: `cmd/tui/prompt.go`; in a Go program call `prompt`
directly.
## Select and copy text in a viewport

Terminals lose their own text selection once an app turns the mouse on. A
viewport brings it back inside the app: with `Mouse` on and `Bounds` set to where
it is drawn, a press and drag selects across lines (following soft wrap and
scrolling at the edges), the `Copy` key (`y`) puts the selection on the clipboard
with OSC 52, and `SelectedText` returns it as plain text.

```go
m := viewport.New(20, 3)
m.Mouse = true
m.Bounds = hittest.Rect{W: 20, H: 3}
// ... after a drag from (6,0) to (4,1) over "alpha beta" and "gamma delta":
fmt.Println(m.SelectedText())
// Output:
// beta
// gamma
```

Source: `ExampleModel_SelectedText`, `viewport/example_test.go`. Clipboard writing
is best effort: it needs a terminal that accepts OSC 52 (and `set-clipboard` in
tmux).

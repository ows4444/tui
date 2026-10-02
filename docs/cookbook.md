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

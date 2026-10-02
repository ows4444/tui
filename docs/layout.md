# Layout

A [`layout.Node`](https://pkg.go.dev/github.com/ows4444/tui/layout#Node) is sized
in two passes: `Measure` reports the size the node wants under some
`Constraints`, then `Render` is handed the final `Size` and must return exactly
that many columns and rows. `layout.Draw` runs both passes and returns the
string your `View` returns. Watch the zero value: `Constraints{}` bounds both
axes to 0, so it draws nothing. Use `layout.Unconstrained()`, `layout.Loose`, or
set `MaxW`/`MaxH` to `layout.Unbounded`.

```go
ui := layout.Column(0,
	layout.FlexChild{Node: layout.BoxNode(layout.NewBox().Border(layout.NormalBorder()), layout.Block("title"))},
	layout.FlexChild{Grow: 1, Node: layout.Row(1,
		layout.FlexChild{Node: layout.Block("nav"), Basis: 5},
		layout.FlexChild{Node: layout.Block("body"), Grow: 1},
	)},
	layout.FlexChild{Node: layout.Block("q: quit")},
)
show(layout.Draw(ui, layout.Constraints{MinW: 24, MaxW: 24, MinH: 6, MaxH: 6}))
// Output:
// ┌──────────────────────┐
// │title                 │
// └──────────────────────┘
// nav   body
//
// q: quit
```

From `Example_measureRender` in `layout/example_test.go`. `show` is that file's
helper: it prints each line with trailing spaces trimmed.

## Drawing a screen in View

Most programs in `examples/` build the screen as a `layout.Node` in a helper
and draw it in `View`:

```go
func (m model) View() string { return layout.Draw(m.screen(), layout.Unconstrained()) }
```

From `examples/focus/main.go` (the same line appears in `examples/pager`,
`examples/login`, `examples/procstream` and others).

`Draw` renders at the measured size. To fill the terminal exactly, keep the
`Width` and `Height` of the last `tui.ResizeMsg` in your model and call
`layout.DrawTight(root, layout.Size{W: w, H: h})`: the root is rendered at
exactly that size, so `Fill` children take all the remaining space.
`examples/dashboard` caps only the width:
`layout.Constraints{MaxW: maxW, MaxH: layout.Unbounded}`.

`Draw` always returns a rectangle: every row is padded to the widest one
(`ExampleDraw_rectangle`). Compare drawn output by content, not raw width.

## Sizing children of a Row or Column

`Row` and `Column` take a gap and a list of
[`FlexChild`](https://pkg.go.dev/github.com/ows4444/tui/layout#FlexChild)
values. The zero `FlexChild` is sized to its content and neither grows nor
shrinks. `Basis`, `BasisLen` (`Pct`, `Fr`), `Grow`, `Shrink`, `Min`, `Max` and
`CrossAlign` are documented on the type. The behaviours that catch people:

- **A growing child that measures taller than the screen pushes its siblings
  out of view.** A viewport or a long log measures to its full content. Wrap it
  in `layout.Fill` (Basis 1, Grow 1, Shrink 1) instead of setting `Grow: 1`.
  Compare `ExampleFill`, which keeps the footer, with `Example_growTrap`,
  which loses it. `FillWeight` splits leftover space unevenly.
- **Without a constrained height, give a scrolling child a `Basis`.** It is
  the window height (`ExampleColumn_window`).
- **Use the gap for blank rows, not `Block("")`.** An empty `Block` is 0x0
  (`ExampleColumn_gap`).
- **Children are stretched across the cross axis by default.** The zero
  `CrossAlign` is `CrossStretch`: a short box beside a tall one in a `Row` grows
  to the row's height, and one long line in a `Column` widens a bordered box
  above it. `CrossStart`, `CrossCenter` and `CrossEnd` keep the child's own
  cross size (`ExampleCrossAlign`, `ExampleColumn_crossStart`).
- **Measure may run more than once.** It must be side-effect free. Within one
  `Draw`, the built-in containers measure each child at most once per distinct
  `Constraints`.

`RowJustify` and `ColumnJustify` add main-axis spacing (`JustifyStart`,
`JustifyEnd`, `JustifyCenter`, `JustifySpaceBetween`, `JustifySpaceAround`,
`JustifySpaceEvenly`).

## Putting a widget in a layout

Every component package's `Model` has a `LayoutNode` method.
`TestEveryModelHasLayoutNodeOrIsAllowlisted` (`layoutnode_conformance_test.go`)
fails if a package with a `Model` lacks one, and its allowlist is empty.
`wizard.Model.LayoutNode` and `markdown.Model.LayoutNode` take a `theme.Theme`.
All the others take no arguments.

A widget's node is not the same picture as its `View`. The node takes whatever
size the layout gives it, and does not change the Model. A `textinput.Model`'s
`Width` limits its `View`, but its node measures the whole value and is cut to
the width the layout allots:

```go
in := textinput.New()
in.Prompt = "Name: "
in.Width = 12
in.SetValue("a value much longer than the width")
fmt.Println("View width:", ansi.Width(in.View()))
fmt.Println("node width:", ansi.Width(layout.Draw(in.LayoutNode(), layout.Unconstrained())))
fixed := layout.Row(0, layout.FlexChild{Node: in.LayoutNode(), Basis: 20})
fmt.Println("node in a Basis-20 Row:", ansi.Width(layout.Draw(fixed, layout.Unconstrained())))
// Output:
// View width: 18
// node width: 41
// node in a Basis-20 Row: 20
```

From `Example_widgetNodeIsNotItsView` in `layout/example_test.go`.

For anything that is already a string, such as the output of a `widgets`
function, use `layout.Block` (`widgets.Node` is the same adapter). `layout.Block`
pads or clips the string to the allotted size without splitting a rune or an
escape sequence. For a view that changes on its own, such as a clock, use
`layout.ViewFunc`, which calls the function each time it is measured or
rendered.

## Choosing a node

| Need | Node |
| --- | --- |
| Columns that line up across rows | `GridNode`, `GridNodeGaps` with `[]Track` (`ExampleGridNode`) |
| Different trees for different terminal sizes | `Responsive` with `[]Break` (`ExampleResponsive`) |
| A "terminal too small" screen | `MinSize(n, min, fallback)` |
| An exact or bounded size | `Fixed` (`ExampleFixed`), `MinMax` |
| Border, padding, title, background | `BoxNode(layout.NewBox()..., child)` |
| Wrapping or styled text | `Text` with `WithWrap`, `WithAlign`, `WithEllipsis`, `WithStyle`; `StyledText` |
| A scrolled window onto a tall child | `Scroll(child, offset)`. Rows wrapped in `Sticky` pin to the top. A child implementing `Windowed` renders only the visible rows |
| Layers | `Stack` with `Layer(z, n)` and `Absolute(x, y, n)`, or `OverlayNode(base, over, x, y)` |

Pre-rendered strings have their own helpers: `layout.Overlay(base, overlay, x, y)`,
`JoinHorizontalJustify`, and `Window`/`WindowRange` for keeping a cursor line in
view. `Box.Render` is deprecated in favour of `BoxNode`.

## Overlays on a drawn screen

The overlay widgets (`dialog`, `drawer`, `popover`, `toast`, `helpscreen`,
`contextmenu`, `menubar`) implement `tui.Overlay`: `Render(base string) string`
composites them onto an already drawn frame. Draw the screen first, then layer
them:

```go
base := layout.Draw(m.screen(), layout.Constraints{MaxW: maxW, MaxH: layout.Unbounded})
return m.notice.Render(m.about.Render(base))
```

From `View` in `examples/dashboard/main.go`, where `about` is a `dialog.Model`
and `notice` a `toast.Model`.

## Finding where a node was drawn

Wrap a node in `layout.Named(name, n)`; layout is unchanged. Then:

- `layout.Rects(root, size)` lists every node with its rectangle, parent before
  children.
- `layout.RectOf(root, size, name)` returns one rectangle.
- `layout.NamedAt(root, size, x, y)` returns the innermost named node at a
  cell.

Pass the size the root was rendered at. These feed mouse hit-testing and tab
order; see [Input](input.md#mouse-and-hit-testing).

## Drawing into cells

A node that also implements `layout.CellNode` draws straight into a
`layout.CellSurface` with `layout.DrawTo`, without building a string.
`cellbuf.Layout` adapts a `cellbuf.Buffer`. See [Rendering](rendering.md).

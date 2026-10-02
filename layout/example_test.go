package layout_test

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/textinput"
)

// show prints out with trailing spaces trimmed (Go example output is
// compared verbatim, and Render pads every row to the full width).
func show(out string) {
	for _, l := range strings.Split(out, "\n") {
		fmt.Println(strings.TrimRight(l, " "))
	}
}

// A Node tree is sized in two passes: Draw measures the tree under the
// given constraints, then renders it at the measured size. Here the
// window is pinned to 24x6, so the body row grows to fill what the header
// and footer leave, and the sidebar keeps a fixed basis.
func Example_measureRender() {
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
}

// GridNode aligns columns across rows and lets one column absorb the
// leftover width.
func ExampleGridNode() {
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
}

// GridNodeGaps gives a grid different gaps between columns and between rows.
// GridNode's single gap would put a blank row between every pair of rows here.
func ExampleGridNodeGaps() {
	g := layout.GridNodeGaps(
		[]layout.Track{{}, {}},
		2, 0, // two blank columns between columns, no blank rows between rows
		layout.Block("api"), layout.Block("up"),
		layout.Block("worker"), layout.Block("down"),
		layout.Block("db"), layout.Block("up"),
	)
	show(layout.Draw(g, layout.Unconstrained()))
	// Output:
	// api     up
	// worker  down
	// db      up
}

// Fill makes a panel take whatever space is left, even when its content is
// far longer than the screen: the header and footer stay visible and the
// long log fills the middle.
func ExampleFill() {
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
}

// The zero Constraints bounds both axes to 0, so nothing fits: use
// Unconstrained, Loose or an explicit Constraints with MaxH set.
func ExampleConstraints_zeroValue() {
	n := layout.Block("hello")
	fmt.Printf("zero value: %q\n", layout.Draw(n, layout.Constraints{}))
	fmt.Printf("unconstrained: %q\n", layout.Draw(n, layout.Unconstrained()))
	// Output:
	// zero value: ""
	// unconstrained: "hello"
}

// A plain Grow child that measures taller than the screen pushes its
// siblings out of view. Compare ExampleFill, which keeps the footer.
func Example_growTrap() {
	lines := make([]string, 100)
	for i := range lines {
		lines[i] = fmt.Sprintf("log %02d", i)
	}
	ui := layout.Column(0,
		layout.FlexChild{Node: layout.Block("== header ==")},
		layout.FlexChild{Node: layout.Block(strings.Join(lines, "\n")), Grow: 1},
		layout.FlexChild{Node: layout.Block("== footer ==")},
	)
	show(layout.Draw(ui, layout.Constraints{MinW: 12, MaxW: 12, MinH: 5, MaxH: 5}))
	// Output:
	// == header ==
	// log 00
	// log 01
	// log 02
	// log 03
}

// A Column's gap inserts the blank row between sections. Do not use
// Block("") as a spacer: it is 0x0, not a blank row.
func ExampleColumn_gap() {
	ui := layout.Column(1,
		layout.FlexChild{Node: layout.Block("top")},
		layout.FlexChild{Node: layout.Block("bottom")},
	)
	show(layout.Draw(ui, layout.Unconstrained()))
	// Output:
	// top
	//
	// bottom
}

// A Row stretches every child to the row's full height, so a short box
// beside a tall one grows to match. CrossStart keeps its own height.
func ExampleCrossAlign() {
	tall := layout.Block("a\nb\nc\nd\ne")
	short := layout.Block("x")
	stretched := layout.Row(1, layout.FlexChild{Node: tall}, layout.FlexChild{Node: layout.BoxNode(layout.NewBox().Border(layout.NormalBorder()), short)})
	kept := layout.Row(1, layout.FlexChild{Node: tall}, layout.FlexChild{Node: layout.BoxNode(layout.NewBox().Border(layout.NormalBorder()), short), CrossAlign: layout.CrossStart})
	show(layout.Draw(stretched, layout.Unconstrained()))
	fmt.Println("--")
	show(layout.Draw(kept, layout.Unconstrained()))
	// Output:
	// a ┌─┐
	// b │x│
	// c │ │
	// d │ │
	// e └─┘
	// --
	// a ┌─┐
	// b │x│
	// c └─┘
	// d
	// e
}

// A Column stretches every child to its own width, so one long line widens the
// bordered box above it. CrossStart keeps the box at its own width.
func ExampleColumn_crossStart() {
	box := layout.BoxNode(layout.NewBox().Border(layout.NormalBorder()), layout.Block("hi"))
	help := layout.Block("a much longer help line")

	show(layout.Draw(layout.Column(0,
		layout.FlexChild{Node: box},
		layout.FlexChild{Node: help},
	), layout.Unconstrained()))
	fmt.Println()
	show(layout.Draw(layout.Column(0,
		layout.FlexChild{Node: box, CrossAlign: layout.CrossStart},
		layout.FlexChild{Node: help},
	), layout.Unconstrained()))
	// Output:
	// ┌─────────────────────┐
	// │hi                   │
	// └─────────────────────┘
	// a much longer help line
	//
	// ┌──┐
	// │hi│
	// └──┘
	// a much longer help line
}

// Fixed gives a node an exact width, height or both, whatever it contains. A
// zero axis is left to the node.
func ExampleFixed() {
	box := layout.BoxNode(layout.NewBox().Border(layout.NormalBorder()), layout.Block("hi"))
	fixed := layout.Fixed(box, layout.Size{W: 12})
	show(layout.Draw(fixed, layout.Unconstrained()))
	// Output:
	// ┌──────────┐
	// │hi        │
	// └──────────┘
}

// A scrolling child measures to its whole content, so on its own it would make
// the view as tall as the content. A Basis on the child is its window height.
func ExampleColumn_window() {
	lines := make([]string, 100)
	for i := range lines {
		lines[i] = fmt.Sprintf("log %02d", i)
	}
	ui := layout.Column(0,
		layout.FlexChild{Node: layout.Block("== header ==")},
		layout.FlexChild{Node: layout.Block(strings.Join(lines, "\n")), Basis: 3},
		layout.FlexChild{Node: layout.Block("== footer ==")},
	)
	show(layout.Draw(ui, layout.Unconstrained()))
	// Output:
	// == header ==
	// log 00
	// log 01
	// log 02
	// == footer ==
}

// Draw returns a rectangle: every row is padded to the widest one. Compare a
// drawn view by its content (trim the trailing spaces), not its raw width.
func ExampleDraw_rectangle() {
	ui := layout.Column(0, layout.FlexChild{Node: layout.Block("ab")}, layout.FlexChild{Node: layout.Block("abcdef")})
	fmt.Printf("%q\n", layout.Draw(ui, layout.Unconstrained()))
	// Output:
	// "ab    \nabcdef"
}

// A widget's LayoutNode is not the same picture as its View. A textinput's
// Width limits its View (a longer value scrolls to fit), but its node ignores
// Width and measures the whole value: in an unconstrained layout it is as wide
// as the value, and it takes the width the layout gives it (see Row's Basis).
func Example_widgetNodeIsNotItsView() {
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
}

func ExampleResponsive() {
	ui := layout.Responsive([]layout.Break{{MaxW: 20}}, layout.Block("menu"), layout.Block("home | search | settings"))
	fmt.Println(layout.Draw(ui, layout.Loose(layout.Size{W: 20, H: 3})))
	fmt.Println(layout.Draw(ui, layout.Loose(layout.Size{W: 40, H: 3})))
	// Output:
	// menu
	// home | search | settings
}

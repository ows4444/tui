package main

import (
	"fmt"
	"strconv"
	"strings"
)

type mode int

const (
	mFlex mode = iota
	mGrid
	mWrap
	mAnchor
	numModes
)

var modeNames = [numModes]string{"Flex", "Grid", "Wrap", "Anchor"}

// param is one setting: a slider over lo..hi, or a choice among opts.
type param struct {
	label, key string
	lo, hi     int
	opts       []string
}

var (
	dirs     = []string{"Row", "Column"}
	justify  = []string{"Start", "End", "Center", "SpaceBetween", "SpaceAround"}
	crosses  = []string{"Stretch", "Start", "Center", "End"}
	corners  = []string{"TopLeft", "TopRight", "BottomLeft", "BottomRight", "Center"}
	maxKids  = 5
	maxTrack = 5
)

// defaults are the settings each mode starts from.
func defaults(md mode) map[string]int {
	v := map[string]int{"width": 60, "height": 12}
	switch md {
	case mFlex:
		v["dir"], v["gap"], v["justify"], v["count"], v["child"] = 0, 1, 0, 3, 0
		for i, b := range []int{10, 0, 12, 0, 8} {
			v["basis."+strconv.Itoa(i)] = b
			v["grow."+strconv.Itoa(i)] = []int{0, 1, 0, 1, 0}[i]
			v["shrink."+strconv.Itoa(i)] = 1
		}
	case mGrid:
		v["cols"], v["col"], v["cells"], v["colgap"], v["rowgap"], v["spanc"], v["spanr"], v["height"] = 3, 0, 7, 1, 0, 2, 1, 14
		for j := 0; j < 5; j++ {
			v["grow."+strconv.Itoa(j)] = 1
		}
	case mWrap:
		v["count"], v["colgap"], v["rowgap"], v["width"], v["height"] = 10, 1, 1, 40, 10
	case mAnchor:
		v["corner"], v["dx"], v["dy"], v["width"], v["height"] = 1, 1, 0, 40, 10
	}
	return v
}

// params lists the settings of mode md given the values v, with the per-child
// and per-column ones for the selected child or column.
func params(md mode, v map[string]int) []param {
	size := []param{{"width", "width", 6, 120, nil}, {"height", "height", 3, 40, nil}}
	switch md {
	case mFlex:
		c := strconv.Itoa(v["child"])
		return append([]param{
			{"direction", "dir", 0, 1, dirs}, {"gap", "gap", 0, 4, nil}, {"justify", "justify", 0, 4, justify}, {"children", "count", 2, maxKids, nil},
			{"child", "child", 0, v["count"] - 1, nil},
			{"  basis", "basis." + c, 0, 40, nil}, {"  grow", "grow." + c, 0, 5, nil}, {"  shrink", "shrink." + c, 0, 5, nil},
			{"  min", "min." + c, 0, 30, nil}, {"  max (0: none)", "max." + c, 0, 60, nil}, {"  cross align", "cross." + c, 0, 3, crosses},
		}, size...)
	case mGrid:
		c := strconv.Itoa(v["col"])
		return append([]param{
			{"columns", "cols", 1, maxTrack, nil}, {"column", "col", 0, v["cols"] - 1, nil},
			{"  size (0: auto)", "size." + c, 0, 30, nil}, {"  grow", "grow." + c, 0, 4, nil},
			{"cells", "cells", 1, 14, nil}, {"column gap", "colgap", 0, 3, nil}, {"row gap", "rowgap", 0, 3, nil},
			{"cell 1 col span", "spanc", 1, 3, nil}, {"cell 1 row span", "spanr", 1, 3, nil},
		}, size...)
	case mWrap:
		return append([]param{{"children", "count", 1, 24, nil}, {"column gap", "colgap", 0, 4, nil}, {"row gap", "rowgap", 0, 3, nil}}, size...)
	}
	return append([]param{{"corner", "corner", 0, 4, corners}, {"dx", "dx", -6, 6, nil}, {"dy", "dy", -4, 4, nil}}, size...)
}

func (p param) text(v int) string {
	if p.opts != nil {
		return p.opts[max(0, min(len(p.opts)-1, v))]
	}
	return strconv.Itoa(v)
}

// code is the Go that builds the layout the settings describe.
func code(md mode, v map[string]int) string {
	var b strings.Builder
	line := func(f string, a ...any) { b.WriteString(fmt.Sprintf(f, a...) + "\n") }
	switch md {
	case mFlex:
		fn := map[int]string{0: "RowJustify", 1: "ColumnJustify"}[v["dir"]]
		line("layout.%s(%d, layout.Justify%s,", fn, v["gap"], justify[v["justify"]])
		for i := 0; i < v["count"]; i++ {
			k := strconv.Itoa(i)
			f := fmt.Sprintf("Node: tile%d", i+1)
			for _, e := range []struct{ name, key string }{{"Basis", "basis."}, {"Grow", "grow."}, {"Shrink", "shrink."}, {"Min", "min."}, {"Max", "max."}} {
				if n := v[e.key+k]; n != 0 {
					f += fmt.Sprintf(", %s: %d", e.name, n)
				}
			}
			if c := v["cross."+k]; c != 0 {
				f += ", CrossAlign: layout.Cross" + crosses[c]
			}
			line("\tlayout.FlexChild{%s},", f)
		}
		line(")")
	case mGrid:
		line("layout.GridNodeGaps([]layout.Track{")
		for j := 0; j < v["cols"]; j++ {
			line("\t{Size: %d, Grow: %d},", v["size."+strconv.Itoa(j)], v["grow."+strconv.Itoa(j)])
		}
		line("}, %d, %d,", v["colgap"], v["rowgap"])
		line("\tlayout.Span(cell1, %d, %d), cell2 ... cell%d)", v["spanc"], v["spanr"], v["cells"])
	case mWrap:
		line("layout.Wrap(%d, %d, chip1 ... chip%d)", v["colgap"], v["rowgap"], v["count"])
	case mAnchor:
		line("layout.Stack(panel,")
		line("\tlayout.Anchor(layout.%s, %d, %d, badge))", corners[v["corner"]], v["dx"], v["dy"])
	}
	return b.String()
}

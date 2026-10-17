package treeview

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

type color int

func (c color) String() string { return [...]string{"red", "green"}[c] }

type inner struct {
	Name string
	N    int
}

type outer struct {
	ID      int
	Tags    []string
	Inner   *inner
	Extra   map[string]float64
	Err     error
	At      time.Time
	Color   color
	Raw     []byte
	secret  string
	Nothing *inner
}

// dump flattens a node tree to indented labels.
func dump(n Node, depth int, out *[]string) {
	*out = append(*out, strings.Repeat("  ", depth)+n.Label)
	for _, c := range n.Children {
		dump(c, depth+1, out)
	}
}

func lines(n Node) string {
	var out []string
	dump(n, 0, &out)
	return strings.Join(out, "\n")
}

func TestValueNodeShowsAStruct(t *testing.T) {
	v := outer{ID: 7, Tags: []string{"a", "b"}, Inner: &inner{"x", 3}, Extra: map[string]float64{"z": 1.5, "a": 2},
		Err: errors.New("boom"), At: time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC), Color: 1, Raw: []byte{1, 2, 3}, secret: "hidden"}
	got := lines(ValueNode("cfg", v))
	want := strings.Join([]string{
		"cfg outer {}",
		"  ID: 7",
		"  Tags []",
		`    0: "a"`,
		`    1: "b"`,
		"  Inner {} inner {}"[:0] + "  Inner inner {}",
		`    Name: "x"`,
		"    N: 3",
		"  Extra {}",
		"    a: 2",
		"    z: 1.5",
		`  Err: "boom"`,
		"  At: 2026-10-17T09:00:00Z",
		`  Color: "green"`,
		"  Raw: 3 bytes",
		"  Nothing: nil",
	}, "\n")
	if got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
	if strings.Contains(got, "hidden") {
		t.Fatal("an unexported field was shown")
	}
}

func TestValueNodeCycleDepthAndLimits(t *testing.T) {
	type node struct {
		Next *node
		V    int
	}
	a := &node{V: 1}
	a.Next = a
	if got := lines(ValueNode("n", a)); !strings.Contains(got, "Next: <cycle>") {
		t.Fatalf("no cycle mark:\n%s", got)
	}
	deep := map[string]any{"a": map[string]any{"b": map[string]any{"c": 1}}}
	if got := lines(ValueNode("d", deep, ValueOptions{MaxDepth: 2})); !strings.Contains(got, "b: ...") {
		t.Fatalf("depth limit:\n%s", got)
	}
	big := make([]int, 10)
	got := lines(ValueNode("s", big, ValueOptions{MaxItems: 3}))
	if !strings.Contains(got, "...: 7 more") || strings.Count(got, "\n") != 4 {
		t.Fatalf("item limit:\n%s", got)
	}
	m := map[int]bool{}
	for i := 0; i < 6; i++ {
		m[i] = true
	}
	if got := lines(ValueNode("m", m, ValueOptions{MaxItems: 2})); !strings.Contains(got, "...: 4 more") {
		t.Fatalf("map limit:\n%s", got)
	}
}

func TestValueNodeNilsAndOddKinds(t *testing.T) {
	var p *inner
	var i any
	var m map[string]int
	var s []int
	var f func()
	for name, v := range map[string]any{"p": p, "i": i, "m": m, "s": s, "f": f} {
		if got := lines(ValueNode(name, v)); got != name+": nil" {
			t.Errorf("%s: %q", name, got)
		}
	}
	if got := lines(ValueNode("f", func() {})); got != "f: func()" {
		t.Errorf("func: %q", got)
	}
	if got := lines(ValueNode("arr", [2]bool{true, false})); got != "arr []\n  0: true\n  1: false" {
		t.Errorf("array: %q", got)
	}
	if got := lines(ValueNode("c", make(chan int))); got != "c: chan int" {
		t.Errorf("chan: %q", got)
	}
}

type zeroPanics struct{ p *int }

func (z zeroPanics) String() string { return fmt.Sprint(*z.p) }

func TestValueNodeSurvivesAStringMethodThatPanics(t *testing.T) {
	got := lines(ValueNode("z", zeroPanics{}))
	if !strings.HasPrefix(got, "z zeroPanics {}") {
		t.Fatalf("%q", got)
	}
}

func TestValueNodeCleansEscapes(t *testing.T) {
	evil := "x\x1b]52;c;ZXZpbA==\x07y"
	got := lines(ValueNode(evil, map[string]string{evil: evil}))
	if strings.Contains(got, "\x1b") || strings.Contains(got, "\x07") {
		t.Fatalf("escape reached a label: %q", got)
	}
}

func TestFromValueIsATreeModel(t *testing.T) {
	m := FromValue(map[string]int{"a": 1})
	if rows := m.VisibleRows(); len(rows) != 1 || !rows[0].HasChildren {
		t.Fatalf("%+v", rows)
	}
}

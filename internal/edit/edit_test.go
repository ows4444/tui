package edit

import (
	"reflect"
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
)

const family = "👨‍👩‍👧"

func TestSplitClusters(t *testing.T) {
	cases := map[string][]string{
		"ab":         {"a", "b"},
		"éx":        {"é", "x"},
		family + "z": {family, "z"},
		"👍🏽!":        {"👍🏽", "!"},
		"世界":         {"世", "界"},
		"🇺🇸🇯🇵":       {"🇺🇸", "🇯🇵"},
		"한":        {"한"},
		"a\tb":       {"a", "\t", "b"},
	}
	for in, want := range cases {
		if got := Split(in); !reflect.DeepEqual(got, want) {
			t.Errorf("Split(%q) = %q, want %q", in, got, want)
		}
	}
}

// #34
func TestBackspaceDeleteRemoveWholeCluster(t *testing.T) {
	var e Editor
	e.Set("a"+family+"é", 0)
	e.Backspace()
	if e.String() != "a"+family {
		t.Fatalf("backspace over e+accent: %q", e.String())
	}
	e.Backspace()
	if e.String() != "a" {
		t.Fatalf("backspace over ZWJ family: %q", e.String())
	}
	e.Set("x"+family+"y", 0)
	e.SetCursor(1)
	e.Delete()
	if e.String() != "xy" {
		t.Fatalf("delete of ZWJ family: %q", e.String())
	}
}

// #35
func TestLeftRightMoveOneCluster(t *testing.T) {
	var e Editor
	e.Set(family+"é", 0)
	if e.Len() != 2 || e.Cursor() != 2 {
		t.Fatalf("len %d cursor %d", e.Len(), e.Cursor())
	}
	e.Left()
	e.Left()
	e.Left()
	if e.Cursor() != 0 {
		t.Fatal("left did not clamp")
	}
	e.Right()
	if e.Cursor() != 1 {
		t.Fatalf("right moved %d clusters", e.Cursor())
	}
}

// #36
func TestWordMotion(t *testing.T) {
	var e Editor
	e.Set("foo  bar baz", 0)
	e.WordLeft()
	if e.Cursor() != 9 {
		t.Fatalf("WordLeft from end = %d", e.Cursor())
	}
	e.WordLeft()
	e.WordLeft()
	if e.Cursor() != 0 {
		t.Fatalf("WordLeft x3 = %d", e.Cursor())
	}
	e.WordRight()
	if e.Cursor() != 3 {
		t.Fatalf("WordRight = %d", e.Cursor())
	}
	e.WordRight()
	if e.Cursor() != 8 {
		t.Fatalf("WordRight 2 = %d", e.Cursor())
	}
}

// #37
func TestLargePasteOneInsert(t *testing.T) {
	var e Editor
	e.Set("ab", 0)
	e.SetCursor(1)
	big := strings.Repeat("x", 100000)
	allocs := testing.AllocsPerRun(1, func() {
		e2 := e
		e2.Insert(big, 0)
	})
	// Split allocates its output; the splice itself must be O(1) allocations
	// beyond that (no per-rune slice copy).
	e.Insert(big, 0)
	if e.Len() != 100002 || e.Cursor() != 100001 || !strings.HasPrefix(e.String(), "axxx") || !strings.HasSuffix(e.String(), "xb") {
		t.Fatalf("len %d cursor %d", e.Len(), e.Cursor())
	}
	if allocs > 200 {
		t.Fatalf("%v allocs for one paste", allocs)
	}
}

func TestInsertJoinsWithPreviousCluster(t *testing.T) {
	var e Editor
	e.Set("e", 0)
	e.Insert("́", 0)
	if e.Len() != 1 {
		t.Fatalf("combining mark not attached: %d clusters", e.Len())
	}
}

func TestInsertLimit(t *testing.T) {
	var e Editor
	e.Set("ab", 3)
	e.Insert("cdef", 3)
	if e.String() != "abc" {
		t.Fatal(e.String())
	}
}

// #33
func TestWindowNeverExceedsCols(t *testing.T) {
	for _, txt := range []string{"世界你好世界你好世界你好", "a世b界c你d好", family + "世" + family + "abc"} {
		for cols := 2; cols <= 12; cols++ {
			var e Editor
			e.Set(txt, 0)
			for pos := 0; pos <= e.Len(); pos++ {
				e.SetCursor(pos)
				s, en := e.Window(cols)
				if s > pos || (pos < e.Len() && en <= pos) || (pos == e.Len() && en != pos) {
					t.Fatalf("%q cols %d pos %d: window [%d,%d) hides cursor", txt, cols, pos, s, en)
				}
				w := ansi.Width(strings.Join(e.cl[s:en], ""))
				if pos == e.Len() {
					w++
				}
				if w > cols {
					t.Fatalf("%q cols %d pos %d: window [%d,%d) is %d cols", txt, cols, pos, s, en, w)
				}
			}
		}
	}
}

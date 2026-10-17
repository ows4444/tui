package main

import (
	"bytes"
	"math"
	"strings"
	"testing"
)

func TestReplaceInsertsDeletesAndOverwrites(t *testing.T) {
	b := &buffer{data: []byte{1, 2, 3, 4}}
	b.replace(1, 1, []byte{9}, false)
	b.replace(2, 0, []byte{7, 8}, false)
	b.replace(0, 1, nil, false)
	if !bytes.Equal(b.data, []byte{9, 7, 8, 3, 4}) {
		t.Fatalf("data = %v", b.data)
	}
	b.replace(5, 0, []byte{5}, false) // append
	if !bytes.Equal(b.data, []byte{9, 7, 8, 3, 4, 5}) {
		t.Fatalf("data = %v", b.data)
	}
}

func TestUndoRedoRoundTripEveryStep(t *testing.T) {
	b := &buffer{data: []byte{1, 2, 3}}
	states := [][]byte{append([]byte(nil), b.data...)}
	for _, step := range []func(){
		func() { b.replace(1, 1, []byte{9}, false) },
		func() { b.replace(0, 0, []byte{0xaa, 0xbb}, false) },
		func() { b.replace(2, 2, nil, false) },
		func() { b.replace(len(b.data), 0, []byte{4}, false) },
	} {
		step()
		states = append(states, append([]byte(nil), b.data...))
	}
	for i := len(states) - 2; i >= 0; i-- {
		if _, ok := b.undoStep(); !ok || !bytes.Equal(b.data, states[i]) {
			t.Fatalf("undo to state %d: %v, want %v", i, b.data, states[i])
		}
	}
	if _, ok := b.undoStep(); ok {
		t.Fatal("undo with nothing left")
	}
	for i := 1; i < len(states); i++ {
		if _, ok := b.redoStep(); !ok || !bytes.Equal(b.data, states[i]) {
			t.Fatalf("redo to state %d: %v, want %v", i, b.data, states[i])
		}
	}
	if _, ok := b.redoStep(); ok {
		t.Fatal("redo with nothing left")
	}
}

func TestJoinedStepsUndoTogether(t *testing.T) {
	b := &buffer{data: []byte{0x00}}
	b.replace(0, 1, []byte{0xa0}, false)
	b.replace(0, 1, []byte{0xab}, true)
	b.replace(0, 0, []byte{0x11}, false)
	b.undoStep() // the insert
	if !bytes.Equal(b.data, []byte{0xab}) {
		t.Fatalf("data = %x", b.data)
	}
	b.undoStep() // both nibbles
	if !bytes.Equal(b.data, []byte{0x00}) {
		t.Fatalf("the two halves of a byte did not undo together: %x", b.data)
	}
	b.redoStep()
	if !bytes.Equal(b.data, []byte{0xab}) {
		t.Fatalf("redo of a joined pair: %x", b.data)
	}
}

func TestNewEditForgetsRedo(t *testing.T) {
	b := &buffer{data: []byte{1}}
	b.replace(0, 1, []byte{2}, false)
	b.undoStep()
	b.replace(0, 1, []byte{3}, false)
	if _, ok := b.redoStep(); ok {
		t.Fatal("redo survived a new edit")
	}
}

func TestUndoIsBounded(t *testing.T) {
	b := &buffer{data: []byte{0}}
	for i := 0; i < undoMax+50; i++ {
		b.replace(0, 1, []byte{byte(i)}, false)
	}
	if len(b.undo) != undoMax {
		t.Fatalf("%d steps kept", len(b.undo))
	}
}

func TestParsePattern(t *testing.T) {
	good := map[string]string{
		"de ad be ef": "deadbeef", "DEADBEEF": "deadbeef", "0xCAFE": "cafe", ` "GIF" `: "474946", `"a\x00b"`: "610062", "  00 ff  ": "00ff",
	}
	for in, want := range good {
		got, err := parsePattern(in)
		if err != nil || hexString(got) != strings.Join(splitPairs(want), " ") {
			t.Errorf("parsePattern(%q) = %v, %v; want %s", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "  ", "abc", "zz", `"unterminated`, "de ad b"} {
		if _, err := parsePattern(bad); err == nil {
			t.Errorf("parsePattern(%q) accepted", bad)
		}
	}
}

func splitPairs(s string) []string {
	var out []string
	for i := 0; i+1 < len(s); i += 2 {
		out = append(out, s[i:i+2])
	}
	return out
}

func TestParseOffset(t *testing.T) {
	for _, c := range []struct {
		in        string
		cur, size int
		want      int
	}{
		{"1f", 0, 100, 0x1f}, {"0x10", 0, 100, 16}, {"#99", 0, 1000, 99}, {"+5", 10, 100, 15}, {"-4", 10, 100, 6},
		{"-50", 10, 100, 0}, {"ffff", 0, 100, 99}, {"end", 0, 100, 99}, {"$", 0, 100, 99}, {"0", 5, 0, 0},
	} {
		got, err := parseOffset(c.in, c.cur, c.size)
		if err != nil || got != c.want {
			t.Errorf("parseOffset(%q, %d, %d) = %d, %v; want %d", c.in, c.cur, c.size, got, err, c.want)
		}
	}
	for _, bad := range []string{"", "xyz", "#abc", "--3", "0x"} {
		if _, err := parseOffset(bad, 0, 10); err == nil {
			t.Errorf("parseOffset(%q) accepted", bad)
		}
	}
}

func TestFind(t *testing.T) {
	data := []byte("abcabcabc")
	pat := []byte("bc")
	for _, c := range []struct {
		from int
		back bool
		want int
	}{{0, false, 1}, {2, false, 4}, {8, false, 1}, {5, true, 4}, {0, true, 7}, {4, true, 1}} {
		if got := find(data, pat, c.from, c.back); got != c.want {
			t.Errorf("find from %d back=%v = %d, want %d", c.from, c.back, got, c.want)
		}
	}
	if find(data, []byte("zz"), 0, false) != -1 || find(nil, pat, 0, false) != -1 || find(data, nil, 0, false) != -1 {
		t.Error("find found something that is not there")
	}
	if find([]byte("a"), []byte("abc"), 0, true) != -1 {
		t.Error("a pattern longer than the data matched")
	}
}

func TestInspect(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03, 0x04, 0x00, 0x00, 0x80, 0x3f}
	out := strings.Join(inspect(data, 0), "\n")
	for _, want := range []string{"0x00000000", "u8      1", "u16 le  513", "u16 be  258", "u32 le  67305985", "u32 be  16909060"} {
		if !strings.Contains(out, want) {
			t.Errorf("inspector lacks %q:\n%s", want, out)
		}
	}
	if f := strings.Join(inspect(data, 4), "\n"); !strings.Contains(f, "f32 le  1") {
		t.Errorf("float:\n%s", f)
	}
	if got := inspect(data, 7); len(got) != 6 {
		t.Errorf("one byte left should show only the one-byte rows, got %d", len(got))
	}
	if got := inspect(data, 99); !strings.Contains(got[1], "end of file") {
		t.Errorf("past the end = %v", got)
	}
	if !math.IsNaN(math.NaN()) {
		t.Fatal("unreachable")
	}
}

func TestDisplayHelpers(t *testing.T) {
	if hexString([]byte{0, 0xab, 0xff}) != "00 ab ff" || hexString(nil) != "" {
		t.Error("hexString")
	}
	if asText([]byte{'a', 0, 0x7f, 'z', 0xff}) != "a··z·" {
		t.Errorf("asText = %q", asText([]byte{'a', 0, 0x7f, 'z', 0xff}))
	}
	if !printable(' ') || !printable('~') || printable(0x7f) || printable(0x1f) {
		t.Error("printable")
	}
}

package render

import (
	"strings"
	"testing"
)

func TestPatchRowRefusesWhatItCannotModel(t *testing.T) {
	c := New()
	parse := func(s string) ([]cell, rowInfo) {
		r, ok := c.parseRow(s, nil)
		if !ok {
			t.Fatalf("parseRow(%q) failed", s)
		}
		return r, rowInfo{esc: c.rowEsc, simple: c.rowSimple}
	}
	old := "\x1b[31mabc\x1b[0m def"
	prev, info := parse(old)
	orig := append([]cell(nil), prev...)
	for name, nw := range map[string]string{
		"escape changed":  "\x1b[32mabc\x1b[0m def",
		"length differs":  old + "x",
		"control":         "\x1b[31mabc\x1b[0m d\x01f",
		"identical":       old,
		"blank tail":      "\x1b[31mabc\x1b[0m de ",
		"new escape byte": "\x1b[31mabc\x1b[0m d\x1bf",
		"byte in escape":  "\x1b[31mabc\x1b[1m def",
		"escape and text": "\x1b[32mabX\x1b[0m def",
	} {
		if _, _, ok := patchRow(old, nw, prev, info, nil, nil); ok {
			t.Errorf("%s: patchRow accepted it", name)
		}
		for i := range orig {
			if prev[i] != orig[i] {
				t.Fatalf("%s: a refused patch changed cell %d", name, i)
			}
		}
	}
	// A row that is not simple is never patched.
	nonSimple := "\x1b[31mab\u00e9\x1b[0m def"
	ns, nsInfo := parse(nonSimple)
	if _, _, ok := patchRow(nonSimple, strings.Replace(nonSimple, "def", "dXf", 1), ns, nsInfo, nil, nil); ok {
		t.Error("patched a row with non-ASCII text")
	}
	idx, pos, ok := patchRow(old, "\x1b[31mabX\x1b[0m dXf", prev, info, nil, nil)
	if !ok || len(idx) != 2 || idx[0] != 2 || idx[1] != 5 || pos[0] != 7 || pos[1] != 14 {
		t.Fatalf("patch = %v %v %v", idx, pos, ok)
	}
	want, _ := parse("\x1b[31mabX\x1b[0m dXf")
	if len(prev) != len(want) {
		t.Fatalf("len %d want %d", len(prev), len(want))
	}
	for i := range want {
		if prev[i] != want[i] {
			t.Errorf("cell %d = %+v, want %+v", i, prev[i], want[i])
		}
	}
}

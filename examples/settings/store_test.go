package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEncodeDecodeRoundTrip(t *testing.T) {
	cats := catalogue()
	v := defaults(cats).with("font", 20).with("wrap", 1).with("lang", 3)
	got := decode(cats, encode(cats, v))
	for k, want := range v {
		if got[k] != want {
			t.Errorf("%s = %v after a round trip, want %v", k, got[k], want)
		}
	}
}

func TestDecodeIgnoresBadLines(t *testing.T) {
	cats := catalogue()
	got := decode(cats, "font=99\nlang=9\nnope=1\nwrap=abc\nfont\n=\ntab=2\n")
	d := defaults(cats)
	if got["font"] != d["font"] || got["lang"] != d["lang"] || got["wrap"] != d["wrap"] {
		t.Fatalf("an out-of-range or malformed line changed a setting: %v", got)
	}
	if got["tab"] != 2 {
		t.Fatalf("a good line was dropped: tab = %v", got["tab"])
	}
	if _, ok := got["nope"]; ok {
		t.Fatal("an unknown key was kept")
	}
}

func TestFromFileMissingStartsWithDefaults(t *testing.T) {
	m, err := fromFile(initialModel(), filepath.Join(t.TempDir(), "none.txt"))
	if err != nil || len(m.dirtyKeys()) != 0 {
		t.Fatalf("err=%v dirty=%v", err, m.dirtyKeys())
	}
}

func TestApplyWritesTheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.txt")
	m, _ := fromFile(initialModel(), path)
	m = m.set(0, 2) // language
	m, cmd := m.apply()
	if cmd == nil {
		t.Fatal("Apply with a file gave no write")
	}
	cmd()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := decode(m.cats, string(b)); got["lang"] != 2 {
		t.Fatalf("file holds lang = %v, want 2", got["lang"])
	}
	m2, _ := fromFile(initialModel(), path)
	if m2.cur["lang"] != 2 || len(m2.dirtyKeys()) != 0 {
		t.Fatal("reading the file back did not restore the setting as saved")
	}
}

func TestSettingMath(t *testing.T) {
	font := catalogue()[1].items[1]
	if got := font.clamp(99); got != 32 {
		t.Errorf("clamp(99) = %v", got)
	}
	if got := font.step(32, 1); got != 32 {
		t.Errorf("a slider stepped past its end: %v", got)
	}
	sp := catalogue()[1].items[2]
	if got := sp.step(1.4, 1); got != 1.5 {
		t.Errorf("step(1.4) = %v, want 1.5", got)
	}
	lang := catalogue()[0].items[0]
	if got := lang.step(0, -1); got != 3 {
		t.Errorf("a choice did not wrap back: %v", got)
	}
}

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const miniClass = `# @missing: 0000..10FFFF; Left_To_Right
# @missing: 0590..05FF; Right_To_Left
# @missing: 20A0..20CF; European_Terminator
0009          ; S # TAB
0030..0039    ; EN # digits
05D0..05D2    ; R # letters
05C0          ; L # an explicit line beats the Hebrew default
0600..0605    ; AN # arabic-indic
`

func TestClassesApplyMissingDefaultsThenExplicitLines(t *testing.T) {
	got, err := classes([]byte(miniClass))
	if err != nil {
		t.Fatal(err)
	}
	at := func(r rune) string {
		for _, c := range got {
			if c.lo <= r && r <= c.hi {
				return c.class
			}
		}
		return "L"
	}
	for r, want := range map[rune]string{
		'a': "L", '\t': "S", '5': "EN",
		0x05D1: "R", // listed
		0x05C8: "R", // unassigned, Hebrew block default
		0x05C0: "L", // listed L inside an R default
		0x20B0: "ET", 0x0602: "AN", 0x0700: "L",
	} {
		if g := at(r); g != want {
			t.Errorf("class of U+%04X = %s, want %s", r, g, want)
		}
	}
	for i := 1; i < len(got); i++ {
		if got[i].lo <= got[i-1].hi {
			t.Fatalf("ranges overlap or are unsorted: %v then %v", got[i-1], got[i])
		}
	}
}

func TestClassesRejectUnknownNames(t *testing.T) {
	if _, err := classes([]byte("0041 ; Bogus # x\n")); err == nil {
		t.Error("an unknown class must be an error")
	}
	if _, err := classes([]byte("# @missing: 0000..10FFFF; Nonsense\n")); err == nil {
		t.Error("an unknown @missing class must be an error")
	}
}

func TestBracketsAndMirrors(t *testing.T) {
	br, err := brackets([]byte("0029; 0028; c # RIGHT\n0028; 0029; o # LEFT\n"))
	if err != nil || len(br) != 2 || br[0].r != 0x28 || br[0].kind != 'o' || br[1].pair != 0x28 {
		t.Fatalf("brackets = %+v, %v", br, err)
	}
	if _, err := brackets([]byte("0028; 0029; x\n")); err == nil {
		t.Error("a bad bracket kind must be an error")
	}
	mi, err := mirrors([]byte("0028; 0029 # LEFT PARENTHESIS\n2209; 220C # [BEST FIT] NOT AN ELEMENT\n003C; 003E # LESS-THAN\n"))
	if err != nil || len(mi) != 2 || mi[0].r != 0x28 || mi[1].r != 0x3C {
		t.Fatalf("mirrors = %+v, %v (best-fit pairs must be left out)", mi, err)
	}
}

// Criterion #68: the tables carry the pinned Unicode version, and a committed
// file that does not match what the sources generate fails the check.
func TestGenerateEmbedsTheVersionAndCheckCatchesAStaleFile(t *testing.T) {
	files := map[string][]byte{
		derivedBidiClass: []byte(miniClass),
		bidiBrackets:     []byte("0028; 0029; o\n0029; 0028; c\n"),
		bidiMirroring:    []byte("0028; 0029\n"),
	}
	src, err := generate(files)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Unicode version: " + unicodeVersion, `const UnicodeVersion = "` + unicodeVersion + `"`, baseURL + derivedBidiClass} {
		if !strings.Contains(string(src), want) {
			t.Errorf("generated source lacks %q", want)
		}
	}
	again, _ := generate(files)
	if string(again) != string(src) {
		t.Error("generation is not deterministic")
	}
	out := filepath.Join(t.TempDir(), "tables.go")
	if err := writeOrCheck(out, src, false); err != nil {
		t.Fatal(err)
	}
	if err := writeOrCheck(out, src, true); err != nil {
		t.Errorf("a fresh file failed the check: %v", err)
	}
	if err := os.WriteFile(out, append(src, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeOrCheck(out, src, true); err == nil || !strings.Contains(err.Error(), "out of date") {
		t.Errorf("a stale file passed the check: %v", err)
	}
}

func TestGenerateRejectsEmptyTables(t *testing.T) {
	if _, err := generate(map[string][]byte{derivedBidiClass: []byte("# nothing\n"), bidiBrackets: nil, bidiMirroring: nil}); err == nil {
		t.Error("empty sources must be an error")
	}
}

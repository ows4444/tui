package main

import (
	"bytes"
	"strings"
	"testing"
)

func fakeFiles() map[string][]byte {
	return map[string][]byte{
		graphemeBreak: []byte("# comment\n000D ; CR # x\n000A ; LF\n0300..0302 ; Extend # marks\n0303 ; Extend\n200D ; ZWJ\n"),
		emojiData:     []byte("00A9 ; Extended_Pictographic# E0.6\n1F600..1F601 ; Emoji_Presentation # not pictographic\n1F600..1F601 ; Extended_Pictographic # grin\n"),
		derivedCore:   []byte("# @missing: 0000..10FFFF; InCB; None\n094D ; InCB; Linker # virama\n0915..0916 ; InCB; Consonant\n0300 ; Alphabetic # other property\n"),
	}
}

func TestGenerateIsDeterministicAndMerged(t *testing.T) {
	a, err := generate(fakeFiles())
	if err != nil {
		t.Fatal(err)
	}
	b, err := generate(fakeFiles())
	if err != nil || !bytes.Equal(a, b) {
		t.Fatalf("two runs differ (err %v)", err)
	}
	src := string(a)
	for _, want := range []string{
		"{0x0300, 0x0303, gbExtend}", // 0300..0302 and 0303 merged
		"{0x000A, 0x000A, gbLF}",
		"{0x200D, 0x200D, gbZWJ}",
		"{0x00A9, 0x00A9},",
		"{0x1F600, 0x1F601},",
		"{0x094D, 0x094D, incbLinker}",
		"{0x0915, 0x0916, incbConsonant}",
		"Unicode version: " + unicodeVersion,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("output lacks %q", want)
		}
	}
	if strings.Contains(src, "Alphabetic") {
		t.Error("an unrelated property leaked into the tables")
	}
}

func TestGenerateRejectsUnknownAndOverlappingAndEmpty(t *testing.T) {
	f := fakeFiles()
	f[graphemeBreak] = []byte("0041 ; Brand_New\n")
	if _, err := generate(f); err == nil || !strings.Contains(err.Error(), "Brand_New") {
		t.Errorf("unknown value: %v", err)
	}
	f = fakeFiles()
	f[graphemeBreak] = []byte("0300..0305 ; Extend\n0304 ; ZWJ\n")
	if _, err := generate(f); err == nil || !strings.Contains(err.Error(), "overlap") {
		t.Errorf("overlap: %v", err)
	}
	f = fakeFiles()
	f[derivedCore] = []byte("0300 ; Alphabetic\n")
	if _, err := generate(f); err == nil {
		t.Error("an empty table should fail")
	}
}

func TestParseRange(t *testing.T) {
	if lo, hi, err := parseRange("0041..005A"); err != nil || lo != 0x41 || hi != 0x5A {
		t.Errorf("got %x %x %v", lo, hi, err)
	}
	for _, bad := range []string{"zz", "0041..zz", "005A..0041", "110000"} {
		if _, _, err := parseRange(bad); err == nil {
			t.Errorf("%q should be rejected", bad)
		}
	}
}

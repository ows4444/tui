package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestOutlineFindsATXHeadingsWithTheirLines(t *testing.T) {
	hs := outline("# One\ntext\n## Two ##\n\n###   Three  \n####### seven\n#nospace\n")
	want := []heading{{1, "One", 0}, {2, "Two", 2}, {3, "Three", 4}}
	if !reflect.DeepEqual(hs, want) {
		t.Fatalf("outline = %+v", hs)
	}
}

func TestOutlineSkipsFencedCode(t *testing.T) {
	md := "# Real\n```sh\n# comment, not a heading\n```\n~~~\n# also not\n~~~\n## After\n"
	hs := outline(md)
	if len(hs) != 2 || hs[1].text != "After" || hs[1].line != 7 {
		t.Fatalf("outline = %+v", hs)
	}
	// An unclosed fence hides the rest of the file, as in CommonMark.
	if hs := outline("# A\n```\n# B\n"); len(hs) != 1 {
		t.Fatalf("outline = %+v", hs)
	}
}

func TestOutlineIgnoresEmptyHeadings(t *testing.T) {
	if hs := outline("# \n#\n## x\n"); len(hs) != 1 || hs[0].text != "x" {
		t.Fatalf("outline = %+v", hs)
	}
}

func TestCurrentHeading(t *testing.T) {
	hs := outline("intro\n# A\nx\n## B\ny\n")
	for line, want := range map[int]int{0: -1, 1: 0, 2: 0, 3: 1, 4: 1, 99: 1} {
		if got := current(hs, line); got != want {
			t.Errorf("current(line %d) = %d, want %d", line, got, want)
		}
	}
	if current(nil, 3) != -1 {
		t.Error("no headings should give -1")
	}
}

func TestPlainAndWords(t *testing.T) {
	if plain("**Bold** and `code` and _it_") != "Bold and code and it" {
		t.Errorf("plain = %q", plain("**Bold** and `code` and _it_"))
	}
	if words("one two\n three\t four") != 4 || words("") != 0 {
		t.Error("words")
	}
}

func TestHighlightKeepsTheTextWhole(t *testing.T) {
	for _, line := range []string{"# Heading", "plain `code` and `more`", "```go", "> quote", "unclosed `tick", "", "  - item"} {
		var b strings.Builder
		for _, sp := range highlightLine(line) {
			b.WriteString(sp.Text)
		}
		if b.String() != line {
			t.Errorf("highlightLine(%q) text = %q", line, b.String())
		}
	}
}

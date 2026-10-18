package main

import (
	"regexp"
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// heading is one "#" heading of the source, with the line it is on.
type heading struct {
	level int
	text  string
	line  int // 0-based line of the source
}

var atx = regexp.MustCompile(`^(#{1,6})[ \t]+(.*?)[ \t#]*$`)

// outline lists the ATX headings of md in order, skipping lines inside fenced
// code blocks. Setext headings (underlined) are not listed.
func outline(md string) []heading {
	var out []heading
	fence := ""
	for i, line := range strings.Split(md, "\n") {
		t := strings.TrimSpace(line)
		switch {
		case fence != "":
			if strings.HasPrefix(t, fence) {
				fence = ""
			}
			continue
		case strings.HasPrefix(t, "```"):
			fence = "```"
			continue
		case strings.HasPrefix(t, "~~~"):
			fence = "~~~"
			continue
		}
		if m := atx.FindStringSubmatch(line); m != nil && m[2] != "" {
			out = append(out, heading{level: len(m[1]), text: m[2], line: i})
		}
	}
	return out
}

// current is the index of the last heading at or above line, or -1.
func current(hs []heading, line int) int {
	at := -1
	for i, h := range hs {
		if h.line <= line {
			at = i
		}
	}
	return at
}

// words counts the words of md.
func words(md string) int { return len(strings.Fields(md)) }

// highlighter returns the editor's Highlighter for a theme.
func highlighter(th theme.Theme) func(string) ansi.Text {
	return func(line string) ansi.Text { return highlightLine(th, line) }
}

// highlightLine colours the markup of one source line: headings, list markers,
// fences, and `code`. The text is unchanged, so the editor's columns hold.
func highlightLine(th theme.Theme, line string) ansi.Text {
	head := ansi.NewStyle().Bold().Foreground(th.Primary)
	code := ansi.NewStyle().Foreground(th.Warning)
	faint := ansi.NewStyle().Faint()
	t := strings.TrimLeft(line, " ")
	switch {
	case atx.MatchString(line):
		return ansi.Text{{Text: line, Style: head}}
	case strings.HasPrefix(t, "```"), strings.HasPrefix(t, "~~~"):
		return ansi.Text{{Text: line, Style: faint}}
	case strings.HasPrefix(t, ">"):
		return ansi.Text{{Text: line, Style: faint}}
	}
	var out ansi.Text
	rest := line
	for rest != "" {
		i := strings.IndexByte(rest, '`')
		if i < 0 {
			break
		}
		j := strings.IndexByte(rest[i+1:], '`')
		if j < 0 {
			break
		}
		out = append(out, ansi.Span{Text: rest[:i]}, ansi.Span{Text: rest[i : i+j+2], Style: code})
		rest = rest[i+j+2:]
	}
	return append(out, ansi.Span{Text: rest})
}

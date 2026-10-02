package highlight

import (
	"strings"
	"testing"
	"time"
)

// Long tokens stay linear: a 1 MB declaration value or attribute must not make
// a lexer rescan its input per byte.
func TestLongInputsStayLinear(t *testing.T) {
	big := strings.Repeat("a", 1<<20)
	cases := map[string]string{
		"css":      "a { background: url(data:" + big + "); }",
		"html":     `<img src="` + big + `">` + big,
		"makefile": "X := " + big,
		"ini":      "k = " + big,
		"ruby":     "x = " + big,
		"lua":      "x = " + big,
		"php":      "<?php $x = " + big,
	}
	for lang, in := range cases {
		start := time.Now()
		Lines(lang, in)
		if el := time.Since(start); el > 2*time.Second {
			t.Errorf("%s: %v for 1 MB, want linear time", lang, el)
		}
	}
}

// Every lexer stays linear on a long run of punctuation, which most of them
// add one byte at a time (the emitter merges the bytes into one span).
func TestPunctuationRunsStayLinear(t *testing.T) {
	in := "x = " + strings.Repeat("-+", 1<<18)
	for name, lang := range goldenLangs {
		start := time.Now()
		Lines(lang, in)
		if el := time.Since(start); el > time.Second {
			t.Errorf("%s: %v for 512 KB of punctuation, want linear time", name, el)
		}
	}
}

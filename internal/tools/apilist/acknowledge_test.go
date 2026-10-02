package main

import (
	"strings"
	"testing"
)

func TestChangedIdentifier(t *testing.T) {
	for line, want := range map[string]string{
		"m/a (package a): func Remove()":                   "Remove",
		"m/a (package a): func (s Style) Render(t string)": "Render",
		"m/a (package a): func (s *Style) Link(u string)":  "Link",
		"m/a (package a): func Map[K comparable](k K)":     "Map",
		"m/a (package a): type Model struct {":             "Model",
		"m/a (package a): type Key = input.Key":            "Key",
		"m/a (package a): var ErrX = errors.New(\"x\")":    "ErrX",
		"m/a (package a): const Max = 3":                   "Max",
		"m/a (package a): \tKeyUp\t\t\t= input.KeyUp":      "KeyUp",
		"m/a (package a): \tRunes []rune":                  "Runes",
	} {
		if got := changedIdentifier(line); got != want {
			t.Errorf("changedIdentifier(%q) = %q, want %q", line, got, want)
		}
	}
}

const changelog = `# Changelog

## [Unreleased]

### Removed

- BREAKING: ` + "`a.Remove`" + ` is gone; use ` + "`a.Keep`" + `.
- BREAKING: ` + "`Style.Change`" + ` now takes two
  arguments, ` + "`x`" + ` and ` + "`y`" + `; the old form
  is removed.
- A non-breaking note that mentions Quiet.

## [0.1.0]

- BREAKING: ` + "`Old`" + ` was removed in 0.1.0.
`

func TestUnacknowledgedNamesWhatTheChangelogDoesNotMention(t *testing.T) {
	changes := []string{
		"m/a (package a): func Remove()",
		"m/a (package a): func (s Style) Change(x int) string",
		"m/a (package a): func Quiet()", // only a non-BREAKING bullet names it
		"m/a (package a): func Old()",   // only an already-released section names it
		"m/a (package a): func Gone()",
	}
	got := unacknowledged(changes, changelog)
	want := []string{
		"m/a (package a): func Quiet()",
		"m/a (package a): func Old()",
		"m/a (package a): func Gone()",
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestUnacknowledgedMatchesWholeWordsOnly(t *testing.T) {
	log := "## [Unreleased]\n\n- BREAKING: `RemoveAll` is gone.\n"
	if got := unacknowledged([]string{"m/a (package a): func Remove()"}, log); len(got) != 1 {
		t.Errorf("Remove was acknowledged by RemoveAll: %q", got)
	}
}

func TestUnacknowledgedWithNoChangesOrNoChangelog(t *testing.T) {
	if got := unacknowledged(nil, changelog); len(got) != 0 {
		t.Errorf("got %q", got)
	}
	if got := unacknowledged([]string{"m/a (package a): func X()"}, ""); len(got) != 1 {
		t.Errorf("an empty changelog acknowledges nothing, got %q", got)
	}
}

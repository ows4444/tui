package main

import (
	"regexp"
	"strings"
)

// changedIdentifier returns the name a breaking-change line from breaking is
// about: the function, method, type, variable or constant being declared, or
// the first name on a line inside a const, var or struct block. Packages are
// not part of the name; a CHANGELOG entry may write `pkg.Name` or `Name`.
func changedIdentifier(change string) string {
	_, decl, ok := strings.Cut(change, "): ")
	if !ok {
		decl = change
	}
	d := strings.TrimLeft(decl, "\t ")
	switch {
	case strings.HasPrefix(d, "func ("):
		if _, rest, ok := strings.Cut(d, ") "); ok {
			return leadingName(rest)
		}
	case strings.HasPrefix(d, "func "):
		return leadingName(d[len("func "):])
	case strings.HasPrefix(d, "type "):
		return leadingName(d[len("type "):])
	case strings.HasPrefix(d, "var "):
		return leadingName(d[len("var "):])
	case strings.HasPrefix(d, "const "):
		return leadingName(d[len("const "):])
	}
	return leadingName(d)
}

// leadingName returns the Go identifier at the start of s.
func leadingName(s string) string {
	i := 0
	for i < len(s) {
		c := s[i]
		if c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || i > 0 && c >= '0' && c <= '9' {
			i++
			continue
		}
		break
	}
	return s[:i]
}

// breakingEntries returns the text of every "- BREAKING" bullet in the first
// (unreleased) section of a changelog: from the bullet's line up to the next
// bullet, heading or blank line, so its indented continuation lines count.
func breakingEntries(changelog string) []string {
	var out []string
	var cur []string
	flush := func() {
		if cur != nil {
			out = append(out, strings.Join(cur, " "))
			cur = nil
		}
	}
	sections := 0
	for _, l := range strings.Split(changelog, "\n") {
		switch {
		case strings.HasPrefix(l, "## "):
			flush()
			if sections++; sections > 1 {
				return out
			}
		case strings.HasPrefix(l, "- BREAKING"):
			flush()
			cur = []string{l}
		case cur != nil && (strings.HasPrefix(l, "  ") || strings.HasPrefix(l, "\t")):
			cur = append(cur, strings.TrimSpace(l))
		default:
			flush()
		}
	}
	flush()
	return out
}

// unacknowledged returns the changes whose identifier no BREAKING entry of the
// changelog's unreleased section names as a whole word.
func unacknowledged(changes []string, changelog string) []string {
	entries := breakingEntries(changelog)
	var out []string
	for _, c := range changes {
		name := changedIdentifier(c)
		if name == "" || !named(entries, name) {
			out = append(out, c)
		}
	}
	return out
}

func named(entries []string, name string) bool {
	re := regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\b`)
	for _, e := range entries {
		if re.MatchString(e) {
			return true
		}
	}
	return false
}

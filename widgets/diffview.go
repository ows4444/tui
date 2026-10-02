package widgets

import (
	"strconv"
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// diffKind classifies one line of a unified diff.
type diffKind int

const (
	diffContext diffKind = iota
	diffAdd
	diffRemove
	diffHunk // "@@ -a,b +c,d @@"
	diffMeta // file headers, index/mode/rename lines, "\ No newline ..."
)

// DiffView renders unified-diff text (as produced by `git diff` or
// `diff -u`) in the same bordered box as CodeBlock, colouring each line by
// kind: added lines in t.Success, removed in t.Error, context in t.Text,
// hunk headers in t.Info, and file/metadata lines in t.Muted. The leading
// +/-/space marker stays visible. Framing, truncation with "…", tab and
// newline handling, and the width rules are exactly CodeBlock's, so every
// line is width columns and the row count is the diff's line count plus
// two.
//
// Inside a hunk, lines are classified by the counts in its @@ header, so a
// removed line whose content begins "--" is not mistaken for a file header.
// Outside any hunk, a line starting with a single + or - counts as added
// or removed, so header-less snippets colour correctly. There is no line
// numbering and no intra-line highlighting in v1.
func DiffView(diff string, width int, t theme.Theme) string {
	lines := codeLines(diff)
	kinds := classifyDiff(lines)
	colors := make([]ansi.Color, len(lines))
	for i, k := range kinds {
		switch k {
		case diffAdd:
			colors[i] = t.Success
		case diffRemove:
			colors[i] = t.Error
		case diffHunk:
			colors[i] = t.Info
		case diffMeta:
			colors[i] = t.Muted
		default:
			colors[i] = t.Text
		}
	}
	return renderCode(lines, colors, width, false, t)
}

// diffMetaPrefixes start git's per-file header lines that are neither
// hunk content nor a ---/+++ file marker.
var diffMetaPrefixes = []string{
	"diff ", "index ", "new file", "deleted file", "old mode", "new mode",
	"similarity ", "dissimilarity ", "rename ", "copy ", "Binary files",
}

func classifyDiff(lines []string) []diffKind {
	kinds := make([]diffKind, len(lines))
	oldLeft, newLeft := 0, 0 // lines still owed by the current hunk
	for i, l := range lines {
		if oldLeft > 0 || newLeft > 0 {
			if k, ok := hunkLineKind(l); ok {
				kinds[i] = k
				switch k {
				case diffAdd:
					newLeft--
				case diffRemove:
					oldLeft--
				case diffContext:
					oldLeft--
					newLeft--
				}
				continue
			}
			oldLeft, newLeft = 0, 0 // malformed hunk: resync outside it
		}
		switch {
		case strings.HasPrefix(l, "@@"):
			kinds[i] = diffHunk
			oldLeft, newLeft = parseHunkCounts(l)
		case strings.HasPrefix(l, "+++"), strings.HasPrefix(l, "---"), strings.HasPrefix(l, `\`):
			kinds[i] = diffMeta
		case strings.HasPrefix(l, "+"):
			kinds[i] = diffAdd
		case strings.HasPrefix(l, "-"):
			kinds[i] = diffRemove
		case hasAnyPrefix(l, diffMetaPrefixes):
			kinds[i] = diffMeta
		default:
			kinds[i] = diffContext
		}
	}
	return kinds
}

// hunkLineKind classifies a line inside a hunk by its marker. An empty
// line counts as context (editors often strip the marker's trailing
// space); a "\ No newline" marker is metadata and consumes no count.
func hunkLineKind(l string) (diffKind, bool) {
	switch {
	case l == "" || l[0] == ' ':
		return diffContext, true
	case l[0] == '+':
		return diffAdd, true
	case l[0] == '-':
		return diffRemove, true
	case l[0] == '\\':
		return diffMeta, true
	}
	return 0, false
}

// parseHunkCounts reads the old and new line counts from an
// "@@ -a[,b] +c[,d] @@ ..." header; an omitted count means 1. It returns
// zeros for a header it can't parse, leaving the hunk empty.
func parseHunkCounts(h string) (oldN, newN int) {
	f := strings.Fields(h)
	if len(f) < 3 || f[0] != "@@" || !strings.HasPrefix(f[1], "-") || !strings.HasPrefix(f[2], "+") {
		return 0, 0
	}
	count := func(s string) int {
		_, n, found := strings.Cut(s[1:], ",")
		if !found {
			return 1
		}
		v, err := strconv.Atoi(n)
		if err != nil || v < 0 {
			return 0
		}
		return v
	}
	return count(f[1]), count(f[2])
}

func hasAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

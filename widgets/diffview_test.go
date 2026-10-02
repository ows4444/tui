package widgets

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

const sampleDiff = `diff --git a/a.go b/a.go
index 111..222 100644
--- a/a.go
+++ b/a.go
@@ -1,3 +1,3 @@
 keep
-old line
+new line
 tail
\ No newline at end of file`

func coloured(c ansi.Color, s string) string {
	return ansi.NewStyle().Foreground(c).Render(s)
}

func TestDiffViewZeroWidth(t *testing.T) {
	for _, w := range []int{0, -1} {
		if got := DiffView(sampleDiff, w, theme.DarkTheme()); got != "" {
			t.Errorf("DiffView width %d = %q, want empty", w, got)
		}
	}
}

func TestDiffViewFramingMatchesCodeBlock(t *testing.T) {
	th := theme.DarkTheme()
	for _, w := range []int{1, 3, 4, 5, 12, 40, 80} {
		out := DiffView(sampleDiff, w, th)
		lines := strings.Split(out, "\n")
		n := len(strings.Split(sampleDiff, "\n"))
		if w >= 5 {
			if len(lines) != n+2 {
				t.Errorf("w=%d: %d lines, want %d", w, len(lines), n+2)
			}
			for i, l := range lines {
				if got := ansi.Width(l); got != w {
					t.Errorf("w=%d line %d: Width = %d", w, i, got)
				}
			}
		} else {
			if len(lines) != n {
				t.Errorf("w=%d: %d lines, want %d (borderless)", w, len(lines), n)
			}
			for i, l := range lines {
				if ansi.Width(l) > w {
					t.Errorf("w=%d line %d wider than width: %q", w, i, l)
				}
			}
		}
	}

	// Same frame as CodeBlock over the same visible text: borders identical.
	d := strings.Split(DiffView("+a", 20, th), "\n")
	c := strings.Split(CodeBlock("+a", 20, false, th), "\n")
	if d[0] != c[0] || d[len(d)-1] != c[len(c)-1] {
		t.Errorf("DiffView frame differs from CodeBlock's")
	}

	// Truncation with an ellipsis.
	long := "+" + strings.Repeat("x", 100)
	if p := plainLines(DiffView(long, 12, th)); p[1] != "│ +xxxxxx… │" {
		t.Errorf("long added line = %q", p[1])
	}
}

func TestDiffViewColoursByLineKind(t *testing.T) {
	th := theme.DarkTheme()
	out := DiffView(sampleDiff, 60, th)
	want := []struct {
		c    ansi.Color
		text string
	}{
		{th.Muted, "diff --git a/a.go b/a.go"},
		{th.Muted, "index 111..222 100644"},
		{th.Muted, "--- a/a.go"},
		{th.Muted, "+++ b/a.go"},
		{th.Info, "@@ -1,3 +1,3 @@"},
		{th.Text, " keep"},
		{th.Error, "-old line"},
		{th.Success, "+new line"},
		{th.Text, " tail"},
		{th.Muted, `\ No newline at end of file`},
	}
	for _, w := range want {
		if !strings.Contains(out, coloured(w.c, w.text)) {
			t.Errorf("%q not rendered in colour %q", w.text, w.c)
		}
	}
}

func TestDiffViewMetaLines(t *testing.T) {
	th := theme.DarkTheme()
	diff := "diff --git a/x b/y\nold mode 100644\nnew mode 100755\nsimilarity index 90%\nrename from x\nrename to y\nnew file mode 100644\ndeleted file mode 100644\nBinary files a and b differ"
	out := DiffView(diff, 60, th)
	for _, l := range strings.Split(diff, "\n") {
		if !strings.Contains(out, coloured(th.Muted, l)) {
			t.Errorf("meta line %q not Muted", l)
		}
	}
}

func TestDiffViewMarkerLookalikesInsideHunk(t *testing.T) {
	th := theme.DarkTheme()
	// The removed line's content is "-- comment" and the added one's is
	// "++x": inside a hunk they must be remove/add, not file headers.
	diff := "@@ -1,2 +1,2 @@\n--- comment\n+++x\n context"
	out := DiffView(diff, 40, th)
	if !strings.Contains(out, coloured(th.Error, "--- comment")) {
		t.Errorf("'--- comment' inside a hunk not rendered as removed")
	}
	if !strings.Contains(out, coloured(th.Success, "+++x")) {
		t.Errorf("'+++x' inside a hunk not rendered as added")
	}

	// After the hunk's counts are used up, ---/+++ are headers again.
	two := "@@ -1 +1 @@\n-a\n+b\n--- a/next\n+++ b/next\n@@ -1 +1 @@\n-c\n+d"
	out = DiffView(two, 40, th)
	if !strings.Contains(out, coloured(th.Muted, "--- a/next")) || !strings.Contains(out, coloured(th.Muted, "+++ b/next")) {
		t.Errorf("headers between hunks not Muted")
	}
}

func TestDiffViewHeaderlessSnippets(t *testing.T) {
	th := theme.DarkTheme()
	out := DiffView("-gone\n+added\n same", 30, th)
	for c, s := range map[ansi.Color]string{th.Error: "-gone", th.Success: "+added", th.Text: " same"} {
		if !strings.Contains(out, coloured(c, s)) {
			t.Errorf("%q not in colour %q", s, c)
		}
	}
	// +++ / --- outside a hunk are headers, not add/remove.
	out = DiffView("--- a\n+++ b", 30, th)
	if !strings.Contains(out, coloured(th.Muted, "--- a")) || !strings.Contains(out, coloured(th.Muted, "+++ b")) {
		t.Errorf("---/+++ outside a hunk not Muted")
	}
}

func TestDiffViewHunkHeaderCountsDefaultToOne(t *testing.T) {
	th := theme.DarkTheme()
	out := DiffView("@@ -5 +5 @@ func f()\n-a\n+b\n--- x", 40, th)
	// Hunk is 1 old + 1 new line, so the trailing "--- x" is a header.
	if !strings.Contains(out, coloured(th.Info, "@@ -5 +5 @@ func f()")) {
		t.Errorf("hunk header with section text not Info")
	}
	if !strings.Contains(out, coloured(th.Muted, "--- x")) {
		t.Errorf("'--- x' after a one-line hunk should be a header")
	}
}

func TestDiffViewNormalisesLikeCodeBlock(t *testing.T) {
	th := theme.DarkTheme()
	out := plainLines(DiffView("+\tx\r\n context\r\n", 20, th))
	if len(out) != 4 {
		t.Fatalf("got %d lines, want 4: %q", len(out), out)
	}
	row := func(body string) string { return "│ " + body + strings.Repeat(" ", 16-len([]rune(body))) + " │" }
	if want := row("+    x"); out[1] != want {
		t.Errorf("tab row = %q, want %q", out[1], want)
	}
	if want := row(" context"); out[2] != want {
		t.Errorf("crlf row = %q, want %q", out[2], want)
	}
	if e := plainLines(DiffView("", 12, th)); len(e) != 3 {
		t.Errorf("empty diff: %d lines, want 3", len(e))
	}

	// Visible text equals the input.
	var got []string
	for _, l := range plainLines(DiffView(sampleDiff, 60, th))[1:11] {
		got = append(got, strings.TrimRight(strings.TrimSuffix(strings.TrimPrefix(l, "│ "), " │"), " "))
	}
	if strings.Join(got, "\n") != sampleDiff {
		t.Errorf("visible text differs from input:\n%s", strings.Join(got, "\n"))
	}
}

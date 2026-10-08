package widgets

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

func TestEmptyCentresItsLines(t *testing.T) {
	th := theme.DarkTheme()
	got := ansi.StripANSI(Empty("No results", "Try another search.", "[n] new", th, 30))
	want := strings.Join([]string{
		"          No results          ",
		"                              ",
		"     Try another search.      ",
		"           [n] new            ",
	}, "\n")
	if got != want {
		t.Fatalf("Empty =\n%s\nwant\n%s", got, want)
	}
}

func TestEmptyLeavesOutWhatIsMissing(t *testing.T) {
	th := theme.DarkTheme()
	if got := ansi.StripANSI(Empty("Nothing here", "", "", th, 0)); got != "Nothing here" {
		t.Errorf("title only = %q", got)
	}
	if got := ansi.StripANSI(Empty("", "No files", "", th, 0)); got != "No files" {
		t.Errorf("description only = %q", got)
	}
	if got := Empty("", "", "", th, 20); got != "" {
		t.Errorf("all empty = %q", got)
	}
	// With no width the lines centre on the widest.
	got := ansi.StripANSI(Empty("Hi", "", "a longer hint", th, 0))
	if want := "     Hi      \n             \na longer hint"; got != want {
		t.Errorf("no width =\n%q\nwant\n%q", got, want)
	}
}

func TestEmptyWrapsToItsWidthAndSanitises(t *testing.T) {
	th := theme.DarkTheme()
	out := Empty("\x1b[2JTitle", "one two three four five six seven", "", th, 12)
	if strings.Contains(out, "\x1b[2J") {
		t.Fatal("an escape sequence in the title reached the output")
	}
	for i, l := range strings.Split(ansi.StripANSI(out), "\n") {
		if ansi.Width(l) != 12 {
			t.Errorf("line %d is %d cells wide, want 12: %q", i, ansi.Width(l), l)
		}
	}
	// A word longer than the width is cut, not allowed to overflow.
	for _, l := range strings.Split(ansi.StripANSI(Empty("Supercalifragilistic", "", "", th, 8)), "\n") {
		if ansi.Width(l) > 8 {
			t.Errorf("a long word overflowed the width: %q", l)
		}
	}
}

func timelineItems() []TimelineItem {
	return []TimelineItem{
		{Title: "Build passed", Time: "09:40", Detail: "122 packages", Status: TimelineDone},
		{Title: "Deploying", Time: "10:02", Status: TimelineCurrent},
		{Title: "Tests failed", Time: "10:05", Detail: "two on windows\nsee the log", Status: TimelineFailed},
		{Title: "Verify", Status: TimelinePending, Detail: "after the deploy"},
	}
}

func TestTimelineMarksEachStatusWithItsOwnGlyph(t *testing.T) {
	got := ansi.StripANSI(Timeline(timelineItems(), theme.DarkTheme(), 0))
	want := strings.Join([]string{
		"✓ 09:40  Build passed",
		"│        122 packages",
		"● 10:02  Deploying",
		"✗ 10:05  Tests failed",
		"│        two on windows",
		"│        see the log",
		"○        Verify",
		"         after the deploy",
	}, "\n")
	if got != want {
		t.Fatalf("Timeline =\n%s\nwant\n%s", got, want)
	}
}

func TestTimelineWithoutTimesHasNoTimeColumn(t *testing.T) {
	items := []TimelineItem{{Title: "One", Status: TimelineDone, Detail: "d"}, {Title: "Two", Status: TimelinePending}}
	got := ansi.StripANSI(Timeline(items, theme.DarkTheme(), 0))
	if want := "✓ One\n│ d\n○ Two"; got != want {
		t.Fatalf("Timeline =\n%s\nwant\n%s", got, want)
	}
	if got := Timeline(nil, theme.DarkTheme(), 40); got != "" {
		t.Errorf("no items = %q", got)
	}
}

func TestTimelineFitsItsWidthAndSanitises(t *testing.T) {
	items := []TimelineItem{
		{Title: "A title that is much too long for the room", Time: "09:40", Detail: "detail text that has to wrap onto more lines", Status: TimelineDone},
		{Title: "\x1b]52;c;x\x07Next", Time: "\x1b[2J10:00", Detail: "\x1b[2Jd", Status: TimelinePending},
	}
	out := Timeline(items, theme.DarkTheme(), 24)
	if strings.Contains(out, "\x1b[2J") || strings.Contains(out, "\x1b]52") {
		t.Fatal("an escape sequence in an item reached the output")
	}
	for i, l := range strings.Split(ansi.StripANSI(out), "\n") {
		if ansi.Width(l) > 24 {
			t.Errorf("line %d is %d cells wide, over 24: %q", i, ansi.Width(l), l)
		}
	}
	// A width smaller than the indent still leaves one cell for the text.
	if narrow := Timeline(items, theme.DarkTheme(), 3); narrow == "" {
		t.Error("a very narrow timeline is empty")
	}
}

func TestTimelineFollowsTheGlyphSet(t *testing.T) {
	th := theme.DarkTheme()
	th.Glyphs = theme.ASCIIGlyphSet()
	got := ansi.StripANSI(Timeline(timelineItems()[:2], th, 0))
	if want := "v 09:40  Build passed\n|        122 packages\n* 10:02  Deploying"; got != want {
		t.Fatalf("ASCII =\n%s\nwant\n%s", got, want)
	}
}

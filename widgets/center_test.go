package widgets

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
)

func TestCenterHorizontalPaddingEven(t *testing.T) {
	// "ab" is 2 wide; width 6 leaves 4 to split 2/2.
	got := Center("ab", 6, 1)
	want := "  ab  "
	if got != want {
		t.Errorf("Center(%q, 6, 1) = %q, want %q", "ab", got, want)
	}
	if w := ansi.Width(got); w != 6 {
		t.Errorf("Width(Center(...)) = %d, want 6", w)
	}
}

func TestCenterHorizontalPaddingOddExtraOnRight(t *testing.T) {
	// "ab" is 2 wide; width 7 leaves 5 to split 2/3 (extra column right).
	got := Center("ab", 7, 1)
	want := "  ab   "
	if got != want {
		t.Errorf("Center(%q, 7, 1) = %q, want %q", "ab", got, want)
	}
}

func TestCenterVerticalPaddingEven(t *testing.T) {
	got := Center("x", 1, 5)
	want := strings.Join([]string{" ", " ", "x", " ", " "}, "\n")
	if got != want {
		t.Errorf("Center(%q, 1, 5) = %q, want %q", "x", got, want)
	}
}

func TestCenterVerticalPaddingOddExtraOnBottom(t *testing.T) {
	got := Center("x", 1, 4)
	want := strings.Join([]string{" ", "x", " ", " "}, "\n")
	if got != want {
		t.Errorf("Center(%q, 1, 4) = %q, want %q", "x", got, want)
	}
}

func TestCenterLeavesAxisUnchangedWhenNotLarger(t *testing.T) {
	content := "hello"
	if got := Center(content, 5, 1); got != content {
		t.Errorf("Center(%q, 5, 1) = %q, want unchanged %q", content, got, content)
	}
	if got := Center(content, 3, 1); got != content {
		t.Errorf("Center(%q, 3, 1) = %q, want unchanged (width < content) %q", content, got, content)
	}

	multi := "ab\ncd"
	if got := Center(multi, 2, 2); got != multi {
		t.Errorf("Center(%q, 2, 2) = %q, want unchanged %q", multi, got, multi)
	}
	if got := Center(multi, 2, 1); got != multi {
		t.Errorf("Center(%q, 2, 1) = %q, want unchanged (height < content) %q", multi, got, multi)
	}
}

func TestCenterDoesNotPanicOrTruncate(t *testing.T) {
	content := "wide content here"
	got := Center(content, 1, 1)
	if !strings.Contains(got, content) {
		t.Errorf("Center(%q, 1, 1) = %q, want content preserved (not truncated)", content, got)
	}
}

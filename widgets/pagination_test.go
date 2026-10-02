package widgets

import (
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// TestPaginationRendersPageXOfY proves criterion #398: current and total
// render as "Page X of Y" (1-indexed current).
func TestPaginationRendersPageXOfY(t *testing.T) {
	cases := []struct {
		current, total int
		want           string
	}{
		{1, 1, "Page 1 of 1"},
		{3, 12, "Page 3 of 12"},
		{12, 12, "Page 12 of 12"},
	}
	for _, c := range cases {
		if got := Pagination(c.current, c.total, theme.DarkTheme()); got != c.want {
			t.Errorf("Pagination(%d, %d) = %q, want %q", c.current, c.total, got, c.want)
		}
	}
}

// TestPaginationClampsOutOfRangeCurrent proves criterion #399: current
// outside [1, total] is clamped into that range rather than rendering a
// nonsensical result.
func TestPaginationClampsOutOfRangeCurrent(t *testing.T) {
	cases := []struct {
		current, total int
		want           string
	}{
		{0, 5, "Page 1 of 5"},
		{-3, 5, "Page 1 of 5"},
		{99, 5, "Page 5 of 5"},
	}
	for _, c := range cases {
		if got := Pagination(c.current, c.total, theme.DarkTheme()); got != c.want {
			t.Errorf("Pagination(%d, %d) = %q, want %q", c.current, c.total, got, c.want)
		}
	}
}

// TestPaginationNonPositiveTotal proves criterion #400: total <= 0 renders
// an empty string without panicking.
func TestPaginationNonPositiveTotal(t *testing.T) {
	for _, total := range []int{0, -1, -5} {
		if got := Pagination(1, total, theme.DarkTheme()); got != "" {
			t.Errorf("Pagination(1, %d) = %q, want empty", total, got)
		}
	}
}

// TestPaginationDotsStylesCurrentDistinctly proves criterion #401:
// PaginationDots renders total dot characters, one per page, styling the
// current page's dot with Primary and the rest with Muted, mirroring
// ProgressBar's filled/track color convention.
func TestPaginationDotsStylesCurrentDistinctly(t *testing.T) {
	dt := theme.DarkTheme()
	filled := ansi.NewStyle().Foreground(dt.Primary)
	track := ansi.NewStyle().Foreground(dt.Muted)

	got := PaginationDots(2, 4, dt)
	want := track.Render("○") + filled.Render("●") + track.Render("○") + track.Render("○")
	if got != want {
		t.Errorf("PaginationDots(2, 4) = %q, want %q", got, want)
	}
}

// TestPaginationDotsClampsOutOfRangeCurrent proves current is clamped into
// [1, total] the same way Pagination is.
func TestPaginationDotsClampsOutOfRangeCurrent(t *testing.T) {
	dt := theme.DarkTheme()
	filled := ansi.NewStyle().Foreground(dt.Primary)
	track := ansi.NewStyle().Foreground(dt.Muted)

	got := PaginationDots(0, 3, dt)
	want := filled.Render("●") + track.Render("○") + track.Render("○")
	if got != want {
		t.Errorf("PaginationDots(0, 3) = %q, want %q", got, want)
	}

	got = PaginationDots(99, 3, dt)
	want = track.Render("○") + track.Render("○") + filled.Render("●")
	if got != want {
		t.Errorf("PaginationDots(99, 3) = %q, want %q", got, want)
	}
}

// TestPaginationDotsNonPositiveTotal proves criterion #402: total <= 0
// renders an empty string without panicking.
func TestPaginationDotsNonPositiveTotal(t *testing.T) {
	for _, total := range []int{0, -1, -5} {
		if got := PaginationDots(1, total, theme.DarkTheme()); got != "" {
			t.Errorf("PaginationDots(1, %d) = %q, want empty", total, got)
		}
	}
}

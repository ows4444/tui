package widgets

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// TestBreadcrumbJoinsWithSeparator proves criterion #393: several items are
// joined with a separator between each, e.g. "a / b / c".
func TestBreadcrumbJoinsWithSeparator(t *testing.T) {
	dt := theme.DarkTheme()
	item := ansi.NewStyle().Foreground(dt.Text)
	current := ansi.NewStyle().Bold().Foreground(dt.Primary)
	sep := ansi.NewStyle().Foreground(dt.Muted).Render(" / ")

	got := Breadcrumb([]string{"a", "b", "c"}, dt)
	want := item.Render("a") + sep + item.Render("b") + sep + current.Render("c")
	if got != want {
		t.Errorf("Breadcrumb([a b c]) = %q, want %q", got, want)
	}
}

// TestBreadcrumbSeparatorStyledDistinctly proves criterion #394: the
// separator is styled distinctly (muted) from the item text.
func TestBreadcrumbSeparatorStyledDistinctly(t *testing.T) {
	dt := theme.DarkTheme()
	got := Breadcrumb([]string{"a", "b"}, dt)

	mutedSep := ansi.NewStyle().Foreground(dt.Muted).Render(" / ")
	if !contains(got, mutedSep) {
		t.Errorf("Breadcrumb output %q does not contain muted-styled separator %q", got, mutedSep)
	}

	itemStyle := ansi.NewStyle().Foreground(dt.Text)
	plainSep := ansi.NewStyle().Foreground(dt.Text).Render(" / ")
	if plainSep == mutedSep {
		t.Fatal("test setup invalid: muted and text-styled separators render identically")
	}
	_ = itemStyle
}

// TestBreadcrumbLastItemStyledDistinctly proves criterion #395: the last
// item is styled distinctly (bold, Primary) from earlier items, marking
// current location.
func TestBreadcrumbLastItemStyledDistinctly(t *testing.T) {
	dt := theme.DarkTheme()
	item := ansi.NewStyle().Foreground(dt.Text)
	current := ansi.NewStyle().Bold().Foreground(dt.Primary)

	got := Breadcrumb([]string{"a", "b", "c"}, dt)

	plainC := item.Render("c")
	styledC := current.Render("c")
	if plainC == styledC {
		t.Fatal("test setup invalid: plain and current styles render identically")
	}
	if !contains(got, styledC) {
		t.Errorf("Breadcrumb output %q does not style last item distinctly: want to contain %q", got, styledC)
	}
	if contains(got, plainC) {
		t.Errorf("Breadcrumb output %q contains last item styled like an earlier item %q", got, plainC)
	}
}

// TestBreadcrumbSingleItem proves criterion #396: a single item renders
// with no separator.
func TestBreadcrumbSingleItem(t *testing.T) {
	dt := theme.DarkTheme()
	current := ansi.NewStyle().Bold().Foreground(dt.Primary)

	got := Breadcrumb([]string{"home"}, dt)
	want := current.Render("home")
	if got != want {
		t.Errorf("Breadcrumb([home]) = %q, want %q", got, want)
	}
}

// TestBreadcrumbEmpty proves criterion #397: an empty items slice renders
// an empty string without panicking.
func TestBreadcrumbEmpty(t *testing.T) {
	if got := Breadcrumb(nil, theme.DarkTheme()); got != "" {
		t.Errorf("Breadcrumb(nil) = %q, want empty", got)
	}
	if got := Breadcrumb([]string{}, theme.DarkTheme()); got != "" {
		t.Errorf("Breadcrumb([]string{}) = %q, want empty", got)
	}
}

func contains(s, sub string) bool {
	return strings.Contains(s, sub)
}

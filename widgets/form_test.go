package widgets

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// TestFormFieldLabelAboveField proves criterion #340: FormField renders
// the label above the field content, styled distinctly from plain body
// text using the given Theme.
func TestFormFieldLabelAboveField(t *testing.T) {
	dt := theme.DarkTheme()
	got := FormField("Name", "John Doe", "", dt)

	wantLabel := ansi.NewStyle().Foreground(dt.Muted).Render("Name")
	if !strings.Contains(got, wantLabel) {
		t.Fatalf("FormField output does not contain label styled with t.Muted %q\ngot:\n%s", wantLabel, got)
	}

	lines := strings.Split(got, "\n")
	labelIdx, fieldIdx := -1, -1
	for i, l := range lines {
		if strings.Contains(l, "Name") {
			labelIdx = i
		}
		if strings.Contains(l, "John Doe") {
			fieldIdx = i
		}
	}
	if labelIdx == -1 || fieldIdx == -1 || labelIdx >= fieldIdx {
		t.Errorf("expected label line before field line, got label at %d, field at %d", labelIdx, fieldIdx)
	}

	// The label must be styled distinctly from plain, unstyled body text.
	if wantLabel == "Name" {
		t.Errorf("label rendering %q is identical to plain text; expected distinct styling", wantLabel)
	}
}

// TestFormFieldError proves criterion #341: a non-empty error message
// renders below the field, styled in t.Error, visually distinct from the
// label and field.
func TestFormFieldError(t *testing.T) {
	dt := theme.DarkTheme()
	got := FormField("Email", "not-an-email", "invalid email address", dt)

	wantErr := ansi.NewStyle().Foreground(dt.Error).Render("invalid email address")
	if !strings.Contains(got, wantErr) {
		t.Fatalf("FormField output does not contain error styled with t.Error %q\ngot:\n%s", wantErr, got)
	}

	lines := strings.Split(got, "\n")
	fieldIdx, errIdx := -1, -1
	for i, l := range lines {
		if strings.Contains(l, "not-an-email") {
			fieldIdx = i
		}
		if strings.Contains(l, "invalid email address") {
			errIdx = i
		}
	}
	if fieldIdx == -1 || errIdx == -1 || errIdx <= fieldIdx {
		t.Errorf("expected error line after field line, got field at %d, error at %d", fieldIdx, errIdx)
	}

	wantLabel := ansi.NewStyle().Foreground(dt.Muted).Render("Email")
	if wantErr == wantLabel {
		t.Errorf("error styling is identical to label styling; expected them to be visually distinct")
	}
}

// TestFormFieldNoError proves criterion #342: an empty error message
// renders no error line at all, not even a blank one.
func TestFormFieldNoError(t *testing.T) {
	dt := theme.DarkTheme()
	got := FormField("Name", "John Doe", "", dt)

	wantLabel := ansi.NewStyle().Foreground(dt.Muted).Render("Name")
	want := wantLabel + "\n" + "John Doe"
	if got != want {
		t.Errorf("FormField with empty err = %q, want %q (no extra error line)", got, want)
	}

	lines := strings.Split(got, "\n")
	if len(lines) != 2 {
		t.Errorf("FormField with empty err produced %d lines, want 2 (label + field, no error line)", len(lines))
	}
	for _, l := range lines {
		if strings.TrimSpace(ansi.StripANSI(l)) == "" {
			t.Errorf("FormField with empty err produced a blank line %q, want none", l)
		}
	}
}

// TestFormStacksFieldsWithGap proves criterion #343: Form stacks multiple
// FormField-rendered blocks vertically with a blank-line separation
// between fields, matching layout.JoinVertical(1, ...).
func TestFormStacksFieldsWithGap(t *testing.T) {
	dt := theme.DarkTheme()
	f1 := FormField("Name", "John Doe", "", dt)
	f2 := FormField("Email", "john@example.com", "", dt)

	got := Form(f1, f2)

	f1Lines := strings.Split(f1, "\n")
	f2Lines := strings.Split(f2, "\n")
	gotLines := strings.Split(got, "\n")

	// f1 lines, then one blank separator line, then f2 lines.
	wantLineCount := len(f1Lines) + 1 + len(f2Lines)
	if len(gotLines) != wantLineCount {
		t.Fatalf("Form produced %d lines, want %d (fields + 1 blank separator)\ngot:\n%s", len(gotLines), wantLineCount, got)
	}

	sepLine := gotLines[len(f1Lines)]
	if strings.TrimSpace(ansi.StripANSI(sepLine)) != "" {
		t.Errorf("expected blank separator line between fields, got %q", sepLine)
	}

	if !strings.Contains(got, "Name") || !strings.Contains(got, "John Doe") {
		t.Errorf("Form output missing first field's content:\n%s", got)
	}
	if !strings.Contains(got, "Email") || !strings.Contains(got, "john@example.com") {
		t.Errorf("Form output missing second field's content:\n%s", got)
	}

	// Name field content must appear before Email field content.
	nameIdx := strings.Index(got, "John Doe")
	emailIdx := strings.Index(got, "john@example.com")
	if nameIdx == -1 || emailIdx == -1 || nameIdx >= emailIdx {
		t.Errorf("expected first field before second field in Form output")
	}
}

// TestFormFieldNoTruncation proves criterion #344 for FormField: label,
// field, and error text are never truncated — output width follows the
// content, unlike widgets.Box/Panel/Card's fixed-width containers.
func TestFormFieldNoTruncation(t *testing.T) {
	dt := theme.DarkTheme()
	longLabel := strings.Repeat("L", 200)
	longField := strings.Repeat("F", 250)
	longErr := strings.Repeat("E", 300)

	got := FormField(longLabel, longField, longErr, dt)

	if !strings.Contains(got, longLabel) {
		t.Errorf("FormField truncated the label; expected full %d-char label present", len(longLabel))
	}
	if !strings.Contains(got, longField) {
		t.Errorf("FormField truncated the field content; expected full %d-char field present", len(longField))
	}
	if !strings.Contains(got, longErr) {
		t.Errorf("FormField truncated the error message; expected full %d-char error present", len(longErr))
	}

	lines := strings.Split(got, "\n")
	if ansi.Width(lines[0]) < len(longLabel) {
		t.Errorf("label line width %d is narrower than label content %d chars", ansi.Width(lines[0]), len(longLabel))
	}
}

// TestFormNoTruncation proves criterion #344 for Form: joining fields
// never truncates content, even when field blocks have very different
// widths from one another.
func TestFormNoTruncation(t *testing.T) {
	dt := theme.DarkTheme()
	longField := strings.Repeat("X", 400)
	f1 := FormField("Short", "s", "", dt)
	f2 := FormField("Long", longField, "", dt)

	got := Form(f1, f2)

	if !strings.Contains(got, longField) {
		t.Errorf("Form truncated a field's long content; expected full %d-char field present", len(longField))
	}
}

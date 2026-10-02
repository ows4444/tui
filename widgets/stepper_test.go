package widgets

import (
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

func TestStepper(t *testing.T) {
	dt := theme.DarkTheme()
	steps := []string{"Setup", "Configure", "Done"}

	done := ansi.NewStyle().Foreground(dt.Success)
	active := ansi.NewStyle().Bold().Foreground(dt.Primary)
	upcoming := ansi.NewStyle().Foreground(dt.Muted)
	sep := upcoming.Render(" → ")

	got := Stepper(steps, 1, dt)
	want := done.Render("✓ Setup") + sep + active.Render("● Configure") + sep + upcoming.Render("○ Done")
	if got != want {
		t.Errorf("Stepper(steps, 1) = %q, want %q", got, want)
	}
}

func TestStepperFirstStepActive(t *testing.T) {
	dt := theme.DarkTheme()
	got := Stepper([]string{"A", "B"}, 0, dt)
	active := ansi.NewStyle().Bold().Foreground(dt.Primary)
	upcoming := ansi.NewStyle().Foreground(dt.Muted)
	want := active.Render("● A") + upcoming.Render(" → ") + upcoming.Render("○ B")
	if got != want {
		t.Errorf("Stepper at step 0 = %q, want %q", got, want)
	}
}

func TestStepperAllDone(t *testing.T) {
	dt := theme.DarkTheme()
	got := Stepper([]string{"A", "B"}, 2, dt) // current past the last index
	done := ansi.NewStyle().Foreground(dt.Success)
	upcoming := ansi.NewStyle().Foreground(dt.Muted)
	want := done.Render("✓ A") + upcoming.Render(" → ") + done.Render("✓ B")
	if got != want {
		t.Errorf("Stepper with current past the end = %q, want %q", got, want)
	}
}

func TestStepperEmpty(t *testing.T) {
	if got := Stepper(nil, 0, theme.DarkTheme()); got != "" {
		t.Errorf("Stepper(nil, ...) = %q, want empty", got)
	}
}

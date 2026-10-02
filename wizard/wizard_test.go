package wizard

import (
	"errors"
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

func TestNextAdvancesOnNilValidate(t *testing.T) {
	m := New("A", "B", "C")
	if got := m.Current(); got != 0 {
		t.Fatalf("Current() = %d, want 0", got)
	}
	if err := m.Next(func() error { return nil }); err != nil {
		t.Fatalf("Next() returned %v, want nil", err)
	}
	if got := m.Current(); got != 1 {
		t.Fatalf("Current() = %d, want 1", got)
	}
}

func TestNextClampsAtLastStep(t *testing.T) {
	m := New("A", "B", "C")
	for i := 0; i < 5; i++ {
		if err := m.Next(func() error { return nil }); err != nil {
			t.Fatalf("Next() returned %v, want nil", err)
		}
	}
	if got := m.Current(); got != 2 {
		t.Fatalf("Current() = %d, want 2 (clamped at last index)", got)
	}
}

func TestNextDoesNotAdvanceOnError(t *testing.T) {
	m := New("A", "B", "C")
	wantErr := errors.New("invalid field")
	if err := m.Next(func() error { return wantErr }); err != wantErr {
		t.Fatalf("Next() returned %v, want %v", err, wantErr)
	}
	if got := m.Current(); got != 0 {
		t.Fatalf("Current() = %d, want 0 (should not advance on error)", got)
	}
}

func TestBackRetreats(t *testing.T) {
	m := New("A", "B", "C")
	m.Next(func() error { return nil })
	m.Next(func() error { return nil })
	if got := m.Current(); got != 2 {
		t.Fatalf("Current() = %d, want 2", got)
	}
	m.Back()
	if got := m.Current(); got != 1 {
		t.Fatalf("Current() = %d, want 1", got)
	}
}

func TestBackClampsAtZero(t *testing.T) {
	m := New("A", "B", "C")
	for i := 0; i < 5; i++ {
		m.Back()
	}
	if got := m.Current(); got != 0 {
		t.Fatalf("Current() = %d, want 0 (clamped at first step)", got)
	}
}

func TestViewRendersViaStepper(t *testing.T) {
	m := New("Name", "Email", "Confirm")
	m.Next(func() error { return nil })

	out := ansi.StripANSI(m.View(theme.DarkTheme()))
	for _, want := range []string{"✓ Name", "● Email", "○ Confirm", " → "} {
		if !strings.Contains(out, want) {
			t.Errorf("View() missing %q\noutput: %q", want, out)
		}
	}
}

func TestZeroStepsDoesNotPanic(t *testing.T) {
	m := New()
	if got := m.Current(); got != 0 {
		t.Fatalf("Current() = %d, want 0", got)
	}
	if err := m.Next(func() error { return nil }); err != nil {
		t.Fatalf("Next() on zero-step Model returned %v, want nil", err)
	}
	if got := m.Current(); got != 0 {
		t.Fatalf("Current() after Next() on zero-step Model = %d, want 0", got)
	}
	m.Back()
	if got := m.Current(); got != 0 {
		t.Fatalf("Current() after Back() on zero-step Model = %d, want 0", got)
	}
	if out := m.View(theme.DarkTheme()); out != "" {
		t.Fatalf("View() on zero-step Model = %q, want empty", out)
	}
}

func TestNextWithNilValidateFunc(t *testing.T) {
	m := New("A", "B")
	if err := m.Next(nil); err != nil {
		t.Fatalf("Next(nil) returned %v, want nil", err)
	}
	if got := m.Current(); got != 1 {
		t.Fatalf("Current() = %d, want 1", got)
	}
}

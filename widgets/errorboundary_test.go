package widgets

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/theme"
)

// TestErrorBoundary proves criteria #501-#503: a normal render passes
// through unchanged, a panicking render is recovered and rendered as a
// styled fallback using the caller-supplied fallback text (not a fixed
// generic message).
func TestErrorBoundary(t *testing.T) {
	dt := theme.DarkTheme()

	t.Run("normal render returns output unchanged", func(t *testing.T) {
		render := func() string { return "hello world" }
		got := ErrorBoundary(render, "fallback message", dt)
		if got != "hello world" {
			t.Errorf("ErrorBoundary() = %q, want %q (unchanged)", got, "hello world")
		}
	})

	t.Run("panicking render recovers instead of propagating", func(t *testing.T) {
		render := func() string { panic("boom") }

		var got string
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic propagated out of ErrorBoundary: %v", r)
				}
			}()
			got = ErrorBoundary(render, "something went wrong", dt)
		}()

		if got == "" {
			t.Fatal("ErrorBoundary() returned empty string after panic recovery")
		}
	})

	t.Run("panicking render uses caller-supplied fallback text", func(t *testing.T) {
		render := func() string { panic("boom") }
		fallback := "custom failure explanation, not a generic message"

		got := ErrorBoundary(render, fallback, dt)
		if !strings.Contains(got, fallback) {
			t.Errorf("ErrorBoundary() = %q, want it to contain caller-supplied fallback %q", got, fallback)
		}

		want := Alert(fallback, VariantError, dt, 0)
		if got != want {
			t.Errorf("ErrorBoundary() = %q, want styled fallback %q", got, want)
		}
	})

	t.Run("different fallback strings produce different output", func(t *testing.T) {
		render := func() string { panic("boom") }

		a := ErrorBoundary(render, "fallback A", dt)
		b := ErrorBoundary(render, "fallback B", dt)
		if a == b {
			t.Errorf("expected different fallback strings to produce different output, got identical %q", a)
		}
	})
}

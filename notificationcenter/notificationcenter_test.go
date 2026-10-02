package notificationcenter

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

// #558: Push drops the oldest entry once the queue would exceed MaxVisible,
// keeping only the most recent notifications, and never exceeds MaxVisible.
func TestPushBoundsQueueToMaxVisible(t *testing.T) {
	tests := []struct {
		name     string
		max      int
		pushes   []string
		wantMsgs []string
	}{
		{
			name:     "under capacity keeps all",
			max:      3,
			pushes:   []string{"a", "b"},
			wantMsgs: []string{"a", "b"},
		},
		{
			name:     "at capacity keeps all",
			max:      2,
			pushes:   []string{"a", "b"},
			wantMsgs: []string{"a", "b"},
		},
		{
			name:     "over capacity drops oldest",
			max:      2,
			pushes:   []string{"a", "b", "c"},
			wantMsgs: []string{"b", "c"},
		},
		{
			name:     "well over capacity keeps only most recent",
			max:      2,
			pushes:   []string{"a", "b", "c", "d", "e"},
			wantMsgs: []string{"d", "e"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(tt.max, 0)
			for _, msg := range tt.pushes {
				m.Push(Notification{Message: msg, Variant: widgets.VariantInfo})
			}

			if len(m.Notifications) > tt.max {
				t.Fatalf("queue length %d exceeds MaxVisible %d", len(m.Notifications), tt.max)
			}

			got := make([]string, len(m.Notifications))
			for i, n := range m.Notifications {
				got[i] = n.Message
			}
			if len(got) != len(tt.wantMsgs) {
				t.Fatalf("got %d notifications, want %d: %v", len(got), len(tt.wantMsgs), got)
			}
			for i := range got {
				if got[i] != tt.wantMsgs[i] {
					t.Errorf("notification %d = %q, want %q (got %v)", i, got[i], tt.wantMsgs[i], got)
				}
			}
		})
	}
}

// #561: Render with an empty queue returns base unchanged rather than
// drawing an empty panel.
func TestRenderEmptyQueueReturnsBaseUnchanged(t *testing.T) {
	m := New(3, 0)
	base := "hello\nworld"

	got := m.Render(base)
	if got != base {
		t.Errorf("Render() with empty queue = %q, want base unchanged %q", got, base)
	}
}

// #559: Render with a non-empty queue composites a panel listing ALL
// currently-queued notifications simultaneously, not one at a time.
func TestRenderShowsAllQueuedNotificationsSimultaneously(t *testing.T) {
	m := New(3, 0)
	m.Theme = theme.DarkTheme()
	m.Push(Notification{Message: "first alert", Variant: widgets.VariantInfo})
	m.Push(Notification{Message: "second alert", Variant: widgets.VariantWarning})
	m.Push(Notification{Message: "third alert", Variant: widgets.VariantError})

	base := strings.TrimSuffix(strings.Repeat(strings.Repeat("x", 40)+"\n", 15), "\n")
	got := m.Render(base)

	for _, msg := range []string{"first alert", "second alert", "third alert"} {
		if !strings.Contains(got, msg) {
			t.Errorf("Render() output missing notification %q; output:\n%s", msg, got)
		}
	}
}

// #560: each notification is colored via the same Variant-to-color
// convention widgets.Alert/toast.Model already use (n.Variant.Color(theme)),
// not a new color scheme.
func TestRenderUsesVariantColorConvention(t *testing.T) {
	th := theme.DarkTheme()
	m := New(2, 0)
	m.Theme = th
	m.Push(Notification{Message: "warn-msg", Variant: widgets.VariantWarning})

	base := strings.TrimSuffix(strings.Repeat(strings.Repeat("x", 40)+"\n", 15), "\n")
	got := m.Render(base)

	wantColor := widgets.VariantWarning.Color(th)
	wantStyled := ansi.NewStyle().Foreground(wantColor).Render("warn-msg")
	if !strings.Contains(got, wantStyled) {
		t.Errorf("Render() output does not contain notification styled via widgets.Variant.Color convention;\nwant substring: %q\ngot:\n%s", wantStyled, got)
	}
}

// Sanity check that pushing beyond capacity with MaxVisible of 1 still
// behaves as a bounded queue (edge case for #558).
func TestPushMaxVisibleOne(t *testing.T) {
	m := New(1, 0)
	m.Push(Notification{Message: "a", Variant: widgets.VariantInfo})
	m.Push(Notification{Message: "b", Variant: widgets.VariantInfo})

	if len(m.Notifications) != 1 {
		t.Fatalf("len(Notifications) = %d, want 1", len(m.Notifications))
	}
	if m.Notifications[0].Message != "b" {
		t.Errorf("Notifications[0].Message = %q, want %q", m.Notifications[0].Message, "b")
	}
}

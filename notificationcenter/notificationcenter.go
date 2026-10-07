// Package notificationcenter is a persistent multi-item panel over a
// bounded notification queue — distinct from toast.Model's transient,
// one-at-a-time popup, this composites ALL currently-queued notifications
// simultaneously in a single panel, colored via the same
// widgets.Variant-to-color convention toast and widgets.Alert already use.
//
// Stability: experimental. Its API may change in any minor release.
package notificationcenter

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/internal/boxdraw"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

// Notification is one queued entry: a message and the Variant that colors
// it, per widgets.Variant's shared convention.
type Notification struct {
	Message string
	Variant widgets.Variant
}

// Model is a bounded queue of Notifications rendered as a single panel
// listing every entry currently queued, unlike toast.Model which shows
// only one notification at a time.
type Model struct {
	// Notifications is the bounded queue, oldest first. Push maintains the
	// MaxVisible bound; Notifications can also be set directly as long as
	// that invariant is respected.
	Notifications []Notification
	// MaxVisible caps how many notifications the queue holds at once. Push
	// drops the oldest entry once the queue would exceed it.
	MaxVisible int
	// Width, if positive, wraps each notification's message so every
	// rendered line's ansi.Width stays within it (mirrors widgets.Alert's
	// width behavior). 0 sizes the panel to its widest content line.
	Width int
	Theme theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens
	// Raw, when true, draws notification messages unchanged. By default they
	// are sanitised (ansi.Sanitize) so untrusted text cannot carry terminal
	// escape sequences.
	Raw bool
}

// New returns an empty Model bounded to maxVisible notifications, rendering
// at width (0 sizes to content).
func New(maxVisible, width int) Model {
	return Model{
		MaxVisible: maxVisible,
		Width:      width,
		Theme:      theme.DarkTheme(),
	}
}

// Push appends n to the queue. If the queue then holds more than
// MaxVisible notifications, the oldest entry (index 0) is dropped so the
// queue never exceeds MaxVisible — only the most recent notifications are
// kept.
func (m *Model) Push(n Notification) {
	m.Notifications = append(m.Notifications, n)
	if m.MaxVisible > 0 && len(m.Notifications) > m.MaxVisible {
		drop := len(m.Notifications) - m.MaxVisible
		m.Notifications = m.Notifications[drop:]
	}
}

// Render composites a panel listing every currently-queued notification
// over base, each line colored via n.Variant.Color(m.Theme) — the same
// convention toast.Model and widgets.Alert use. With an empty queue it
// returns base unchanged rather than drawing an empty panel.
func (m Model) Render(base string) string {
	if len(m.Notifications) == 0 {
		return base
	}

	lines := make([]string, len(m.Notifications))
	for i, n := range m.Notifications {
		color := n.Variant.Color(m.themed())
		text := ansi.Clean(m.Raw, n.Message)
		if m.Width > 0 {
			text = ansi.WrapStyled(text, m.Width)
		}
		lines[i] = ansi.NewStyle().Foreground(color).Render(text)
	}
	body := strings.Join(lines, "\n")

	panel := boxdraw.Draw(layout.NewBox().BorderColor(m.themed().BorderColor), m.themed().Border, 1, m.Width, body)

	baseLines := strings.Split(base, "\n")
	baseWidth := 0
	for _, l := range baseLines {
		if w := ansi.Width(l); w > baseWidth {
			baseWidth = w
		}
	}
	baseHeight := len(baseLines)

	panelLines := strings.Split(panel, "\n")
	panelWidth := ansi.Width(panelLines[0])
	panelHeight := len(panelLines)

	const margin = 1
	x := baseWidth - panelWidth - margin
	y := baseHeight - panelHeight - margin
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	return layout.Overlay(base, panel, x, y)
}

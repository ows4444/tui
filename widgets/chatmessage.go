package widgets

import (
	"time"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// Sender identifies who a ChatMessage is from, so ChatMessage can style
// the header distinctly per sender via widgets.Variant's existing
// color mapping rather than inventing a parallel one.
type Sender int

const (
	// SenderUser is a message from the user (Muted header colour). It is
	// the zero value.
	SenderUser Sender = iota
	// SenderAssistant is a message from the assistant (Success colour).
	SenderAssistant
	// SenderSystem is a system notice (Info colour).
	SenderSystem
	// SenderError is an error message (Error colour).
	SenderError
)

// Variant maps s to the Variant whose color best fits it: SenderError to
// VariantError (so it reads as a failure the same way Alert/Badge do),
// SenderSystem to VariantInfo, and SenderUser/SenderAssistant to
// VariantNeutral/VariantSuccess respectively, since neither is a severity.
func (s Sender) Variant() Variant {
	switch s {
	case SenderAssistant:
		return VariantSuccess
	case SenderSystem:
		return VariantInfo
	case SenderError:
		return VariantError
	default:
		return VariantNeutral
	}
}

// label returns the sender's default display label, used when name is
// empty.
func (s Sender) label() string {
	switch s {
	case SenderAssistant:
		return "Assistant"
	case SenderSystem:
		return "System"
	case SenderError:
		return "Error"
	default:
		return "User"
	}
}

// streamCursor is the trailing visual indicator ChatMessage appends to
// content while streaming, matching streamtext.Model's default cursor
// ("▌" rendered in the theme's Primary color) so a ChatMessage rendered
// mid-stream by a caller looks consistent with a streamtext.Model
// rendering the same content — without importing streamtext, which is a
// stateful Model owned by the caller, not something this stateless
// render function should depend on.
var streamCursor = theme.UnicodeGlyphSet().Cursor

// ChatMessage renders a single chat message: a header naming sender (name
// if non-empty, else Sender's default label) and timestamp, styled by
// sender via Sender.Variant's Variant color, followed by content. content
// may be empty; the header still renders. If streaming is true, a
// trailing cursor is appended after content instead of any
// character-reveal timing — ChatMessage is a stateless pure-render
// function, so the actual reveal animation stays owned by a caller's
// streamtext.Model elsewhere.
//
// name and content are untrusted (model output): tabs are expanded and every
// escape sequence and control character except SGR styling is removed. Use
// ChatMessageRaw to render them unchanged.
func ChatMessage(sender Sender, name, content string, timestamp time.Time, streaming bool, t theme.Theme) string {
	clean := func(s string) string { return ansi.SanitizeKeepSGR(ansi.ExpandTabs(s)) }
	return ChatMessageRaw(sender, clean(name), clean(content), timestamp, streaming, t)
}

// ChatMessageRaw is ChatMessage without sanitising: name and content are
// used unchanged. Only use it for trusted text.
func ChatMessageRaw(sender Sender, name, content string, timestamp time.Time, streaming bool, t theme.Theme) string {
	label := name
	if label == "" {
		label = sender.label()
	}

	color := sender.Variant().Color(t)
	header := ansi.NewStyle().Bold().Foreground(color).Render(label)
	ts := ansi.NewStyle().Foreground(t.Muted).Render(timestamp.Format("15:04:05"))
	out := header + " " + ts

	if content != "" {
		out += "\n" + content
	}
	if streaming {
		out += ansi.NewStyle().Foreground(t.Primary).Render(t.GlyphSet().Cursor)
	}
	return out
}

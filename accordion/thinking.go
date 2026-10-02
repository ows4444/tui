package accordion

import (
	"time"

	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

// ThinkingSection builds an Section for a single reasoning/thinking
// block, rather than a new Model type: round 12's gap analysis found
// Model with one Section already covers this (title always
// shown, content shown/hidden on toggle), so the only new code here is the
// token-count/duration metadata formatting on the title line. Feed the
// result into New(...); the returned section (like any built by
// New) starts collapsed.
//
// label becomes the Title's leading text, defaulting to "Reasoning" when
// empty. tokenCount and duration are appended to the title in parentheses
// as compact metadata, e.g. "Reasoning (1.2k tokens, 3.2s)", reusing
// compactCount (TokenCounter's own formatting) for the count and
// time.Duration's default String() for the duration.
//
// reasoning becomes Content unchanged; empty reasoning is valid and
// produces an empty Content, not a panic. If streaming is true, streamCursor
// (the same trailing cursor glyph ChatMessage appends while streaming) is
// appended to Content so a ThinkingSection rendered mid-stream matches
// ChatMessage's visual convention instead of inventing a new one.
func ThinkingSection(label, reasoning string, tokenCount int, duration time.Duration, streaming bool) Section {
	return ThinkingSectionWith(label, reasoning, tokenCount, duration, streaming, theme.Theme{})
}

// ThinkingSectionWith is ThinkingSection with the streaming cursor taken from t
// (theme.Glyphs.Cursor), so an ASCII theme gets an ASCII cursor.
func ThinkingSectionWith(label, reasoning string, tokenCount int, duration time.Duration, streaming bool, t theme.Theme) Section {
	title := label
	if title == "" {
		title = "Reasoning"
	}
	title += " (" + widgets.CompactCount(tokenCount) + " tokens, " + duration.String() + ")"

	content := reasoning
	if streaming {
		content += t.GlyphSet().Cursor
	}

	return Section{Title: title, Content: content}
}

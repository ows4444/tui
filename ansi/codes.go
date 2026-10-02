// Package ansi provides the raw terminal escape sequences and a small
// styling builder used to render colored, styled text.
//
// Width and Truncate count terminal columns using tables generated from
// Unicode 17.0.0 (see internal/tools/genwidth and `go generate`): East Asian Wide and
// Fullwidth runes and emoji-presentation runes take two columns; nonspacing
// marks, enclosing marks and format characters take none; everything else
// takes one. Text is measured by extended grapheme cluster (Unicode Standard
// Annex #29, tables from internal/tools/gengrapheme): a ZWJ sequence, a flag, an emoji
// with a skin-tone modifier or a base letter with combining marks is one unit,
// as wide as its widest rune, and Truncate never cuts inside one. For a
// terminal that draws the parts separately, call SetClusterWidth(false) or set
// TUI_NO_CLUSTERS=1 to count each rune on its own.
package ansi

import (
	"encoding/base64"
	"fmt"
	"strings"
)

const (
	// CSI is the control sequence introducer, ESC [.
	CSI = "\x1b["
	// Reset ends all SGR styling (ESC[0m).
	Reset = "\x1b[0m"

	// AltScreenEnable switches to the alternate screen buffer (mode 1049).
	AltScreenEnable = CSI + "?1049h"
	// AltScreenDisable returns to the normal screen buffer.
	AltScreenDisable = CSI + "?1049l"

	// CursorHide hides the text cursor.
	CursorHide = CSI + "?25l"
	// CursorShow shows the text cursor.
	CursorShow = CSI + "?25h"

	// ClearScreen erases the whole screen.
	ClearScreen = CSI + "2J"
	// ClearLine erases the cursor's whole line.
	ClearLine = CSI + "2K"
	// CursorHome moves the cursor to row 1, column 1.
	CursorHome = CSI + "H"

	// EraseDown clears from the cursor's current position to the end of
	// the screen. Used when committing a Println: after moving the
	// cursor up to the top of the live region via relative addressing,
	// there is no absolute row to address the region's remaining rows
	// by, so the only way to guarantee no stale live-region content is
	// left behind (if the committed text is shorter than the region it
	// replaces) is to erase everything below the cursor before writing.
	EraseDown = CSI + "0J"

	// BracketedPasteEnable makes the terminal wrap pasted text in
	// ESC[200~ ... ESC[201~ so it arrives as one PasteEvent.
	BracketedPasteEnable = CSI + "?2004h"
	// BracketedPasteDisable reverses BracketedPasteEnable.
	BracketedPasteDisable = CSI + "?2004l"

	// MouseSGREnable/Disable select SGR mouse-coordinate encoding, needed
	// so coordinates beyond 223 work and so press/release are unambiguous.
	// It must be combined with one of the tracking modes below.
	MouseSGREnable = CSI + "?1006h"
	// MouseSGRDisable reverses MouseSGREnable.
	MouseSGRDisable = CSI + "?1006l"

	// MouseClickEnable/Disable report only button press/release, no drag.
	MouseClickEnable = CSI + "?1000h"
	// MouseClickDisable reverses MouseClickEnable.
	MouseClickDisable = CSI + "?1000l"

	// MouseCellMotionEnable/Disable additionally report motion while a
	// button is held (drag), but not free-standing mouse movement.
	MouseCellMotionEnable = CSI + "?1002h"
	// MouseCellMotionDisable reverses MouseCellMotionEnable.
	MouseCellMotionDisable = CSI + "?1002l"

	// MouseAllMotionEnable/Disable report every mouse movement, held button
	// or not — the most complete, and the most event traffic.
	MouseAllMotionEnable = CSI + "?1003h"
	// MouseAllMotionDisable reverses MouseAllMotionEnable.
	MouseAllMotionDisable = CSI + "?1003l"

	// KittyKeyboardEnable pushes progressive-enhancement flag 1
	// ("disambiguate escape codes") onto the terminal's kitty-keyboard-
	// protocol stack: enough to make Shift+Enter, Ctrl+Shift+<letter>, and
	// similar combinations distinguishable from their unmodified sibling,
	// while leaving arrows/Home/End/function keys on their existing
	// legacy encoding. A terminal that doesn't support the protocol simply
	// ignores it. KittyKeyboardDisable pops one level off the stack,
	// restoring whatever was active before (nothing, for a terminal that
	// was never in kitty mode).
	KittyKeyboardEnable = CSI + ">1u"
	// KittyKeyboardDisable reverses KittyKeyboardEnable (see above).
	KittyKeyboardDisable = CSI + "<u"

	// FocusReportingEnable/Disable (DECSET 1004) make the terminal send
	// ESC[I when its window gains focus and ESC[O when it loses it, which
	// input.Reader decodes as an input.FocusEvent. A terminal that doesn't
	// support the mode ignores both sequences.
	FocusReportingEnable = CSI + "?1004h"
	// FocusReportingDisable reverses FocusReportingEnable.
	FocusReportingDisable = CSI + "?1004l"

	// SyncOutputEnable/Disable (mode 2026) bracket a frame write so the
	// terminal buffers it and paints the whole thing atomically instead of
	// character-by-character, preventing a torn/partial frame from being
	// visible mid-repaint. A terminal that doesn't recognize mode 2026
	// simply ignores both sequences, so this degrades to the previous
	// unwrapped-write behavior with no feature detection needed.
	SyncOutputEnable = CSI + "?2026h"
	// SyncOutputDisable ends a synchronized frame write (see
	// SyncOutputEnable).
	SyncOutputDisable = CSI + "?2026l"
)

// CursorPosition returns the escape sequence to move the cursor to the
// given 1-indexed row/column.
func CursorPosition(row, col int) string {
	return fmt.Sprintf("%s%d;%dH", CSI, row, col)
}

// CursorUp returns the escape sequence to move the cursor up n rows.
func CursorUp(n int) string { return fmt.Sprintf("%s%dA", CSI, n) }

// CursorDown returns the escape sequence to move the cursor down n rows.
func CursorDown(n int) string { return fmt.Sprintf("%s%dB", CSI, n) }

// CursorForward returns the escape sequence to move the cursor right n columns.
func CursorForward(n int) string { return fmt.Sprintf("%s%dC", CSI, n) }

// CursorBack returns the escape sequence to move the cursor left n columns.
func CursorBack(n int) string { return fmt.Sprintf("%s%dD", CSI, n) }

// SafeLinkTarget reports whether url may be embedded in an OSC 8 hyperlink.
// It must be non-empty, contain no control characters (C0, DEL, C1: ESC, BEL
// and the ST terminator can all end the OSC early and inject sequences), no
// spaces or surrounding whitespace, and start with a scheme of http, https,
// mailto or file (matched case-insensitively). Scheme-less targets, including
// relative paths and "//host" forms, are rejected: a terminal has no base to
// resolve them against, so they are ambiguous at best.
func SafeLinkTarget(url string) bool {
	for _, r := range url {
		if r <= 0x20 || (r >= 0x7f && r <= 0x9f) {
			return false
		}
	}
	i := strings.IndexByte(url, ':')
	if i < 0 {
		return false
	}
	switch strings.ToLower(url[:i]) {
	case "http", "https", "mailto", "file":
		return true
	}
	return false
}

// Hyperlink returns text wrapped in an OSC 8 clickable-hyperlink escape
// sequence pointing at url, for terminals that support it (many modern
// terminal emulators do; terminals that don't simply show text unadorned,
// ignoring the surrounding escapes). If url fails SafeLinkTarget, text is
// returned unchanged with no OSC 8 sequence.
func Hyperlink(text, url string) string {
	if !SafeLinkTarget(url) {
		return text
	}
	return "\x1b]8;;" + url + "\x1b\\" + text + "\x1b]8;;\x1b\\"
}

// OSC52Copy returns the OSC 52 escape sequence that sets the system
// clipboard to text, for terminals that support it (support isn't
// universal; unsupported terminals typically just ignore it). The payload
// is base64-encoded per the OSC 52 spec.
//
// Under tmux a bare OSC 52 is governed by the "set-clipboard" option (on or
// external; off discards it), not by "allow-passthrough", which only applies
// to DCS tmux;-wrapped sequences. A dropped copy looks like a success and
// cannot be detected from inside the program. See package clipboard.
func OSC52Copy(text string) string {
	return "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(text)) + "\x07"
}

// QueryBackgroundColor asks the terminal for its background color (OSC 11).
// A supporting terminal replies with ESC ] 11 ; rgb:RRRR/GGGG/BBBB, which
// input.Reader decodes as an input.BackgroundColorEvent. Terminals that
// don't support it stay silent, so callers must time the wait out.
const QueryBackgroundColor = "\x1b]11;?\x1b\\"

// QueryPalette asks the terminal for ANSI colours 0-15 (OSC 4). A supporting
// terminal answers each slot with ESC ] 4 ; N ; rgb:RRRR/GGGG/BBBB, which
// input.Reader decodes as an input.PaletteColorEvent. Terminals that don't
// support it stay silent, so callers fall back to the xterm palette.
const QueryPalette = "\x1b]4;0;?;1;?;2;?;3;?;4;?;5;?;6;?;7;?;8;?;9;?;10;?;11;?;12;?;13;?;14;?;15;?\x1b\\"

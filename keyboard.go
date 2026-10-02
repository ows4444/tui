package tui

import "github.com/ows4444/tui/input"

// KeyboardFlags is a bitmask of kitty keyboard protocol progressive
// enhancement flags; see WithKeyboard.
type KeyboardFlags int

// The kitty keyboard flags WithKeyboard accepts. Values match the protocol.
const (
	// KeyboardDisambiguate (flag 1) is what WithKittyKeyboard(true) requests.
	KeyboardDisambiguate KeyboardFlags = 1
	// KeyboardReportEvents (flag 2) asks the terminal to report key repeat
	// and release. Update then receives them as KeyRepeatMsg and
	// KeyReleaseMsg, never as Key. Without this flag they never arrive.
	KeyboardReportEvents KeyboardFlags = 2
	// KeyboardReportAlternates (flag 4) asks for alternate key codes. They
	// are consumed and ignored by the decoder.
	KeyboardReportAlternates KeyboardFlags = 4
	// KeyboardReportAllKeys (flag 8) asks for every key as an escape code.
	KeyboardReportAllKeys KeyboardFlags = 8
)

// keyboardMask is the set of flags WithKeyboard passes through.
const keyboardMask = KeyboardDisambiguate | KeyboardReportEvents | KeyboardReportAlternates | KeyboardReportAllKeys

// WithKeyboard opts into the kitty keyboard protocol with the given flags
// (OR them together). Bits outside 1/2/4/8 are dropped. Adding
// KeyboardReportEvents makes key repeat and release arrive as KeyRepeatMsg
// and KeyReleaseMsg (never as Key, so `case tui.Key` sees presses only and
// chords ignore them); without it a release is dropped as before and
// existing Update code never sees either. The flags
// are pushed on start and Resume after Suspend, and popped on quit, panic
// and Suspend. WithKittyKeyboard(true) is equivalent to
// WithKeyboard(KeyboardDisambiguate) and the two combine by OR. A terminal
// without the protocol ignores it.
func WithKeyboard(flags KeyboardFlags) ProgramOption {
	return func(p *Program) { p.keyboardFlags = flags & keyboardMask }
}

// kittyFlags is the effective flag set: WithKeyboard plus WithKittyKeyboard.
func (p *Program) kittyFlags() KeyboardFlags {
	f := p.keyboardFlags
	if p.kittyKeyboard {
		f |= KeyboardDisambiguate
	}
	return f
}

// KeyReleaseMsg reports a kitty key release (WithKeyboard with
// KeyboardReportEvents). It is delivered instead of a Key so that existing
// `case tui.Key` code only ever sees presses.
type KeyReleaseMsg struct{ Key Key }

// KeyRepeatMsg reports a kitty key auto-repeat (WithKeyboard with
// KeyboardReportEvents). It is delivered instead of a Key; handle it to
// react to held keys.
type KeyRepeatMsg struct{ Key Key }

// splitKeyAction is the single choke point that keeps non-press Keys away
// from chords and Update: releases and repeats become KeyReleaseMsg and
// KeyRepeatMsg; everything else is returned unchanged.
func splitKeyAction(msg Msg) Msg {
	k, ok := msg.(Key)
	if !ok {
		return msg
	}
	switch k.Action {
	case KeyRelease:
		return KeyReleaseMsg{Key: k}
	case KeyRepeat:
		return KeyRepeatMsg{Key: k}
	}
	return msg
}

// KeyAction is an alias of input.KeyAction; see Key.Action.
type KeyAction = input.KeyAction

// The key actions, aliased from input. They appear in KeyRepeatMsg.Key and
// KeyReleaseMsg.Key when WithKeyboard included KeyboardReportEvents.
const (
	KeyPress   = input.KeyPress
	KeyRepeat  = input.KeyRepeat
	KeyRelease = input.KeyRelease
)

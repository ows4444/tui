// Package input decodes raw terminal bytes into structured key events,
// including multi-byte ANSI escape sequences (arrows, function keys, etc.)
// and UTF-8 runes, without any external terminfo/termcap dependency.
package input

import (
	"strconv"
	"strings"
)

// KeyType identifies which key a Key event represents.
type KeyType int

// The key types. KeyRunes carries printable input in Key.Text; the
// navigation, editing and function keys carry no runes. KeyCtrl is a
// generic ctrl+letter with the letter in Key.Code, and Ctrl+C is reported
// separately as KeyCtrlC. KeyUnknown is an escape sequence the reader
// consumed but does not recognise.
const (
	KeyRunes KeyType = iota // printable input; see Key.Text
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeyEnter
	KeyEsc
	KeyTab
	KeyBackspace
	KeyDelete
	KeySpace
	KeyHome
	KeyEnd
	KeyPgUp
	KeyPgDown
	KeyF1
	KeyF2
	KeyF3
	KeyF4
	KeyCtrlC
	KeyCtrl // generic ctrl+letter; the letter is in Key.Code
	KeyUnknown

	// Appended after KeyUnknown so existing values are unchanged. KeyF5..KeyF24
	// are contiguous.
	KeyF5
	KeyF6
	KeyF7
	KeyF8
	KeyF9
	KeyF10
	KeyF11
	KeyF12
	KeyF13
	KeyF14
	KeyF15
	KeyF16
	KeyF17
	KeyF18
	KeyF19
	KeyF20
	KeyF21
	KeyF22
	KeyF23
	KeyF24
	KeyInsert

	// Kitty media keys (private-use codepoints 57428-57440), decoded by
	// the kitty CSI-u path. They carry no runes.
	KeyMediaPlay
	KeyMediaPause
	KeyMediaPlayPause
	KeyMediaReverse
	KeyMediaStop
	KeyMediaFastForward
	KeyMediaRewind
	KeyMediaTrackNext
	KeyMediaTrackPrevious
	KeyMediaRecord
	KeyMediaVolumeDown
	KeyMediaVolumeUp
	KeyMediaMute
)

// Mod is a bitmask of modifier keys held down alongside a Key.
//
// Plain Ctrl+letter arrives from the terminal as its own control byte (e.g.
// Ctrl+A is literally byte 0x01) and is represented by KeyCtrl, not Mod —
// Mod exists for the keys that have no such direct byte encoding and are
// only distinguishable via an xterm CSI modifier parameter, chiefly
// Ctrl/Shift/Alt combined with arrows, Home/End, PgUp/PgDn, and Alt+rune.
type Mod uint8

// ModNone is the zero Mod: no modifier held.
const ModNone Mod = 0

// The modifier bits of a Mod, combined with |. ModSuper below is set only
// by the kitty keyboard protocol.
const (
	ModShift Mod = 1 << iota
	ModAlt
	ModCtrl
	// ModSuper is only ever set by the kitty keyboard protocol (see
	// parseCSIu) — no legacy xterm CSI-modifier sequence distinguishes it.
	ModSuper
)

// Shift reports whether Shift was held.
func (m Mod) Shift() bool { return m&ModShift != 0 }

// Alt reports whether Alt was held.
func (m Mod) Alt() bool { return m&ModAlt != 0 }

// Ctrl reports whether Ctrl was held.
func (m Mod) Ctrl() bool { return m&ModCtrl != 0 }

// Super reports whether Super (the Windows/Cmd/Meta key) was held. Only
// ever true when decoded from a kitty-keyboard-protocol sequence.
func (m Mod) Super() bool { return m&ModSuper != 0 }

// String renders the held modifiers joined by "+", e.g. "ctrl+alt", or ""
// for ModNone.
func (m Mod) String() string {
	var parts []string
	if m.Ctrl() {
		parts = append(parts, "ctrl")
	}
	if m.Alt() {
		parts = append(parts, "alt")
	}
	if m.Shift() {
		parts = append(parts, "shift")
	}
	if m.Super() {
		parts = append(parts, "super")
	}
	return strings.Join(parts, "+")
}

// modFromXterm decodes an xterm CSI modifier parameter (as sent after the
// ';' in sequences like "\x1b[1;5A" for Ctrl+Up) into a Mod bitmask. The
// wire encoding is 1 + a 4-bit field (1=Shift, 2=Alt, 4=Ctrl, 8=Meta); Meta
// isn't represented in Mod. A value < 1 (no modifier parameter present)
// yields ModNone.
func modFromXterm(n int) Mod {
	if n < 1 {
		return ModNone
	}
	n--
	var m Mod
	if n&1 != 0 {
		m |= ModShift
	}
	if n&2 != 0 {
		m |= ModAlt
	}
	if n&4 != 0 {
		m |= ModCtrl
	}
	return m
}

// modFromKitty decodes a kitty-keyboard-protocol modifier parameter (see
// parseCSIu) into a Mod bitmask. The wire encoding extends xterm's
// 1+bitmask scheme with a 5th bit for Super (1=Shift, 2=Alt, 4=Ctrl,
// 8=Super — kitty's own Hyper/Meta/CapsLock/NumLock bits aren't
// represented, the same way xterm's Meta bit isn't). A value < 1 (no
// modifier parameter present) yields ModNone.
func modFromKitty(n int) Mod {
	if n < 1 {
		return ModNone
	}
	n--
	var m Mod
	if n&1 != 0 {
		m |= ModShift
	}
	if n&2 != 0 {
		m |= ModAlt
	}
	if n&4 != 0 {
		m |= ModCtrl
	}
	if n&8 != 0 {
		m |= ModSuper
	}
	return m
}

// decodeKittyKey turns a parsed kitty-keyboard-protocol CSI-u sequence's
// parameter groups (built by parseCSI's numeric-CSI loop, which also
// handles ':'-separated sub-values for this format) into a Key. groups[0]
// is the codepoint group: its first value is the base Unicode codepoint —
// this library only requests the kitty "disambiguate escape codes"
// enhancement flag, which keeps arrows/Home/End/function keys on their
// existing legacy encoding and only routes through CSI-u for keys that
// need disambiguation (letters/digits/symbols with modifiers, and
// modified Enter/Tab/Backspace/Esc/Space), so the higher functional-key
// codepoint range from other kitty flags never needs handling here. Any
// further
// ':'-separated alternates (shifted-key, base-layout-key) are ignored,
// since this library doesn't do layout-aware key matching. groups[1], if
// present, is [modifier, event-type]; event-type 3 (key release) yields
// KeyUnknown, the same way any other unrecognized sequence does, since
// this library has no key-release event.
func decodeKittyKey(groups [][]int) Key {
	if len(groups) == 0 || len(groups[0]) == 0 {
		return Key{Type: KeyUnknown}
	}
	codepoint := groups[0][0]

	var mod Mod
	// The event type (groups[1][1]) is applied by Reader.applyAction, which
	// knows whether the caller asked for release/repeat events.
	if len(groups) >= 2 && len(groups[1]) >= 1 {
		mod = modFromKitty(groups[1][0])
	}

	switch codepoint {
	case 13:
		return Key{Type: KeyEnter, Mod: mod}
	case 9:
		return Key{Type: KeyTab, Mod: mod}
	case 127, 8:
		return Key{Type: KeyBackspace, Mod: mod}
	case 27:
		return Key{Type: KeyEsc, Mod: mod}
	case 32:
		return Key{Type: KeySpace, Text: " ", Code: ' ', Mod: mod}
	}

	if codepoint <= 0 || codepoint > 0x10FFFF {
		return Key{Type: KeyUnknown}
	}
	// Kitty reports functional keys as Unicode private-use codepoints.
	// F13-F24 are 57376-57387; every other PUA codepoint (keypad, media,
	// modifier keys ...) has no Key here and must not surface as text.
	if codepoint >= 57376 && codepoint <= 57387 {
		return Key{Type: KeyF13 + KeyType(codepoint-57376), Mod: mod}
	}
	if k, ok := kittyPUAKey(codepoint, mod); ok {
		return k
	}
	if isPrivateUse(codepoint) {
		return Key{Type: KeyUnknown}
	}
	r := rune(codepoint)

	// Ctrl+c with no other modifier matches KeyCtrlC exactly the way the
	// plain-byte path (reader.go) does for the legacy 0x03 control byte,
	// so existing app code keeping a single "quit" case keeps working
	// whether or not the terminal speaks the kitty protocol.
	if r == 'c' && mod == ModCtrl {
		return Key{Type: KeyCtrlC}
	}
	// Ctrl+<letter> with no other modifier matches KeyCtrl exactly the way
	// the plain-byte path's control-byte range does. Only a modifier
	// combination the legacy encoding can't represent (e.g.
	// Ctrl+Shift+<letter>) falls through to the generic KeyRunes+Mod case
	// below — the actual new capability this decoder adds.
	if mod == ModCtrl && ((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')) {
		lower := r
		if lower >= 'A' && lower <= 'Z' {
			lower += 'a' - 'A'
		}
		return Key{Type: KeyCtrl, Code: lower}
	}

	// Kitty flag 16 sends the text the key produced as a third group of
	// colon-separated codepoints (ESC[97;2;65u is Shift+a producing "A").
	// Use it: it is what the user typed, with Shift and the layout applied.
	if len(groups) >= 3 {
		var text []rune
		for _, cp := range groups[2] {
			if cp >= 0x20 && cp <= 0x10FFFF && !isPrivateUse(cp) && !(cp >= 0xD800 && cp <= 0xDFFF) {
				text = append(text, rune(cp))
			}
		}
		if len(text) > 0 {
			return Key{Type: KeyRunes, Text: string(text), Code: text[0], Mod: mod &^ ModShift}
		}
	}

	return runeKey(r, mod)
}

// kittyKeypadRunes maps kitty keypad codepoints 57399-57416 (KP_0..KP_9,
// decimal, divide, multiply, subtract, add, enter, equal, separator) to the
// character they type; enter is handled separately.
var kittyKeypadRunes = [...]rune{
	'0', '1', '2', '3', '4', '5', '6', '7', '8', '9',
	'.', '/', '*', '-', '+', '\r', '=', ',',
}

// kittyKeypadNav maps keypad codepoints 57417-57426 to their navigation keys.
var kittyKeypadNav = [...]KeyType{
	KeyLeft, KeyRight, KeyUp, KeyDown, KeyPgUp, KeyPgDown,
	KeyHome, KeyEnd, KeyInsert, KeyDelete,
}

// kittyPUAKey decodes the kitty keypad (57399-57426) and media
// (57428-57440) codepoints. KP_BEGIN (57427) and the rest stay unknown.
func kittyPUAKey(cp int, mod Mod) (Key, bool) {
	switch {
	case cp >= 57399 && cp < 57399+len(kittyKeypadRunes):
		r := kittyKeypadRunes[cp-57399]
		if r == '\r' {
			return Key{Type: KeyEnter, Mod: mod, Keypad: true}, true
		}
		k := runeKey(r, mod)
		k.Keypad = true
		return k, true
	case cp >= 57417 && cp <= 57426:
		return Key{Type: kittyKeypadNav[cp-57417], Mod: mod, Keypad: true}, true
	case cp >= 57428 && cp <= 57440:
		return Key{Type: KeyMediaPlay + KeyType(cp-57428), Mod: mod}, true
	}
	return Key{}, false
}

// isPrivateUse reports whether cp is in a Unicode private-use area.
func isPrivateUse(cp int) bool {
	return (cp >= 0xE000 && cp <= 0xF8FF) ||
		(cp >= 0xF0000 && cp <= 0xFFFFD) ||
		(cp >= 0x100000 && cp <= 0x10FFFD)
}

// Key is a single decoded key event.
type Key struct {
	Type KeyType
	// Text is what a KeyRunes or KeySpace key typed: one character, or the
	// whole text when the terminal reports a key with associated text (a
	// grapheme cluster, a composed character). It is empty for every other
	// type. A Text of one ASCII character shares a table, so decoding it
	// allocates nothing.
	Text string
	// Code is the first rune of Text for a KeyRunes or KeySpace key, and the
	// letter (or space, \\, ], ^, _) of a KeyCtrl key; 0 for every other type.
	Code rune
	Mod  Mod
	// Action is KeyPress (the zero value) unless the Reader was asked to
	// report kitty event types, when it can also be KeyRepeat or KeyRelease.
	Action KeyAction
	// Keypad is true when the key came from the numeric keypad, as reported
	// by the kitty protocol (always with flag 8, KeyboardReportAll, and
	// usually with flag 1). Keypad Enter is {Type: KeyEnter, Keypad: true};
	// keypad digits and operators are KeyRunes with the character in Text;
	// keypad arrows, Home/End, PgUp/PgDn, Insert and Delete use the
	// matching Type. Legacy terminals cannot tell keypad keys apart.
	Keypad bool
}

// KeyAction is the kind of a Key event.
type KeyAction uint8

// The key actions. Only KeyPress is ever delivered unless release/repeat
// reporting is on (Reader.SetReportEvents, tui.WithKeyboard with
// KeyboardReportEvents).
const (
	KeyPress KeyAction = iota
	KeyRepeat
	KeyRelease
)

// String names the action: "press", "repeat" or "release".
func (a KeyAction) String() string {
	switch a {
	case KeyRepeat:
		return "repeat"
	case KeyRelease:
		return "release"
	}
	return "press"
}

// String renders k as a human-readable name, e.g. "ctrl+alt+shift+right"
// or "a", prefixed with any held modifiers.
func (k Key) String() string {
	prefix := ""
	if k.Mod != ModNone {
		prefix = k.Mod.String() + "+"
	}
	switch k.Type {
	case KeyRunes:
		return prefix + k.Text
	case KeyCtrl:
		if k.Code == ' ' {
			return "ctrl+space"
		}
		return "ctrl+" + string(k.Code)
	case KeyCtrlC:
		return "ctrl+c"
	case KeyUp:
		return prefix + "up"
	case KeyDown:
		return prefix + "down"
	case KeyLeft:
		return prefix + "left"
	case KeyRight:
		return prefix + "right"
	case KeyEnter:
		return prefix + "enter"
	case KeyEsc:
		return prefix + "esc"
	case KeyTab:
		return prefix + "tab"
	case KeyBackspace:
		return prefix + "backspace"
	case KeyDelete:
		return prefix + "delete"
	case KeySpace:
		return prefix + "space"
	case KeyHome:
		return prefix + "home"
	case KeyEnd:
		return prefix + "end"
	case KeyPgUp:
		return prefix + "pgup"
	case KeyPgDown:
		return prefix + "pgdown"
	case KeyF1:
		return prefix + "f1"
	case KeyF2:
		return prefix + "f2"
	case KeyF3:
		return prefix + "f3"
	case KeyF4:
		return prefix + "f4"
	case KeyInsert:
		return prefix + "insert"
	case KeyMediaPlay:
		return prefix + "media-play"
	case KeyMediaPause:
		return prefix + "media-pause"
	case KeyMediaPlayPause:
		return prefix + "media-play-pause"
	case KeyMediaReverse:
		return prefix + "media-reverse"
	case KeyMediaStop:
		return prefix + "media-stop"
	case KeyMediaFastForward:
		return prefix + "media-fast-forward"
	case KeyMediaRewind:
		return prefix + "media-rewind"
	case KeyMediaTrackNext:
		return prefix + "media-next"
	case KeyMediaTrackPrevious:
		return prefix + "media-previous"
	case KeyMediaRecord:
		return prefix + "media-record"
	case KeyMediaVolumeDown:
		return prefix + "volume-down"
	case KeyMediaVolumeUp:
		return prefix + "volume-up"
	case KeyMediaMute:
		return prefix + "mute"
	default:
		if k.Type >= KeyF5 && k.Type <= KeyF24 {
			return prefix + "f" + strconv.Itoa(int(k.Type-KeyF5)+5)
		}
		return "unknown"
	}
}

// asciiLimit is the number of ASCII characters asciiText holds.
const asciiLimit = 128

// asciiText backs textOf for ASCII so decoding a printable key allocates no
// string. A string is immutable, so sharing it is safe.
var asciiText = func() string {
	var b [asciiLimit]byte
	for i := range b {
		b[i] = byte(i)
	}
	return string(b[:])
}()

// textOf returns r as a string. For ASCII it is a one-byte slice of a shared
// string (no allocation).
func textOf(r rune) string {
	if r >= 0 && r < asciiLimit {
		return asciiText[r : r+1]
	}
	return string(r)
}

// runeKey is the KeyRunes key for the character r.
func runeKey(r rune, mod Mod) Key {
	return Key{Type: KeyRunes, Text: textOf(r), Code: r, Mod: mod}
}

package tui

import (
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/input"
)

// Key and KeyType are aliased from tui/input so callers only need to
// import the root package for the common case.
type Key = input.Key

// KeyType is an alias of input.KeyType.
type KeyType = input.KeyType

// The key types, aliased from input (see input.KeyType) so the common case
// needs only the root import.
const (
	KeyRunes              = input.KeyRunes
	KeyUp                 = input.KeyUp
	KeyDown               = input.KeyDown
	KeyLeft               = input.KeyLeft
	KeyRight              = input.KeyRight
	KeyEnter              = input.KeyEnter
	KeyEsc                = input.KeyEsc
	KeyTab                = input.KeyTab
	KeyBackspace          = input.KeyBackspace
	KeyDelete             = input.KeyDelete
	KeySpace              = input.KeySpace
	KeyHome               = input.KeyHome
	KeyEnd                = input.KeyEnd
	KeyPgUp               = input.KeyPgUp
	KeyPgDown             = input.KeyPgDown
	KeyF1                 = input.KeyF1
	KeyF2                 = input.KeyF2
	KeyF3                 = input.KeyF3
	KeyF4                 = input.KeyF4
	KeyF5                 = input.KeyF5
	KeyF6                 = input.KeyF6
	KeyF7                 = input.KeyF7
	KeyF8                 = input.KeyF8
	KeyF9                 = input.KeyF9
	KeyF10                = input.KeyF10
	KeyF11                = input.KeyF11
	KeyF12                = input.KeyF12
	KeyF13                = input.KeyF13
	KeyF14                = input.KeyF14
	KeyF15                = input.KeyF15
	KeyF16                = input.KeyF16
	KeyF17                = input.KeyF17
	KeyF18                = input.KeyF18
	KeyF19                = input.KeyF19
	KeyF20                = input.KeyF20
	KeyF21                = input.KeyF21
	KeyF22                = input.KeyF22
	KeyF23                = input.KeyF23
	KeyF24                = input.KeyF24
	KeyCtrlC              = input.KeyCtrlC
	KeyCtrl               = input.KeyCtrl
	KeyUnknown            = input.KeyUnknown
	KeyInsert             = input.KeyInsert
	KeyMediaPlay          = input.KeyMediaPlay
	KeyMediaPause         = input.KeyMediaPause
	KeyMediaPlayPause     = input.KeyMediaPlayPause
	KeyMediaReverse       = input.KeyMediaReverse
	KeyMediaStop          = input.KeyMediaStop
	KeyMediaFastForward   = input.KeyMediaFastForward
	KeyMediaRewind        = input.KeyMediaRewind
	KeyMediaTrackNext     = input.KeyMediaTrackNext
	KeyMediaTrackPrevious = input.KeyMediaTrackPrevious
	KeyMediaRecord        = input.KeyMediaRecord
	KeyMediaVolumeDown    = input.KeyMediaVolumeDown
	KeyMediaVolumeUp      = input.KeyMediaVolumeUp
	KeyMediaMute          = input.KeyMediaMute
)

// MouseEvent, MouseButton, and MouseAction are aliased from tui/input the
// same way Key is. MouseEvent Msgs only arrive when the Program was started
// with WithMouse.
type MouseEvent = input.MouseEvent

// MouseButton is an alias of input.MouseButton.
type MouseButton = input.MouseButton

// MouseAction is an alias of input.MouseAction.
type MouseAction = input.MouseAction

// The mouse buttons and actions, aliased from input (see input.MouseButton
// and input.MouseAction).
const (
	MouseButtonNone      = input.MouseButtonNone
	MouseButtonLeft      = input.MouseButtonLeft
	MouseButtonMiddle    = input.MouseButtonMiddle
	MouseButtonRight     = input.MouseButtonRight
	MouseButtonWheelUp   = input.MouseButtonWheelUp
	MouseButtonWheelDown = input.MouseButtonWheelDown

	MouseButtonWheelLeft  = input.MouseButtonWheelLeft
	MouseButtonWheelRight = input.MouseButtonWheelRight
	MouseButtonBack       = input.MouseButtonBack
	MouseButtonForward    = input.MouseButtonForward
	MouseButton10         = input.MouseButton10
	MouseButton11         = input.MouseButton11

	MouseActionPress   = input.MouseActionPress
	MouseActionRelease = input.MouseActionRelease
	MouseActionMotion  = input.MouseActionMotion
)

// PasteEvent carries the full text of a bracketed paste as one Msg, rather
// than as individual Key events. It arrives unless the Program was started
// with WithBracketedPaste(false).
type PasteEvent = input.PasteEvent

// FocusEvent reports the terminal window gaining or losing focus. It only
// arrives if the terminal has focus reporting (DECSET 1004) enabled. A
// repeat of the state last delivered is dropped by the Program.
type FocusEvent = input.FocusEvent

// ChordDef configures one multi-key chord for WithChords: Name is what
// ChordMsg reports and Keys is the ordered sequence of Key.String() forms
// (for example "g", "g" or "ctrl+x", "ctrl+s").
type ChordDef = input.ChordDef

// ChordMsg is delivered to Update when the keys of a configured chord have
// been pressed in sequence within the chord timeout. It replaces the Key
// events for those keys. It only arrives when the Program was started with
// WithChords.
type ChordMsg = input.ChordMsg

// BackgroundColorEvent is the terminal's reply to ansi.QueryBackgroundColor.
type BackgroundColorEvent = input.BackgroundColorEvent

// PaletteColorEvent is the terminal's reply to one slot of ansi.QueryPalette
// (OSC 4). theme.Palette.Observe consumes it.
type PaletteColorEvent = input.PaletteColorEvent

// ReplyEvent is a terminal string-sequence reply (OSC/DCS/SOS/PM/APC) that
// is not otherwise decoded.
type ReplyEvent = input.ReplyEvent

// BackgroundUnknownMsg is sent once, to Update, when WithBackgroundDetection
// is on and the terminal did not answer the background-colour query within
// the timeout: the terminal doesn't support OSC 11 (or is too slow), so the
// app should keep its default theme. It is never sent after a reply arrived.
type BackgroundUnknownMsg = input.BackgroundUnknownMsg

// ResizeMsg is sent once at startup and again on every terminal resize
// (SIGWINCH).
//
// Measurer says how this Program's terminal draws grapheme clusters, which a
// capability probe (WithCapabilityProbe) can change after start-up; the Program
// then sends the same size again with the new Measurer. It is the zero value,
// which follows the process-wide ansi.SetClusterWidth setting, until a probe
// says otherwise. A component that measures text for this terminal uses
// msg.Measurer instead of the package-level ansi.Width, so two Programs in one
// process (an SSH server, parallel tests) never measure for each other.
type ResizeMsg struct {
	Width, Height int
	Measurer      ansi.Measurer
}

// QuitMsg, once it reaches the event loop, ends Program.Run. Applications
// don't construct it directly — return tui.Quit() from Update instead.
type QuitMsg struct{}

// InputErrorMsg is delivered once if reading terminal input fails after
// startup (including io.EOF when the input is closed). Input delivery stops
// after it — no further Key/Mouse/Paste Msgs arrive — but the Program keeps
// running; Update decides whether to react, typically by returning tui.Quit().
type InputErrorMsg struct {
	Err error
}

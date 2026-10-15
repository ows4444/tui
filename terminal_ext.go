package tui

import (
	"encoding/base64"
	"errors"
	"strconv"
	"strings"

	"github.com/ows4444/tui/ansi"
)

const (
	titlePush = "\x1b[22;0t" // save the window title on the terminal's stack
	titlePop  = "\x1b[23;0t" // restore the saved window title
)

// saveTitleOnce returns the sequence that pushes the current window title,
// the first time it is called, and "" afterwards; the matching pop is written
// by leaveModes and by the fixed restore string.
func (p *Program) saveTitleOnce() string {
	p.outMu.Lock()
	defer p.outMu.Unlock()
	if p.titleSaved {
		return ""
	}
	p.titleSaved = true
	return titlePush
}

// titleAndShapeRestoreLocked is what the exit paths append to undo
// SetWindowTitle and SetCursorShape. The caller holds outMu.
func (p *Program) titleAndShapeRestoreLocked() string {
	s := ""
	if p.cursorShaped {
		s += cursorShapeSeq(CursorShapeDefault)
	}
	if p.titleSaved {
		s += titlePop
	}
	if p.progressSet {
		s += progressSeq(ProgressNone, 0)
	}
	return s
}

// takeTitleAndShapeRestore is titleAndShapeRestoreLocked for a caller that
// does not hold outMu; it also clears the flags, so a Suspend that restored
// the title does not pop it twice, and a later SetWindowTitle pushes again.
func (p *Program) takeTitleAndShapeRestore() string {
	p.outMu.Lock()
	defer p.outMu.Unlock()
	s := p.titleAndShapeRestoreLocked()
	p.titleSaved, p.cursorShaped, p.progressSet = false, false, false
	return s
}

// CursorShape is a hardware cursor shape for SetCursorShape (DECSCUSR).
type CursorShape int

// The cursor shapes, numbered as DECSCUSR's parameter.
const (
	// CursorShapeDefault is the terminal's own configured shape.
	CursorShapeDefault CursorShape = iota
	CursorShapeBlinkingBlock
	CursorShapeSteadyBlock
	CursorShapeBlinkingUnderline
	CursorShapeSteadyUnderline
	CursorShapeBlinkingBar
	CursorShapeSteadyBar
)

func cursorShapeSeq(s CursorShape) string {
	if s < CursorShapeDefault || s > CursorShapeSteadyBar {
		s = CursorShapeDefault
	}
	return "\x1b[" + string(rune('0'+int(s))) + " q" // #nosec G115 -- s is clamped to 0..6 above
}

// SetCursorShape returns a Cmd that changes the hardware cursor's shape (CSI n
// SP q). The Program puts the terminal's default shape back on every exit
// path. It affects the cursor a CursorPlacer shows, or any cursor the terminal
// draws; terminals that do not know DECSCUSR ignore it.
func SetCursorShape(s CursorShape) Cmd {
	return func() Msg {
		return modeMsg{apply: func(p *Program) {
			p.outMu.Lock()
			p.cursorShaped = s != CursorShapeDefault
			p.outMu.Unlock()
			p.write(cursorShapeSeq(s))
		}}
	}
}

// ProgressState is what a terminal's progress indicator shows (OSC 9;4), which
// terminals such as Windows Terminal, iTerm2, WezTerm and Ghostty draw in the
// tab or the taskbar.
type ProgressState int

// The progress states, numbered as the sequence's state parameter.
const (
	// ProgressNone removes the indicator.
	ProgressNone ProgressState = iota
	// ProgressNormal shows a bar at the given percent.
	ProgressNormal
	// ProgressError shows a bar at the given percent in the error colour.
	ProgressError
	// ProgressIndeterminate shows a busy indicator with no value.
	ProgressIndeterminate
	// ProgressPaused shows a bar at the given percent in the warning colour.
	ProgressPaused
)

// progressSeq is OSC 9;4;state;percent, percent clamped to 0..100.
func progressSeq(st ProgressState, percent int) string {
	if st < ProgressNone || st > ProgressPaused {
		st = ProgressNone
	}
	if st == ProgressNone || st == ProgressIndeterminate {
		percent = 0
	}
	percent = min(max(percent, 0), 100)
	return "\x1b]9;4;" + strconv.Itoa(int(st)) + ";" + strconv.Itoa(percent) + "\x07"
}

// SetProgress returns a Cmd that sets the terminal's progress indicator (OSC
// 9;4): a bar at percent (0 to 100), an error or paused bar, a busy indicator
// with no value, or none. The Program removes the indicator on every exit
// path. Terminals that do not know the sequence ignore it.
func SetProgress(state ProgressState, percent int) Cmd {
	return func() Msg {
		return modeMsg{apply: func(p *Program) {
			p.outMu.Lock()
			p.progressSet = state != ProgressNone
			p.outMu.Unlock()
			p.write(progressSeq(state, percent))
		}}
	}
}

// ClipboardMsg is delivered to Update with the answer to a ReadClipboard.
type ClipboardMsg struct {
	// Text is the clipboard contents the terminal reported.
	Text string
	// Err is non-nil when the terminal's answer could not be decoded.
	Err error
}

// ReadClipboard returns a Cmd that asks the terminal for the system clipboard
// through OSC 52 and delivers a ClipboardMsg with the decoded text. It is
// best effort: many terminals refuse to reveal the clipboard (it is a privacy
// risk they leave off by default), and under tmux the query needs
// allow-passthrough. When the terminal does not answer, no message arrives, so
// do not wait on one.
func ReadClipboard() Cmd {
	return func() Msg {
		return modeMsg{apply: func(p *Program) {
			p.clipReads.Add(1)
			p.write("\x1b]52;c;?\x07")
		}}
	}
}

// WriteClipboard returns a Cmd that copies text: the Program writes the OSC 52
// sequence that sets the system clipboard to its own output, and keeps text
// as its copy buffer for PasteCopied. The buffer belongs to the Program, so
// two Programs in one process do not see each other's copies. Whether the
// system clipboard changes is up to the terminal; see package clipboard.
func WriteClipboard(text string) Cmd {
	return func() Msg {
		return modeMsg{apply: func(p *Program) {
			p.copied = text
			p.write(ansi.OSC52Copy(text))
		}}
	}
}

// pasteCopiedMsg asks the loop to deliver the Program's copy buffer.
type pasteCopiedMsg struct{}

// PasteCopied returns a Cmd that delivers the text of the Program's last
// WriteClipboard to Update as a PasteEvent, the Msg a bracketed paste arrives
// as. Nothing is delivered when nothing has been copied. It does not read the
// system clipboard; ReadClipboard asks the terminal for that.
func PasteCopied() Cmd { return func() Msg { return pasteCopiedMsg{} } }

var errClipboardReply = errors.New("tui: malformed OSC 52 clipboard reply")

// clipboardReply turns an OSC reply into a ClipboardMsg when it is the answer
// to an outstanding ReadClipboard. A reply arriving with no query outstanding
// is left alone, so stray terminal output cannot inject a clipboard message.
func (p *Program) clipboardReply(ev any) (ClipboardMsg, bool) {
	r, ok := ev.(ReplyEvent)
	if !ok || r.Kind != ']' || !strings.HasPrefix(r.Data, "52;") {
		return ClipboardMsg{}, false
	}
	for {
		n := p.clipReads.Load()
		if n <= 0 {
			return ClipboardMsg{}, false
		}
		if p.clipReads.CompareAndSwap(n, n-1) {
			break
		}
	}
	// Data is "52;<selection>;<base64>".
	parts := strings.SplitN(r.Data, ";", 3)
	if len(parts) != 3 {
		return ClipboardMsg{Err: errClipboardReply}, true
	}
	raw, err := base64.StdEncoding.DecodeString(parts[2])
	if err != nil {
		return ClipboardMsg{Err: errClipboardReply}, true
	}
	return ClipboardMsg{Text: string(raw)}, true
}

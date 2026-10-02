package tui

import (
	"encoding/base64"
	"errors"
	"strings"
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
	return s
}

// takeTitleAndShapeRestore is titleAndShapeRestoreLocked for a caller that
// does not hold outMu; it also clears the flags, so a Suspend that restored
// the title does not pop it twice, and a later SetWindowTitle pushes again.
func (p *Program) takeTitleAndShapeRestore() string {
	p.outMu.Lock()
	defer p.outMu.Unlock()
	s := p.titleAndShapeRestoreLocked()
	p.titleSaved, p.cursorShaped = false, false
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

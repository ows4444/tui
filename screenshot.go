package tui

import (
	"strings"

	"github.com/ows4444/tui/ansi"
)

// Screenshot is the screen a Program last drew: what a user sees, for a bug
// report or a picture in the docs.
type Screenshot struct {
	// Text is the rows without styling, each with its trailing spaces cut and
	// blank rows at the end dropped, joined by "\n".
	Text string
	// ANSI is the rows with their colours and attributes, joined by "\n". Feed
	// it to tuitest.SVGOf for an SVG image.
	ANSI string
	// Width and Height are the terminal's size in cells.
	Width, Height int
}

// snapshotMsg asks the loop for a Screenshot, to be wrapped into the Msg the
// asker wants delivered to Update.
type snapshotMsg struct{ wrap func(Screenshot) Msg }

// ScreenshotCmd is a Cmd that delivers the screen the Program last drew to
// Update, as the Msg wrap makes from it: the loop takes the Screenshot, so it
// is never a race with a frame being drawn, and a model can ask from Update
// without any lock. A nil wrap gives a nil Cmd.
func ScreenshotCmd(wrap func(Screenshot) Msg) Cmd {
	if wrap == nil {
		return nil
	}
	return func() Msg { return snapshotMsg{wrap: wrap} }
}

// WithScreenshot makes the key (a Key.String name such as "ctrl+p") hand save
// the current Screenshot. The key is consumed by the Program and never reaches
// Update. save runs on the event loop, so it should return quickly: write the
// file, or start a goroutine to. Without this option nothing is kept for it.
func WithScreenshot(key string, save func(Screenshot)) ProgramOption {
	return func(p *Program) { p.screenshotKey, p.screenshotFn = key, save }
}

// takeShot reads the retained frame; only the event loop may call it.
func (p *Program) takeShot() Screenshot {
	rows := make([]string, len(p.lastFrame))
	plain := make([]string, len(p.lastFrame))
	for i, l := range p.lastFrame {
		rows[i] = l
		plain[i] = strings.TrimRight(ansi.StripANSI(l), " ")
	}
	for len(plain) > 0 && plain[len(plain)-1] == "" {
		plain = plain[:len(plain)-1]
	}
	return Screenshot{
		Text: strings.Join(plain, "\n"), ANSI: strings.Join(rows, "\n"),
		Width: p.width, Height: p.height,
	}
}

// screenshotKeyPressed hands save the screen when msg is the screenshot key,
// and reports whether it was (the caller drops msg).
func (p *Program) screenshotKeyPressed(msg Msg) bool {
	if p.screenshotKey == "" || p.screenshotFn == nil {
		return false
	}
	k, ok := msg.(Key)
	if !ok || k.Action == KeyRelease || k.String() != p.screenshotKey {
		return false
	}
	p.screenshotFn(p.takeShot())
	return true
}

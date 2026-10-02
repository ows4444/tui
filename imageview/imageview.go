// Package imageview draws a PNG with the kitty graphics protocol when the
// terminal supports it, with Sixel when it supports only that, and a bordered
// text placeholder otherwise.
//
// Stability: experimental. Its API may change in any minor release.
package imageview

import (
	"encoding/base64"
	"hash/fnv"
	"strconv"
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// MaxChunk is the largest base64 payload, in bytes, put in one kitty
// graphics escape sequence (the protocol's limit).
const MaxChunk = 4096

// Model shows PNG as a Width by Height cell area. Set Kitty from
// tui.Capabilities.KittyGraphics (the startup probe's answer); leave it false
// when the probe is off or the terminal did not confirm support.
type Model struct {
	// PNG is the encoded image, transmitted as is.
	PNG []byte
	// Width and Height are the size in terminal cells.
	Width, Height int
	// Alt is the text shown in the placeholder and spoken by Linearize.
	Alt string
	// Kitty reports that the terminal supports kitty graphics.
	Kitty bool
	// Sixel reports that the terminal supports Sixel graphics (set it from
	// tui.Capabilities.Sixel). It is used only when Kitty is false, so kitty
	// stays preferred when a terminal has both.
	Sixel bool
	// CellWidth and CellHeight are the pixel size of a terminal cell, used to
	// scale a Sixel image to Width by Height cells. Zero means
	// DefaultCellWidth by DefaultCellHeight.
	CellWidth, CellHeight int
	// ID is the kitty image id (1 to 4294967295). Leave it 0 to derive one
	// from the PNG and the size; give each simultaneously visible image of the
	// same bytes its own ID. The renderer deletes the placement with this id
	// (a=d) when the image leaves the View or moves, and re-places it whenever
	// the row it is anchored to is rewritten.
	ID    uint32
	Theme theme.Theme
	// Getenv reads $TMUX and $STY, to wrap the graphics sequences for a
	// terminal multiplexer (see ansi.Passthrough). Nil means os.Getenv; set it
	// to the Program's environment, or to a stub in tests. Delete does not
	// wrap: a multiplexer drops it, which leaves the image until the next
	// redraw.
	Getenv func(string) string

	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens
}

// imageID returns the kitty id of m: ID, or a nonzero FNV-1a hash of the PNG
// and the size.
func (m Model) imageID() uint32 {
	if m.ID != 0 {
		return m.ID
	}
	h := fnv.New32a()
	_, _ = h.Write(m.PNG)
	_, _ = h.Write([]byte{byte(m.Width), byte(m.Width >> 8), byte(m.Height), byte(m.Height >> 8)}) // #nosec G115 -- hashing the low 16 bits of each size on purpose
	if id := h.Sum32(); id != 0 {
		return id
	}
	return 1
}

// New returns a Model for png at width by height cells, using theme.Dark.
func New(png []byte, width, height int, alt string) Model {
	return Model{PNG: png, Width: width, Height: height, Alt: alt, Theme: theme.DarkTheme()}
}

// View renders Height rows, each exactly Width cells wide. With Kitty or Sixel
// set (Kitty wins) and a non-empty PNG, the first row carries the graphics
// escape sequences (zero width) that display the image, followed by blank
// cells; the other rows are blank and reserve the image's area. Otherwise, or
// when a Sixel PNG does not decode, it draws the placeholder. Width <= 0 or Height <= 0 renders "".
func (m Model) View() string {
	if m.Width <= 0 || m.Height <= 0 {
		return ""
	}
	if len(m.PNG) == 0 || !m.Kitty && !m.Sixel {
		return m.placeholder()
	}
	var graphics string
	if m.Kitty {
		graphics = ansi.Passthrough(TransmitID(m.imageID(), m.PNG, m.Width, m.Height), m.Getenv)
	} else if graphics = Sixel(m.PNG, m.Width, m.Height, m.CellWidth, m.CellHeight); graphics == "" {
		return m.placeholder() // the PNG does not decode
	}
	blank := strings.Repeat(" ", m.Width)
	rows := make([]string, m.Height)
	for i := range rows {
		rows[i] = blank
	}
	rows[0] = graphics + blank
	return strings.Join(rows, "\n")
}

// Transmit returns the kitty graphics sequences that transmit png and display
// it over cols by rows cells at the cursor, without moving the cursor (C=1).
// The base64 payload is split into chunks of at most MaxChunk bytes; the
// control keys are on the first chunk only and m=1 marks all but the last.
// The image has no id; see TransmitID.
func Transmit(png []byte, cols, rows int) string { return TransmitID(0, png, cols, rows) }

// Delete returns the kitty graphics command that deletes every placement of
// image id and frees its data, with the terminal's reply suppressed. The
// renderer sends it by itself when a view stops showing the image; this is for
// code that writes to the terminal directly.
func Delete(id uint32) string {
	return "\x1b_Ga=d,d=I,i=" + strconv.FormatUint(uint64(id), 10) + ",q=2\x1b\\"
}

// TransmitID is Transmit for the image with kitty id (i= and p= keys, replies
// suppressed with q=2), so showing it again replaces its placement instead of
// adding one. id 0 gives Transmit.
func TransmitID(id uint32, png []byte, cols, rows int) string {
	enc := base64.StdEncoding.EncodeToString(png)
	var b strings.Builder
	first := true
	for {
		n := min(len(enc), MaxChunk)
		chunk := enc[:n]
		enc = enc[n:]
		more := "0"
		if len(enc) > 0 {
			more = "1"
		}
		b.WriteString("\x1b_G")
		if first {
			b.WriteString("a=T,f=100,C=1,c=" + strconv.Itoa(cols) + ",r=" + strconv.Itoa(rows) + ",")
			if id != 0 {
				n := strconv.FormatUint(uint64(id), 10)
				b.WriteString("i=" + n + ",p=" + n + ",q=2,")
			}
			first = false
		}
		b.WriteString("m=" + more + ";" + chunk + "\x1b\\")
		if len(enc) == 0 {
			return b.String()
		}
	}
}

// placeholder draws an ASCII box (when at least 3 by 3; plain rows
// otherwise) with the alt text on its middle row, truncated to fit.
func (m Model) placeholder() string {
	style := ansi.NewStyle().Foreground(m.themed().Muted)
	alt := asciiOnly(ansi.Sanitize(m.Alt))
	rows := make([]string, m.Height)
	mid := m.Height / 2
	if m.Width < 3 || m.Height < 3 {
		for i := range rows {
			rows[i] = strings.Repeat(" ", m.Width)
		}
		rows[mid] = style.Render(fit(alt, m.Width))
		return strings.Join(rows, "\n")
	}
	inner := m.Width - 2
	edge := style.Render("+" + strings.Repeat("-", inner) + "+")
	for i := range rows {
		switch {
		case i == 0 || i == m.Height-1:
			rows[i] = edge
		case i == mid:
			rows[i] = style.Render("|" + fit(alt, inner) + "|")
		default:
			rows[i] = style.Render("|" + strings.Repeat(" ", inner) + "|")
		}
	}
	return strings.Join(rows, "\n")
}

// fit truncates s to w cells and centres it in exactly w cells.
func fit(s string, w int) string {
	if len(s) > w {
		s = s[:w]
	}
	left := (w - len(s)) / 2
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", w-len(s)-left)
}

// asciiOnly replaces every non-ASCII or control rune of s with '?', so each
// byte is one cell.
func asciiOnly(s string) string {
	b := make([]byte, 0, len(s))
	for _, r := range s {
		if r < 0x20 || r > 0x7e {
			r = '?'
		}
		b = append(b, byte(r))
	}
	return string(b)
}

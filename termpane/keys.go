package termpane

import (
	"strings"

	"github.com/ows4444/tui"
)

// encodeKey is the bytes a terminal sends for k. appCursor is the program's
// cursor-key mode (DECCKM). It returns "" for a key that sends nothing.
func encodeKey(k tui.Key, appCursor bool) string {
	alt := k.Mod.Alt()
	// xterm's modifier parameter: 1 plus shift (1), alt (2) and ctrl (4).
	mod := 1
	if k.Mod.Shift() {
		mod += 1
	}
	if k.Mod.Alt() {
		mod += 2
	}
	if k.Mod.Ctrl() {
		mod += 4
	}
	csi := func(final string) string { // an arrow or Home/End
		if mod > 1 {
			return "\x1b[1;" + itoa(mod) + final
		}
		if appCursor {
			return "\x1bO" + final
		}
		return "\x1b[" + final
	}
	tilde := func(n string) string {
		if mod > 1 {
			return "\x1b[" + n + ";" + itoa(mod) + "~"
		}
		return "\x1b[" + n + "~"
	}
	ss3 := func(final string) string { // F1 to F4
		if mod > 1 {
			return "\x1b[1;" + itoa(mod) + final
		}
		return "\x1bO" + final
	}
	prefixAlt := func(s string) string {
		if alt && s != "" {
			return "\x1b" + s
		}
		return s
	}
	switch k.Type {
	case tui.KeyRunes:
		if k.Mod.Ctrl() {
			if r := []rune(k.Text); len(r) == 1 {
				return prefixAlt(ctrlByte(r[0]))
			}
		}
		return prefixAlt(k.Text)
	case tui.KeySpace:
		if k.Mod.Ctrl() {
			return prefixAlt("\x00")
		}
		return prefixAlt(" ")
	case tui.KeyEnter:
		return prefixAlt("\r")
	case tui.KeyTab:
		if k.Mod.Shift() {
			return "\x1b[Z"
		}
		return prefixAlt("\t")
	case tui.KeyBackspace:
		if k.Mod.Ctrl() {
			return prefixAlt("\x08")
		}
		return prefixAlt("\x7f")
	case tui.KeyEsc:
		return "\x1b"
	case tui.KeyCtrlC:
		return prefixAlt("\x03")
	case tui.KeyCtrl:
		return prefixAlt(ctrlByte(k.Code))
	case tui.KeyUp:
		return csi("A")
	case tui.KeyDown:
		return csi("B")
	case tui.KeyRight:
		return csi("C")
	case tui.KeyLeft:
		return csi("D")
	case tui.KeyHome:
		return csi("H")
	case tui.KeyEnd:
		return csi("F")
	case tui.KeyInsert:
		return tilde("2")
	case tui.KeyDelete:
		return tilde("3")
	case tui.KeyPgUp:
		return tilde("5")
	case tui.KeyPgDown:
		return tilde("6")
	case tui.KeyF1:
		return ss3("P")
	case tui.KeyF2:
		return ss3("Q")
	case tui.KeyF3:
		return ss3("R")
	case tui.KeyF4:
		return ss3("S")
	}
	if k.Type >= tui.KeyF5 && k.Type <= tui.KeyF12 {
		return tilde([]string{"15", "17", "18", "19", "20", "21", "23", "24"}[k.Type-tui.KeyF5])
	}
	return ""
}

// ctrlByte is the control character for Ctrl and r, or "" when there is none.
func ctrlByte(r rune) string {
	switch {
	case r >= 'a' && r <= 'z':
		return string(r - 'a' + 1)
	case r >= 'A' && r <= 'Z':
		return string(r - 'A' + 1)
	case r == '@' || r == ' ':
		return "\x00"
	case r >= '[' && r <= '_':
		return string(r - '@')
	case r == '?':
		return "\x7f"
	}
	return ""
}

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}

// encodePaste is the bytes for pasted text: wrapped in markers when the program
// asked for bracketed paste, with the marker itself cut out of the text so a
// paste cannot end the bracket early.
func encodePaste(text string, bracketed bool) string {
	text = strings.ReplaceAll(text, "\x1b[201~", "")
	text = strings.ReplaceAll(text, "\x1b[200~", "")
	if bracketed {
		return "\x1b[200~" + text + "\x1b[201~"
	}
	return strings.NewReplacer("\r\n", "\r", "\n", "\r").Replace(text)
}

// encodeMouse is an SGR mouse report for ev at cell (x, y) of the pane, or ""
// when the event is not one that is reported. Coordinates are 1-based.
func encodeMouse(ev tui.MouseEvent, x, y int, mode int) string {
	var b int
	switch ev.Button {
	case tui.MouseButtonLeft:
		b = 0
	case tui.MouseButtonMiddle:
		b = 1
	case tui.MouseButtonRight:
		b = 2
	case tui.MouseButtonWheelUp:
		b = 64
	case tui.MouseButtonWheelDown:
		b = 65
	case tui.MouseButtonWheelLeft:
		b = 66
	case tui.MouseButtonWheelRight:
		b = 67
	default:
		return ""
	}
	final := "M"
	switch ev.Action {
	case tui.MouseActionRelease:
		final = "m"
	case tui.MouseActionMotion:
		if mode < 2 { // a program that asked for clicks only wants no drags
			return ""
		}
		b += 32
	}
	if ev.Mod.Shift() {
		b += 4
	}
	if ev.Mod.Alt() {
		b += 8
	}
	if ev.Mod.Ctrl() {
		b += 16
	}
	return "\x1b[<" + itoa(b) + ";" + itoa(x+1) + ";" + itoa(y+1) + final
}

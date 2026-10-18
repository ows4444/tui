package main

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/ows4444/tui"
)

type kind int

const (
	kKey kind = iota
	kMouse
	kPaste
	kOther
	numKinds
)

var kindNames = [numKinds]string{"keys", "mouse", "paste", "other"}

var buttonNames = map[tui.MouseButton]string{
	tui.MouseButtonNone: "none", tui.MouseButtonLeft: "left", tui.MouseButtonMiddle: "middle", tui.MouseButtonRight: "right",
	tui.MouseButtonWheelUp: "wheel-up", tui.MouseButtonWheelDown: "wheel-down", tui.MouseButtonWheelLeft: "wheel-left",
	tui.MouseButtonWheelRight: "wheel-right", tui.MouseButtonBack: "back", tui.MouseButtonForward: "forward",
	tui.MouseButton10: "button10", tui.MouseButton11: "button11",
}

var actionNames = [...]string{"press", "release", "motion"}

func modText(m tui.Key) string {
	if m.Mod == 0 {
		return "-"
	}
	return m.Mod.String()
}

// show makes control characters and spaces visible in a string.
func show(s string, limit int) string {
	var b strings.Builder
	n := 0
	for _, r := range s {
		if n >= limit {
			b.WriteString("…")
			break
		}
		switch {
		case r == '\n':
			b.WriteString("⏎")
		case r == '\t':
			b.WriteString("⇥")
		case r == ' ':
			b.WriteString("␠")
		case unicode.IsControl(r):
			fmt.Fprintf(&b, "\\x%02x", r)
		default:
			b.WriteRune(r)
		}
		n++
	}
	return b.String()
}

// keyText describes a key press, repeat or release.
func keyText(verb string, k tui.Key) string {
	text := ""
	if k.Text != "" {
		text = fmt.Sprintf(" text=%q", k.Text)
	}
	extra := ""
	if k.Keypad {
		extra = " keypad"
	}
	return fmt.Sprintf("%-7s %-18s type=%d%s code=%d mod=%s%s", verb, k.String(), int(k.Type), text, k.Code, modText(k), extra)
}

func mouseText(ev tui.MouseEvent) string {
	mod := "-"
	if ev.Mod != 0 {
		mod = ev.Mod.String()
	}
	return fmt.Sprintf("%-7s %-11s at %d,%d mod=%s clicks=%d", actionNames[ev.Action], buttonNames[ev.Button], ev.X, ev.Y, mod, ev.Clicks)
}

func pasteText(p tui.PasteEvent) string {
	flags := ""
	if p.Incomplete {
		flags += " incomplete"
	}
	if p.Truncated {
		flags += " truncated"
	}
	return fmt.Sprintf("paste   %d bytes, %d lines%s: %s", len(p.Text), strings.Count(p.Text, "\n")+1, flags, show(p.Text, 40))
}

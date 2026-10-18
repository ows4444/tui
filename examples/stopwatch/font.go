package main

import "strings"

// glyphs are digits three rows high, drawn with box characters.
var glyphs = map[rune][3]string{
	'0': {"┌─┐", "│ │", "└─┘"}, '1': {" ╷ ", " │ ", " ╵ "}, '2': {"╶─┐", "┌─┘", "└─╴"}, '3': {"╶─┐", "╶─┤", "╶─┘"}, '4': {"╷ ╷", "└─┤", "  ╵"},
	'5': {"┌─╴", "└─┐", "╶─┘"}, '6': {"┌─╴", "├─┐", "└─┘"}, '7': {"╶─┐", "  │", "  ╵"}, '8': {"┌─┐", "├─┤", "└─┘"}, '9': {"┌─┐", "└─┤", "╶─┘"},
	':': {" ", ":", " "}, '.': {" ", " ", "."},
}

// big draws s (digits, colon and full stop) three rows high. A character it
// has no glyph for is left out.
func big(s string) []string {
	rows := make([]string, 3)
	for _, r := range s {
		g, ok := glyphs[r]
		if !ok {
			continue
		}
		for i := range rows {
			if rows[i] != "" {
				rows[i] += " "
			}
			rows[i] += g[i]
		}
	}
	return rows
}

func bigWidth(s string) int { return len([]rune(strings.TrimRight(big(s)[0], " "))) }

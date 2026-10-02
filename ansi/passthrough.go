package ansi

import (
	"os"
	"strings"
)

// screenChunk is the longest payload GNU screen accepts in one DCS string; it
// drops longer ones, so Passthrough splits a payload into pieces of this size.
const screenChunk = 768

// Passthrough wraps seq, a DCS, APC or other string sequence meant for the
// outer terminal, so a terminal multiplexer forwards it instead of dropping it.
// It reads the environment through getenv (nil means os.Getenv):
//
//   - $TMUX set: seq becomes a "DCS tmux;" string, every ESC in seq doubled.
//     tmux also needs "set -g allow-passthrough on" to forward it.
//   - else $STY set: seq becomes DCS strings, split every 768 bytes, which is
//     how GNU screen passes data through.
//   - otherwise seq is returned unchanged.
//
// Wrap the whole sequence once; wrapping twice nests the wrapper.
func Passthrough(seq string, getenv func(string) string) string {
	if seq == "" {
		return seq
	}
	if getenv == nil {
		getenv = os.Getenv
	}
	switch {
	case getenv("TMUX") != "":
		return "\x1bPtmux;" + strings.ReplaceAll(seq, "\x1b", "\x1b\x1b") + "\x1b\\"
	case getenv("STY") != "":
		var b strings.Builder
		for rest := seq; rest != ""; {
			n := min(len(rest), screenChunk)
			b.WriteString("\x1bP" + rest[:n] + "\x1b\\")
			rest = rest[n:]
		}
		return b.String()
	}
	return seq
}

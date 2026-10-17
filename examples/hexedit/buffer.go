package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

const undoMax = 1000

// op is one change: at pos, the bytes old became the bytes new. join marks an
// op that belongs with the one before it (the second half of a byte typed in
// two keystrokes), so undo takes both back together.
type op struct {
	pos      int
	old, new []byte
	join     bool
}

// buffer is the bytes being edited, with the history to undo them. Its methods
// change it in place; the model keeps one buffer and never copies the bytes.
type buffer struct {
	data       []byte
	undo, redo []op
}

func (b *buffer) apply(pos, n int, with []byte) {
	tail := append([]byte(nil), b.data[pos+n:]...)
	b.data = append(append(b.data[:pos], with...), tail...)
}

// replace swaps the n bytes at pos for with, and records it.
func (b *buffer) replace(pos, n int, with []byte, join bool) {
	o := op{pos: pos, old: append([]byte(nil), b.data[pos:pos+n]...), new: append([]byte(nil), with...), join: join}
	b.apply(pos, n, with)
	b.undo = append(b.undo, o)
	if len(b.undo) > undoMax {
		b.undo = b.undo[1:]
	}
	b.redo = nil
}

// undoStep reverses the last change (and the one it is joined to) and reports
// where the cursor belongs, or false if there is nothing to undo.
func (b *buffer) undoStep() (int, bool) {
	if len(b.undo) == 0 {
		return 0, false
	}
	at := 0
	for len(b.undo) > 0 {
		o := b.undo[len(b.undo)-1]
		b.undo = b.undo[:len(b.undo)-1]
		b.apply(o.pos, len(o.new), o.old)
		b.redo = append(b.redo, o)
		at = o.pos
		if !o.join {
			break
		}
	}
	return at, true
}

func (b *buffer) redoStep() (int, bool) {
	if len(b.redo) == 0 {
		return 0, false
	}
	at := 0
	for len(b.redo) > 0 {
		o := b.redo[len(b.redo)-1]
		b.redo = b.redo[:len(b.redo)-1]
		b.apply(o.pos, len(o.old), o.new)
		b.undo = append(b.undo, o)
		at = o.pos + len(o.new)
		if len(b.redo) == 0 || !b.redo[len(b.redo)-1].join {
			break
		}
	}
	return at, true
}

// ---- search and parsing ----------------------------------------------------

// parsePattern reads what the person typed to search for: hex bytes
// ("de ad be ef" or "deadbeef"), or text in double quotes ("\"GIF89a\"").
func parsePattern(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, errors.New("type bytes in hex, or text in quotes")
	}
	if strings.HasPrefix(s, `"`) {
		t, err := strconv.Unquote(s)
		if err != nil {
			return nil, errors.New("text needs a closing quote")
		}
		return []byte(t), nil
	}
	compact := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, strings.TrimPrefix(strings.ToLower(s), "0x"))
	if len(compact)%2 != 0 {
		return nil, errors.New("hex bytes come in pairs")
	}
	out := make([]byte, len(compact)/2)
	for i := range out {
		v, err := strconv.ParseUint(compact[2*i:2*i+2], 16, 8)
		if err != nil {
			return nil, fmt.Errorf("%q is not a hex byte", compact[2*i:2*i+2])
		}
		out[i] = byte(v)
	}
	return out, nil
}

// parseOffset reads a place in the file: hex by default ("1f0", "0x1f0"),
// decimal with a leading #, relative to the cursor with + or -.
func parseOffset(s string, cur, size int) (int, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	sign := 0
	switch {
	case s == "end" || s == "$":
		return max(0, size-1), nil
	case strings.HasPrefix(s, "+"):
		sign, s = 1, s[1:]
	case strings.HasPrefix(s, "-"):
		sign, s = -1, s[1:]
	}
	base := 16
	if strings.HasPrefix(s, "#") {
		base, s = 10, s[1:]
	}
	s = strings.TrimPrefix(s, "0x")
	n, err := strconv.ParseInt(s, base, 64)
	if err != nil || n < 0 {
		return 0, errors.New("type an offset in hex, #123 in decimal, +n or -n, or end")
	}
	pos := int(n)
	if sign != 0 {
		pos = cur + sign*pos
	}
	return max(0, min(pos, max(0, size-1))), nil
}

// find returns the first place at or after from where pat occurs, wrapping
// round to the start; -1 if it is nowhere. back searches backwards instead.
func find(data, pat []byte, from int, back bool) int {
	if len(pat) == 0 || len(data) < len(pat) {
		return -1
	}
	if !back {
		if i := bytes.Index(data[min(from, len(data)):], pat); i >= 0 {
			return from + i
		}
		return bytes.Index(data, pat)
	}
	end := min(from+len(pat)-1, len(data))
	if i := bytes.LastIndex(data[:end], pat); i >= 0 {
		return i
	}
	return bytes.LastIndex(data, pat)
}

// ---- showing bytes ---------------------------------------------------------

func printable(b byte) bool { return b >= 0x20 && b < 0x7f }

func asciiOf(b byte) string {
	if printable(b) {
		return string(rune(b))
	}
	return "·"
}

// hexString is bytes as "de ad be ef".
func hexString(b []byte) string {
	parts := make([]string, len(b))
	for i, x := range b {
		parts[i] = fmt.Sprintf("%02x", x)
	}
	return strings.Join(parts, " ")
}

// asText is bytes as text, with anything not printable shown as a dot.
func asText(b []byte) string {
	var sb strings.Builder
	for _, x := range b {
		sb.WriteString(asciiOf(x))
	}
	return sb.String()
}

// inspect reads the bytes at pos as numbers, the way a data inspector does.
func inspect(data []byte, pos int) []string {
	if pos < 0 || pos >= len(data) {
		return []string{fmt.Sprintf("offset  0x%08x", pos), "(end of file)"}
	}
	b := data[pos]
	out := []string{
		fmt.Sprintf("offset  0x%08x (%d)", pos, pos),
		fmt.Sprintf("hex     0x%02x", b),
		fmt.Sprintf("u8      %d", b),
		fmt.Sprintf("i8      %d", int8(b)),
		fmt.Sprintf("binary  %08b", b),
		fmt.Sprintf("char    %s", asciiOf(b)),
	}
	rest := data[pos:]
	if len(rest) >= 2 {
		out = append(out, fmt.Sprintf("u16 le  %d", binary.LittleEndian.Uint16(rest)), fmt.Sprintf("u16 be  %d", binary.BigEndian.Uint16(rest)))
	}
	if len(rest) >= 4 {
		le := binary.LittleEndian.Uint32(rest)
		out = append(out, fmt.Sprintf("u32 le  %d", le), fmt.Sprintf("u32 be  %d", binary.BigEndian.Uint32(rest)),
			fmt.Sprintf("i32 le  %d", int32(le)), fmt.Sprintf("f32 le  %g", math.Float32frombits(le)))
	}
	return out
}

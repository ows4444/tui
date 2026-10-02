package input

import (
	"bufio"
	"errors"
	"io"
	"os"
	"time"
	"unicode/utf8"
)

// Event is whatever ReadEvent decodes next: a Key, a MouseEvent, or a
// PasteEvent. There's no exported interface to implement — a type switch
// on the concrete type is the intended way to handle it, the same way
// tui.Msg is handled in a Model's Update.
type Event = any

// Reader decodes a raw byte stream (a terminal in raw mode) into Events.
type Reader struct {
	r        *bufio.Reader
	src      io.Reader
	escWait  time.Duration
	pending  chan error // in-flight fallback peek from a timed-out ESC wait
	deadline deadliner  // non-nil when src supports read deadlines

	reportEvents bool // kitty event types surface as Key.Action; see SetReportEvents
	reportCSI    bool // CSI replies surface as ReplyEvent{Kind: '['}; see SetReportCSIReplies
}

// DefaultEscTimeout is how long a lone ESC waits for the rest of an escape
// sequence before it is reported as the Escape key.
const DefaultEscTimeout = 30 * time.Millisecond

type deadliner interface{ SetReadDeadline(time.Time) error }

// NewReader wraps r, buffering reads for decoding. If r supports
// SetReadDeadline (an *os.File, net.Conn) the ESC timeout uses it; otherwise
// a single background peek is used, so any io.Reader works.
func NewReader(r io.Reader) *Reader {
	rd := &Reader{r: bufio.NewReaderSize(r, 256), src: r, escWait: DefaultEscTimeout}
	if d, ok := r.(deadliner); ok {
		rd.deadline = d
	}
	return rd
}

// SetEscTimeout sets how long a lone ESC waits for a following byte. Zero or
// negative disables waiting (a bare ESC with nothing buffered is Escape).
func (rd *Reader) SetEscTimeout(d time.Duration) { rd.escWait = d }

// SetReportEvents chooses how a kitty-protocol key event type is decoded.
// Off (the default), a key release decodes as KeyUnknown and a repeat as a
// plain press, exactly as before the option existed. On, they decode as the
// key itself with Key.Action set to KeyRelease or KeyRepeat.
func (rd *Reader) SetReportEvents(on bool) { rd.reportEvents = on }

// applyAction applies the kitty event type (second value of the second
// parameter group, 1 press, 2 repeat, 3 release) to a decoded Key.
func (rd *Reader) applyAction(ev Event, groups [][]int) Event {
	k, ok := ev.(Key)
	if !ok {
		return ev
	}
	et := 1
	if len(groups) >= 2 && len(groups[1]) >= 2 {
		et = groups[1][1]
	}
	switch et {
	case 3:
		if !rd.reportEvents || k.Type == KeyUnknown {
			return Key{Type: KeyUnknown}
		}
		k.Action = KeyRelease
	case 2:
		if rd.reportEvents {
			k.Action = KeyRepeat
		}
	}
	return k
}

// EscTimeout reports the current ESC timeout.
func (rd *Reader) EscTimeout() time.Duration { return rd.escWait }

// waitMore reports whether at least one more byte is available within the
// ESC timeout. A false result leaves no byte consumed.
func (rd *Reader) waitMore() bool {
	if rd.r.Buffered() > 0 {
		return true
	}
	if rd.escWait <= 0 {
		return false
	}
	return rd.waitBytes(1, rd.escWait)
}

// waitBytes reports whether at least n bytes are buffered or arrive within d.
// A false result leaves no byte consumed; a Peek still blocked at the
// deadline is joined by the next ReadEvent (see joinPending).
func (rd *Reader) waitBytes(n int, d time.Duration) bool {
	ok, _ := rd.waitBytesErr(n, d)
	return ok
}

// waitBytesErr is waitBytes that also returns the read error when the stream
// ended or failed (nil when the wait merely timed out).
func (rd *Reader) waitBytesErr(n int, d time.Duration) (bool, error) {
	if rd.r.Buffered() >= n {
		return true, nil
	}
	if rd.deadline != nil {
		if rd.deadline.SetReadDeadline(time.Now().Add(d)) == nil {
			_, err := rd.r.Peek(n)
			_ = rd.deadline.SetReadDeadline(time.Time{})
			if errors.Is(err, os.ErrDeadlineExceeded) {
				err = nil
			}
			return err == nil && rd.r.Buffered() >= n, err
		}
		rd.deadline = nil // deadlines unsupported on this file (e.g. blocking tty)
	}
	ch := make(chan error, 1)
	go func() { _, err := rd.r.Peek(n); ch <- err }()
	select {
	case err := <-ch:
		return err == nil, err
	case <-time.After(d):
		rd.pending = ch // the peek is still blocked; the next read must join it
		return false, nil
	}
}

// joinPending waits for a peek left in flight by a timed-out waitMore, so the
// bufio.Reader is never used from two goroutines.
func (rd *Reader) joinPending() {
	if rd.pending != nil {
		<-rd.pending
		rd.pending = nil
	}
}

// ReadEvent blocks until the next event is available.
func (rd *Reader) ReadEvent() (Event, error) {
	rd.joinPending()
	b, err := rd.r.ReadByte()
	if err != nil {
		return Key{}, err
	}

	switch {
	case b == 0x1b:
		return rd.parseEscape()
	case b == 0x03:
		return c0Key(b), nil
	case b == 0x0d, b == 0x0a:
		return Key{Type: KeyEnter}, nil
	case b == 0x09:
		return Key{Type: KeyTab}, nil
	case b == 0x7f, b == 0x08:
		return Key{Type: KeyBackspace}, nil
	case b == 0x20:
		return Key{Type: KeySpace, Text: " ", Code: ' '}, nil
	case b < 0x20:
		return c0Key(b), nil
	default:
		r, err := rd.decodeRune(b)
		if err != nil {
			return Key{}, err
		}
		return runeKey(r, ModNone), nil
	}
}

// c0Key is the key for a control byte other than Enter, Tab, Backspace and
// Escape: ctrl+letter and the ctrl+ punctuation range.
func c0Key(b byte) Key {
	switch {
	case b == 0x03:
		return Key{Type: KeyCtrlC}
	case b == 0x00:
		return Key{Type: KeyCtrl, Code: ' '}
	case b >= 0x1c && b <= 0x1f:
		// ctrl+\ ctrl+] ctrl+^ ctrl+_
		return Key{Type: KeyCtrl, Code: rune(b - 0x1c + '\\')}
	default:
		return Key{Type: KeyCtrl, Code: rune(b + 'a' - 1)}
	}
}

// decodeRune completes a UTF-8 sequence that started with first. Invalid
// input (stray continuation byte, bad lead byte, truncated or overlong
// sequence) yields U+FFFD and never consumes a byte that is not a
// continuation byte.
func (rd *Reader) decodeRune(first byte) (rune, error) {
	var n int
	switch {
	case first < 0x80:
		return rune(first), nil
	case first&0xE0 == 0xC0:
		n = 1
	case first&0xF0 == 0xE0:
		n = 2
	case first&0xF8 == 0xF0:
		n = 3
	default:
		return utf8.RuneError, nil
	}
	buf := make([]byte, 1, n+1)
	buf[0] = first
	for i := 0; i < n; i++ {
		p, err := rd.r.Peek(1)
		if err != nil || p[0]&0xC0 != 0x80 {
			return utf8.RuneError, nil
		}
		buf = append(buf, p[0])
		_, _ = rd.r.ReadByte()
	}
	r, size := utf8.DecodeRune(buf)
	if r == utf8.RuneError || size != len(buf) {
		return utf8.RuneError, nil
	}
	return r, nil
}

// parseEscape handles a byte after a leading ESC (0x1b). Terminals write an
// entire escape sequence in one syscall, so if nothing else is buffered yet
// it's a bare Escape keypress rather than the start of a sequence.
//
// Known limitations: ESC followed by a letter that is already buffered is
// indistinguishable from Alt+letter, so a fast Esc-then-key while the reader
// is behind reads as Alt+key; and a sequence split across reads reads as a
// bare Esc. Neither is fixable without a timed read.
func (rd *Reader) parseEscape() (Event, error) {
	if !rd.waitMore() {
		return Key{Type: KeyEsc}, nil
	}
	b1, err := rd.r.ReadByte()
	if err != nil {
		return Key{Type: KeyEsc}, nil
	}

	switch b1 {
	case '[':
		return rd.parseCSI()
	case 'O':
		return rd.parseSS3()
	case ']':
		// OSC strings start with a digit. Decide with the same timed wait as
		// ESC, never a blocking Peek: a lone Alt+] must not swallow the next
		// keys typed later.
		if rd.waitMore() {
			if next, err := rd.r.Peek(1); err == nil && next[0] >= '0' && next[0] <= '9' {
				return rd.parseString(']')
			}
		}
		return runeKey(']', ModAlt), nil
	case 0x0d, 0x0a:
		return Key{Type: KeyEnter, Mod: ModAlt}, nil
	case 0x7f, 0x08:
		return Key{Type: KeyBackspace, Mod: ModAlt}, nil
	case 0x09:
		return Key{Type: KeyTab, Mod: ModAlt}, nil
	case 0x1b:
		// ESC ESC [ A / ESC ESC O P: legacy Alt+<sequence>.
		if rd.waitMore() {
			if next, err := rd.r.Peek(1); err == nil && (next[0] == '[' || next[0] == 'O') {
				ev, err := rd.parseEscape()
				if k, ok := ev.(Key); ok && k.Type != KeyEsc {
					k.Mod |= ModAlt
					return k, err
				}
				return ev, err
			}
		}
		return Key{Type: KeyEsc, Mod: ModAlt}, nil
	default:
		if b1 < 0x20 {
			// ESC + control byte: Alt+Ctrl+letter, the same key as the bare
			// control byte with Alt added.
			k := c0Key(b1)
			k.Mod |= ModAlt
			return k, nil
		}
		if b1 >= 0x21 && b1 <= 0x2f && rd.waitMore() {
			// nF sequence (charset designation ESC ( B, ESC # 8, ...):
			// intermediates 0x20-0x2f, then one final byte 0x30-0x7e. It is
			// consumed whole, not read as Alt+punctuation followed by text.
			return rd.parseNF()
		}
		if b1 < 0x80 {
			// DCS/SOS/PM/APC replies arrive in one write, so the body is
			// already buffered; a lone Alt+Shift+letter is not.
			if (b1 == 'P' || b1 == 'X' || b1 == '^' || b1 == '_') && rd.r.Buffered() > 0 {
				return rd.parseString(b1)
			}
			return runeKey(rune(b1), ModAlt), nil
		}
		r, err := rd.decodeRune(b1)
		if err != nil {
			return Key{Type: KeyEsc}, nil
		}
		return runeKey(r, ModAlt), nil
	}
}

// parseNF consumes the rest of an nF escape sequence after its first
// intermediate byte. A byte that cannot continue it is left unread.
func (rd *Reader) parseNF() (Event, error) {
	for {
		p, err := rd.r.Peek(1)
		if err != nil {
			return Key{Type: KeyUnknown}, nil
		}
		switch {
		case p[0] >= 0x20 && p[0] <= 0x2f:
			_, _ = rd.r.ReadByte()
		case p[0] >= 0x30 && p[0] <= 0x7e:
			_, _ = rd.r.ReadByte()
			return Key{Type: KeyUnknown}, nil
		default:
			return Key{Type: KeyUnknown}, nil
		}
	}
}

// parseSS3 handles ESC O x sequences (function keys on some terminals).
func (rd *Reader) parseSS3() (Key, error) {
	b, err := rd.r.ReadByte()
	if err != nil {
		return Key{Type: KeyEsc}, nil
	}
	// Optional numeric modifier parameters: ESC O 5 P, ESC O 1;5 P.
	var mod Mod
	if b >= '0' && b <= '9' || b == ';' {
		var params []byte
		for b >= '0' && b <= '9' || b == ';' {
			params = append(params, b)
			if b, err = rd.r.ReadByte(); err != nil {
				return Key{Type: KeyUnknown}, nil
			}
		}
		last := params
		for i, c := range params {
			if c == ';' {
				last = params[i+1:]
			}
		}
		n := 0
		for _, c := range last {
			if c == ';' {
				break
			}
			if n < 1<<20 {
				n = n*10 + int(c-'0')
			}
		}
		mod = modFromXterm(n)
	}
	k := rd.ss3Key(b)
	if k.Type != KeyUnknown {
		k.Mod = mod
	}
	return k, nil
}

func (rd *Reader) ss3Key(b byte) Key {
	switch b {
	case 'P':
		return Key{Type: KeyF1}
	case 'Q':
		return Key{Type: KeyF2}
	case 'R':
		return Key{Type: KeyF3}
	case 'S':
		return Key{Type: KeyF4}
	case 'A':
		return Key{Type: KeyUp}
	case 'B':
		return Key{Type: KeyDown}
	case 'C':
		return Key{Type: KeyRight}
	case 'D':
		return Key{Type: KeyLeft}
	case 'H':
		return Key{Type: KeyHome}
	case 'F':
		return Key{Type: KeyEnd}
	default:
		return Key{Type: KeyUnknown}
	}
}

// parseCSI handles ESC [ ... sequences: arrows, home/end, insert, delete,
// page up/down, function keys, kitty CSI-u keys, SGR mouse reports, and
// bracketed-paste start markers. Every sequence is consumed through its
// final byte (ECMA-48: parameter bytes 0x30-0x3f, intermediate bytes
// 0x20-0x2f, final byte 0x40-0x7e) even when it isn't recognised, so replies
// such as DA1 (ESC[?62;22c) never leak out as literal keypresses.
func (rd *Reader) parseCSI() (Event, error) {
	b, err := rd.r.ReadByte()
	if err != nil {
		return Key{Type: KeyEsc}, nil
	}
	if b == '<' {
		return rd.parseSGRMouse()
	}
	if b == 'M' {
		// X10/normal mouse: ESC [ M Cb Cx Cy, three raw bytes.
		for i := 0; i < 3; i++ {
			if _, err := rd.r.ReadByte(); err != nil {
				break
			}
		}
		return Key{Type: KeyUnknown}, nil
	}

	var params []byte
	var inter bool
	var raw []byte // every parameter/intermediate byte, for SetReportCSIReplies
	for b < 0x40 {
		if b >= 0x20 && rd.reportCSI {
			raw = append(raw, b)
		}
		// A private prefix (? > < =) makes "$y" a DECRPM reply, not rxvt.
		if b == '$' && len(params) > 0 && !inter && params[0] < '<' {
			// rxvt shifted keys end in '$' (ESC[23$): no further final byte.
			return rd.rxvtKey(params, ModShift)
		}
		switch {
		case b >= 0x30:
			params = append(params, b)
		case b >= 0x20:
			inter = true
		default:
			// A control byte aborts the sequence.
			return Key{Type: KeyUnknown}, nil
		}
		b, err = rd.r.ReadByte()
		if err != nil {
			return Key{Type: KeyUnknown}, nil
		}
	}
	final := b
	if final == 0x7f || final > 0x7e {
		return Key{Type: KeyUnknown}, nil
	}
	if final == '^' || final == '@' {
		// rxvt ctrl (^) and ctrl+shift (@) keys: ESC[23^ / ESC[23@.
		if !inter && len(params) > 0 {
			m := ModCtrl
			if final == '@' {
				m |= ModShift
			}
			return rd.rxvtKey(params, m)
		}
	}
	// Private-parameter prefixes (? > < =) and intermediates mark replies
	// and commands that are not keys.
	if inter || (len(params) > 0 && params[0] >= '<' && params[0] <= '?') {
		if rd.reportCSI {
			return ReplyEvent{Kind: '[', Data: string(raw) + string(final)}, nil
		}
		return Key{Type: KeyUnknown}, nil
	}

	// GROUP (';' GROUP)*, each GROUP being NUM (':' NUM)*: e.g. "3" (Delete),
	// "1;5" (xterm modifier code in the second group), or a kitty
	// "97;6" whose groups can carry ':' sub-values.
	groups := [][]int{{0}}
	for _, c := range params {
		last := len(groups) - 1
		switch {
		case c >= '0' && c <= '9':
			n := len(groups[last]) - 1
			if v := groups[last][n]*10 + int(c-'0'); v < 1<<30 {
				groups[last][n] = v
			}
		case c == ':':
			groups[last] = append(groups[last], 0)
		case c == ';':
			groups = append(groups, []int{0})
		default:
			return Key{Type: KeyUnknown}, nil
		}
	}

	if final == 'u' {
		return rd.applyAction(decodeKittyKey(groups), groups), nil
	}
	ev, err := rd.csiFinal(final, params, groups)
	return rd.applyAction(ev, groups), err
}

// rxvtKey decodes rxvt's ESC[n$ / ESC[n^ / ESC[n@ modified "~" keys.
func (rd *Reader) rxvtKey(params []byte, mod Mod) (Event, error) {
	n := 0
	for _, c := range params {
		if c < '0' || c > '9' {
			return Key{Type: KeyUnknown}, nil
		}
		if n < 1<<20 {
			n = n*10 + int(c-'0')
		}
	}
	if n == 200 {
		return Key{Type: KeyUnknown}, nil
	}
	return rd.decodeTilde(n, mod)
}

// csiFinal decodes the non-kitty CSI keys (arrows, Home/End, F-keys, ~ keys,
// focus).
func (rd *Reader) csiFinal(final byte, params []byte, groups [][]int) (Event, error) {
	nparams := len(groups)
	if len(params) == 0 {
		nparams = 0
	}
	var mod Mod
	if nparams >= 2 {
		mod = modFromXterm(groups[1][0])
	}
	p0 := groups[0][0]

	switch final {
	case 'I':
		if nparams == 0 {
			return FocusEvent{Focused: true}, nil
		}
	case 'O':
		if nparams == 0 {
			return FocusEvent{Focused: false}, nil
		}
	case '~':
		return rd.decodeTilde(p0, mod)
	case 'A':
		return Key{Type: KeyUp, Mod: mod}, nil
	case 'B':
		return Key{Type: KeyDown, Mod: mod}, nil
	case 'C':
		return Key{Type: KeyRight, Mod: mod}, nil
	case 'D':
		return Key{Type: KeyLeft, Mod: mod}, nil
	case 'H':
		return Key{Type: KeyHome, Mod: mod}, nil
	case 'F':
		return Key{Type: KeyEnd, Mod: mod}, nil
	case 'P':
		return Key{Type: KeyF1, Mod: mod}, nil
	case 'Q':
		return Key{Type: KeyF2, Mod: mod}, nil
	case 'R':
		return Key{Type: KeyF3, Mod: mod}, nil
	case 'S':
		return Key{Type: KeyF4, Mod: mod}, nil
	case 'Z':
		return Key{Type: KeyTab, Mod: mod | ModShift}, nil
	}
	return Key{Type: KeyUnknown}, nil
}

// tildeFKeys maps the xterm/VT220 "ESC [ n ~" function-key codes to key types.
var tildeFKeys = map[int]KeyType{
	11: KeyF1, 12: KeyF2, 13: KeyF3, 14: KeyF4,
	15: KeyF5, 17: KeyF6, 18: KeyF7, 19: KeyF8, 20: KeyF9,
	21: KeyF10, 23: KeyF11, 24: KeyF12,
	25: KeyF13, 26: KeyF14, 28: KeyF15, 29: KeyF16,
	31: KeyF17, 32: KeyF18, 33: KeyF19, 34: KeyF20,
}

// decodeTilde handles the ESC [ n ; mod ~ family.
func (rd *Reader) decodeTilde(code int, mod Mod) (Event, error) {
	switch code {
	case 200:
		return rd.readPaste()
	case 1, 7:
		return Key{Type: KeyHome, Mod: mod}, nil
	case 2:
		return Key{Type: KeyInsert, Mod: mod}, nil
	case 3:
		return Key{Type: KeyDelete, Mod: mod}, nil
	case 4, 8:
		return Key{Type: KeyEnd, Mod: mod}, nil
	case 5:
		return Key{Type: KeyPgUp, Mod: mod}, nil
	case 6:
		return Key{Type: KeyPgDown, Mod: mod}, nil
	}
	if t, ok := tildeFKeys[code]; ok {
		return Key{Type: t, Mod: mod}, nil
	}
	return Key{Type: KeyUnknown}, nil
}

// parseSGRMouse handles ESC [ < Cb ; Cx ; Cy M/m sequences (SGR mouse mode,
// ansi.MouseSGREnable). It consumes the full sequence regardless of
// whether it parses cleanly, for the same reason parseCSI's numeric
// parameters must always be fully consumed: leftover bytes would otherwise
// leak out as bogus literal keypresses.
func (rd *Reader) parseSGRMouse() (Event, error) {
	params := []int{0}
	for {
		b, err := rd.r.ReadByte()
		if err != nil {
			return Key{Type: KeyUnknown}, nil
		}
		switch {
		case b >= '0' && b <= '9':
			params[len(params)-1] = params[len(params)-1]*10 + int(b-'0')
		case b == ';':
			params = append(params, 0)
		case b == 'M' || b == 'm':
			if len(params) != 3 {
				return Key{Type: KeyUnknown}, nil
			}
			return decodeSGRMouse(params[0], params[1], params[2], b == 'm'), nil
		default:
			return Key{Type: KeyUnknown}, nil
		}
	}
}

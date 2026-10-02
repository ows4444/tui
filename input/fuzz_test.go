package input

import (
	"bytes"
	"testing"
)

// FuzzReadEvent feeds arbitrary byte streams into Reader.ReadEvent, the
// hand-rolled ANSI/CSI/SS3 parser that decodes raw terminal input — the
// one piece of this library that must survive genuinely adversarial bytes
// (a terminal, or anything relaying a terminal's raw input, isn't a
// trusted source). ReadEvent is documented as tolerant of malformed
// sequences (parseCSI/parseSGRMouse always fully consume a sequence's
// bytes even when it isn't recognized) — this proves that holds for
// inputs no handwritten test would think to try, not just that it
// doesn't panic.
func FuzzReadEvent(f *testing.F) {
	seeds := [][]byte{
		{},
		{0x1b},
		{0x1b, '['},
		{0x1b, '[', 'A'},
		{0x1b, 'O', 'P'},
		{0x1b, '[', '<', '0', ';', '1', ';', '1', 'M'},
		{0x1b, '[', '2', '0', '0', '~', 'h', 'i', 0x1b, '[', '2', '0', '1', '~'},
		{0x1b, '[', '2', '0', '0', '~', 'u', 'n', 't', 'e', 'r', 'm', 'i', 'n', 'a', 't', 'e', 'd'},
		{0x1b, '[', '1', ';', '5', 'A'},
		{0x1b, '[', '9', '9', '9', '9', '9', '9', '9', '9', '9', '9', '~'},
		{0x1b, 0x1b},
		{0x1b, 0x1b, '[', 'A'},
		{0x1b, 0x1b, 'O', 'P'},
		{0x1b, 0x1b, '[', '1', ';', '3', 'A'},
		{0x1b, 0x1b, 0x1b, '[', 'A'},
		{0x1b, 'O', '5', 'P'},
		{0x1b, 'O', '1', ';', '5', 'P'},
		{0x1b, 'O', ';', ';', ';'},
		{0x1b, 'O', '9', '9', '9', '9', '9', '9', '9', '9', '9', '9', 'A'},
		{0x1b, 'O', '5'},
		{0x1b, '[', '2', '3', '$'},
		{0x1b, '[', '2', '3', '$', 'x'},
		{0x1b, '[', '2', '3', '^'},
		{0x1b, '[', '2', '3', '@'},
		{0x1b, '[', '$'},
		{0x1b, '[', '2', '0', '0', '$'},
		{0xff, 0xfe, 0x80},
		[]byte("hello, 世界"),
	}
	for _, s := range seeds {
		f.Add(s)
	}
	// Every decode fix (C1, C2, C3, C6, C9) is a seed as well.
	for _, c := range decodeFixes {
		f.Add([]byte(c.in))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		rd := NewReader(bytes.NewReader(data))
		// Every ReadEvent call consumes at least one byte from a bounded
		// bufio.Reader, so this always terminates — no iteration cap needed.
		for {
			if _, err := rd.ReadEvent(); err != nil {
				return
			}
		}
	})
}

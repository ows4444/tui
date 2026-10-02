package term

import (
	"errors"
	"testing"
)

func TestRawInputMode(t *testing.T) {
	const cooked = enableProcessedInput | enableLineInput | enableEchoInput
	cases := []struct {
		name  string
		orig  uint32
		mouse bool
		want  uint32
	}{
		{"mouse off keeps quickedit", cooked | enableQuickEditMode | enableExtendedFlags, false,
			enableQuickEditMode | enableExtendedFlags | enableVirtualTerminalInput | enableWindowInput},
		{"mouse off leaves extended flags unset", cooked, false,
			enableVirtualTerminalInput | enableWindowInput},
		{"mouse on clears quickedit, sets extended", cooked | enableQuickEditMode | enableExtendedFlags, true,
			enableExtendedFlags | enableVirtualTerminalInput | enableWindowInput},
		{"mouse on, quickedit already off", cooked, true,
			enableExtendedFlags | enableVirtualTerminalInput | enableWindowInput},
		{"mouse on, quickedit without extended", cooked | enableQuickEditMode, true,
			enableExtendedFlags | enableVirtualTerminalInput | enableWindowInput},
		{"unrelated bits preserved", 0x0100 | cooked | enableQuickEditMode, true,
			0x0100 | enableExtendedFlags | enableVirtualTerminalInput | enableWindowInput},
	}
	for _, c := range cases {
		if got := rawInputMode(c.orig, c.mouse); got != c.want {
			t.Errorf("%s: rawInputMode(%#x, %v) = %#x, want %#x", c.name, c.orig, c.mouse, got, c.want)
		}
	}
}

type fakeCon struct {
	mode    uint32
	sets    []uint32
	modeErr error
	setErr  error
}

func (f *fakeCon) Mode(int) (uint32, error) { return f.mode, f.modeErr }
func (f *fakeCon) SetMode(_ int, m uint32) error {
	if f.setErr != nil {
		return f.setErr
	}
	f.sets = append(f.sets, m)
	f.mode = m
	return nil
}

func TestMakeRawOnReturnsExactOriginal(t *testing.T) {
	for _, mouse := range []bool{false, true} {
		for _, orig := range []uint32{0, 0x00E7, 0x01FF, enableQuickEditMode, enableExtendedFlags | enableQuickEditMode | enableLineInput} {
			f := &fakeCon{mode: orig}
			got, err := makeRawOn(f, 1, mouse)
			if err != nil || got != orig {
				t.Fatalf("makeRawOn(%#x, %v) = %#x, %v; want original", orig, mouse, got, err)
			}
			if len(f.sets) != 1 || f.mode != rawInputMode(orig, mouse) {
				t.Fatalf("mode = %#x after %d sets, want %#x", f.mode, len(f.sets), rawInputMode(orig, mouse))
			}
			// Restoring means writing the returned mode back verbatim.
			if err := f.SetMode(1, got); err != nil || f.mode != orig {
				t.Fatalf("restore left mode %#x, want %#x", f.mode, orig)
			}
		}
	}
}

func TestMakeRawOnErrors(t *testing.T) {
	boom := errors.New("boom")
	if _, err := makeRawOn(&fakeCon{modeErr: boom}, 1, true); !errors.Is(err, boom) {
		t.Fatalf("Mode error = %v", err)
	}
	f := &fakeCon{setErr: boom}
	if _, err := makeRawOn(f, 1, true); !errors.Is(err, boom) {
		t.Fatalf("SetMode error = %v", err)
	}
}

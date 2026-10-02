package term

import (
	"errors"
	"testing"
)

type fakeConsole struct {
	mode            uint32
	modeErr, setErr error
	sets            []uint32
}

func (f *fakeConsole) Mode(int) (uint32, error) { return f.mode, f.modeErr }
func (f *fakeConsole) SetMode(_ int, m uint32) error {
	if f.setErr != nil {
		return f.setErr
	}
	f.sets = append(f.sets, m)
	f.mode = m
	return nil
}

func TestEnableOutputVTSetsFlagAndRestores(t *testing.T) {
	f := &fakeConsole{mode: 0x0003}
	restore, err := EnableOutputVTOn(f, 1)
	if err != nil {
		t.Fatal(err)
	}
	if f.mode != 0x0003|enableVirtualTerminalProcessing {
		t.Fatalf("mode = %#x, want VT flag set", f.mode)
	}
	if err := restore(); err != nil {
		t.Fatal(err)
	}
	if f.mode != 0x0003 {
		t.Fatalf("mode after restore = %#x, want original", f.mode)
	}
}

func TestEnableOutputVTAlreadySet(t *testing.T) {
	f := &fakeConsole{mode: 0x0007}
	restore, err := EnableOutputVTOn(f, 1)
	if err != nil {
		t.Fatal(err)
	}
	_ = restore()
	if len(f.sets) != 0 {
		t.Fatalf("unexpected SetMode calls %v", f.sets)
	}
}

func TestEnableOutputVTErrors(t *testing.T) {
	boom := errors.New("boom")
	if _, err := EnableOutputVTOn(&fakeConsole{setErr: boom}, 1); !errors.Is(err, boom) || err == nil {
		t.Fatalf("set error = %v", err)
	}
	if _, err := EnableOutputVTOn(&fakeConsole{modeErr: boom}, 1); !errors.Is(err, boom) {
		t.Fatalf("mode error = %v", err)
	}
}

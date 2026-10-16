package prompt

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/picker"
)

// drive runs a prompt with keys as its input and returns what it drew.
func drive(keys string) (Config, *bytes.Buffer) {
	out := &bytes.Buffer{}
	return Config{Program: []tui.ProgramOption{
		tui.WithInput(strings.NewReader(keys)),
		tui.WithOutput(out),
		tui.WithReducedMotion(true),
	}}, out
}

func TestInputReturnsTheTypedLine(t *testing.T) {
	o, out := drive("Ada Lovelace\r")
	got, err := Input("Name:", o)
	if err != nil || got != "Ada Lovelace" {
		t.Fatalf("Input = %q, %v", got, err)
	}
	if !strings.Contains(out.String(), "Name:") {
		t.Errorf("the question was not drawn: %q", out.String())
	}
}

func TestInputDefaultAndPlaceholder(t *testing.T) {
	o, _ := drive("\r")
	o.Default = "guest"
	got, err := Input("Name:", o)
	if err != nil || got != "guest" {
		t.Errorf("default: %q, %v", got, err)
	}
}

func TestInputValidationKeepsThePromptOpen(t *testing.T) {
	o, out := drive("\rok\r")
	o.Validate = func(s string) error {
		if s == "" {
			return errors.New("required")
		}
		return nil
	}
	got, err := Input("Name:", o)
	if err != nil || got != "ok" {
		t.Fatalf("Input = %q, %v", got, err)
	}
	if !strings.Contains(out.String(), "required") {
		t.Errorf("the validation error was not shown: %q", out.String())
	}
}

func TestEscCancels(t *testing.T) {
	o, _ := drive("abc\x1b")
	if _, err := Input("Name:", o); !errors.Is(err, ErrCanceled) {
		t.Errorf("err = %v, want ErrCanceled", err)
	}
	o, _ = drive("\x03")
	if _, err := Confirm("Sure?", false, o); !errors.Is(err, ErrCanceled) {
		t.Errorf("Ctrl+C err = %v", err)
	}
}

func TestPasswordReturnsTheRealValueButDrawsItMasked(t *testing.T) {
	o, out := drive("hunter2\r")
	got, err := Password("Secret:", o)
	if err != nil || got != "hunter2" {
		t.Fatalf("Password = %q, %v", got, err)
	}
	if strings.Contains(out.String(), "hunter2") {
		t.Errorf("the secret was drawn: %q", out.String())
	}
}

func TestConfirm(t *testing.T) {
	for _, c := range []struct {
		keys string
		def  bool
		want bool
	}{
		{"y", false, true},
		{"n", true, false},
		{"\r", true, true},
		{"\r", false, false},
	} {
		o, _ := drive(c.keys)
		got, err := Confirm("Deploy?", c.def, o)
		if err != nil || got != c.want {
			t.Errorf("keys %q def %v: %v, %v; want %v", c.keys, c.def, got, err, c.want)
		}
	}
}

func TestSelectReturnsTheIndexAndFilters(t *testing.T) {
	o, _ := drive("\x1b[B\r") // Down, Enter
	i, err := Select("Region:", []string{"eu", "us", "ap"}, o)
	if err != nil || i != 1 {
		t.Fatalf("Select = %d, %v", i, err)
	}
	o, _ = drive("ap\r") // type to narrow, Enter
	i, err = Select("Region:", []string{"eu", "us", "ap"}, o)
	if err != nil || i != 2 {
		t.Errorf("filtered Select = %d, %v; want 2", i, err)
	}
	if _, err := Select("x", nil); err == nil {
		t.Error("Select with no choices did not fail")
	}
}

func TestMultiSelect(t *testing.T) {
	o, _ := drive(" \x1b[B\x1b[B \r") // tick 0, down twice, tick 2, confirm
	got, err := MultiSelect("Tools:", []string{"a", "b", "c"}, o)
	if err != nil || len(got) != 2 || got[0] != 0 || got[1] != 2 {
		t.Errorf("MultiSelect = %v, %v", got, err)
	}
}

func TestSpinRunsTheFunctionAndReturnsItsError(t *testing.T) {
	boom := errors.New("boom")
	err := Spin("Working", func(ctx context.Context) error {
		time.Sleep(20 * time.Millisecond)
		return boom
	}, Config{Program: []tui.ProgramOption{
		tui.WithInput(blockingReader{}),
		tui.WithOutput(&bytes.Buffer{}),
		tui.WithReducedMotion(true),
	}})
	if !errors.Is(err, boom) {
		t.Errorf("Spin err = %v, want boom", err)
	}
}

// blockingReader never returns data, so the Program stays up until the work
// finishes (a reader at EOF would quit it at once).
type blockingReader struct{}

func (blockingReader) Read([]byte) (int, error) {
	time.Sleep(time.Hour)
	return 0, nil
}

func TestSelectAndMultiSelectCancel(t *testing.T) {
	o, _ := drive("\x1b")
	if _, err := Select("x", []string{"a", "b"}, o); !errors.Is(err, ErrCanceled) {
		t.Errorf("Select err = %v", err)
	}
	o, _ = drive("\x03")
	if _, err := MultiSelect("x", []string{"a", "b"}, o); !errors.Is(err, ErrCanceled) {
		t.Errorf("MultiSelect err = %v", err)
	}
	if _, err := MultiSelect("x", nil); err == nil {
		t.Error("MultiSelect with no choices did not fail")
	}
}

// Esc with a query typed clears the query and does not cancel; with none it
// cancels. (Driven by messages: ESC then a byte reads as Alt+key in a stream.)
func TestEscClearsAQueryBeforeItCancels(t *testing.T) {
	p := picker.NewStrings("a", "b")
	p.Filter = true
	var m tui.Model = selectModel{p: p}
	for _, r := range "zz" {
		m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: string(r), Code: r})
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyEsc})
	sm := m.(selectModel)
	if sm.canceled || sm.p.Query() != "" {
		t.Fatalf("first Esc: canceled %v query %q", sm.canceled, sm.p.Query())
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyEsc})
	if !m.(selectModel).canceled {
		t.Error("second Esc did not cancel")
	}
}

func TestPasswordValidationAndCancel(t *testing.T) {
	o, _ := drive("ab\rabcdefgh\r")
	o.Validate = func(s string) error {
		if len(s) < 8 {
			return errors.New("too short")
		}
		return nil
	}
	got, err := Password("pw:", o) // "ab" stays after the refusal and the rest is added to it
	if err != nil || got != "ababcdefgh" {
		t.Errorf("Password = %q, %v", got, err)
	}
	o, _ = drive("\x1b")
	if _, err := Password("pw:", o); !errors.Is(err, ErrCanceled) {
		t.Errorf("err = %v", err)
	}
}

func TestSpinCancelStopsTheFunction(t *testing.T) {
	cfg := Config{Program: []tui.ProgramOption{
		tui.WithInput(&oneThenBlock{data: "\x03"}),
		tui.WithOutput(&bytes.Buffer{}),
		tui.WithReducedMotion(true),
	}}
	err := Spin("Working", func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}, cfg)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Spin err = %v, want context.Canceled", err)
	}
}

// oneThenBlock returns data once and then blocks, so the Program is not
// ended by the reader reaching EOF while the work is still running.
type oneThenBlock struct {
	data string
	sent bool
}

func (r *oneThenBlock) Read(p []byte) (int, error) {
	if !r.sent {
		r.sent = true
		return copy(p, r.data), nil
	}
	time.Sleep(time.Hour)
	return 0, nil
}

// A label or an error from the caller is drawn as text: an escape sequence in
// it does not reach the terminal.
func TestLabelsAndErrorsAreSanitised(t *testing.T) {
	o, out := drive("\rx\r")
	o.Validate = func(s string) error {
		if s == "" {
			return errors.New("bad\x1b[2Jinput")
		}
		return nil
	}
	if _, err := Input("Name\x1b]0;pwned\x07:", o); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"\x1b]0;pwned", "\x1b[2J"} {
		if strings.Contains(out.String(), bad) {
			t.Errorf("output holds the caller's escape %q", bad)
		}
	}
}

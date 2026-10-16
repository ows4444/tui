package main

import (
	"bytes"
	"errors"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui"
)

// withKeys makes the prompts read keys and draw into a buffer, for one test.
func withKeys(t *testing.T, keys string) *bytes.Buffer {
	t.Helper()
	drawn := &bytes.Buffer{}
	old := ui
	ui = func() []tui.ProgramOption {
		return []tui.ProgramOption{tui.WithInput(strings.NewReader(keys)), tui.WithOutput(drawn), tui.WithReducedMotion(true)}
	}
	t.Cleanup(func() { ui = old })
	return drawn
}

// blockingReader never returns data, so a Program reading it stays up until
// the work finishes (a reader at EOF would quit it at once).
type blockingReader struct{}

func (blockingReader) Read([]byte) (int, error) {
	time.Sleep(time.Hour)
	return 0, nil
}

// withIdleInput is withKeys for prompts that need no keys.
func withIdleInput(t *testing.T) {
	t.Helper()
	old := ui
	ui = func() []tui.ProgramOption {
		return []tui.ProgramOption{tui.WithInput(blockingReader{}), tui.WithOutput(&bytes.Buffer{}), tui.WithReducedMotion(true)}
	}
	t.Cleanup(func() { ui = old })
}

func exitCode(err error) int {
	var ee *exitError
	if errors.As(err, &ee) {
		return ee.code
	}
	if err != nil {
		return -1
	}
	return 0
}

func TestPromptInputPrintsTheAnswerOnly(t *testing.T) {
	drawn := withKeys(t, "Ada\r")
	var out bytes.Buffer
	if err := run([]string{"prompt", "input", "Name:"}, &out); err != nil {
		t.Fatal(err)
	}
	if out.String() != "Ada\n" {
		t.Errorf("stdout = %q, want just the answer", out.String())
	}
	if !strings.Contains(drawn.String(), "Name:") {
		t.Errorf("the question was not drawn on the UI writer: %q", drawn.String())
	}
}

func TestPromptInputDefault(t *testing.T) {
	withKeys(t, "\r")
	var out bytes.Buffer
	if err := run([]string{"prompt", "input", "-default", "guest", "Name:"}, &out); err != nil || out.String() != "guest\n" {
		t.Errorf("out=%q err=%v", out.String(), err)
	}
}

func TestPromptConfirmUsesTheExitCode(t *testing.T) {
	var out bytes.Buffer
	withKeys(t, "y")
	if err := run([]string{"prompt", "confirm", "Sure?"}, &out); err != nil {
		t.Errorf("yes: %v", err)
	}
	withKeys(t, "n")
	if got := exitCode(run([]string{"prompt", "confirm", "Sure?"}, &out)); got != exitNo {
		t.Errorf("no: exit %d, want %d", got, exitNo)
	}
	withKeys(t, "\r")
	if err := run([]string{"prompt", "confirm", "-yes", "Sure?"}, &out); err != nil {
		t.Errorf("-yes with Enter: %v", err)
	}
	if out.Len() != 0 {
		t.Errorf("confirm printed %q", out.String())
	}
}

func TestPromptSelectAndMultiSelectPrintTheChoices(t *testing.T) {
	var out bytes.Buffer
	withKeys(t, "\x1b[B\r") // Down, Enter
	if err := run([]string{"prompt", "select", "Region", "eu", "us", "ap"}, &out); err != nil || out.String() != "us\n" {
		t.Errorf("select: out=%q err=%v", out.String(), err)
	}
	out.Reset()
	withKeys(t, " \x1b[B\x1b[B \r") // tick eu, down twice, tick ap, Enter
	if err := run([]string{"prompt", "multiselect", "Regions", "eu", "us", "ap"}, &out); err != nil || out.String() != "eu\nap\n" {
		t.Errorf("multiselect: out=%q err=%v", out.String(), err)
	}
}

func TestPromptCancelIs130AndQuiet(t *testing.T) {
	withKeys(t, "\x1b")
	var out, errOut bytes.Buffer
	if code := cli([]string{"prompt", "input", "Name:"}, &out, &errOut); code != exitCanceled {
		t.Errorf("exit = %d, want %d", code, exitCanceled)
	}
	if out.Len() != 0 || errOut.Len() != 0 {
		t.Errorf("a canceled prompt printed: out=%q err=%q", out.String(), errOut.String())
	}
}

func TestPromptSpinPassesTheExitCodeThrough(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	var out bytes.Buffer
	withIdleInput(t)
	if err := run([]string{"prompt", "spin", "working", "--", "sh", "-c", "exit 0"}, &out); err != nil {
		t.Errorf("success: %v", err)
	}
	withIdleInput(t)
	if got := exitCode(run([]string{"prompt", "spin", "working", "--", "sh", "-c", "exit 3"}, &out)); got != 3 {
		t.Errorf("failing command: exit %d, want 3", got)
	}
	withIdleInput(t)
	if got := exitCode(run([]string{"prompt", "spin", "working", "--", "no-such-command-xyz"}, &out)); got == 0 {
		t.Error("a missing command did not fail")
	}
}

func TestPromptRefusesBadUsage(t *testing.T) {
	var out bytes.Buffer
	for _, args := range [][]string{
		{"prompt"}, {"prompt", "nope"}, {"prompt", "input"}, {"prompt", "select", "label"},
		{"prompt", "multiselect", "label"}, {"prompt", "spin", "label"}, {"prompt", "spin", "--", "true"},
		{"prompt", "input", "-nope", "x"},
	} {
		if err := run(args, &out); err == nil {
			t.Errorf("run(%q) did not fail", args)
		}
	}
}

func TestSplitDash(t *testing.T) {
	l, c, ok := splitDash([]string{"build", "the", "app", "--", "go", "build", "--", "x"})
	if !ok || l != "build the app" || strings.Join(c, " ") != "go build -- x" {
		t.Errorf("got %q %q %v", l, c, ok)
	}
}

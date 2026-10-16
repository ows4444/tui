package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/prompt"
)

// keys feeds question i its own input, so each prompt's Program reads only its
// own keys.
func keys(per ...string) func(int) prompt.Config {
	return func(i int) prompt.Config {
		return prompt.Config{Program: []tui.ProgramOption{
			tui.WithInput(strings.NewReader(per[i])),
			tui.WithOutput(io.Discard),
			tui.WithReducedMotion(true),
		}}
	}
}

func TestAskCollectsEveryAnswer(t *testing.T) {
	a, err := ask(keys(
		"demo-app\r", // name
		"\x1b[B\r",   // Down, Enter: Rust
		" \x1b[B \r", // tick README, down, tick LICENSE, Enter
		"\r",         // Enter: the default, yes
	))
	if err != nil {
		t.Fatal(err)
	}
	if a.Name != "demo-app" || a.Language != "Rust" || !a.Confirm {
		t.Errorf("answers = %+v", a)
	}
	if strings.Join(a.Features, "|") != "README|LICENSE (MIT)" {
		t.Errorf("features = %v", a.Features)
	}
}

// A name that fails validation is never accepted: with the keys ending while
// the prompt still holds a bad name, the answer is ErrCanceled, not "Bad Name".
func TestABadNameIsNeverAccepted(t *testing.T) {
	a, err := ask(keys("Bad Name\r", "", "", ""))
	if !errors.Is(err, prompt.ErrCanceled) || a.Name != "" {
		t.Errorf("answers %+v err %v", a, err)
	}
}

func TestCancelReturnsErrCanceled(t *testing.T) {
	_, err := ask(keys("\x03", "", "", ""))
	if !errors.Is(err, prompt.ErrCanceled) {
		t.Errorf("err = %v", err)
	}
}

func TestCreateWritesOnlyWhatWasTicked(t *testing.T) {
	root := t.TempDir()
	a := answers{Name: "demo", Language: "Go", Features: []string{"README", ".gitignore"}, Confirm: true}
	var out bytes.Buffer
	if err := create(root, a, false, &out); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"README.md", ".gitignore"} {
		if _, err := os.Stat(filepath.Join(root, "demo", f)); err != nil {
			t.Errorf("%s not written: %v", f, err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "demo", "Dockerfile")); err == nil {
		t.Error("an unticked file was written")
	}
	if got := out.String(); !strings.Contains(got, filepath.Join("demo", "README.md")) {
		t.Errorf("listing = %q", got)
	}
}

func TestDryRunWritesNothing(t *testing.T) {
	root := t.TempDir()
	a := answers{Name: "demo", Language: "Go", Features: []string{"README"}}
	var out bytes.Buffer
	if err := create(root, a, true, &out); err != nil {
		t.Fatal(err)
	}
	if entries, _ := os.ReadDir(root); len(entries) != 0 {
		t.Errorf("a dry run wrote %d entries", len(entries))
	}
	if !strings.Contains(out.String(), "README.md") {
		t.Errorf("listing = %q", out.String())
	}
	var none bytes.Buffer
	if err := create(root, answers{Name: "x"}, false, &none); err != nil || !strings.Contains(none.String(), "nothing ticked") {
		t.Errorf("no features: %v %q", err, none.String())
	}
}

func TestValidName(t *testing.T) {
	for _, ok := range []string{"a", "my-project", "x1"} {
		if validName(ok) != nil {
			t.Errorf("%q refused", ok)
		}
	}
	for _, bad := range []string{"", "1abc", "Has Space", "UPPER", "-x"} {
		if validName(bad) == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}

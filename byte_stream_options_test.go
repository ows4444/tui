package tui

import (
	"os"
	"strings"
	"testing"
)

// Criterion #71: a file given to WithInput is the terminal: Run needs it to be
// one, as it did with the old file option.
func TestWithInputFileIsReadAsATerminal(t *testing.T) {
	r, w := mustPipe(t)
	defer r.Close()
	defer w.Close()
	p := NewProgram(keyLogger{}, WithInput(r), WithOutput(mustDevNull(t)))
	if p.input != r || p.inReader != nil {
		t.Fatalf("input=%v inReader=%v, want the file as the terminal input", p.input, p.inReader)
	}
	if _, err := p.Run(); err == nil || !strings.Contains(err.Error(), "not a terminal") {
		t.Errorf("Run = %v, want the not-a-terminal error for a non-terminal file", err)
	}
}

// Criterion #72: a non-file reader behaves as WithInputReader did: no terminal
// needed, EOF quits.
func TestWithInputPlainReaderNeedsNoTerminal(t *testing.T) {
	var out strings.Builder
	p := NewProgram(keyLogger{}, WithInput(strings.NewReader("")), WithOutput(&out))
	if p.inReader == nil {
		t.Fatal("a plain reader must be the reader source")
	}
	if _, err := p.Run(); err != nil {
		t.Errorf("Run = %v, want a clean quit at EOF", err)
	}
}

// Criterion #73: the CHANGELOG has a BREAKING entry naming each removed option.
func TestChangelogNamesEveryRemovedByteStreamOption(t *testing.T) {
	raw, err := os.ReadFile("CHANGELOG.md")
	if err != nil {
		t.Fatal(err)
	}
	log := string(raw)
	i := strings.Index(log, "- BREAKING: `WithInputReader`")
	if i < 0 {
		t.Fatal("no BREAKING entry for the removed byte-stream options")
	}
	entry := log[i:]
	if j := strings.Index(entry[2:], "\n- "); j >= 0 {
		entry = entry[:j+2]
	}
	for _, name := range []string{"WithInputReader", "WithOutputWriter", "WithErrWriter"} {
		if !strings.Contains(entry, name) {
			t.Errorf("the entry does not name %s", name)
		}
	}
}

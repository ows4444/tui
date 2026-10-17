package tui

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeEditor writes a shell script that runs body with the file path in $1 and
// points $EDITOR at it. It returns the directory the script can write to.
func fakeEditor(t *testing.T, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fake editor is a shell script")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "ed.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil { // #nosec G306 -- a test script
		t.Fatal(err)
	}
	old := lookupEnv
	lookupEnv = func(k string) string {
		if k == "EDITOR" {
			return script + " --flag"
		}
		return ""
	}
	t.Cleanup(func() { lookupEnv = old })
	return dir
}

// edit runs EditInEditor's Cmd the way the loop does: fn, then the hook.
func edit(t *testing.T, id, text, ext string) EditedMsg {
	t.Helper()
	sm, ok := EditInEditor(id, text, ext)().(suspendMsg)
	if !ok || sm.then == nil {
		t.Fatalf("EditInEditor did not produce a suspend with a hook")
	}
	m, ok := sm.then(sm.fn()).(EditedMsg)
	if !ok {
		t.Fatal("the hook did not produce an EditedMsg")
	}
	return m
}

func TestEditInEditorReturnsTheEditedText(t *testing.T) {
	fakeEditor(t, `printf 'added\n' >> "$2"`) // $1 is --flag: arguments in $EDITOR come first
	m := edit(t, "bio", "hello\n", ".md")
	if m.Err != nil || !m.Changed || m.Text != "hello\nadded\n" || m.ID != "bio" {
		t.Errorf("EditedMsg = %+v", m)
	}
}

func TestEditInEditorNoChange(t *testing.T) {
	fakeEditor(t, `true`)
	m := edit(t, "x", "same", "")
	if m.Err != nil || m.Changed || m.Text != "same" {
		t.Errorf("EditedMsg = %+v", m)
	}
}

func TestEditInEditorFailureKeepsTheOriginal(t *testing.T) {
	fakeEditor(t, `printf 'half' > "$2"; exit 3`)
	m := edit(t, "x", "original", "")
	if m.Err == nil || m.Changed || m.Text != "original" {
		t.Errorf("EditedMsg = %+v", m)
	}
	old := lookupEnv
	lookupEnv = func(k string) string {
		if k == "VISUAL" {
			return "/no/such/editor"
		}
		return ""
	}
	defer func() { lookupEnv = old }()
	if m := edit(t, "x", "original", ""); m.Err == nil || m.Text != "original" {
		t.Errorf("a missing editor: %+v", m)
	}
}

func TestEditorChoice(t *testing.T) {
	old := lookupEnv
	defer func() { lookupEnv = old }()
	env := map[string]string{}
	lookupEnv = func(k string) string { return env[k] }
	want := "vi"
	if runtime.GOOS == "windows" {
		want = "notepad"
	}
	if n, a := editorCommand(); n != want || len(a) != 0 {
		t.Errorf("no variables: %q %v, want %q", n, a, want)
	}
	env["EDITOR"] = "nano -w"
	if n, a := editorCommand(); n != "nano" || len(a) != 1 || a[0] != "-w" {
		t.Errorf("EDITOR: %q %v", n, a)
	}
	env["VISUAL"] = "  code   --wait "
	if n, a := editorCommand(); n != "code" || len(a) != 1 || a[0] != "--wait" {
		t.Errorf("VISUAL wins: %q %v", n, a)
	}
	env["VISUAL"] = "   "
	if n, _ := editorCommand(); n != "nano" {
		t.Errorf("a blank VISUAL did not fall through: %q", n)
	}
}

func TestTheFileGetsTheExtensionAndIsRemoved(t *testing.T) {
	dir := fakeEditor(t, `printf '%s' "$2" > "`+"$(dirname \"$0\")"+`/seen"`)
	for ext, wantSuffix := range map[string]string{".md": ".md", ".json": ".json", "": "", "md": "", "/../x": "", ".a b": "", ".": "", ".waytoolongextension1": ""} {
		if m := edit(t, "x", "t", ext); m.Err != nil {
			t.Fatalf("ext %q: %v", ext, m.Err)
		}
		b, err := os.ReadFile(filepath.Join(dir, "seen"))
		if err != nil {
			t.Fatal(err)
		}
		path := string(b)
		base := filepath.Base(path)
		if !strings.HasPrefix(base, "tui-edit-") || filepath.Dir(path) != filepath.Clean(os.TempDir()) && !strings.HasPrefix(path, os.TempDir()) {
			t.Errorf("ext %q: the file was %q", ext, path)
		}
		if wantSuffix != "" && !strings.HasSuffix(base, wantSuffix) {
			t.Errorf("ext %q: file %q lacks the suffix", ext, base)
		}
		if wantSuffix == "" && strings.ContainsAny(base, " /") {
			t.Errorf("ext %q reached the file name: %q", ext, base)
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("ext %q: the temporary file %q was not removed", ext, path)
		}
	}
}

// editRec is a model that starts an edit on 'e', records what it is given and
// quits on 'q'.
type editRec struct {
	mu  *sync.Mutex
	got *[]EditedMsg
}

func (editRec) Init() Cmd { return nil }
func (m editRec) Update(msg Msg) (Model, Cmd) {
	switch v := msg.(type) {
	case EditedMsg:
		m.mu.Lock()
		*m.got = append(*m.got, v)
		m.mu.Unlock()
	case Key:
		switch v.Text {
		case "e":
			return m, EditInEditor("note", "one\n", ".txt")
		case "q":
			return m, Quit()
		}
	}
	return m, nil
}
func (editRec) View() string { return "x" }

func TestEditedMsgReachesUpdateThroughTheProgram(t *testing.T) {
	fakeEditor(t, `printf 'two\n' >> "$2"`)
	pr, pw := mustPipe(t)
	t.Cleanup(func() { pr.Close(); pw.Close() })
	dn := mustDevNull(t)
	t.Cleanup(func() { dn.Close() })
	m := editRec{mu: &sync.Mutex{}, got: new([]EditedMsg)}
	p := NewProgram(m, WithInput(pr), WithOutput(dn))
	ch := startSendLoop(t, p)

	pw.WriteString("e")
	deadline := time.Now().Add(30 * time.Second) // generous: the whole module is testing at once in CI
	for {
		m.mu.Lock()
		n := len(*m.got)
		m.mu.Unlock()
		if n > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("no EditedMsg")
		}
		time.Sleep(time.Millisecond)
	}
	pw.WriteString("q")
	if res := <-ch; res.err != nil {
		t.Fatal(res.err)
	}
	if got := (*m.got)[0]; got.ID != "note" || got.Text != "one\ntwo\n" || !got.Changed || got.Err != nil {
		t.Errorf("EditedMsg = %+v", got)
	}
}

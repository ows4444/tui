package tuitest_test

import (
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/tuitest"
)

// echo renders typed keys, pasted text and the last size it saw.
type echo struct {
	typed, pasted string
	w, h          int
	custom        string
}

type customMsg string

func (m echo) Init() tui.Cmd { return nil }
func (m echo) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.Key:
		switch msg.String() {
		case "enter":
			m.typed += "|"
		case "q":
			return m, tui.Quit()
		default:
			m.typed += msg.String()
		}
	case tui.PasteEvent:
		m.pasted += msg.Text
	case tui.ResizeMsg:
		m.w, m.h = msg.Width, msg.Height
	case customMsg:
		m.custom = string(msg)
	}
	return m, nil
}
func (m echo) View() string {
	return "typed:" + m.typed + "\npaste:" + m.pasted + "\nsize:" + itoa(m.w) + "x" + itoa(m.h) + "\ncustom:" + m.custom
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for ; n > 0; n /= 10 {
		s = string(rune('0'+n%10)) + s
	}
	return s
}

func newSession(t *testing.T) *tuitest.Session {
	t.Helper()
	s := tuitest.New(echo{}, 30, 6)
	t.Cleanup(s.Close)
	return s
}

// Criterion 1: Keys drive the model and Screen returns VT-emulated lines.
func TestKeysReturnVTScreenLines(t *testing.T) {
	s := newSession(t)
	s.Keys("h", "i", "enter", "x")
	want := []string{"typed:hi|x", "paste:", "size:30x6", "custom:", "", ""}
	if got := s.Screen(); !reflect.DeepEqual(got, want) {
		t.Fatalf("screen = %q, want %q", got, want)
	}
}

func TestPasteResizeSend(t *testing.T) {
	s := newSession(t)
	s.Paste("hello world")
	s.Send(customMsg("x"))
	got := s.Screen()
	if got[1] != "paste:hello world" || got[3] != "custom:x" {
		t.Fatalf("screen = %q", got)
	}
	s.Resize(40, 8)
	got = s.Screen()
	if len(got) != 8 || got[2] != "size:40x8" {
		t.Fatalf("after resize screen = %q", got)
	}
}

func TestQuitEndsSession(t *testing.T) {
	s := newSession(t)
	s.Keys("q")
	if !s.Done() {
		t.Fatal("session should be done after the model quit")
	}
}

// The test binary declares -update itself; tuitest registers no flag.
var _ = flag.Bool("update", false, "rewrite golden files")

// Criterion 2: Golden writes with -update and diffs otherwise.
func TestGoldenUpdateAndDiff(t *testing.T) {
	s := newSession(t)
	s.Keys("a", "b")
	name := "golden_probe"
	path := filepath.Join("testdata", name+".golden")
	t.Cleanup(func() { os.Remove(path) })
	os.Remove(path)

	up := flag.Lookup("update")
	if up == nil {
		t.Fatal("-update flag not declared by this test package")
	}
	old := up.Value.String()
	defer up.Value.Set(old)

	// Without -update and no file: fails (captured via a sub-recorder).
	up.Value.Set("false")
	if failed := runFake(func(ft tuitest.TB) { s.Golden(ft, name) }); !failed {
		t.Fatal("Golden without a file must fail")
	}

	up.Value.Set("true")
	if failed := runFake(func(ft tuitest.TB) { s.Golden(ft, name) }); failed {
		t.Fatal("Golden -update must not fail")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("golden not written: %v", err)
	}
	if !strings.HasPrefix(string(b), "typed:ab\n") {
		t.Fatalf("golden = %q", b)
	}

	up.Value.Set("false")
	if failed := runFake(func(ft tuitest.TB) { s.Golden(ft, name) }); failed {
		t.Fatal("Golden must match its own output")
	}
	s.Keys("c")
	if failed := runFake(func(ft tuitest.TB) { s.Golden(ft, name) }); !failed {
		t.Fatal("Golden must fail on a diff")
	}
}

type fakeTB struct {
	tuitest.TB
	failed bool
}

func (f *fakeTB) Helper()               {}
func (f *fakeTB) Errorf(string, ...any) { f.failed = true }
func (f *fakeTB) Fatalf(string, ...any) { f.failed = true }
func (f *fakeTB) Logf(string, ...any)   {}

func runFake(fn func(tuitest.TB)) bool {
	f := &fakeTB{}
	fn(f)
	return f.failed
}

// lastKey shows the name of the last key it received.
type lastKey struct{ name string }

func (lastKey) Init() tui.Cmd { return nil }
func (m lastKey) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	if k, ok := msg.(tui.Key); ok {
		m.name = k.String()
	}
	return m, nil
}
func (m lastKey) View() string { return "key=" + m.name + "." }

// A key named the way Key.String names it reaches the model as that key, not
// as its letters.
func TestKeysDeliversNamedKeysAsKeys(t *testing.T) {
	s := tuitest.New(lastKey{}, 40, 3)
	defer s.Close()
	for _, name := range []string{"f1", "f12", "ctrl+left", "shift+up", "insert", "ctrl+shift+a", "volume-up"} {
		s.Keys(name)
		if got := strings.Join(s.Screen(), "\n"); !strings.Contains(got, "key="+name+".") {
			t.Errorf("Keys(%q): screen shows %q", name, got)
		}
	}
}

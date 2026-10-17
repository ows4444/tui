package autocomplete_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/autocomplete"
	"github.com/ows4444/tui/tuitest"
)

type app struct{ f autocomplete.Model }

func (a app) Init() tui.Cmd { return nil }
func (a app) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	var cmd tui.Cmd
	a.f, cmd = a.f.Update(msg)
	return a, cmd
}
func (a app) View() string { return a.f.View() }

func typed(s *tuitest.Session, text string) {
	for _, r := range text {
		s.Keys(string(r))
	}
}

func TestSourceSuggestionsJoinTheDropdown(t *testing.T) {
	f := autocomplete.New("alpha")
	f.Source = func(_ context.Context, text string) []string { return []string{text + "-one", text + "-two"} }
	s := tuitest.New(app{f}, 30, 6)
	defer s.Close()
	typed(s, "al")
	if !s.WaitForText("al-one", 2*time.Second) {
		t.Fatalf("no Source suggestion:\n%s", strings.Join(s.Screen(), "\n"))
	}
	if !strings.Contains(strings.Join(s.Screen(), "\n"), "alpha") {
		t.Fatal("the static suggestion is gone")
	}
	s.Keys("down", "down", "tab") // alpha, al-one, al-two: the third row
	if !s.WaitForText("al-two", time.Second) || strings.Contains(strings.Join(s.Screen(), "\n"), "al-one") {
		t.Fatalf("accepting the third row should leave only 'al-two':\n%s", strings.Join(s.Screen(), "\n"))
	}
}

func TestNewTextCancelsTheOldRun(t *testing.T) {
	var started, cancelled atomic.Int32
	f := autocomplete.New()
	f.Source = func(ctx context.Context, text string) []string {
		if text == "a" {
			started.Add(1)
			<-ctx.Done()
			cancelled.Add(1)
			return []string{"STALE"}
		}
		return []string{text + "!"}
	}
	s := tuitest.New(app{f}, 30, 6)
	defer s.Close()
	s.Keys("a")
	for i := 0; started.Load() == 0 && i < 500; i++ {
		time.Sleep(2 * time.Millisecond)
	}
	s.Keys("b")
	if !s.WaitForText("ab!", 2*time.Second) {
		t.Fatalf("fresh result missing:\n%s", strings.Join(s.Screen(), "\n"))
	}
	for i := 0; cancelled.Load() == 0 && i < 500; i++ {
		time.Sleep(2 * time.Millisecond)
	}
	if cancelled.Load() != 1 || strings.Contains(strings.Join(s.Screen(), "\n"), "STALE") {
		t.Fatalf("cancelled=%d; screen:\n%s", cancelled.Load(), strings.Join(s.Screen(), "\n"))
	}
}

func TestPathSource(t *testing.T) {
	dir := t.TempDir()
	for _, d := range []string{"src", "scripts", ".hidden"} {
		if err := os.Mkdir(filepath.Join(dir, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range []string{"setup.go", "Readme.md", "src/main.go"} {
		if err := os.WriteFile(filepath.Join(dir, f), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	src := autocomplete.PathSource()
	sep := string(filepath.Separator)
	got := src(context.Background(), filepath.Join(dir, "s"))
	want := []string{filepath.Join(dir, "scripts") + sep, filepath.Join(dir, "src") + sep, filepath.Join(dir, "setup.go")}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("got %q\nwant %q", got, want)
	}
	if got := src(context.Background(), dir+sep); len(got) != 4 { // scripts, src, Readme.md, setup.go; .hidden is skipped
		t.Fatalf("listing %q", got)
	}
	if got := src(context.Background(), dir+sep+"."); len(got) != 1 || !strings.HasSuffix(got[0], ".hidden"+sep) {
		t.Fatalf("a dot prefix should list hidden entries: %q", got)
	}
	if got := src(context.Background(), filepath.Join(dir, "src")+sep+"M"); len(got) != 1 || !strings.HasSuffix(got[0], "main.go") {
		t.Fatalf("the prefix should ignore case: %q", got)
	}
	if got := src(context.Background(), filepath.Join(dir, "nope")+sep+"x"); got != nil {
		t.Fatalf("an unreadable directory gave %q", got)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got := src(ctx, dir+sep); got != nil {
		t.Fatalf("a cancelled run returned %q", got)
	}
}

func TestUnrelatedTaskDoneIsPassedOn(t *testing.T) {
	f := autocomplete.New("a")
	f.Source = func(context.Context, string) []string { return nil }
	m, cmd := f.Update(tui.TaskDoneMsg{Key: "other", Msg: 1})
	if cmd != nil || m.IsOpen() {
		t.Fatal("an unrelated TaskDoneMsg changed the field")
	}
}

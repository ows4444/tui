package commandpalette_test

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/commandpalette"
	"github.com/ows4444/tui/tuitest"
)

// app forwards every Msg to a palette, as a real program does, and lists
// the commands picked.
type app struct {
	pal    commandpalette.Model
	picked []string
}

func (a app) Init() tui.Cmd { return nil }
func (a app) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	if sel, ok := msg.(commandpalette.SelectedMsg); ok {
		a.picked = append(a.picked, sel.Command.Name)
		return a, nil
	}
	var cmd tui.Cmd
	a.pal, cmd = a.pal.Update(msg)
	return a, cmd
}
func (a app) View() string {
	return a.pal.View() + "\npicked: " + strings.Join(a.picked, ",")
}

func typeText(s *tuitest.Session, text string) {
	for _, r := range text {
		s.Keys(string(r))
	}
}

func TestSourceCommandsAppear(t *testing.T) {
	p := commandpalette.New(commandpalette.Command{Name: "Quit"})
	p.Sources = []commandpalette.Source{func(_ context.Context, q string) []commandpalette.Command {
		return []commandpalette.Command{{Name: "Open " + q + ".go", Description: "file"}}
	}}
	s := tuitest.New(app{pal: p}, 40, 8)
	defer s.Close()
	typeText(s, "qu")
	if !s.WaitForText("Open qu.go", 2*time.Second) {
		t.Fatalf("a Source's command never showed:\n%s", strings.Join(s.Screen(), "\n"))
	}
	if !strings.Contains(strings.Join(s.Screen(), "\n"), "Quit") {
		t.Fatal("the static match is gone")
	}
	s.Keys("down", "enter")
	if !s.WaitForText("picked: Open qu.go", 2*time.Second) {
		t.Fatalf("a Source command could not be picked:\n%s", strings.Join(s.Screen(), "\n"))
	}
}

func TestNewQueryCancelsTheOldRun(t *testing.T) {
	var started, cancelled atomic.Int32
	p := commandpalette.New()
	p.Sources = []commandpalette.Source{func(ctx context.Context, q string) []commandpalette.Command {
		if q == "a" {
			started.Add(1)
			<-ctx.Done()
			cancelled.Add(1)
			return []commandpalette.Command{{Name: "STALE"}}
		}
		return []commandpalette.Command{{Name: "fresh " + q}}
	}}
	s := tuitest.New(app{pal: p}, 40, 6)
	defer s.Close()
	s.Keys("a")
	for i := 0; started.Load() == 0 && i < 500; i++ {
		time.Sleep(2 * time.Millisecond)
	}
	s.Keys("b")
	if !s.WaitForText("fresh ab", 2*time.Second) {
		t.Fatalf("fresh result missing:\n%s", strings.Join(s.Screen(), "\n"))
	}
	for i := 0; cancelled.Load() == 0 && i < 500; i++ {
		time.Sleep(2 * time.Millisecond)
	}
	if cancelled.Load() != 1 {
		t.Fatal("the older query's run was not cancelled")
	}
	if strings.Contains(strings.Join(s.Screen(), "\n"), "STALE") {
		t.Fatal("a cancelled run's answer showed")
	}
}

func TestClearingTheInputDropsSourceResults(t *testing.T) {
	p := commandpalette.New()
	p.Sources = []commandpalette.Source{func(_ context.Context, q string) []commandpalette.Command {
		return []commandpalette.Command{{Name: "hit " + q}}
	}}
	s := tuitest.New(app{pal: p}, 40, 6)
	defer s.Close()
	s.Keys("x")
	if !s.WaitForText("hit x", 2*time.Second) {
		t.Fatal("no result")
	}
	s.Keys("backspace")
	time.Sleep(50 * time.Millisecond)
	if strings.Contains(ansi.StripANSI(strings.Join(s.Screen(), "\n")), "hit") {
		t.Fatal("results stayed after the input was cleared")
	}
}

func TestUnrelatedTaskDoneIsPassedOn(t *testing.T) {
	p := commandpalette.New()
	p.Sources = []commandpalette.Source{func(context.Context, string) []commandpalette.Command { return nil }}
	m, cmd := p.Update(tui.TaskDoneMsg{Key: "other", Msg: "x"})
	if cmd != nil || m.IsOpen() {
		t.Fatal("an unrelated TaskDoneMsg changed the palette")
	}
}

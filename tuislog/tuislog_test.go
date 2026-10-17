package tuislog

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
)

// lines collects what a Handler's sink receives.
type lines struct {
	mu  sync.Mutex
	got []string
}

func (l *lines) add(s string) {
	l.mu.Lock()
	l.got = append(l.got, s)
	l.mu.Unlock()
}

func (l *lines) plain() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]string, len(l.got))
	for i, s := range l.got {
		out[i] = ansi.StripANSI(s)
	}
	return out
}

func logger(l *lines, o *Options) *slog.Logger {
	if o == nil {
		o = &Options{}
	}
	if o.TimeFormat == "" {
		o.TimeFormat = "-" // a time in a golden line would never match
	}
	return slog.New(New(l.add, o))
}

func TestOneLinePerRecordWithLevelMessageAndAttributes(t *testing.T) {
	var l lines
	lg := logger(&l, nil)
	lg.Info("started", "port", 8080, "name", "web 1")
	lg.Warn("disk low")
	lg.Error("failed", "err", io.EOF)
	want := []string{
		`INFO  started port=8080 name="web 1"`,
		`WARN  disk low`,
		`ERROR failed err=EOF`,
	}
	got := l.plain()
	if len(got) != len(want) {
		t.Fatalf("got %d lines, want %d: %q", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestLevelsAreColouredApart(t *testing.T) {
	var l lines
	lg := logger(&l, &Options{Level: slog.LevelDebug})
	lg.Debug("d")
	lg.Info("i")
	lg.Warn("w")
	lg.Error("e")
	seen := map[string]bool{}
	for _, s := range l.got {
		head := s[:strings.Index(s, " ")]
		seen[head] = true
	}
	if len(seen) != 4 {
		t.Errorf("the four levels share a style: %q", l.got)
	}
}

func TestLevelFilter(t *testing.T) {
	var l lines
	lg := logger(&l, nil)
	lg.Debug("hidden")
	lg.Info("shown")
	if got := l.plain(); len(got) != 1 || !strings.Contains(got[0], "shown") {
		t.Errorf("default level logged %q", got)
	}
	l = lines{}
	lg = logger(&l, &Options{Level: slog.LevelError})
	lg.Warn("hidden")
	if got := l.plain(); len(got) != 0 {
		t.Errorf("Error level logged a warning: %q", got)
	}
	var lv slog.LevelVar
	lv.Set(slog.LevelWarn)
	h := New(l.add, &Options{Level: &lv})
	if h.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("Info enabled at Warn")
	}
	lv.Set(slog.LevelInfo)
	if !h.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("a LevelVar change did not take effect")
	}
}

func TestWithAttrsAndGroups(t *testing.T) {
	var l lines
	lg := logger(&l, nil).With("req", 7).WithGroup("db").With("host", "a")
	lg.Info("query", "rows", 3, slog.Group("tx", "id", 9))
	lg.Info("empty group", slog.Group("none"))
	lg.Info("blank", slog.Attr{})
	want := []string{
		`INFO  query req=7 db.host=a db.rows=3 db.tx.id=9`,
		`INFO  empty group req=7 db.host=a`,
		`INFO  blank req=7 db.host=a`,
	}
	got := l.plain()
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, got[i], want[i])
		}
	}
	if h := New(l.add, nil); h.WithGroup("") != slog.Handler(h) {
		t.Error("WithGroup(\"\") returned a new handler")
	}
}

func TestQuotingKeepsPairsReadable(t *testing.T) {
	var l lines
	logger(&l, nil).Info("m", "empty", "", "eq", "a=b", "quote", `say "hi"`, "nl", "a\nb", "plain", "ok")
	want := `INFO  m empty="" eq="a=b" quote="say \"hi\"" nl="a\nb" plain=ok`
	if got := l.plain()[0]; got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestEscapeSequencesCannotReachTheScreen(t *testing.T) {
	var l lines
	logger(&l, nil).Info("hello\x1b[2Jworld\x1b]0;title\x07", "k\x1b[31m", "v\x1b[41mred")
	for _, s := range l.got {
		stripped := ansi.StripANSI(s)
		if strings.ContainsAny(stripped, "\x1b\x07") {
			t.Errorf("control characters survived: %q", s)
		}
	}
	if got := l.plain()[0]; !strings.Contains(got, "helloworld") {
		t.Errorf("message = %q", got)
	}
}

func TestTimeColumn(t *testing.T) {
	var l lines
	h := New(l.add, &Options{TimeFormat: "2006"})
	r := slog.NewRecord(time.Date(2026, 10, 17, 1, 2, 3, 0, time.UTC), slog.LevelInfo, "x", 0)
	_ = h.Handle(context.Background(), r)
	h = New(l.add, &Options{})
	_ = h.Handle(context.Background(), r)
	h = New(l.add, &Options{TimeFormat: "-"})
	_ = h.Handle(context.Background(), r)
	_ = h.Handle(context.Background(), slog.NewRecord(time.Time{}, slog.LevelInfo, "zero", 0))
	got := l.plain()
	want := []string{"2026 INFO  x", "01:02:03 INFO  x", "INFO  x", "INFO  zero"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestSourceAddsFileAndLine(t *testing.T) {
	var l lines
	logger(&l, &Options{AddSource: true}).Info("here")
	if got := l.plain()[0]; !strings.Contains(got, "tuislog_test.go:") {
		t.Errorf("no source in %q", got)
	}
}

func TestMessageWithNewlineStaysOneRecord(t *testing.T) {
	var l lines
	logger(&l, nil).Info("a\nb")
	if len(l.got) != 1 || !strings.Contains(l.plain()[0], "a\nb") {
		t.Errorf("got %q", l.got)
	}
}

// lockedBuffer is a bytes.Buffer that the Program and the test can share.
type lockedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (l *lockedBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *lockedBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}

type idle struct{}

func (idle) Init() tui.Cmd                       { return nil }
func (idle) Update(tui.Msg) (tui.Model, tui.Cmd) { return idle{}, nil }
func (idle) View() string                        { return "live" }

func TestToProgramCommitsTheLineAboveTheLiveRegion(t *testing.T) {
	in, release := io.Pipe()
	out := &lockedBuffer{}
	p := tui.NewProgram(idle{}, tui.WithInput(in), tui.WithOutput(out), tui.WithReducedMotion(true), tui.WithAltScreen(false))
	lg := slog.New(ToProgram(p, &Options{TimeFormat: "-"}))
	lg.Warn("disk low", "free", "3 GB") // queued: Run has not started

	done := make(chan error, 1)
	go func() { _, err := p.Run(); done <- err }()

	deadline := time.After(5 * time.Second)
	for !strings.Contains(ansi.StripANSI(out.String()), `WARN  disk low free="3 GB"`) {
		select {
		case <-deadline:
			t.Fatalf("the line never reached the terminal:\n%q", ansi.StripANSI(out.String()))
		case <-time.After(10 * time.Millisecond):
		}
	}
	lg.Info("while running")
	p.Quit()
	release.Close()
	if err := <-done; err != nil {
		t.Fatalf("Run: %v", err)
	}
	lg.Info("after exit") // dropped, and must not block or panic
	if s := ansi.StripANSI(out.String()); strings.Contains(s, "after exit") {
		t.Errorf("a line after Run returned was printed:\n%s", s)
	}
}

func TestNestedGroupsKeepTheirOuterNames(t *testing.T) {
	var l lines
	logger(&l, nil).WithGroup("a").WithGroup("b").Info("m", "k", 1)
	if got, want := l.plain()[0], "INFO  m a.b.k=1"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

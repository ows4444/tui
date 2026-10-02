package tui

import (
	"bytes"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ows4444/tui/motion"
)

// Criterion #81: a pending motion wait ends when Run returns.
func TestMotionWaitsEndOnExplicitContext(t *testing.T) {
	cases := map[string]CtxCmd{
		"after": func(ctx context.Context) Msg {
			return motion.After(time.Hour, func(time.Time) Msg { return nil })(ctx)
		},
		"clock": func(ctx context.Context) Msg {
			return motion.NewClock(time.Hour).Start()(ctx)
		},
		"tick": TickCtx(time.Hour, func(time.Time) Msg { return nil }),
	}
	for name, fn := range cases {
		t.Run(name, func(t *testing.T) {
			pr, pw := mustPipe(t)
			defer pr.Close()
			defer pw.Close()
			var out bytes.Buffer
			m := recModel{mu: &sync.Mutex{}, got: &[]Msg{}, init: FromCtx(fn)}
			p := NewProgram(m, WithInput(readerOnly{pr}), WithOutput(&out))
			done := make(chan struct{})
			go func() { _, _ = p.Run(); close(done) }()
			time.Sleep(50 * time.Millisecond)
			p.msgs <- QuitMsg{}
			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.Fatal("Run did not return with a motion wait pending")
			}
		})
	}
}

// Criterion #80: no non-test source calls runtime.Stack. Panic reporting uses
// debug.Stack, which is not a goroutine-id lookup.
func TestNoRuntimeStackOutsideTests(t *testing.T) {
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if n := d.Name(); n == "testdata" || n == "vendor" || (path != "." && strings.HasPrefix(n, ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(b), "runtime.Stack(") {
			t.Errorf("%s calls runtime.Stack", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

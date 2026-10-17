package prompt

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

// progressWidth is how many columns the bar is.
const progressWidth = 30

type progressModel struct {
	label  string
	total  int
	done   *atomic.Int64
	stats  *widgets.ProgressStats
	fn     func(context.Context, func(done int)) error
	ctx    context.Context
	cancel context.CancelFunc
	err    error
	finish bool
	theme  theme.Theme
}

type progressTick struct{ at time.Time }
type progressDone struct{ err error }

func (m progressModel) Init() tui.Cmd {
	report := func(done int) { m.done.Store(int64(done)) }
	work := tui.Go(func(context.Context) tui.Msg { return progressDone{m.fn(m.ctx, report)} })
	tick, _ := tui.Every(100*time.Millisecond, func(t time.Time) tui.Msg { return progressTick{t} })
	return tui.Batch(work, tick)
}

func (m progressModel) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch v := msg.(type) {
	case progressTick:
		m.stats.Update(v.at, float64(m.done.Load()), float64(m.total))
	case progressDone:
		m.err, m.finish = v.err, true
		if v.err == nil {
			m.done.Store(int64(m.total))
		}
		m.stats.Update(time.Now(), float64(m.done.Load()), float64(m.total))
		return m, tui.Quit()
	case tui.Key:
		if v.Type == tui.KeyCtrlC {
			m.cancel()
		}
	}
	return m, nil
}

func (m progressModel) View() string {
	return ansi.Sanitize(m.label) + "\n" + widgets.ProgressBarStats(m.stats, progressWidth, "", m.theme)
}

// Progress shows a bar for fn while it runs, and returns fn's error. total is
// how many units of work there are. fn reports how many it has finished with
// report (from any goroutine; later calls replace earlier ones, so it can be
// called as often as is convenient), and gets a context that is cancelled when
// the person presses Ctrl+C, which it should honour. Under the bar are the
// elapsed time, the rate and the time left (see widgets.ProgressStats). A total
// under 1 shows the elapsed time and rate only.
func Progress(label string, total int, fn func(ctx context.Context, report func(done int)) error, cfg ...Config) error {
	c := newConfig(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m := progressModel{
		label: label, total: total, done: new(atomic.Int64), stats: widgets.NewProgressStats(time.Now()),
		fn: fn, ctx: ctx, cancel: cancel, theme: theme.DarkTheme(),
	}
	out, err := run(m, c)
	if err != nil {
		return err
	}
	res := out.(progressModel)
	if !res.finish {
		return ErrCanceled
	}
	return res.err
}

// Track runs fn for each item in order with a bar that advances after each, and
// stops at the first error or when the person presses Ctrl+C (then it returns
// the context's error). It is Progress for the common case of a loop.
func Track[T any](label string, items []T, fn func(ctx context.Context, item T) error, cfg ...Config) error {
	return Progress(label, len(items), func(ctx context.Context, report func(int)) error {
		for i, it := range items {
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := fn(ctx, it); err != nil {
				return err
			}
			report(i + 1)
		}
		return nil
	}, cfg...)
}

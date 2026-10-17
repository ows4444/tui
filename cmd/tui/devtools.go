package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"os/signal"
	"sort"
	"strings"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/motion"
	"github.com/ows4444/tui/theme"
)

// ---- tui keys: a live view of what the terminal sends ----

// keyViewer shows the last events the program receives, decoded.
type keyViewer struct {
	events []string
	w, h   int
}

const keyViewerHeader = "tui keys: press keys, click, scroll, paste; Ctrl+Q quits"

func (keyViewer) Init() tui.Cmd { return nil }

func (v keyViewer) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	var line string
	switch m := msg.(type) {
	case tui.ResizeMsg:
		v.w, v.h = m.Width, m.Height
		return v, nil
	case tui.Key:
		if m.Type == tui.KeyCtrl && m.Code == 'q' {
			return v, tui.Quit()
		}
		line = fmt.Sprintf("key    %-18s type=%d text=%q code=%q mod=%q action=%d keypad=%v",
			m.String(), m.Type, m.Text, m.Code, m.Mod.String(), m.Action, m.Keypad)
	case tui.MouseEvent:
		line = fmt.Sprintf("mouse  button=%d action=%d at %d,%d mod=%q clicks=%d", m.Button, m.Action, m.X, m.Y, m.Mod.String(), m.Clicks)
	case tui.PasteEvent:
		line = fmt.Sprintf("paste  %d bytes %q", len(m.Text), shorten(m.Text, 40))
	case tui.FocusEvent:
		line = fmt.Sprintf("focus  focused=%v", m.Focused)
	default:
		return v, nil
	}
	v.events = append(v.events, ansi.Sanitize(line))
	if keep := max(v.h-2, 5); len(v.events) > keep {
		v.events = v.events[len(v.events)-keep:]
	}
	return v, nil
}

func (v keyViewer) View() string {
	return keyViewerHeader + "\n\n" + strings.Join(v.events, "\n")
}

func shorten(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n]) + "..."
	}
	return s
}

// runKeyViewer runs the viewer on the terminal.
func runKeyViewer() error {
	_, err := tui.NewProgram(keyViewer{},
		tui.WithAltScreen(true), tui.WithMouse(tui.MouseAllMotion), tui.WithFocusReporting(true),
		tui.WithKeyboard(tui.KeyboardDisambiguate|tui.KeyboardReportEvents|tui.KeyboardReportAlternates|tui.KeyboardReportAllKeys),
	).Run()
	return err
}

// ---- tui log: read and follow the file tui.LogToFile writes ----

// runLog prints the last lines of a log file, those that contain every -grep
// word, and with -f keeps printing what is appended until stop is closed.
func runLog(args []string, out io.Writer, stop <-chan struct{}) error {
	fs := flag.NewFlagSet("log", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	n := fs.Int("n", 20, "how many of the last lines to print first (0 prints none, -1 all)")
	follow := fs.Bool("f", false, "keep printing lines as they are appended")
	var grep multiFlag
	fs.Var(&grep, "grep", "only lines containing this text (repeat for more; all must match)")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("%v\n%s", err, usage)
	}
	if fs.NArg() != 1 {
		return errors.New(usage)
	}
	path := fs.Arg(0)
	fh, err := os.Open(path) // #nosec G304 -- the file the user named
	if err != nil {
		return err
	}
	defer fh.Close()
	match := func(line string) bool {
		for _, g := range grep {
			if !strings.Contains(line, g) {
				return false
			}
		}
		return true
	}
	// Print the tail first.
	var tail []string
	sc := bufio.NewScanner(fh)
	sc.Buffer(make([]byte, 64<<10), 1<<20)
	for sc.Scan() {
		if line := sc.Text(); match(line) {
			tail = append(tail, line)
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}
	if *n >= 0 && len(tail) > *n {
		tail = tail[len(tail)-*n:]
	}
	for _, l := range tail {
		fmt.Fprintln(out, ansi.Sanitize(l))
	}
	if !*follow {
		return nil
	}
	rd := bufio.NewReader(fh)
	var partial string
	for {
		line, err := rd.ReadString('\n')
		partial += line
		if err == nil {
			if l := strings.TrimRight(partial, "\r\n"); match(l) {
				fmt.Fprintln(out, ansi.Sanitize(l))
			}
			partial = ""
			continue
		}
		select {
		case <-stop:
			return nil
		case <-time.After(100 * time.Millisecond):
		}
	}
}

type multiFlag []string

func (m *multiFlag) String() string     { return strings.Join(*m, ",") }
func (m *multiFlag) Set(s string) error { *m = append(*m, s); return nil }

// interrupted returns a channel closed on Ctrl+C.
func interrupted() (<-chan struct{}, func()) {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	return ctx.Done(), cancel
}

// ---- tui colors, borders and easings: look at what a theme or an effect is ----

// runColors prints each colour role of a preset as a swatch, its hex value
// where it has one and its contrast against the background.
func runColors(args []string, out io.Writer) error {
	if len(args) > 1 {
		return errors.New(usage)
	}
	names := theme.PresetNames()
	if len(args) == 1 {
		names = []string{args[0]}
	}
	for _, n := range names {
		t, ok := theme.Preset(n)
		if !ok {
			return fmt.Errorf("unknown preset %q (one of %s)", n, strings.Join(theme.PresetNames(), ", "))
		}
		fmt.Fprintln(out, n)
		roles := []struct {
			name string
			c    ansi.Color
		}{
			{"primary", t.Primary}, {"secondary", t.Secondary}, {"success", t.Success}, {"warning", t.Warning},
			{"error", t.Error}, {"info", t.Info}, {"muted", t.Muted}, {"text", t.Text},
			{"text_inverse", t.TextInverse}, {"border_color", t.BorderColor}, {"focus", t.Focus},
			{"selection", t.Selection}, {"background", t.Background}, {"surface", t.Surface}, {"overlay", t.Overlay},
		}
		for _, r := range roles {
			if r.c == nil {
				continue
			}
			swatch := ansi.NewStyle().Background(r.c).Render("      ")
			hex := ""
			if c, ok := r.c.(ansi.RGB); ok {
				hex = fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
			}
			ratio := ""
			if t.Background != nil && r.name != "background" {
				ratio = fmt.Sprintf("%.1f:1", theme.Contrast(r.c, t.Background))
			}
			fmt.Fprintf(out, "  %s %-13s %-8s %s\n", swatch, r.name, hex, ratio)
		}
	}
	return nil
}

// runBorders prints a small box in each border preset of the layout package.
func runBorders(out io.Writer) error {
	presets := []struct {
		name string
		b    layout.Border
	}{
		{"normal", layout.NormalBorder()}, {"rounded", layout.RoundedBorder()}, {"double", layout.DoubleBorder()},
		{"thick", layout.ThickBorder()}, {"ascii", layout.ASCIIBorder()}, {"block", layout.BlockBorder()},
		{"outer-half-block", layout.OuterHalfBlockBorder()}, {"inner-half-block", layout.InnerHalfBlockBorder()},
		{"dashed", layout.DashedBorder()}, {"heavy-dashed", layout.HeavyDashedBorder()},
		{"markdown", layout.MarkdownBorder()}, {"hidden", layout.HiddenBorder()},
	}
	for _, p := range presets {
		fmt.Fprintln(out, p.name)
		box := layout.BoxNode(layout.NewBox().Border(p.b).Padding(0, 1, 0, 1), layout.Block("sample"))
		fmt.Fprintln(out, layout.Draw(box, layout.Unconstrained()))
	}
	return nil
}

// runEasings plots each easing curve as a small grid of dots: time left to
// right, progress bottom to top.
func runEasings(args []string, out io.Writer) error {
	names := motion.EaseNames()
	if len(args) == 1 {
		if _, ok := motion.ByName(args[0]); !ok {
			return fmt.Errorf("unknown easing %q (one of %s)", args[0], strings.Join(names, ", "))
		}
		names = []string{args[0]}
	} else if len(args) > 1 {
		return errors.New(usage)
	}
	sort.Strings(names)
	const w, h = 24, 6
	for _, n := range names {
		e, _ := motion.ByName(n)
		grid := make([][]byte, h)
		for y := range grid {
			grid[y] = []byte(strings.Repeat(" ", w))
		}
		for x := 0; x < w; x++ {
			v := e(float64(x) / float64(w-1))
			row := int(math.Round(v * float64(h-1)))
			row = min(max(row, 0), h-1)
			grid[h-1-row][x] = '*'
		}
		fmt.Fprintln(out, n)
		for _, r := range grid {
			fmt.Fprintln(out, "  |"+strings.TrimRight(string(r), " "))
		}
	}
	return nil
}

// Command probe shows what your terminal supports, using the library's
// opt-in capability options: colour depth (with NO_COLOR, COLORTERM and TERM
// honoured), focus in/out reporting, and OSC 11 background detection. It runs
// inline, so the report stays in your scrollback: run it, click to another
// window and back, then press q and copy the "probe:" line it leaves behind.
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

const bgTimeout = 700 * time.Millisecond

type model struct {
	profile  ansi.Profile
	focusIn  int
	focusOut int
	focused  string // "unknown", "in" or "out": the last report seen
	bg       string // "waiting", "rgb(r,g,b)" or "no answer"
	theme    string // "-", "dark" or "light"
	lastKey  string
	keys     []string // every key seen, in order
	size     string   // "-" until the first ResizeMsg, then "WxH"
	width    int      // terminal width from the last ResizeMsg; 0 until known
	log      focusLog
}

func newModel(profile ansi.Profile) model {
	return model{profile: profile, focused: "unknown", bg: "waiting", theme: "-", lastKey: "-", size: "-", log: newFocusLog(nil)}
}

func (m model) Init() tui.Cmd { return nil }

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.FocusEvent:
		m.log.record(msg.Focused)
		if msg.Focused {
			m.focusIn++
			m.focused = "in"
		} else {
			m.focusOut++
			m.focused = "out"
		}
	case tui.BackgroundColorEvent:
		m.bg = fmt.Sprintf("rgb(%d,%d,%d)", msg.R, msg.G, msg.B)
		m.theme = themeName(msg)
	case tui.BackgroundUnknownMsg:
		m.bg = "no answer"
	case tui.ResizeMsg:
		m.size = fmt.Sprintf("%dx%d", msg.Width, msg.Height)
		m.width = msg.Width
	case tui.Key:
		m.lastKey = msg.String()
		m.keys = append(m.keys, m.lastKey)
		if msg.Type == tui.KeyCtrlC || (msg.Type == tui.KeyRunes && msg.Text == "q") {
			return m, tui.Quit()
		}
	}
	return m, nil
}

// themeName reports whether theme.Detect would pick the light or dark theme.
func themeName(ev tui.BackgroundColorEvent) string {
	if th, ok := theme.Detect(ev, theme.DarkTheme()); ok && th == theme.LightTheme() {
		return "light"
	}
	return "dark"
}

func profileName(p ansi.Profile) string {
	switch p {
	case ansi.NoColor:
		return "none"
	case ansi.ANSI16:
		return "16-colour"
	case ansi.ANSI256:
		return "256-colour"
	default:
		return "truecolor"
	}
}

// gradient is a truecolor strip; under a lower profile the program rewrites
// it, so its appearance shows what the downgrade did.
func gradient(width int) string {
	var b strings.Builder
	for i := 0; i < width; i++ {
		f := float64(i) / float64(width-1)
		b.WriteString(ansi.NewStyle().Background(ansi.RGB{R: uint8(255 * (1 - f)), G: uint8(255 * f * (1 - f) * 4), B: uint8(255 * f)}).Render(" "))
	}
	return b.String()
}

func (m model) summary() string {
	return fmt.Sprintf("probe: profile=%s focus_in=%d focus_out=%d background=%s theme=%s",
		profileName(m.profile), m.focusIn, m.focusOut, m.bg, m.theme)
}

// hardWrap breaks every line of s into rows of at most width columns, indenting
// continuation rows by two spaces (when there is room). Unlike ansi.Wrap it keeps the runs of
// spaces that line the report up, and it never splits a wide character. A
// width of 0 or less leaves s alone.
func hardWrap(s string, width int) string {
	if width <= 0 {
		return s
	}
	indent := "  "
	if width <= len(indent)+1 {
		indent = "" // too narrow to indent and still make progress
	}
	var out []string
	for _, line := range strings.Split(s, "\n") {
		for ansi.Width(line) > width {
			head := ansi.Truncate(line, width)
			consumed := ansi.Width(head)
			if consumed == 0 { // a single cell wider than width: give up on it
				break
			}
			skip := 0
			rest := ansi.TrimLeftWidth(line, consumed)
			if !strings.HasPrefix(rest, " ") {
				// Mid-word: back up to the last space past the indentation.
				lead := len(head) - len(strings.TrimLeft(head, " "))
				if i := strings.LastIndex(head, " "); i > lead {
					consumed, skip = ansi.Width(head[:i]), 1
					head = strings.TrimRight(head[:i], " ")
				}
			}
			out = append(out, head)
			line = indent + strings.TrimLeft(ansi.TrimLeftWidth(line, consumed+skip), " ")
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func (m model) View() string {
	title := ansi.NewStyle().Bold().Render("tui probe: what does your terminal do?")
	env := fmt.Sprintf("TERM=%q  COLORTERM=%q  NO_COLOR=%q", os.Getenv("TERM"), os.Getenv("COLORTERM"), os.Getenv("NO_COLOR"))
	return hardWrap(strings.Join([]string{
		title,
		"",
		"  " + env,
		"  Colour profile:  " + profileName(m.profile) + "   (detected; drives the strip below)",
		"  " + gradient(m.stripWidth()),
		"",
		fmt.Sprintf("  Focus reports:   in=%d out=%d  last=%s   (click another window, then back)", m.focusIn, m.focusOut, m.focused),
		fmt.Sprintf("  Background:      %s  theme=%s   (asked once at start, %v timeout)", m.bg, m.theme, bgTimeout),
		"  Last key:        " + m.lastKey,
		"  Size (resize):   " + m.size,
		"",
		"  " + ansi.NewStyle().Faint().Render("q or ctrl+c to quit; a summary line stays in your scrollback"),
		"  " + m.summary(),
		"  " + ansi.NewStyle().Faint().Render("focus timeline:"),
		"    " + strings.Join(m.log.lines(), "\n    "),
	}, "\n"), m.width)
}

// stripWidth is the colour strip's width: 48, or what fits the terminal.
func (m model) stripWidth() int {
	if m.width > 0 {
		return max(2, min(48, m.width-2))
	}
	return 48
}

func main() {
	profile := ansi.DetectColorProfile()
	clusters := runClusterProbe()
	start := newModel(profile)
	// Stamp just before Run: the program enables focus reporting as it starts.
	start.log.markEnabled()
	final, err := tui.NewProgram(start,
		tui.WithAltScreen(false),
		tui.WithColorProfile(profile),
		tui.WithFocusReporting(true),
		tui.WithBackgroundDetection(bgTimeout),
	).Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if m, ok := final.(model); ok {
		fmt.Print(clusters)
		fmt.Println(m.summary())
		// Stable lines for the Windows ConPTY CI job (conpty_windows_test.go).
		fmt.Println("probe: keys=" + strings.Join(m.keys, ","))
		fmt.Println("probe: size=" + m.size)
		fmt.Println("probe: focus timeline: " + m.log.String())
	}
}

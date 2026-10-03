package tui

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// ProgramOption configures a Program at construction time.
type ProgramOption func(*Program)

// WithMaxConcurrentCmds limits how many Cmds run at the same time to n. Cmds
// beyond the limit wait and start in the order the event loop received them
// (a Batch's Cmds count in their argument order). n <= 0, the default, means
// no limit. The limit applies to Cmds the loop dispatches; the steps of a
// Sequence and Every timers run outside it.
func WithMaxConcurrentCmds(n int) ProgramOption {
	return func(p *Program) { p.cmdPool.max = n }
}

// WithExitOnSignal chooses what happens on SIGTERM or SIGHUP. The terminal is
// always restored first. When enabled (the default) the process then exits
// with status 1. When disabled, Run cancels its context and returns
// ErrInterrupted so the application can run its own shutdown.
func WithExitOnSignal(enabled bool) ProgramOption {
	return func(p *Program) { p.exitOnSignal = enabled }
}

// WithAltScreen enables or disables the terminal's alternate screen buffer
// (on by default) so the app doesn't scroll the user's normal history.
func WithAltScreen(enabled bool) ProgramOption {
	return func(p *Program) { p.altScreen, p.modeSet = enabled, true }
}

// WithOutput sets where the program renders to (default os.Stdout). Give it an
// *os.File for a terminal; the Program queries its size and switches its modes.
// Any other io.Writer, such as a bytes.Buffer in a test, has no terminal size:
// the Program assumes 80x24 (a ResizeMsg can still deliver another size) and
// never queries or resizes it. Mode sequences (alternate screen, cursor,
// synchronized output) are written to w like any other output.
func WithOutput(w io.Writer) ProgramOption {
	return func(p *Program) { p.output = w }
}

// WithInput sets where the program reads keys from (default os.Stdin). Given an
// *os.File, the Program reads it as a terminal: raw mode when it is one, and a
// read that Quit can cancel. Given any other io.Reader, Run does not require a
// terminal and does not switch raw mode, so a program can be driven from a
// strings.Reader in a test; a file that should be read that way can be wrapped
// (struct{ io.Reader }{f}). When the reader reaches io.EOF the Program quits, as
// if the model had returned Quit, once the events before EOF have been handled;
// any other read error is delivered to Update as an InputErrorMsg. Cancelling a
// blocked read is not possible for a plain io.Reader: if r blocks forever, Run
// still returns on Quit, but the reading goroutine stays parked in r.Read. A
// reader that is also an io.Closer is closed when the Program quits, unless
// WithInputCloser names another. Of several WithInput options, the last wins.
func WithInput(r io.Reader) ProgramOption {
	return func(p *Program) {
		if f, ok := r.(*os.File); ok {
			p.input, p.inReader = f, nil
			return
		}
		p.inReader = r
		if c, ok := r.(io.Closer); ok && p.inCloser == nil {
			p.inCloser = c
		}
	}
}

// WithInputCloser registers c to be closed, exactly once, when the Program
// quits. Closing it unblocks a read parked in a WithInput source that is not a
// file (an io.Pipe reader, say), so Run returns without waiting out the reader
// stop timeout. A WithInput source that itself implements io.Closer is closed
// the same way without this option; c takes precedence over it.
func WithInputCloser(c io.Closer) ProgramOption {
	return func(p *Program) { p.inCloser = c }
}

// WithErrOutput sets where Eprintln writes to (default os.Stderr). Given an
// *os.File it is the terminal's error stream; any other io.Writer, a
// bytes.Buffer in a test or a log, receives the text as it is.
func WithErrOutput(w io.Writer) ProgramOption {
	return func(p *Program) { p.errOutput = w }
}

// MouseMode selects how much mouse activity the terminal reports; see
// WithMouse.
type MouseMode int

const (
	// MouseOff (the default) reports no mouse activity at all.
	MouseOff MouseMode = iota
	// MouseClick reports only button press/release, no drag or movement.
	MouseClick
	// MouseCellMotion additionally reports motion while a button is held.
	MouseCellMotion
	// MouseAllMotion reports every mouse movement, a button held or not.
	MouseAllMotion
)

// WithMouse enables mouse reporting at the given MouseMode, delivered to
// Update as MouseEvent. Off (the default) reports nothing.
func WithMouse(mode MouseMode) ProgramOption {
	return func(p *Program) { p.mouseMode = mode }
}

// WithBracketedPaste sets bracketed-paste mode (DEC 2004). It is on by
// default: a paste is delivered to Update as a single PasteEvent carrying the
// whole pasted text, rather than as a flood of individual Key events that a
// pasted newline could turn into Enter presses. Pass false to opt out and
// receive pasted text as keys.
func WithBracketedPaste(enabled bool) ProgramOption {
	return func(p *Program) { p.bracketedPaste = enabled }
}

// WithKittyKeyboard opts into the kitty keyboard protocol's "disambiguate
// escape codes" enhancement (CSI ?u), letting input.Key events distinguish
// modifier combinations the legacy xterm CSI-modifier encoding can't
// represent — Shift+Enter vs. plain Enter, Ctrl+Shift+<letter> vs.
// Ctrl+<letter> — without changing how arrows, Home/End, or function keys
// are decoded. A terminal that doesn't support the protocol ignores the
// enable/disable sequences entirely and every key continues to decode via
// the existing legacy parsing, unaffected. Off by default: enabling it
// costs nothing on a supporting terminal, but it's opt-in rather than
// always-on so a Program's observable input behavior doesn't change
// silently between runs based on what terminal happens to be attached.
func WithKittyKeyboard(enabled bool) ProgramOption {
	return func(p *Program) { p.kittyKeyboard = enabled }
}

// WithReducedMotion overrides the Program's reduced-motion preference,
// read back via (*Program).ReducedMotion. Without this option, the
// preference defaults to motion.Detect() (the NO_ANIMATION and REDUCE_MOTION
// environment variables), so existing motion.Preference-aware code keeps working
// whether or not a Program explicitly opts in.
//
// This is a signal, not automatic enforcement: Program has no way to know
// which of a widget's returned Cmds are animation-driven, or which part of
// its View() output is decorative rather than content, so it can't
// suppress either on its own. An app that wants to honor reduced motion
// combines ReducedMotion() with the library's existing cooperative
// pieces — motion.Preference-aware widgets (spinner.Model, skeleton.Model,
// streamtext.Model's Skip) for animation, and theme.Theme.Plain for
// box-drawing decoration — the same "framework provides the signal, caller
// decides" pattern WithContext already uses for cancellation.
func WithReducedMotion(enabled bool) ProgramOption {
	return func(p *Program) { p.reducedMotion = enabled }
}

// WithAccessible turns on accessible (linearized) output for screen-reader
// and plain-text use; it is off by default and changes nothing when unset.
// While on, the Program never uses the alternate screen (even if
// WithAltScreen(true) is also given) and emits no cursor-movement,
// clear-line, synchronized-output, styling (SGR) or OSC sequences. Output
// is append-only: each frame prints only the lines that changed since the
// previous one, once, in order, so a screen reader reads new text as it
// arrives instead of re-reading a repainted screen. The text comes from the
// root model's Linearize method if it implements Linearizer, and otherwise
// from View() with ANSI sequences stripped.
//
// Accessible() implies ReducedMotion(): WithReducedMotion(false) cannot
// override it. Program does not own the theme, so dropping decorative
// borders stays cooperative: apps call theme.Theme.Plain() when
// Accessible() is true.
func WithAccessible(enabled bool) ProgramOption {
	return func(p *Program) { p.accessible, p.modeSet = enabled, true }
}

// WithFocusReporting turns terminal focus reporting (DECSET 1004) on for
// the life of the Program, so Update receives a FocusEvent whenever the
// terminal window gains or loses focus. It is re-enabled after Suspend and
// switched off again when the Program restores the terminal. A FocusEvent
// whose state equals the last one delivered is dropped, so Update sees only
// changes; the first event is always delivered. Off by default: without it
// no focus sequence is emitted.
func WithFocusReporting(enabled bool) ProgramOption {
	return func(p *Program) { p.focusReporting = enabled }
}

// WithBackgroundDetection asks the terminal for its background colour
// (OSC 11, ansi.QueryBackgroundColor) once, at startup. A supporting
// terminal answers and Update receives a BackgroundColorEvent, which
// theme.ForBackground turns into a Light or Dark theme. If no answer
// arrives within timeout, Update receives a single BackgroundUnknownMsg
// instead, so the app can stop waiting. Detection also queries ANSI colours
// 0-15 (OSC 4, ansi.QueryPalette); each answer reaches Update as a
// PaletteColorEvent, which theme.Palette.Observe records so contrast checks
// use the real palette. A timeout <= 0 leaves detection
// off (the default): nothing is sent.
func WithBackgroundDetection(timeout time.Duration) ProgramOption {
	return func(p *Program) { p.bgTimeout = timeout }
}

// Themeable is implemented by a Model that wants the Program's theme (see
// WithTheme). SetTheme returns the model with the theme applied, the way
// Update returns the next model; a root model forwards it to its widgets.
type Themeable interface {
	SetTheme(theme.Theme) Model
}

// defaultThemeDetectTimeout is how long WithTheme waits for the terminal's
// background colour when WithBackgroundDetection did not set a timeout.
const defaultThemeDetectTimeout = 300 * time.Millisecond

// WithTheme makes the Program own the theme. Before Init (and so before the
// first frame) a model that implements Themeable gets auto.Dark through
// SetTheme; the Program then
// asks the terminal for its background colour (OSC 11, as
// WithBackgroundDetection does, with a 300ms timeout unless that option set
// one) and, if the terminal answers, calls SetTheme again with auto.Light or
// auto.Dark to match. A terminal that does not answer keeps auto.Dark. The
// BackgroundColorEvent and BackgroundUnknownMsg still reach Update.
func WithTheme(auto theme.Auto) ProgramOption {
	return func(p *Program) { p.themeAuto, p.themeSet = auto, true }
}

// WithChords makes the Program recognise multi-key chords such as "g" "g" or
// "ctrl+x" "ctrl+s". Every Key passes through a chord matcher before Update:
// when keys complete a chord, Update receives one ChordMsg instead of those
// Keys; a key that cannot start or continue any chord is delivered at once,
// unchanged; and if a prefix turns out not to be a chord (or the timeout,
// WithChordTimeout, passes with nothing more typed) its keys are delivered in
// order, exactly as if chords were off. A key that is the start of some chord
// is therefore held back for up to the timeout, which is inherent to chords.
// Without this option keys are delivered as before.
func WithChords(defs ...ChordDef) ProgramOption {
	return func(p *Program) { p.chordDefs = append(p.chordDefs, defs...) }
}

// WithChordTimeout sets how long a chord prefix waits for its next key
// (default input.DefaultChordTimeout, 500ms). It has no effect without
// WithChords.
func WithChordTimeout(d time.Duration) ProgramOption {
	return func(p *Program) { p.chordTimeout = d }
}

// WithColorProfile makes the Program rewrite the colours in everything it
// paints to what p can display: truecolor and 256-colour escapes are
// mapped to the nearest colour p supports, and under ansi.NoColor all
// colour is removed while bold, underline and other attributes stay. It
// applies to frames and to text committed with Println and Eprintln, so it
// works with every widget without their cooperation. Typical use:
//
//	tui.NewProgram(m, tui.WithColorProfile(ansi.TrueColor))
//
// Without this option NewProgram uses ansi.DetectColorProfileFor on the
// output, which reads, in order: NO_COLOR (non-empty gives ansi.NoColor, per
// https://no-color.org); CLICOLOR_FORCE (non-empty, not "0") to colour a
// non-terminal; otherwise output that is not a terminal, or CLICOLOR=0, gives
// ansi.NoColor; TERM=dumb gives ansi.NoColor; COLORTERM (truecolor or 24bit),
// WT_SESSION and TERM_PROGRAM (iTerm.app, WezTerm, vscode, ghostty, Hyper) give
// ansi.TrueColor, and TERM_PROGRAM=Apple_Terminal gives ansi.ANSI256; then the
// TERM name (xterm-kitty, xterm-ghostty, alacritty and wezterm give
// ansi.TrueColor; 256color gives ansi.ANSI256; empty gives ansi.NoColor, or
// ansi.ANSI16 on Windows; anything else ansi.ANSI16).
// Truecolor terminals that export none of these (some SSH sessions, tmux with
// TERM=screen) are detected as lower depth; pass
// WithColorProfile(ansi.TrueColor) to opt out. Passing this option, even
// ansi.TrueColor, overrides NO_COLOR.
func WithColorProfile(p ansi.Profile) ProgramOption {
	return func(prog *Program) { prog.colorProfile, prog.colorSet = p, true }
}

// WithContext supplies the parent context.Context for the Program's run
// (default context.Background(), also used for a nil ctx). Program derives
// its own cancelable context from it and cancels that derived context when Run returns — on a
// normal quit, an error, a panic (via the same defer chain that restores
// the terminal), or a SIGTERM/SIGHUP — so a Cmd built with the context
// returned by (*Program).Context can observe shutdown instead of leaking
// past it. This never changes the Cmd func() Msg signature: a Cmd that
// wants cancellation just closes over that context itself, e.g.:
//
//	p := tui.NewProgram(model{})
//	ctx := p.Context()
//	// thread ctx into the model (a field, a closure) before p.Run()
//
//	func (m model) Init() tui.Cmd {
//	    return func() tui.Msg {
//	        req, _ := http.NewRequestWithContext(m.ctx, http.MethodGet, url, nil)
//	        resp, err := http.DefaultClient.Do(req)
//	        // ...
//	    }
//	}
func WithContext(ctx context.Context) ProgramOption {
	return func(p *Program) {
		if ctx != nil { // a nil parent keeps the default
			p.parentCtx = ctx
		}
	}
}

// WithFrameLog writes one line to w for every frame the renderer draws, so a
// flicker, an over-eager repaint or a slow frame can be seen without touching
// the terminal: point it at a file and tail it while the app runs.
//
//	n=7 t=1042 kind=diff rows=24 changed=2 bytes=118 us=61
//
// The fields are, in order: n, the frame number from 1; t, milliseconds since
// the event loop started; kind, "diff" for a frame diffed against the previous
// one, "full" for one drawn with no previous frame to diff against (the first
// frame, the repaint after a resize, and the frame after a Suspend or a
// Println) and "accessible" for a WithAccessible transcript write; rows, the
// height of the live region (the lines written, in accessible mode); changed,
// how many of those rows were rewritten (cleared rows count, and an unchanged
// frame reports 0); bytes, what the frame wrote to the terminal, including the
// synchronized-output wrapper; and us, the microseconds spent building and
// writing it. With WithCellRenderer, a frame the cell renderer could not draw
// and handed to the line renderer is kind=fallback (whatever it would
// otherwise have been) and the line ends with a further field, reason=<slug>,
// for example "reason=view_taller_than_terminal"; no other kind has that field.
// A row the cell renderer cannot represent (a control character, an unknown
// escape) is drawn alone with the line strategy and the frame stays diff or
// full; the line then ends with fallback_rows=<row>:<slug>[,...] (0-based rows
// drawn that way this frame), for example "fallback_rows=3:control_character".
//
// The line is written from the event loop after the frame, so it cannot
// reorder terminal output, and the terminal receives exactly the same bytes as
// without the option. An error from w is ignored: the log is a diagnostic and
// never stops the Program. w must not be the terminal's own output, which it
// would corrupt; a nil w disables the log.
func WithFrameLog(w io.Writer) ProgramOption {
	return func(p *Program) { p.frameLog = w }
}

// WithCellRenderer selects the cell-buffer renderer, which is the default
// (WithCellRenderer(false) selects the line renderer).
// Each frame's View is parsed once into a grid of cells and only
// the cells that changed are written, with relative cursor movement, which
// writes fewer bytes than rewriting whole changed rows. The screen it
// produces is the same as the line renderer's, with one difference: rows are
// style-independent, so a style a View row leaves open does not carry into
// the next row. Tabs are expanded. A row it cannot represent (control
// characters, escapes other than SGR and OSC 8, unknown SGR codes) is
// rewritten whole, as the line renderer would; the rest of the frame is still
// diffed. The whole frame is drawn by the line renderer when the view is
// taller than the terminal, or when an unrepresentable row appears in a frame
// that has wide characters.
func WithCellRenderer(enabled bool) ProgramOption {
	return func(p *Program) { p.cellRender = enabled }
}

// WithBidi turns on display reordering per the Unicode Bidirectional Algorithm
// (UAX #9) for terminals that draw text in logical order. Each line of the View
// is reordered by itself, taking its base direction from its first strong
// character, with every character keeping its style and link and brackets in
// right-to-left runs mirrored. It changes only what is drawn: Update still sees
// logical text, the cursor and the models' own editing are not remapped, and a
// frame drawn straight from a cell grid is not reordered. A line with no
// right-to-left character is written exactly as without the option, and with
// the option off (the default) every frame is byte-identical to one without it.
// Terminals that run the algorithm themselves must leave it off, or the text is
// reordered twice. The tables are from Unicode 17.0.0.
func WithBidi(enabled bool) ProgramOption {
	return func(p *Program) { p.bidi = enabled }
}

// Clock supplies the tickers behind Every. The default is the wall clock;
// tuitest's FakeClock is a Clock a test advances by hand.
type Clock interface {
	// NewTicker returns a channel that receives the time every d and a
	// func that stops it.
	NewTicker(d time.Duration) (<-chan time.Time, func())
}

// WithClock makes Every read its ticks from c instead of the wall clock.
func WithClock(c Clock) ProgramOption {
	return func(p *Program) { p.clock = c }
}

// defaultMaxFPS is the frame cap applied to non-input Msgs (ticks, Send,
// Cmd results) when WithMaxFPS is never called. Input Msgs (keys, mouse,
// paste, focus, chords) and resizes always render immediately.
const defaultMaxFPS = 60

// WithMaxFPS caps how often render() repaints in response to ordinary
// Msgs, coalescing a burst of rapid updates (e.g. a fast tui.Tick) into
// fewer terminal writes. When WithMaxFPS is never called a default cap of
// 60 fps applies to non-input Msgs only (a trailing flush still draws the
// latest state; input Msgs render immediately). Calling WithMaxFPS with
// fps <= 0 turns every cap off: every Msg renders. A positive fps caps
// all ordinary Msgs, input included. Update always still runs for every Msg regardless of the cap —
// only the repaint is skipped, so a skipped render's Msg still advances
// the model, and the next render paints its current, not stale, state. A
// printMsg (tui.Println) or QuitMsg always renders immediately, bypassing
// the cap, so a permanent scrollback commit or the final quit frame is
// never delayed or dropped. A repaint the cap skipped is not lost: if no
// later Msg arrives, the latest View is drawn once the frame interval has
// passed, so the screen never keeps a stale frame after a burst.
func WithMaxFPS(fps int) ProgramOption {
	return func(p *Program) { p.maxFPS, p.fpsSet = fps, true }
}

// Terminal is the control plane of the terminal a Program runs on: its size,
// raw mode and output virtual-terminal processing. Bytes still flow through the
// reader and writer given with WithInput and WithOutput (or WithInput
// and WithOutput), so a remote session supplies all three: a Terminal that
// answers for the far end, plus the connection as reader and writer. A Terminal
// that also implements ResizeNotifier reports resizes through the same port.
//
// Every restore func undoes exactly the change its call made, and is called at
// most once. Methods may be called from more than one goroutine (a signal
// handler restores while the loop runs).
type Terminal interface {
	// IsTerminal reports whether the input is an interactive terminal. Run
	// returns an error without it, unless WithInput was used, in which
	// case Run does not ask.
	IsTerminal() bool
	// Size reports the size in cells. ok is false when it is unknown; a
	// Program then keeps the size it has (80x24 for a plain writer) and
	// follows tui.ResizeMsg.
	Size() (w, h int, ok bool)
	// MakeRaw puts the input in raw mode. mouse is true when mouse reporting
	// is on; Windows turns QuickEdit off for it, other platforms ignore it.
	MakeRaw(mouse bool) (restore func() error, err error)
	// EnableOutputVT turns on the output's virtual-terminal processing.
	// restore may be nil when there was nothing to enable.
	EnableOutputVT() (restore func() error, err error)
}

// ResizeNotifier is an optional interface for a Terminal that can say when its
// size changed, so a remote session (SSH, a websocket) delivers resizes through
// the same port as everything else instead of calling Send(ResizeMsg) itself.
//
// Run calls Resizes once, and each value received means "the size may have
// changed": the Program re-reads Size and delivers one ResizeMsg. A burst of
// values while the Program is busy is coalesced into one message carrying the
// last size, so a sender should not block on the channel; a buffered channel of
// one, written with a non-blocking send, is enough. The channel may be closed
// when the session ends. A Terminal without this interface is watched the OS
// way (SIGWINCH on unix, console events on Windows) and keeps working with
// Send(ResizeMsg).
type ResizeNotifier interface {
	Resizes() <-chan struct{}
}

// WithTerminal replaces the terminal the Program controls. The default is the
// OS terminal behind the input and output files. See Terminal.
func WithTerminal(t Terminal) ProgramOption {
	return func(p *Program) { p.term = t }
}

// PanicError is returned by Run when WithRecover(true) is set and the model
// (Init, Update, View) or a Cmd panicked. Value is the recovered value and
// Stack the goroutine stack at the panic.
type PanicError struct {
	Value any
	Stack []byte
}

// Error reports the recovered value.
func (e *PanicError) Error() string { return fmt.Sprintf("tui: panic: %v", e.Value) }

// WithRecover makes Run recover panics from Init, Update, View and Cmds,
// restore the terminal, and return a *PanicError. Without it (the default)
// the terminal is restored and the panic continues.
func WithRecover(enabled bool) ProgramOption {
	return func(p *Program) { p.recoverPanics = enabled }
}

// WithEscTimeout sets how long the input parser waits for the rest of an
// escape sequence after a lone ESC byte before reporting a bare Escape key.
// d <= 0 keeps the default (input.DefaultEscTimeout).
func WithEscTimeout(d time.Duration) ProgramOption {
	return func(p *Program) { p.escTimeout = d }
}

// WithResizePoll sets how often the terminal size is polled where polling is
// used to detect resizes (Windows; Unix platforms rely on SIGWINCH and ignore
// it). d <= 0 keeps the default of 250ms.
func WithResizePoll(d time.Duration) ProgramOption {
	return func(p *Program) { p.resizePoll = d }
}

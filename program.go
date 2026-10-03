package tui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/cellbuf"
	"github.com/ows4444/tui/input"
	"github.com/ows4444/tui/internal/announce"
	"github.com/ows4444/tui/internal/render"
	"github.com/ows4444/tui/internal/termio"
	"github.com/ows4444/tui/motion"
	"github.com/ows4444/tui/theme"
)

// ErrInterrupted is returned by Run when SIGTERM or SIGHUP arrived and
// WithExitOnSignal(false) is set. The terminal has been restored.
var ErrInterrupted = errors.New("tui: interrupted by signal")

// interruptMsg is the runLoop instruction sent by the signal goroutine when
// WithExitOnSignal(false) is set. It never reaches Update.
type interruptMsg struct{}

// panicMsg carries a recovered Cmd panic to runLoop, which returns it.
type panicMsg struct{ err *PanicError }

// applyTheme hands t to the model when it is Themeable and t is not the theme
// it already has.
func (p *Program) applyTheme(t theme.Theme) {
	if p.themeGiven && p.theme == t {
		return
	}
	p.theme, p.themeGiven = t, true
	p.model, _ = setTheme(p.model, t, 0)
}

// Measurer reports how this Program's terminal measures grapheme clusters: the
// zero Measurer (the process-wide ansi.SetClusterWidth setting) until a
// capability probe (WithCapabilityProbe) has decided, then the probed answer.
// It is the same value a ResizeMsg carries. Safe to call from any goroutine.
func (p *Program) Measurer() ansi.Measurer {
	if m := p.measurer.Load(); m != nil {
		return *m
	}
	return ansi.Measurer{}
}

// resizeMsg is the ResizeMsg for a w x h terminal, carrying the current Measurer.
func (p *Program) resizeMsg(w, h int) ResizeMsg {
	m := p.Measurer()
	p.sentMeasurer = m
	return ResizeMsg{Width: w, Height: h, Measurer: m}
}

// ColorProfile reports the colour profile applied to output (see
// WithColorProfile); detected from the environment by default.
func (p *Program) ColorProfile() ansi.Profile { return p.colorProfile }

// Accessible reports whether accessible output is on (see WithAccessible).
// Valid immediately after NewProgram.
func (p *Program) Accessible() bool { return p.accessible }

// ReducedMotion reports the Program's reduced-motion preference — the
// value passed to WithReducedMotion, or motion.Detect().Reduced() if that
// option was never used. Valid immediately after NewProgram, the same as
// Context, so it can be threaded into the model before Run.
func (p *Program) ReducedMotion() bool { return p.reducedMotion }

// Clock returns the Clock given to WithClock, or nil for the wall clock.
func (p *Program) Clock() Clock { return p.clock }

// termSize reports the size of the terminal p renders to. ok is false when
// the output is not a terminal file (a WithOutput writer, a pipe).
func (p *Program) termSize() (w, h int, ok bool) { return p.terminal().Size() }

// terminal is the port to the terminal: the one set with withTerminal, else
// the OS terminal behind p.input and p.output. It is built on use because the
// options that set those files can run in any order.
func (p *Program) terminal() Terminal {
	if p.term != nil {
		return p.term
	}
	return termio.OS(p.input, p.output)
}

// write sends s to the output, ignoring errors like the rest of the renderer.
func (p *Program) write(s string) { p.writeTo(p.output, s) }

// Program drives a Model: it owns the terminal, the event loop, and the
// renderer.
type Program struct {
	clock         Clock         // nil: the wall clock
	cmdPool       cmdPool       // WithMaxConcurrentCmds limiter
	recoverPanics bool          // WithRecover: panics become *PanicError
	escTimeout    time.Duration // >0 overrides input.DefaultEscTimeout
	resizePoll    time.Duration // >0 overrides the Windows resize polling interval
	exitOnSignal  bool          // false: SIGTERM/SIGHUP makes Run return ErrInterrupted
	model         Model
	initModel     Model     // what NewProgram was given; model belongs to the loop once it runs
	output        io.Writer // *os.File unless WithOutput was used
	errOutput     io.Writer
	input         *os.File
	inReader      io.Reader // non-nil when WithInput was used; input is then unused
	inCloser      io.Closer // closed once when runLoop exits, to unblock a parked inReader
	inCloseOnce   sync.Once
	altScreen     bool
	mouseMode     MouseMode
	// resizeKick carries "the console reported a resize" from the input
	// reader to watchResize (Windows); a size re-read follows each kick.
	resizeKick       chan struct{}
	bracketedPaste   bool
	started          atomic.Bool           // set by the first Run; a second Run is ErrProgramReused
	keymapModel      atomic.Pointer[Model] // the model as of the last frame, for Keymap
	frameBuf         []byte                // render's reusable frame buffer; loop goroutine only
	altPrints        []printMsg            // Println made on the alternate screen, held until it is left
	kittyKeyboard    bool
	keyboardFlags    KeyboardFlags
	reducedMotion    bool
	accessible       bool
	modeSet          bool           // WithAltScreen or WithAccessible was given (see applyDumbTermDefaults)
	polite           announce.Queue // accessible polite lines queued and de-duplicated; loop goroutine only
	fullLine         bool           // WithLinearizeFullLine
	announce         announce.Region
	colorProfile     ansi.Profile
	colorSet         bool // WithColorProfile was used, so NO_COLOR is not consulted
	focusReporting   bool
	bgTimeout        time.Duration
	themeAuto        theme.Auto // WithTheme
	themeSet         bool
	theme            theme.Theme // the theme last given to the model
	themeGiven       bool
	bgSeen           atomic.Bool                   // a BackgroundColorEvent has been read
	clusterMode      bool                          // mode 2027 set by applyCapabilities; guarded by outMu
	measurer         atomic.Pointer[ansi.Measurer] // how this terminal measures clusters; nil until a probe says
	sentMeasurer     ansi.Measurer                 // the Measurer last put in a ResizeMsg; loop goroutine only
	inspectorKey     string                        // WithInspector Panel; "" means off
	inspectorOn      bool                          // the overlay is showing; loop goroutine only
	outlineKey       string                        // WithInspector Layout; "" means off
	outlineOn        bool                          // the layout outlines are showing; loop goroutine only
	lastFrameBytes   int                           // the last frame's size and duration, for the inspector
	lastFrameDur     time.Duration
	recCast, recSide io.Writer    // WithRecorder, WithRecorderSidecar
	rec              *recorder    // non-nil while recording
	tickSeq          int          // Every tickers started, for the recorder; loop goroutine only
	msgKey, modelKey string       // WithInspector Messages / Model; "" means off
	msgOn, modelOn   bool         // the message and model panes are showing; loop goroutine only
	msgLog           msgRing      // recent messages, for the inspector; loop goroutine only
	titleSaved       bool         // SetWindowTitle pushed the title (CSI 22;0t); guarded by outMu
	cursorShaped     bool         // SetCursorShape changed the cursor shape; guarded by outMu
	clipReads        atomic.Int32 // ReadClipboard queries not yet answered
	capProbe         capProbe     // WithCapabilityProbe; see capabilities.go
	chordDefs        []ChordDef
	chordTimeout     time.Duration
	chords           *input.ChordMatcher // nil unless WithChords was used
	chordTimer       *time.Timer

	// parentCtx is WithContext's chosen parent (context.Background() if
	// unset); ctx/cancel are derived from it in NewProgram, so ctx is valid
	// as soon as the caller has a *Program, before Run is ever called.
	parentCtx context.Context
	ctx       context.Context
	cancel    context.CancelFunc

	msgs      chan Msg
	lastFrame []string
	// frameSpare and splitBuf are reused storage for renderBody (see there).
	frameSpare, splitBuf []string

	// liveLines is how many terminal rows the live region (the last
	// painted frame) currently occupies. After render() the cursor rests
	// on the last of those rows (see toRegionTop); after printLines it's
	// on the row below the committed text, with liveLines back at 0. It's
	// how render() knows how far to move the cursor up (via relative
	// addressing) to get back to the top of the region before repainting,
	// and how printLines knows how far to move up to erase the region
	// before committing text above it. It's 0 until the first render().
	liveLines int

	// curShown and curUp track the hardware cursor a CursorPlacer asked
	// for: whether it is currently shown, and how many rows above the last
	// live row it was parked. Every write that assumes the cursor rests on
	// the last row starts with unparkCursor, which undoes both. curWant is
	// the cell the last frame parked it at, so a frame skipped because the
	// View is unchanged can still follow a cursor that moved.
	curShown bool
	curUp    int
	curWant  cursorCell

	// committed is how many leading rows of the current inline View have
	// been committed to scrollback because the view exceeded the terminal
	// height; only the rows after them are live (see splitOverflow).
	committed int

	width, height int

	// bidi is WithBidi: reorder each view line for display.
	bidi bool
	// cellRender is WithCellRenderer; cells is its state (nil until used).
	cellRender bool
	cells      *render.Cells
	drawBuf    *cellbuf.Buffer // the grid a CellDrawer root draws into, reused across frames
	// regionTopFn and fitLineFn are p.toRegionTop and p.fitLine as func values,
	// made once so each direct frame does not allocate them; topUp and topSeq
	// memoise toRegionTop's last result.
	regionTopFn func() string
	fitLineFn   func(string) string
	topUp       int
	topSeq      string

	// frameLog, frameN and loopStart implement WithFrameLog; frameStats is
	// what renderBody recorded about the frame it just built.
	frameLog   io.Writer
	frameN     int
	loopStart  time.Time
	frameStats frameStats

	// maxFPS and lastRenderAt implement WithMaxFPS's render-rate cap; see
	// allowRender. maxFPS <= 0 means unthrottled.
	maxFPS int
	// lastView is the raw View string of the last frame renderBody built;
	// peekedView carries a View result from renderIfChanged into renderBody.
	lastView, peekedView string
	haveView, hasPeeked  bool
	fpsSet               bool // WithMaxFPS was called; otherwise defaultMaxFPS gates non-input Msgs
	lastRenderAt         time.Time
	// renderPending is set when allowRender throttled a render away, and
	// cleared by any render; flushTimer is the single timer that draws the
	// pending frame once the interval has passed (nil while none is armed).
	renderPending bool
	flushTimer    *time.Timer

	// term is the terminal port; nil means the OS terminal (see terminal).
	term Terminal
	// termRestore undoes Run's raw mode. It is promoted here (rather than
	// staying local to Run) so the suspend handler inside runLoop can restore
	// and re-enter raw mode. It is nil until Run's MakeRaw succeeds (in
	// particular, in a test that drives runLoop directly without calling Run
	// against a real terminal) — suspend treats nil as "nothing to
	// restore/re-raw," skipping just those two calls while still running
	// everything else (the ansi setup toggling, fn, and the frame-state
	// reset), which is what lets Suspend be tested without a real terminal.
	//
	// termMu guards termRestore (and termRestored): the signal goroutine's
	// restoreTerminal reads it while a Suspend on the loop goroutine
	// rewrites it. termRestored is set once restoreTerminal has run, so a
	// Suspend still inside fn does not put the terminal back into raw mode
	// after a SIGTERM/SIGHUP already restored it.
	termMu       sync.Mutex
	termRestore  func() error
	termRestored bool

	// suspendOnCtrlZ is WithSuspendOnCtrlZ; stopProcess replaces the
	// platform's stop-and-wait-for-SIGCONT in tests.
	suspendOnCtrlZ bool
	stopProcess    func() error

	// stopInput/startInput are set by runLoop while it owns a stoppable input
	// reader (nil otherwise, e.g. WithInput or a test calling suspend
	// directly). stopInput returns only once the reader goroutine has exited
	// and holds no byte; startInput begins a new one. rdDone is the current
	// reader's exit channel. They are touched only on the loop goroutine.
	stopInput  func()
	startInput func()
	rdDone     chan struct{}

	// wg tracks the background goroutines started by runLoop (input reader,
	// resize watcher) purely for observability: tests in this package wait
	// on it, with a timeout, to confirm those goroutines actually exit
	// after runLoop returns rather than leaking. Run itself never waits on
	// it — that would defeat the point of returning promptly on quit.
	wg sync.WaitGroup

	// restoreOnce guards restoreTerminal so it's safe to call it both from
	// Run's normal defer chain and from the SIGTERM/SIGHUP handler racing
	// against it — whichever runs first does the actual work.
	restoreOnce sync.Once

	// outMu serialises every terminal write (frames, prints, mode changes and
	// the restore sequence); outClosed, guarded by it, is set once the
	// terminal has been restored so no later write can land after the restore
	// bytes. See program_restore.go.
	outMu     sync.Mutex
	outClosed bool

	// loopMu guards loopDone: non-nil, and closed on exit, while runLoop is
	// draining msgs. See Send.
	loopMu   sync.RWMutex
	loopDone chan struct{}
	// loopStarted is closed (once, via loopStartedOnce) when the loop first
	// starts or Run returns, so a Send blocked on a full queue before Run
	// wakes without polling.
	loopStarted     chan struct{}
	loopStartedOnce sync.Once
	// loopEnded is set once Run has returned; Send then drops messages.
	loopEnded bool

	// restoreOutputVT undoes the output console's virtual-terminal
	// processing (see Terminal.EnableOutputVT) in restoreTerminal.
	restoreOutputVT func() error
}

// enterOutputVT enables VT processing on the output when it is a file,
// remembering how to undo it. The error names the failing setting.
func (p *Program) enterOutputVT() error {
	restore, err := p.terminal().EnableOutputVT()
	if err != nil {
		return fmt.Errorf("tui: enable output virtual terminal processing: %w", err)
	}
	p.restoreOutputVT = restore
	return nil
}

// enterModes switches on every terminal mode the Program's options ask
// for, in a fixed order: alternate screen, hidden cursor, mouse reporting,
// bracketed paste, kitty keyboard, focus reporting. Run calls it at
// startup and suspend calls it again to redo the setup, so the two can't
// drift apart.
func (p *Program) enterModes() {
	if p.accessible {
		// Accessible mode writes no escape sequences at all.
		p.curShown, p.curUp, p.curWant = false, 0, cursorCell{}
		return
	}
	if p.altScreen {
		p.write(ansi.AltScreenEnable)
	}
	if !p.accessible {
		p.write(ansi.CursorHide)
	}
	p.curShown, p.curUp, p.curWant = false, 0, cursorCell{}
	if p.mouseMode != MouseOff {
		enable, _ := mouseModeCodes(p.mouseMode)
		p.write(enable + ansi.MouseSGREnable)
	}
	if p.bracketedPaste {
		p.write(ansi.BracketedPasteEnable)
	}
	if f := p.kittyFlags(); f != 0 {
		p.write(ansi.KittyKeyboardEnableFlags(int(f)))
	}
	if p.focusReporting {
		p.write(ansi.FocusReportingEnable)
	}
	if p.clusterModeOn() {
		p.write(clusterModeEnable)
	}
}

// leaveModes undoes enterModes in exactly the reverse order, and shows the
// cursor. restoreTerminal and suspend both use it.
func (p *Program) leaveModes() {
	if p.accessible {
		return
	}
	p.write(p.unparkCursor())
	if p.clusterModeOn() {
		p.write(clusterModeDisable)
	}
	if p.focusReporting {
		p.write(ansi.FocusReportingDisable)
	}
	if p.kittyFlags() != 0 {
		p.write(ansi.KittyKeyboardDisable)
	}
	if p.bracketedPaste {
		p.write(ansi.BracketedPasteDisable)
	}
	if p.mouseMode != MouseOff {
		_, disable := mouseModeCodes(p.mouseMode)
		p.write(ansi.MouseSGRDisable + disable)
	}
	p.write(ansi.CursorShow)
	if p.altScreen {
		p.write(ansi.AltScreenDisable)
		p.flushAltPrints()
	}
	p.write(p.takeTitleAndShapeRestore())
}

// restoreTerminal undoes Run's raw-mode/alt-screen/mouse/bracketed-paste
// terminal setup exactly once. Run's defer chain calls it on every normal
// or panicking return; Run also calls it directly from a SIGTERM/SIGHUP
// handler, since Go's default disposition for those signals terminates the
// process without running deferred functions — leaving the terminal in
// raw/alt-screen/mouse-reporting state for a plain `kill <pid>` otherwise.
func (p *Program) restoreTerminal() {
	p.restoreOnce.Do(func() {
		p.leaveModes()
		if !p.altScreen && !p.accessible && p.liveLines > 0 {
			// End below the final frame so the shell prompt starts on a fresh row.
			p.write("\r\n")
		}
		if p.restoreOutputVT != nil {
			_ = p.restoreOutputVT()
		}
		p.termMu.Lock()
		p.termRestored = true
		if p.termRestore != nil {
			_ = p.termRestore()
		}
		p.termMu.Unlock()
		p.closeOutput()
	})
}

// NewProgram returns a Program running m, writing to stdout in the
// alternate screen buffer by default; opts override those defaults.
func NewProgram(m Model, opts ...ProgramOption) *Program {
	p := &Program{
		model:         m,
		initModel:     m,
		output:        os.Stdout,
		errOutput:     os.Stderr,
		input:         os.Stdin,
		altScreen:     true,
		parentCtx:     context.Background(),
		reducedMotion: motion.Detect().Reduced(),
		msgs:          make(chan Msg, 64),
		resizeKick:    make(chan struct{}, 1),
		loopStarted:   make(chan struct{}),
		exitOnSignal:  true,
		cellRender:    true, // the default renderer; WithCellRenderer(false) opts out

		bracketedPaste: true,
	}
	for _, opt := range opts {
		opt(p)
	}
	if !p.colorSet {
		p.colorProfile = ansi.DetectColorProfileFor(p.output, os.Getenv)
	}
	p.applyDumbTermDefaults()
	p.setupRecorder()
	if p.accessible {
		// Applied after every option so option order can't undo it.
		p.reducedMotion = true
		p.altScreen = false
	}
	if len(p.chordDefs) > 0 {
		p.chords = input.NewChordMatcher(p.chordDefs...)
		if p.chordTimeout > 0 {
			p.chords.Timeout = p.chordTimeout
		}
	}
	p.ctx, p.cancel = context.WithCancel(p.parentCtx)
	p.keymapModel.Store(&m)
	return p
}

// Context returns the Program's run-scoped context.Context, derived from
// the context passed to WithContext (or context.Background(), if none was
// given). It's valid immediately — call it any time after NewProgram, in
// particular before Run, so the value can be threaded into the model — and
// is cancelled when Run returns, by whatever path (see WithContext).
func (p *Program) Context() context.Context { return p.ctx }

// ErrProgramReused is returned by Run when the Program has already run (or is
// running). A Program is single-use: its context is cancelled and its message
// queue drained when Run returns, so build a new one with NewProgram.
var ErrProgramReused = errors.New("tui: Program.Run called more than once; build a new Program with NewProgram")

// Run puts the terminal in raw mode, starts the event loop, and blocks
// until the model quits (via tui.Quit) or an unrecoverable error occurs.
// The terminal is always restored before Run returns. A Program runs once:
// a second call returns ErrProgramReused without touching the terminal.
func (p *Program) Run() (Model, error) {
	if !p.started.CompareAndSwap(false, true) {
		// The first Run may still be in its loop, which owns p.model.
		if _, ended := p.loopState(); ended {
			return p.model, ErrProgramReused
		}
		return p.initModel, ErrProgramReused
	}
	// Every return path, including the ones before the loop starts, cancels
	// Context and releases a Send waiting for the loop.
	defer func() {
		p.cancel()
		p.setLoopDone(nil)
	}()
	if p.inReader == nil {
		if !p.terminal().IsTerminal() {
			return p.model, errors.New("tui: input is not a terminal (stdin is piped or redirected); run from a terminal, or supply input with WithInput")
		}

		// Clear any read deadline a previous Run() on this same file may have
		// left in the past (see the deadline runLoop sets before returning) —
		// without this, a second sequential Run() sharing the input file would
		// find every read failing immediately.
		_ = p.input.SetReadDeadline(time.Time{})

		// With the mouse on, Windows QuickEdit is switched off (it would eat
		// clicks); the State keeps the original mode, so every restore path
		// brings it back. A no-op elsewhere.
		restore, err := p.terminal().MakeRaw(p.mouseMode != MouseOff)
		if err != nil {
			return p.model, fmt.Errorf("tui: cannot enter raw mode on the input terminal: %w; if input is not an interactive terminal, supply it with WithInput", err)
		}
		p.termRestore = restore
	}
	defer p.restoreTerminal()
	// Cancels the context returned by (*Program).Context, so a Cmd built
	// from it (see WithContext) observes shutdown on every return path:
	// normal quit, error, or panic (defers still run during ordinary panic
	// unwinding), same as restoreTerminal above.
	defer p.cancel()

	if err := p.enterOutputVT(); err != nil {
		return p.model, err
	}

	// SIGTERM/SIGHUP terminate a process by default without running
	// deferred functions, so restoreTerminal/cancel above would never fire
	// for a plain `kill <pid>` or a closed controlling terminal without
	// this: watch for them explicitly, restore and cancel, then exit.
	// signal.Stop plus sigWatchDone (checked first, so it wins any
	// shutdown-vs-signal race) keep this goroutine from outliving Run.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGHUP)
	sigWatchDone := make(chan struct{})
	defer func() {
		signal.Stop(sigCh)
		close(sigWatchDone)
	}()
	go func() {
		select {
		case <-sigWatchDone:
		case <-sigCh:
			p.restoreViaLoop()
			p.cancel()
			if p.exitOnSignal {
				os.Exit(1)
			}
			select {
			case p.msgs <- interruptMsg{}:
			case <-sigWatchDone:
			}
		}
	}()

	if w, h, ok := p.termSize(); ok {
		p.width, p.height = w, h
	} else if _, isFile := p.output.(*os.File); !isFile {
		p.width, p.height = 80, 24 // a plain writer has no terminal to ask
	}

	p.rec.begin(p.width, p.height)
	p.enterModes()

	return p.runLoop()
}

func mouseModeCodes(mode MouseMode) (enable, disable string) {
	switch mode {
	case MouseClick:
		return ansi.MouseClickEnable, ansi.MouseClickDisable
	case MouseCellMotion:
		return ansi.MouseCellMotionEnable, ansi.MouseCellMotionDisable
	case MouseAllMotion:
		return ansi.MouseAllMotionEnable, ansi.MouseAllMotionDisable
	default:
		return "", ""
	}
}

// The OS adapter and the test fake implement the one Terminal port.
var (
	_ Terminal = termio.OSTerminal{}
	_ Terminal = (*termio.Fake)(nil)

	_ ResizeNotifier = (*termio.Fake)(nil)
)

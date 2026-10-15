package tui

// ViewState is what a Model asks of the terminal for the frame it is drawing,
// stated beside its View. Every field is a pointer: nil means "no opinion",
// and the mode keeps whatever the program options and runtime Cmds left it at.
// A field that is set is applied before the frame is drawn, when it differs
// from the terminal's current state.
//
//	func (m model) ViewState() tui.ViewState {
//		return tui.ViewState{
//			AltScreen: new(m.fullscreen),
//			Title:     new("editing " + m.file),
//		}
//	}
//
// It is the declarative counterpart of the EnableMouse, EnterAltScreen,
// ExitAltScreen and SetWindowTitle Cmds, which stay available.
type ViewState struct {
	// AltScreen is whether the alternate screen is in use.
	AltScreen *bool
	// Mouse is the mouse reporting mode.
	Mouse *MouseMode
	// Title is the terminal window title.
	Title *string
}

// ViewStater is implemented by a Model that states a ViewState. The Program
// calls ViewState before each frame is drawn, so it should be cheap and free
// of side effects. A model wrapped with Unwrap is found through its wrappers,
// as CursorPlacer is. It is ignored in accessible mode, which has no screen.
type ViewStater interface {
	ViewState() ViewState
}

// applyViewState brings the terminal to the state the model asks for. It runs
// on the event loop goroutine, where the mode Cmds run, before a frame is drawn.
func (p *Program) applyViewState() {
	if p.accessible {
		return
	}
	vs, ok := find[ViewStater](p.model)
	if !ok {
		return
	}
	st := vs.ViewState()
	run := func(c Cmd) {
		if m, ok := c().(modeMsg); ok {
			m.apply(p)
		}
	}

	p.outMu.Lock()
	alt, mouse := p.altScreen, p.mouseMode
	p.outMu.Unlock()

	if st.AltScreen != nil && *st.AltScreen != alt {
		if *st.AltScreen {
			run(EnterAltScreen())
		} else {
			run(ExitAltScreen())
		}
	}
	if st.Mouse != nil && *st.Mouse != mouse {
		run(EnableMouse(*st.Mouse))
	}
	if st.Title != nil && (!p.viewTitleSet || *st.Title != p.viewTitle) {
		p.viewTitle, p.viewTitleSet = *st.Title, true
		run(SetWindowTitle(*st.Title))
	}
}

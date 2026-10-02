package tui

import "context"

// CtxCmd is a Cmd that receives the Program's context. See FromCtx.
type CtxCmd func(context.Context) Msg

// ctxMsg asks the event loop to run fn with the Program's context.
type ctxMsg struct{ fn CtxCmd }

// String makes a ctxMsg that escaped the Program (a Cmd called directly, or
// its result wrapped in another Msg) explain itself in logs and test output.
func (ctxMsg) String() string {
	return "tui: unrun context Cmd (FromCtx, Tick or a motion wait); return it to the Program or run it with tui.RunCmd"
}

// FromCtx adapts fn to a Cmd. The Program runs fn on its own goroutine and
// passes it Program.Context, which is cancelled as Run returns (by any path,
// including cancellation of the context given to WithContext), so fn can stop
// early. A Cmd that fn itself calls does not see Run returning; use the
// context instead. A nil fn gives a nil Cmd.
func FromCtx(fn CtxCmd) Cmd {
	if fn == nil {
		return nil
	}
	return func() Msg { return ctxMsg{fn: fn} }
}

// dispatchCtx runs fn with the Program context and delivers its result.
func (p *Program) dispatchCtx(fn CtxCmd, done <-chan struct{}) {
	defer func() {
		if r := recover(); r != nil {
			p.restoreViaLoop()
			panic(r)
		}
	}()
	msg := fn(p.ctx)
	if msg == nil {
		return
	}
	select {
	case p.msgs <- msg:
	case <-done:
	}
}

// RunCmd runs cmd the way a Program does and returns the Msg it produces, for
// tests that call a Cmd directly. A Cmd made by FromCtx, Tick or the motion
// package waits on a context, which a plain cmd() call cannot supply: RunCmd
// runs it with ctx, so cancelling ctx ends the wait and gives a nil Msg. A nil
// cmd gives a nil Msg. Batch and Sequence are not expanded; they are the
// Program's to run.
func RunCmd(ctx context.Context, cmd Cmd) Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if cm, ok := msg.(ctxMsg); ok {
		return cm.fn(ctx)
	}
	return msg
}

package tui

import (
	"context"
	"time"
)

// Map wraps every Msg that cmd produces with f, so a parent can tag or convert
// the messages of a child's Cmd without a hand-written envelope Cmd. It is the
// same idea as Bubble Tea's pattern of wrapping a child's Cmd, done once.
//
// Map sees through Batch, Sequence, FromCtx, Go, Tick and Every: it wraps the
// message those eventually deliver, and keeps running them as the Program
// would. Instructions to the Program (Quit, Println, Suspend and the terminal
// mode Cmds) pass through unchanged, since they are not application messages.
// If f returns nil the message is dropped. A nil cmd gives a nil Cmd, and a nil
// f gives cmd unchanged.
func Map(cmd Cmd, f func(Msg) Msg) Cmd {
	if cmd == nil || f == nil {
		return cmd
	}
	return func() Msg { return mapMsg(cmd(), f) }
}

// MapAs is Map for one message type: a Msg that is a T is replaced with f(t);
// anything else passes through as it is. Use it when the parent cares about a
// child's own result (a datatable's SelectedMsg, say) and wants it as its own
// type without touching the child's other messages.
func MapAs[T any](cmd Cmd, f func(T) Msg) Cmd {
	if f == nil {
		return cmd
	}
	return Map(cmd, func(m Msg) Msg {
		if t, ok := m.(T); ok {
			return f(t)
		}
		return m
	})
}

func mapMsg(msg Msg, f func(Msg) Msg) Msg {
	switch m := msg.(type) {
	case nil:
		return nil
	case batchMsg:
		out := make(batchMsg, len(m))
		for i, c := range m {
			out[i] = Map(c, f)
		}
		return out
	case sequenceMsg:
		out := make(sequenceMsg, len(m))
		for i, c := range m {
			out[i] = Map(c, f)
		}
		return out
	case ctxMsg:
		inner := m.fn
		return ctxMsg{fn: func(ctx context.Context) Msg { return mapMsg(inner(ctx), f) }}
	case everyMsg:
		if inner := m.fn; inner != nil {
			m.fn = func(t time.Time) Msg { return mapMsg(inner(t), f) }
		}
		return m
	case QuitMsg, printMsg, suspendMsg, modeMsg:
		return msg
	}
	return f(msg)
}

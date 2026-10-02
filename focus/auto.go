package focus

import (
	"github.com/ows4444/tui"
	"github.com/ows4444/tui/hittest"
)

// RouteAuto is Route with the two rules a screen otherwise hand-writes:
//
//   - Tab and Shift+Tab move focus (wrapping, skipping disabled items) unless
//     the focused item's Consumes reports it wants the key, in which case the
//     key goes to that item's Update and focus stays.
//   - A left-button press on a zones region focuses the item whose index is the
//     region's ID (blurring the old one), then delivers the press to it. A
//     press outside every region, or on a disabled or unknown item, goes to
//     the focused item as any other message would.
//
// Every other message goes to the focused item only, as in Route. Pass a zero
// hittest.Map to turn mouse focusing off. Programs that do not call RouteAuto are
// unaffected.
func (r Ring) RouteAuto(msg tui.Msg, zones hittest.Map[int], fields ...Field) (Ring, tui.Cmd) {
	switch m := msg.(type) {
	case tui.Key:
		if m.Type == tui.KeyTab {
			if f, ok := fieldAt(fields, r.Current()); ok && f.Consumes != nil && f.Consumes(msg) {
				return r.deliver(msg, fields)
			}
		}
	case tui.MouseEvent:
		if m.Button == tui.MouseButtonLeft && m.Action == tui.MouseActionPress {
			if h, ok := zones.AtEvent(m); ok && r.Enabled(h.ID) && h.ID != r.cur {
				next := r.Set(h.ID)
				blur(fields, r.cur)
				focusCmd := focusField(fields, next.cur)
				_, cmd := next.deliver(msg, fields)
				return next, tui.Batch(focusCmd, cmd)
			}
		}
	}
	return r.Route(msg, fields...)
}

func (r Ring) deliver(msg tui.Msg, fields []Field) (Ring, tui.Cmd) {
	if f, ok := fieldAt(fields, r.Current()); ok && f.Update != nil {
		return r, f.Update(msg)
	}
	return r, nil
}

package input

// MouseButton identifies which mouse button (or wheel direction) a
// MouseEvent refers to.
type MouseButton int

// The mouse buttons. The wheel is reported as four buttons, one per
// direction, with MouseActionPress. Back, Forward, MouseButton10 and
// MouseButton11 are the extra buttons of SGR buttons 8-11.
const (
	MouseButtonNone MouseButton = iota
	MouseButtonLeft
	MouseButtonMiddle
	MouseButtonRight
	MouseButtonWheelUp
	MouseButtonWheelDown
	MouseButtonWheelLeft
	MouseButtonWheelRight
	MouseButtonBack    // SGR button 8 (Cb 128)
	MouseButtonForward // SGR button 9 (Cb 129)
	MouseButton10      // SGR button 10 (Cb 130)
	MouseButton11      // SGR button 11 (Cb 131)
)

// MouseAction is what happened to MouseButton.
type MouseAction int

// The mouse actions: a button pressed or released, or the pointer moved
// (a drag while a button is held, or free movement, depending on the
// tracking mode the Program enabled).
const (
	MouseActionPress MouseAction = iota
	MouseActionRelease
	MouseActionMotion
)

// MouseEvent is a decoded SGR mouse report. X and Y are 0-indexed terminal
// cell coordinates.
type MouseEvent struct {
	X, Y   int
	Button MouseButton
	Action MouseAction
	Mod    Mod
}

// decodeSGRMouse decodes an SGR mouse report's Cb (button+modifier byte),
// Cx/Cy (1-indexed coordinates, converted to 0-indexed here), and whether
// the sequence terminated with 'm' (release) rather than 'M' (press/drag).
//
// Cb layout: bits 0-1 select the button number; bit 2 (4) = Shift, bit 3
// (8) = Alt, bit 4 (16) = Ctrl; bit 5 (32) set means this is a drag/motion
// report rather than a discrete press; bit 6 (64) set means it's a wheel
// event, with bits 0-1 then selecting up (0) or down (1) instead of a
// button number, and then bit 1 selects left (2) or right (3) wheel scrolling
// as well; bit 7 (128) set means one of the extra buttons 8-11, with bits 0-1
// selecting which.
func decodeSGRMouse(cb, x, y int, isRelease bool) MouseEvent {
	ev := MouseEvent{X: x - 1, Y: y - 1}
	if cb&4 != 0 {
		ev.Mod |= ModShift
	}
	if cb&8 != 0 {
		ev.Mod |= ModAlt
	}
	if cb&16 != 0 {
		ev.Mod |= ModCtrl
	}

	button := func() MouseButton {
		if cb&128 != 0 {
			return MouseButtonBack + MouseButton(cb&3)
		}
		switch cb & 3 {
		case 0:
			return MouseButtonLeft
		case 1:
			return MouseButtonMiddle
		case 2:
			return MouseButtonRight
		default:
			return MouseButtonNone
		}
	}

	switch {
	case cb&64 != 0:
		ev.Action = MouseActionPress
		ev.Button = [4]MouseButton{MouseButtonWheelUp, MouseButtonWheelDown, MouseButtonWheelLeft, MouseButtonWheelRight}[cb&3]
	case cb&32 != 0:
		ev.Action = MouseActionMotion
		ev.Button = button()
	case isRelease:
		ev.Action = MouseActionRelease
		ev.Button = button()
	default:
		ev.Action = MouseActionPress
		ev.Button = button()
	}
	return ev
}

package input

// FocusEvent reports the terminal window gaining (Focused true, ESC[I) or
// losing (Focused false, ESC[O) focus. Terminals only send it after focus
// reporting is enabled (DECSET 1004).
type FocusEvent struct {
	Focused bool
}

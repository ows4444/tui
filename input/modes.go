package input

// ColorSchemeEvent is the terminal telling the program whether its colour
// scheme is dark or light, ESC[?997;1n (dark) or ESC[?997;2n (light). A
// terminal sends it after colour-scheme reporting is enabled (DECSET 2031) and
// again whenever the user switches scheme, so a program can follow a terminal
// that flips between light and dark while it runs.
type ColorSchemeEvent struct {
	Dark bool
}

// InBandResizeEvent is the terminal reporting its size, ESC[48;rows;cols;hpx;wpx t,
// after in-band resize reporting is enabled (DECSET 2048). It reaches the
// program through the input stream, so it works where a window-size signal
// does not (over SSH, in some multiplexers). The pixel sizes are 0 when the
// terminal does not know them.
type InBandResizeEvent struct {
	Rows, Cols        int
	HeightPx, WidthPx int
}

// privateReply decodes the CSI replies with a private prefix that carry a
// terminal state change and not a key.
func privateReply(final byte, params []byte) (Event, bool) {
	if final != 'n' {
		return nil, false
	}
	switch string(params) {
	case "?997;1":
		return ColorSchemeEvent{Dark: true}, true
	case "?997;2":
		return ColorSchemeEvent{Dark: false}, true
	}
	return nil, false
}

// inBandResize decodes ESC[48;rows;cols[;hpx;wpx]t from its parameter groups.
func inBandResize(groups [][]int) (Event, bool) {
	if len(groups) < 3 || groups[0][0] != 48 {
		return nil, false
	}
	ev := InBandResizeEvent{Rows: groups[1][0], Cols: groups[2][0]}
	if len(groups) >= 5 {
		ev.HeightPx, ev.WidthPx = groups[3][0], groups[4][0]
	}
	return ev, true
}

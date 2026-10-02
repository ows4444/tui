// Package capprobe holds the terminal capability probe's wire format: the
// queries a Program writes at start-up and the parser that turns the replies
// into a Result. It does no I/O and keeps no locks: the runtime owns the
// timing, the goroutines and the delivery of the result to the model, and
// this package only knows what the bytes mean, so it can be tested against
// recorded replies from real terminals.
package capprobe

import (
	"strconv"
	"strings"
)

// Result is what the terminal reported. The zero value means "nothing was
// confirmed", which is also what a terminal that never answers yields. The
// fields are documented on tui.Capabilities, which has the same shape.
type Result struct {
	SyncOutput       bool
	GraphemeClusters bool
	KittyKeyboard    bool
	KittyGraphics    bool
	Sixel            bool
	StyledUnderline  bool
	Notifications    bool
	XTVersion        string
}

// The probe queries. DA1 goes last: terminals answer in order, so its reply
// means every earlier query has been answered or ignored.
const (
	QueryDECRQM2026    = "\x1b[?2026$p"
	QueryDECRQM2027    = "\x1b[?2027$p"
	QueryKittyKeyboard = "\x1b[?u"
	QueryKittyGraphics = "\x1b_Gi=31,s=1,v=1,a=q,t=d,f=24;AAAA\x1b\\"
	QueryXTVersion     = "\x1b[>0q"
	QueryNotify        = "\x1b]99;i=32:p=?;\x1b\\"
	QueryDA1           = "\x1b[c"

	// Queries is every query, in the order they are written.
	Queries = QueryDECRQM2026 + QueryDECRQM2027 + QueryKittyKeyboard +
		QueryKittyGraphics + QueryXTVersion + QueryNotify + QueryDA1
)

// Mode 2027 (grapheme cluster width) set and reset.
const (
	ClusterModeEnable  = "\x1b[?2027h"
	ClusterModeDisable = "\x1b[?2027l"
)

// Parser accumulates probe replies. The zero value is ready. It is not safe
// for concurrent use; the caller serialises Observe.
type Parser struct {
	acc Result
}

// Observe feeds one terminal reply: kind is the introducer of a string reply
// (']' OSC, 'P' DCS, '_' APC) or '[' for a CSI reply, and data is the payload
// without introducer or terminator (for CSI, the bytes between "ESC[" and the
// final byte inclusive).
//
// consumed reports that the reply belongs to the probe and must not reach the
// model. done reports that the DA1 sentinel arrived: res is then the final
// Result and the probe is over.
func (p *Parser) Observe(kind byte, data string) (consumed, done bool, res Result) {
	acc := &p.acc
	switch kind {
	case '[':
		consumed = true
		switch {
		case strings.HasPrefix(data, "?") && strings.HasSuffix(data, "$y"):
			mode, val, ok := parseDECRPM(data)
			if ok {
				supported := val == 1 || val == 2 || val == 3
				switch mode {
				case 2026:
					acc.SyncOutput = supported
				case 2027:
					acc.GraphemeClusters = supported
				}
			}
		case strings.HasPrefix(data, "?") && strings.HasSuffix(data, "u"):
			acc.KittyKeyboard = true
		case strings.HasPrefix(data, "?") && strings.HasSuffix(data, "c"):
			done = true
			acc.Sixel = da1HasSixel(data)
		default:
			consumed = false // some other CSI reply: not ours
		}
	case 'P':
		if v, ok := strings.CutPrefix(data, ">|"); ok {
			acc.XTVersion, consumed = v, true
		}
	case ']':
		if strings.HasPrefix(data, "99;") && strings.Contains(data, "i=32") {
			acc.Notifications = true
			consumed = true
		}
	case '_':
		if strings.HasPrefix(data, "G") && strings.Contains(data, "i=31") {
			acc.KittyGraphics = strings.Contains(data, ";OK")
			consumed = true
		}
	}
	if done {
		res = *acc
		res.StyledUnderline = InferStyledUnderline(res)
	}
	return consumed, done, res
}

// da1HasSixel reports whether the DA1 reply "?<terminal>;<feature>;...c" lists
// feature 4, Sixel graphics.
func da1HasSixel(data string) bool {
	attrs := strings.TrimSuffix(strings.TrimPrefix(data, "?"), "c")
	for _, a := range strings.Split(attrs, ";") {
		if a == "4" {
			return true
		}
	}
	return false
}

// parseDECRPM parses "?<mode>;<value>$y".
func parseDECRPM(d string) (mode, val int, ok bool) {
	body := strings.TrimSuffix(strings.TrimPrefix(d, "?"), "$y")
	m, v, found := strings.Cut(body, ";")
	if !found {
		return 0, 0, false
	}
	mode, err1 := strconv.Atoi(m)
	val, err2 := strconv.Atoi(v)
	return mode, val, err1 == nil && err2 == nil
}

var styledUnderlineTerminals = []string{"kitty", "wezterm", "foot", "ghostty", "iterm2", "contour", "mintty", "alacritty", "vte", "konsole"}

// InferStyledUnderline reports whether extended underlines (SGR 4:n) are
// believed to render: true when the terminal spoke a kitty protocol or its
// XTVERSION names a terminal known to support them. It is inferred, not queried.
func InferStyledUnderline(r Result) bool {
	if r.KittyKeyboard || r.KittyGraphics {
		return true
	}
	name := strings.ToLower(r.XTVersion)
	for _, t := range styledUnderlineTerminals {
		if strings.HasPrefix(name, t) {
			return true
		}
	}
	return false
}

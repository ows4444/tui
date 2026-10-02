package ansi

import "strings"

// penState is the SGR/OSC 8 state active at a point in a string: the SGR
// sequences applied since the last reset, and the open OSC 8 hyperlink
// sequence (empty when no link is open).
type penState struct {
	sgr  string
	link string
}

func (p penState) active() bool { return p.sgr != "" || p.link != "" }

// apply folds one complete escape sequence into the state.
func (p *penState) apply(seq string) {
	if len(seq) >= 3 && seq[1] == '[' && seq[len(seq)-1] == 'm' {
		params := seq[2 : len(seq)-1]
		// A leading 0 (or an empty parameter list) resets the pen first.
		if params == "" {
			p.sgr = ""
			return
		}
		first, rest, more := strings.Cut(params, ";")
		if strings.Trim(first, "0") == "" {
			p.sgr = ""
			if more && rest != "" {
				p.sgr = "\x1b[" + rest + "m"
			}
			return
		}
		p.sgr += seq
		return
	}
	if isLink, opens := osc8(seq); isLink {
		if opens {
			p.link = seq
		} else {
			p.link = ""
		}
	}
}

func (p penState) open(b *strings.Builder) {
	b.WriteString(p.sgr)
	b.WriteString(p.link)
}

func (p penState) close(b *strings.Builder) {
	if p.link != "" {
		b.WriteString(linkClose)
	}
	if p.sgr != "" {
		b.WriteString(Reset)
	}
}

// WrapStyled word-wraps s to width visible columns like Wrap, but tracks the
// SGR and OSC 8 hyperlink state as it goes, so every output line is
// self-contained: a line starts by re-opening the style active at that point
// and, if a style or link is still open at its end, closes it. A line drawn
// on its own therefore keeps its style, and none leaks into neighbouring
// cells. Existing newlines are paragraph breaks; a word longer than width is
// never split. It runs in time linear in len(s).
func WrapStyled(s string, width int) string {
	var (
		out       strings.Builder
		state     penState // state at the current parse position
		lineWidth int
		lineOpen  bool // the current line has been started
		lineEnd   penState
		wordStart penState
		word      strings.Builder
		wordW     int
		wordAny   bool
	)
	out.Grow(len(s) + len(s)/8)

	endLine := func() {
		if lineOpen {
			lineEnd.close(&out)
		}
		lineOpen, lineWidth = false, 0
	}
	flush := func() {
		if !wordAny {
			return
		}
		switch {
		case !lineOpen:
			wordStart.open(&out)
			lineOpen = true
			lineWidth = wordW
		case wordW == 0:
			// Escapes only: never forces a wrap or a separating space.
		case lineWidth+1+wordW <= width:
			out.WriteByte(' ')
			lineWidth += 1 + wordW
		default:
			endLine()
			out.WriteByte('\n')
			wordStart.open(&out)
			lineOpen = true
			lineWidth = wordW
		}
		out.WriteString(word.String())
		lineEnd = state
		word.Reset()
		wordW, wordAny = 0, false
	}

	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c == '\n':
			flush()
			endLine()
			out.WriteByte('\n')
			i++
		case c == ' ' || c == '\t' || c == '\r' || c == '\v' || c == '\f':
			flush()
			i++
		case c == 0x1b:
			end, _ := escEnd(s, i)
			if end == i {
				end = i + 1
			}
			if !wordAny {
				wordStart, wordAny = state, true
			}
			state.apply(s[i:end])
			word.WriteString(s[i:end])
			i = end
		default:
			j := i + 1
			for j < len(s) {
				b := s[j]
				if b == '\n' || b == ' ' || b == '\t' || b == '\r' || b == '\v' || b == '\f' || b == 0x1b {
					break
				}
				j++
			}
			if !wordAny {
				wordStart, wordAny = state, true
			}
			wordW += Width(s[i:j])
			word.WriteString(s[i:j])
			i = j
		}
	}
	flush()
	endLine()
	return out.String()
}

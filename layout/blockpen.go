package layout

import "strings"

// pen is the SGR and OSC 8 state open at a point in a string.
type pen struct {
	sgr  string // SGR sequences applied since the last reset
	link string // open OSC 8 opener, "" when none
}

func (p pen) active() bool { return p.sgr != "" || p.link != "" }

// scan folds every escape sequence in s into p.
func (p *pen) scan(s string) {
	for i := 0; i < len(s); {
		if s[i] != 0x1b || i+1 >= len(s) {
			i++
			continue
		}
		end := escEnd(s, i)
		p.apply(s[i:end])
		i = end
	}
}

// escEnd returns the index just past the escape sequence starting at s[i].
func escEnd(s string, i int) int {
	switch s[i+1] {
	case '[':
		for j := i + 2; j < len(s); j++ {
			if s[j] >= 0x40 && s[j] <= 0x7e {
				return j + 1
			}
		}
	case ']':
		for j := i + 2; j < len(s); j++ {
			if s[j] == 0x07 {
				return j + 1
			}
			if s[j] == 0x1b && j+1 < len(s) && s[j+1] == '\\' {
				return j + 2
			}
		}
	default:
		return i + 2
	}
	return len(s)
}

func (p *pen) apply(seq string) {
	if len(seq) >= 3 && seq[1] == '[' && seq[len(seq)-1] == 'm' {
		params := seq[2 : len(seq)-1]
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
	if strings.HasPrefix(seq, "\x1b]8;") {
		// OSC 8 ; params ; URI: an empty URI closes the link.
		body := strings.TrimSuffix(strings.TrimSuffix(seq[2:], "\a"), "\x1b\\")
		if i := strings.LastIndexByte(body, ';'); i >= 0 && body[i+1:] == "" {
			p.link = ""
		} else {
			p.link = seq
		}
	}
}

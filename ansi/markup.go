package ansi

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// EscapeMarkup returns s with every '[' escaped, so ParseMarkup gives back s as
// plain, unstyled text.
func EscapeMarkup(s string) string {
	return strings.ReplaceAll(s, "[", `\[`)
}

// ParseMarkup parses s as Markup does, and returns the styled Text. It returns an
// error for an unknown word in a tag and for a closing tag that matches no
// open one, and the Text up to that point.
func ParseMarkup(s string) (Text, error) { return parseMarkup(s, true) }

// Markup parses a string with inline style tags, as most one-line messages
// are written: "[bold red]error[/] in [link=https://example.com]the docs[/]".
//
// A tag is "[" words "]" and starts with a letter, '#' or '/'; anything else in
// brackets ("[1]", "[ ]") is text. The words, separated by spaces, are:
//
//	bold faint italic underline blink reverse strike conceal overline
//	black red green yellow blue magenta cyan white   (and bright_ forms)
//	#rgb #rrggbb                                      (a foreground colour)
//	on <colour>                                       (a background colour)
//	link=<url>                                        (a hyperlink)
//
// A tag adds its styles to the ones already open. "[/]" closes the last open
// tag; "[/bold]" closes the last open tag that is exactly "[bold]". Tags still
// open at the end close themselves. A backslash before '[' makes the bracket
// text: "\[bold]" is the six characters "[bold]". EscapeMarkup does that for
// any string, which is how text that is not yours (a file name, an error from
// a library) goes into a markup string.
//
// Text is cleaned as NewText cleans it: escape sequences and control characters
// are removed, so markup cannot smuggle terminal codes. A link target that
// fails SafeLinkTarget is dropped.
//
// Markup never fails: a tag it does not understand, and a closing tag with
// nothing to close, stay in the text as written. ParseMarkup is the strict form.
func Markup(s string) Text {
	t, _ := parseMarkup(s, false)
	return t
}

type openTag struct {
	raw   string // the words as written, for "[/words]" matching
	style Style  // the style before this tag opened
}

var errMarkup = errors.New("ansi: markup")

func parseMarkup(s string, strict bool) (Text, error) {
	var (
		out   Text
		cur   Style
		stack []openTag
		lit   strings.Builder
	)
	flush := func() {
		if lit.Len() > 0 {
			out = out.Append(lit.String(), cur)
			lit.Reset()
		}
	}
	for i := 0; i < len(s); {
		c := s[i]
		if c == '\\' && i+1 < len(s) && s[i+1] == '[' {
			lit.WriteByte('[')
			i += 2
			continue
		}
		if c != '[' {
			lit.WriteByte(c)
			i++
			continue
		}
		end := strings.IndexByte(s[i:], ']')
		if end < 0 || !isTag(s[i+1:i+end]) {
			lit.WriteByte(c)
			i++
			continue
		}
		body := s[i+1 : i+end]
		raw := s[i : i+end+1]
		if body[0] == '/' {
			words := strings.Join(strings.Fields(body[1:]), " ")
			k := len(stack) - 1
			for ; k >= 0 && words != "" && stack[k].raw != words; k-- {
			}
			if k < 0 {
				if strict {
					flush()
					return out, fmt.Errorf("%w: [%s] closes nothing", errMarkup, body)
				}
				lit.WriteString(raw)
				i += end + 1
				continue
			}
			flush()
			cur = stack[k].style
			stack = stack[:k]
			i += end + 1
			continue
		}
		next, err := applyTag(cur, body)
		if err != nil {
			if strict {
				flush()
				return out, err
			}
			lit.WriteString(raw)
			i += end + 1
			continue
		}
		flush()
		stack = append(stack, openTag{raw: strings.Join(strings.Fields(body), " "), style: cur})
		cur = next
		i += end + 1
	}
	flush()
	return out, nil
}

// isTag reports whether the text between brackets reads as a tag.
func isTag(body string) bool {
	if body == "" {
		return false
	}
	c := body[0]
	return c == '/' || c == '#' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// applyTag returns st with the words of body applied.
func applyTag(st Style, body string) (Style, error) {
	words := strings.Fields(body)
	for i := 0; i < len(words); i++ {
		w := words[i]
		switch lw := strings.ToLower(w); {
		case lw == "bold" || lw == "b":
			st = st.Bold()
		case lw == "faint" || lw == "dim":
			st = st.Faint()
		case lw == "italic" || lw == "i":
			st = st.Italic()
		case lw == "underline" || lw == "u":
			st = st.Underline()
		case lw == "blink":
			st = st.Blink()
		case lw == "reverse":
			st = st.Reverse()
		case lw == "strike" || lw == "strikethrough":
			st = st.Strikethrough()
		case lw == "conceal":
			st = st.Conceal()
		case lw == "overline":
			st = st.Overline()
		case lw == "on":
			if i+1 >= len(words) {
				return st, fmt.Errorf("%w: [%s] has \"on\" without a colour", errMarkup, body)
			}
			i++
			c, ok := markupColor(words[i])
			if !ok {
				return st, fmt.Errorf("%w: [%s] has the unknown colour %q", errMarkup, body, words[i])
			}
			st = st.Background(c)
		case strings.HasPrefix(lw, "link="):
			st = st.Link(w[len("link="):])
		default:
			c, ok := markupColor(w)
			if !ok {
				return st, fmt.Errorf("%w: [%s] has the unknown word %q", errMarkup, body, w)
			}
			st = st.Foreground(c)
		}
	}
	return st, nil
}

var markupColors = map[string]BasicColor{
	"black": Black, "red": Red, "green": Green, "yellow": Yellow,
	"blue": Blue, "magenta": Magenta, "cyan": Cyan, "white": White,
	"bright_black": BrightBlack, "bright_red": BrightRed, "bright_green": BrightGreen,
	"bright_yellow": BrightYellow, "bright_blue": BrightBlue,
	"bright_magenta": BrightMagenta, "bright_cyan": BrightCyan, "bright_white": BrightWhite,
}

// markupColor reads a colour name or a #rgb or #rrggbb value.
func markupColor(w string) (Color, bool) {
	lw := strings.ToLower(w)
	if c, ok := markupColors[strings.ReplaceAll(lw, "-", "_")]; ok {
		return c, true
	}
	if len(lw) == 4 || len(lw) == 7 {
		if lw[0] != '#' {
			return nil, false
		}
		hex := lw[1:]
		if len(hex) == 3 {
			hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
		}
		v, err := strconv.ParseUint(hex, 16, 32)
		if err != nil {
			return nil, false
		}
		return RGB{R: uint8(v >> 16 & 0xff), G: uint8(v >> 8 & 0xff), B: uint8(v & 0xff)}, true // #nosec G115 -- masked to a byte
	}
	return nil, false
}

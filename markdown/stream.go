package markdown

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// Stream renders Markdown that arrives in pieces (an LLM token stream)
// without re-rendering the whole document on every append. Blocks that a
// blank line has closed are rendered once and frozen; only the open tail is
// parsed and rendered again. View always equals RenderWith on the full text
// appended so far, so a finished stream matches a one-shot Render.
//
// A Stream is not safe for concurrent use.
type Stream struct {
	width  int
	t      theme.Theme
	opts   Options
	tail   string          // raw text after the frozen prefix
	chunks [][]block       // parsed blocks of the frozen prefix
	text   strings.Builder // rendered frozen output, built incrementally
	src    strings.Builder
}

// NewStream returns an empty Stream drawn at width with t and o.
func NewStream(width int, t theme.Theme, o Options) *Stream {
	return &Stream{width: width, t: t, opts: o}
}

// Write appends p to the document.
func (s *Stream) Write(p string) {
	s.tail += p
	s.src.WriteString(p)
}

// Source returns all Markdown appended so far.
func (s *Stream) Source() string { return s.src.String() }

// Resize sets the wrap width and theme, re-rendering frozen blocks.
func (s *Stream) Resize(width int, t theme.Theme) {
	if width == s.width && t == s.t {
		return
	}
	s.width, s.t = width, t
	s.text.Reset()
	if width <= 0 {
		return
	}
	for _, c := range s.chunks {
		s.addLines(s.renderChunk(c))
	}
}

// Reset empties the stream, keeping its width, theme and options.
func (s *Stream) Reset() {
	s.tail, s.chunks = "", nil
	s.text.Reset()
	s.src.Reset()
}

func (s *Stream) renderChunk(blocks []block) []string {
	out := renderBlocks(blocks, s.width, s.t, 0, s.opts)
	for i, l := range out {
		if ansi.Width(l) > s.width {
			out[i] = ansi.Truncate(l, s.width)
		}
	}
	return out
}

// addLines appends a chunk's rendered lines to the frozen output.
func (s *Stream) addLines(ls []string) {
	if len(ls) == 0 {
		return
	}
	if s.text.Len() > 0 {
		s.text.WriteString("\n\n")
	}
	s.text.WriteString(strings.Join(ls, "\n"))
}

// Frozen returns the rendering of the closed blocks. It is built
// incrementally, so each call costs only the newly closed blocks; a caller
// that draws every frame can pair it with Open to avoid copying the whole
// document.
func (s *Stream) Frozen() string {
	if s.width <= 0 {
		return ""
	}
	s.freeze()
	return s.text.String()
}

// Open returns the rendering of the still-open tail (call after Frozen).
func (s *Stream) Open() string {
	if s.width <= 0 || strings.TrimSpace(s.tail) == "" {
		return ""
	}
	lines := splitLines(sanitize(s.tail))
	return strings.Join(s.renderChunk(parseBlocks(lines, 0, 0)), "\n")
}

// View returns the rendering of everything appended so far: Frozen and Open
// joined by one blank line.
func (s *Stream) View() string {
	frozen, tail := s.Frozen(), s.Open()
	switch {
	case frozen == "":
		return tail
	case tail == "":
		return frozen
	}
	return frozen + "\n\n" + tail
}

// freeze moves every block closed by a blank line out of the tail.
func (s *Stream) freeze() {
	lines := splitLines(sanitize(s.tail))
	var chunks [][]block
	start, cut := 0, 0
	var fch byte
	var fn int
	inFence := false
	for i := 0; i < len(lines); i++ {
		ln := lines[i]
		if inFence {
			if isFenceClose(ln, fch, fn) {
				inFence = false
			}
			continue
		}
		if _, ch, n, ok := fenceOpen(ln); ok {
			inFence, fch, fn = true, ch, n
			continue
		}
		if !isBlank(ln) {
			continue
		}
		j := i
		for j < len(lines) && isBlank(lines[j]) {
			j++
		}
		if j == len(lines) {
			break // no following content yet: the block may still grow
		}
		if i == start { // leading blanks
			start, i = j, j-1
			continue
		}
		blocks := parseBlocks(lines[start:i], 0, 0)
		if len(blocks) > 0 {
			if _, ok := blocks[len(blocks)-1].(list); ok {
				nx := lines[j]
				if leadingSpaces(nx) > 0 || isListStart(nx) {
					i = j - 1
					continue // the list may continue
				}
			}
		}
		chunks = append(chunks, blocks)
		start, cut, i = j, j, j-1
	}
	if len(chunks) == 0 {
		return
	}
	off := rawOffset(s.tail, cut)
	if off < 0 || len(splitLines(sanitize(s.tail[:off]))) != cut+1 {
		return
	}
	s.tail = s.tail[off:]
	for _, c := range chunks {
		s.chunks = append(s.chunks, c)
		s.addLines(s.renderChunk(c))
	}
}

// rawOffset returns the offset in raw just after its n-th line break (CRLF,
// CR and LF each count once), or -1.
func rawOffset(raw string, n int) int {
	for i := 0; i < len(raw) && n > 0; i++ {
		switch raw[i] {
		case '\r':
			if i+1 < len(raw) && raw[i+1] == '\n' {
				i++
			}
			n--
		case '\n':
			n--
		}
		if n == 0 {
			return i + 1
		}
	}
	return -1
}

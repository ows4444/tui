package ansi

import (
	"math/rand"
	"strconv"
	"strings"
	"testing"
)

// refSequence and refRender are the original, allocation-heavy Style.Render,
// kept verbatim as the reference the optimised implementation must match
// byte for byte.
func refSequence(s Style) string {
	var codes []string
	add := func(cond bool, code string) {
		if cond {
			codes = append(codes, code)
		}
	}
	add(s.bold, "1")
	add(s.faint, "2")
	add(s.italic, "3")
	if s.underline && s.underlineStyle != 0 {
		codes = append(codes, "4:"+strconv.Itoa(int(s.underlineStyle)))
	} else {
		add(s.underline, "4")
	}
	add(s.blink, "5")
	add(s.reverse, "7")
	add(s.strike, "9")
	add(s.conceal, "8")
	if s.fg != nil {
		codes = append(codes, s.fg.fgCode())
	}
	if s.bg != nil {
		codes = append(codes, s.bg.bgCode())
	}
	if s.underlineColor != nil {
		codes = append(codes, s.underlineColor.ulCode())
	}
	if len(codes) == 0 {
		return ""
	}
	return CSI + strings.Join(codes, ";") + "m"
}

func refRender(s Style, text string) string {
	seq := refSequence(s)
	if seq == "" {
		return text
	}
	if !strings.Contains(text, "\n") {
		return seq + text + Reset
	}
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = seq + line + Reset
		}
	}
	return strings.Join(lines, "\n")
}

func randomColor(r *rand.Rand) Color {
	switch r.Intn(4) {
	case 0:
		return nil
	case 1:
		return BasicColor(r.Intn(16))
	case 2:
		return Color256(r.Intn(256))
	}
	return RGB{uint8(r.Intn(256)), uint8(r.Intn(256)), uint8(r.Intn(256))}
}

func randomStyle(r *rand.Rand) Style {
	s := NewStyle()
	if r.Intn(2) == 0 {
		s = s.Bold()
	}
	if r.Intn(4) == 0 {
		s = s.Faint()
	}
	if r.Intn(3) == 0 {
		s = s.Italic()
	}
	switch r.Intn(4) {
	case 0:
		s = s.Underline()
	case 1:
		s = s.UnderlineStyle([]UnderlineStyle{UnderlineCurly, UnderlineDotted, UnderlineDashed}[r.Intn(3)])
	}
	if r.Intn(6) == 0 {
		s = s.Blink()
	}
	if r.Intn(3) == 0 {
		s = s.Reverse()
	}
	if r.Intn(6) == 0 {
		s = s.Strikethrough()
	}
	if r.Intn(8) == 0 {
		s = s.Conceal()
	}
	if c := randomColor(r); c != nil {
		s = s.Foreground(c)
	}
	if c := randomColor(r); c != nil {
		s = s.Background(c)
	}
	if c := randomColor(r); c != nil {
		s = s.UnderlineColor(c)
	}
	return s
}

func randomText(r *rand.Rand) string {
	pieces := []string{"", "a", "hello world", "你好", "\n", "\n\n", "line", " ", "x\ty", "\x1b[1mbold\x1b[0m", "trailing\n", "\nleading"}
	var b strings.Builder
	for i, n := 0, r.Intn(6); i < n; i++ {
		b.WriteString(pieces[r.Intn(len(pieces))])
	}
	return b.String()
}

// TestRenderMatchesReferenceImplementation is the guard for any rewrite of
// Style.Render: 200,000 random styles and texts (empty, single-line,
// multi-line, blank lines, trailing newlines, wide runes, embedded escapes)
// must produce exactly the reference's bytes.
func TestRenderMatchesReferenceImplementation(t *testing.T) {
	r := rand.New(rand.NewSource(42))
	for i := 0; i < 200000; i++ {
		s, text := randomStyle(r), randomText(r)
		if got, want := s.Render(text), refRender(s, text); got != want {
			t.Fatalf("case %d: Render(%q) =\n%q\nwant\n%q", i, text, got, want)
		}
	}
	// The zero Style never adds a sequence.
	if got := NewStyle().Render("x\ny"); got != "x\ny" {
		t.Errorf("zero style changed text: %q", got)
	}
}

package ansi

import (
	"strings"
	"testing"
)

func TestStyleRenderNoStyle(t *testing.T) {
	got := NewStyle().Render("plain")
	if got != "plain" {
		t.Errorf("Render() = %q, want %q (no escape codes for an empty style)", got, "plain")
	}
}

func TestStyleRenderWrapsAndResets(t *testing.T) {
	s := NewStyle().Bold().Foreground(Red)
	got := s.Render("hi")
	if !strings.HasPrefix(got, CSI) {
		t.Errorf("Render() = %q, want it to start with CSI %q", got, CSI)
	}
	if !strings.HasSuffix(got, Reset) {
		t.Errorf("Render() = %q, want it to end with Reset %q", got, Reset)
	}
	if !strings.Contains(got, "hi") {
		t.Errorf("Render() = %q, want it to contain the original text", got)
	}
}

func TestStyleSGRCodes(t *testing.T) {
	tests := []struct {
		name  string
		style Style
		want  string
	}{
		{"bold", NewStyle().Bold(), CSI + "1m"},
		{"faint", NewStyle().Faint(), CSI + "2m"},
		{"italic", NewStyle().Italic(), CSI + "3m"},
		{"underline", NewStyle().Underline(), CSI + "4m"},
		{"blink", NewStyle().Blink(), CSI + "5m"},
		{"reverse", NewStyle().Reverse(), CSI + "7m"},
		{"strikethrough", NewStyle().Strikethrough(), CSI + "9m"},
		{"conceal", NewStyle().Conceal(), CSI + "8m"},
		{"bold+underline order", NewStyle().Bold().Underline(), CSI + "1;4m"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.style.sequence(); got != tt.want {
				t.Errorf("sequence() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStyleUnderlineStyleSGRCodes(t *testing.T) {
	tests := []struct {
		name  string
		style Style
		want  string
	}{
		{"curly", NewStyle().UnderlineStyle(UnderlineCurly), CSI + "4:3m"},
		{"dotted", NewStyle().UnderlineStyle(UnderlineDotted), CSI + "4:4m"},
		{"dashed", NewStyle().UnderlineStyle(UnderlineDashed), CSI + "4:5m"},
		{"UnderlineStyle implies Underline", NewStyle().UnderlineStyle(UnderlineCurly), CSI + "4:3m"},
		{"plain Underline unaffected", NewStyle().Underline(), CSI + "4m"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.style.sequence(); got != tt.want {
				t.Errorf("sequence() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStyleUnderlineColorSGRCode(t *testing.T) {
	tests := []struct {
		name  string
		style Style
		want  string
	}{
		{"basic color", NewStyle().Underline().UnderlineColor(Red), CSI + "4;58;5;1m"},
		{"256 color", NewStyle().Underline().UnderlineColor(Color256(200)), CSI + "4;58;5;200m"},
		{"rgb color", NewStyle().Underline().UnderlineColor(RGB{R: 10, G: 20, B: 30}), CSI + "4;58;2;10;20;30m"},
		{
			"independent of foreground",
			NewStyle().Underline().Foreground(Green).UnderlineColor(Red),
			CSI + "4;32;58;5;1m",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.style.sequence(); got != tt.want {
				t.Errorf("sequence() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStyleIsImmutable(t *testing.T) {
	base := NewStyle().Bold()
	_ = base.Foreground(Red)
	if base.sequence() != CSI+"1m" {
		t.Errorf("base style was mutated by a chained call: got %q", base.sequence())
	}
}

func TestColorCodes(t *testing.T) {
	tests := []struct {
		name   string
		color  Color
		fgWant string
		bgWant string
	}{
		{"black", Black, "30", "40"},
		{"white", White, "37", "47"},
		{"bright black", BrightBlack, "90", "100"},
		{"bright white", BrightWhite, "97", "107"},
		{"256 color", Color256(201), "38;5;201", "48;5;201"},
		{"truecolor", RGB{R: 10, G: 20, B: 30}, "38;2;10;20;30", "48;2;10;20;30"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.color.fgCode(); got != tt.fgWant {
				t.Errorf("fgCode() = %q, want %q", got, tt.fgWant)
			}
			if got := tt.color.bgCode(); got != tt.bgWant {
				t.Errorf("bgCode() = %q, want %q", got, tt.bgWant)
			}
		})
	}
}

func TestStyleRenderMultiLine(t *testing.T) {
	s := NewStyle().Bold().Foreground(Red)
	seq := s.sequence()

	got := s.Render("a\nb\nc")
	want := seq + "a" + Reset + "\n" + seq + "b" + Reset + "\n" + seq + "c" + Reset
	if got != want {
		t.Errorf("Render multi-line = %q, want %q", got, want)
	}

	for i, line := range strings.Split(got, "\n") {
		if !strings.HasPrefix(line, seq) || !strings.HasSuffix(line, Reset) {
			t.Errorf("line %d = %q, want it self-contained (seq ... Reset)", i, line)
		}
	}

	if got := s.Render("hi"); got != seq+"hi"+Reset {
		t.Errorf("single line Render = %q, want unchanged shape", got)
	}

	if got := s.Render("a\n\nb"); got != seq+"a"+Reset+"\n\n"+seq+"b"+Reset {
		t.Errorf("empty middle line = %q, want it left bare", got)
	}

	if got := NewStyle().Render("a\nb"); got != "a\nb" {
		t.Errorf("empty style Render = %q, want text unchanged", got)
	}

	plain := "one\ntwo\n\nthree"
	styled := s.Render(plain)
	if StripANSI(styled) != plain {
		t.Errorf("StripANSI(Render) = %q, want %q", StripANSI(styled), plain)
	}
	for i, line := range strings.Split(styled, "\n") {
		if Width(line) != Width(strings.Split(plain, "\n")[i]) {
			t.Errorf("line %d Width = %d, want %d", i, Width(line), Width(strings.Split(plain, "\n")[i]))
		}
	}
}

func TestStyleOverlineEmitsSGR53AndResets(t *testing.T) {
	got := NewStyle().Overline().Render("x")
	if want := "\x1b[53mx" + Reset; got != want {
		t.Errorf("Overline().Render = %q, want %q", got, want)
	}
	if got := NewStyle().Bold().Overline().Render("x"); got != "\x1b[1;53mx"+Reset {
		t.Errorf("Bold().Overline().Render = %q", got)
	}
}

func TestStyleLinkWrapsRenderInOSC8(t *testing.T) {
	got := NewStyle().Bold().Link("https://example.com/a").Render("hi")
	want := "\x1b]8;;https://example.com/a\x1b\\" + "\x1b[1mhi" + Reset + "\x1b]8;;\x1b\\"
	if got != want {
		t.Errorf("Link.Render = %q, want %q", got, want)
	}
	// An unstyled link still links.
	if got, want := NewStyle().Link("https://e.com").Render("hi"), Hyperlink("hi", "https://e.com"); got != want {
		t.Errorf("unstyled Link.Render = %q, want %q", got, want)
	}
}

func TestStyleLinkUnsafeTargetRendersPlain(t *testing.T) {
	for _, url := range []string{"", "javascript:alert(1)", "https://x.com/\x1b]52;c;AAAA\x07", "no scheme", "https://a b"} {
		s := NewStyle().Bold().Link(url)
		if got, want := s.Render("hi"), NewStyle().Bold().Render("hi"); got != want {
			t.Errorf("Link(%q).Render = %q, want %q", url, got, want)
		}
		if got := NewStyle().Link(url).Render("hi"); got != "hi" {
			t.Errorf("unstyled Link(%q).Render = %q, want plain text", url, got)
		}
	}
}

func TestStyleLinkIsPerLineAndSkipsEmpty(t *testing.T) {
	s := NewStyle().Link("https://e.com")
	got := s.Render("a\n\nb")
	want := Hyperlink("a", "https://e.com") + "\n\n" + Hyperlink("b", "https://e.com")
	if got != want {
		t.Errorf("multi-line Link.Render = %q, want %q", got, want)
	}
	if got := s.Render(""); got != "" {
		t.Errorf("Link.Render(\"\") = %q, want empty", got)
	}
}

func TestStyleLinkSpan(t *testing.T) {
	got := NewStyle().Italic().Link("https://e.com").Span("a ", "b")
	want := Hyperlink(NewStyle().Italic().Span("a b"), "https://e.com")
	if got != want {
		t.Errorf("Link.Span = %q, want %q", got, want)
	}
}

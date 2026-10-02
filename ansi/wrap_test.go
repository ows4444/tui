package ansi

import "testing"

func TestWrap(t *testing.T) {
	tests := []struct {
		name  string
		input string
		width int
		want  string
	}{
		{"fits on one line", "hello world", 20, "hello world"},
		{"wraps at word boundary", "the quick brown fox", 10, "the quick\nbrown fox"},
		{"preserves existing newlines as paragraph breaks", "para one\npara two is longer", 10, "para one\npara two\nis longer"},
		{"blank line preserved", "a\n\nb", 10, "a\n\nb"},
		{"single word longer than width is not split", "supercalifragilistic", 5, "supercalifragilistic"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Wrap(tt.input, tt.width); got != tt.want {
				t.Errorf("Wrap(%q, %d) = %q, want %q", tt.input, tt.width, got, tt.want)
			}
		})
	}
}

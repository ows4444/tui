package layout

import (
	"strings"
	"testing"
)

func TestMinSizeFallsBackBelowMin(t *testing.T) {
	body := Text("main view")
	n := MinSize(body, Size{W: 20, H: 5}, Text("too small"))

	small := n.Render(Size{W: 12, H: 3})
	if !strings.Contains(small, "too small") || strings.Contains(small, "main") {
		t.Fatalf("below min rendered %q, want the fallback", small)
	}
	assertExact(t, small, Size{W: 12, H: 3})

	for _, s := range []Size{{W: 19, H: 5}, {W: 20, H: 4}} {
		if out := n.Render(s); !strings.Contains(out, "too small") {
			t.Fatalf("%v: want fallback, got %q", s, out)
		}
	}

	big := n.Render(Size{W: 20, H: 5})
	if !strings.Contains(big, "main view") || strings.Contains(big, "too small") {
		t.Fatalf("at min rendered %q, want the main node", big)
	}
	assertExact(t, big, Size{W: 20, H: 5})
}

func TestMinSizeMeasureAgreesWithRender(t *testing.T) {
	n := MinSize(Text("main view"), Size{W: 20, H: 5}, Text("small"))
	if got := n.Measure(Loose(Size{W: 10, H: 10})); got != (Size{W: 5, H: 1}) {
		t.Fatalf("narrow Measure = %v, want fallback size 5x1", got)
	}
	if got := n.Measure(Loose(Size{W: 30, H: 10})); got != (Size{W: 9, H: 1}) {
		t.Fatalf("roomy Measure = %v, want main size 9x1", got)
	}
}

func TestMinSizeNilFallbackIsBlank(t *testing.T) {
	out := MinSize(Text("x"), Size{W: 10, H: 2}, nil).Render(Size{W: 4, H: 2})
	if strings.TrimSpace(out) != "" {
		t.Fatalf("nil fallback rendered %q, want blanks", out)
	}
	assertExact(t, out, Size{W: 4, H: 2})
}

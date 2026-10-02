package theme

import "testing"

func TestCheckHighContrastClean(t *testing.T) {
	if got := HighContrastTheme().Check(4.5); len(got) != 0 {
		t.Fatalf("HighContrast.Check(4.5) = %v, want none", got)
	}
}

func TestCheckNamesRole(t *testing.T) {
	th := HighContrastTheme()
	th.Text = th.TextInverse
	got := th.Check(4.5)
	if len(got) != 1 || got[0].Role != "Text" {
		t.Fatalf("Check = %v, want one issue for Text", got)
	}
}

func TestContrastExtremes(t *testing.T) {
	if r := Contrast(HighContrastTheme().Text, HighContrastTheme().TextInverse); r < 20.9 || r > 21.1 {
		t.Fatalf("white/black = %.2f, want 21", r)
	}
	if Contrast(nil, HighContrastTheme().Text) != 0 {
		t.Fatal("nil colour should give 0")
	}
}

package theme

import (
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/input"
)

func TestForProfileDowngradesAndKeepsBorder(t *testing.T) {
	th := DarkTheme()
	th.Primary = ansi.RGB{R: 255, G: 0, B: 0}

	if got := th.ForProfile(ansi.ANSI256).Primary; got != ansi.Color256(196) {
		t.Errorf("ANSI256 Primary = %v, want 196", got)
	}
	if got := th.ForProfile(ansi.ANSI16).Primary; got != ansi.BrightRed {
		t.Errorf("ANSI16 Primary = %v, want BrightRed", got)
	}
	if got := th.ForProfile(ansi.TrueColor); got != th {
		t.Errorf("TrueColor should leave theme unchanged")
	}
	nc := th.ForProfile(ansi.NoColor)
	if nc.Primary != nil || nc.Text != nil || nc.Selection != nil {
		t.Errorf("NoColor should nil all colors: %+v", nc)
	}
	if nc.Border != th.Border {
		t.Errorf("NoColor should keep Border")
	}
}

func TestForBackground(t *testing.T) {
	if got := ForBackground(ansi.RGB{R: 255, G: 255, B: 255}); got != LightTheme() {
		t.Error("white background should pick Light")
	}
	if got := ForBackground(ansi.RGB{R: 30, G: 30, B: 30}); got != DarkTheme() {
		t.Error("near-black background should pick Dark")
	}
}

func TestDetect(t *testing.T) {
	light, ok := Detect(input.BackgroundColorEvent{R: 250, G: 250, B: 250}, DarkTheme())
	if !ok || light != LightTheme() {
		t.Errorf("light reply -> ok=%v, theme is Light: %v", ok, light == LightTheme())
	}
	dark, ok := Detect(input.BackgroundColorEvent{R: 10, G: 10, B: 10}, LightTheme())
	if !ok || dark != DarkTheme() {
		t.Errorf("dark reply with Light fallback -> ok=%v, theme is Dark: %v", ok, dark == DarkTheme())
	}
	unknown, ok := Detect(input.BackgroundUnknownMsg{}, LightTheme())
	if !ok || unknown != LightTheme() {
		t.Errorf("BackgroundUnknownMsg should return the fallback with ok=true, got ok=%v", ok)
	}
	other, ok := Detect(struct{ Width, Height int }{1, 1}, DarkTheme())
	if ok || other != DarkTheme() {
		t.Errorf("unrelated Msg should return the fallback with ok=false, got ok=%v", ok)
	}
}

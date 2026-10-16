package theme_test

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

func TestParseStartsFromTheBaseAndChangesWhatItNames(t *testing.T) {
	got, err := theme.Parse(strings.NewReader(`{"base": "nord", "primary": "#ff8800", "border": "ascii"}`))
	if err != nil {
		t.Fatal(err)
	}
	want := theme.NordTheme()
	if got.Primary != (ansi.RGB{R: 0xff, G: 0x88, B: 0x00}) {
		t.Errorf("Primary = %v", got.Primary)
	}
	if got.Secondary != want.Secondary || got.Text != want.Text {
		t.Error("a colour the file did not name changed")
	}
	if got.Border.Top != "-" {
		t.Errorf("Border = %+v, want ascii", got.Border)
	}
}

func TestParseEmptyObjectIsTheDarkTheme(t *testing.T) {
	got, err := theme.Parse(strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if got != theme.DarkTheme() {
		t.Error(`"{}" is not the Dark preset unchanged`)
	}
}

func TestParseShortHexAndCase(t *testing.T) {
	got, err := theme.Parse(strings.NewReader(`{"base":"Dracula","text":"#FFF"}`))
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != (ansi.RGB{R: 255, G: 255, B: 255}) {
		t.Errorf("Text = %v", got.Text)
	}
}

func TestParseDerivesSurfacesFromNewColoursUnlessSet(t *testing.T) {
	base := theme.NordTheme()
	got, err := theme.Parse(strings.NewReader(`{"base":"nord","text_inverse":"#000000"}`))
	if err != nil {
		t.Fatal(err)
	}
	if got.Background != (ansi.RGB{}) || got.Surface == base.Surface {
		t.Errorf("Background/Surface not derived from the new text_inverse: %v %v", got.Background, got.Surface)
	}
	set, err := theme.Parse(strings.NewReader(`{"base":"nord","text_inverse":"#000000","surface":"#112233"}`))
	if err != nil {
		t.Fatal(err)
	}
	if set.Surface != (ansi.RGB{R: 0x11, G: 0x22, B: 0x33}) {
		t.Errorf("an explicit surface was overwritten: %v", set.Surface)
	}
}

func TestParseErrorsNameTheProblem(t *testing.T) {
	for in, want := range map[string]string{
		`{"base":"nope"}`:            `unknown base "nope"`,
		`{"primry":"#fff"}`:          "primry",
		`{"primary":"red"}`:          "primary",
		`{"primary":"#12345"}`:       "primary",
		`{"border":"dotted"}`:        "border",
		`{"primary":"#fff"} {"a":1}`: "more than one",
		`[1]`:                        "theme:",
	} {
		_, err := theme.Parse(strings.NewReader(in))
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("Parse(%s) error = %v, want it to mention %q", in, err, want)
		}
	}
}

func TestPresetLookupCoversEveryName(t *testing.T) {
	names := theme.PresetNames()
	if len(names) != 20 {
		t.Errorf("%d preset names, want 20: %v", len(names), names)
	}
	for _, n := range names {
		if _, ok := theme.Preset(n); !ok {
			t.Errorf("Preset(%q) not found", n)
		}
	}
	if _, ok := theme.Preset("  NORD "); !ok {
		t.Error("Preset ignores neither case nor spaces")
	}
	if _, ok := theme.Preset("nope"); ok {
		t.Error(`Preset("nope") found something`)
	}
}

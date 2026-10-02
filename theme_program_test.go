package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// themedModel records every theme the Program hands it.
type themedModel struct {
	recorderModel
	themes []theme.Theme
}

func (m themedModel) Update(msg Msg) (Model, Cmd) {
	r, c := m.recorderModel.Update(msg)
	m.recorderModel = r.(recorderModel)
	return m, c
}

func (m themedModel) SetTheme(t theme.Theme) Model {
	m.themes = append(m.themes, t)
	return m
}

var (
	testDark  = theme.NordTheme()
	testLight = theme.SolarizedLightTheme()
	testAuto  = theme.Auto{Dark: testDark, Light: testLight}
)

// Criterion #106: the theme reaches a Themeable model on start, and again on
// each change.
// Criterion #107: the change is the OSC 11 answer, mapped through Auto.
func TestWithThemeDeliversOnStartAndOnDetection(t *testing.T) {
	write, read, wait := bgRun(t, themedModel{recorderModel: recorderModel{quitAfter: 2}}, WithTheme(testAuto))
	write("\x1b]11;rgb:ffff/ffff/ffff\x07")
	got := wait().model.(themedModel).themes
	if len(got) != 2 || got[0] != testDark || got[1] != testLight {
		t.Fatalf("themes delivered = %d, want [Dark, Light]: first %v", len(got), got)
	}
	if n := strings.Count(string(read()), ansi.QueryBackgroundColor); n != 1 {
		t.Errorf("background query sent %d times, want 1 (WithTheme turns detection on)", n)
	}
}

// Criterion #107: a dark answer keeps Dark, and is not delivered twice.
func TestWithThemeDarkAnswerDeliversOnce(t *testing.T) {
	write, _, wait := bgRun(t, themedModel{recorderModel: recorderModel{quitAfter: 2}}, WithTheme(testAuto))
	write("\x1b]11;rgb:0000/0000/0000\x07")
	got := wait().model.(themedModel).themes
	if len(got) != 1 || got[0] != testDark {
		t.Fatalf("themes delivered = %d, want just Dark", len(got))
	}
}

// Criterion #107: with no answer the model keeps Dark, and Update still gets
// BackgroundUnknownMsg.
func TestWithThemeNoAnswerKeepsDark(t *testing.T) {
	_, _, wait := bgRun(t, themedModel{recorderModel: recorderModel{quitAfter: 2}},
		WithTheme(testAuto), WithBackgroundDetection(50*time.Millisecond))
	m := wait().model.(themedModel)
	if len(m.themes) != 1 || m.themes[0] != testDark {
		t.Fatalf("themes delivered = %d, want just Dark", len(m.themes))
	}
	if _, ok := m.received[1].(BackgroundUnknownMsg); !ok {
		t.Fatalf("message 1 = %#v, want BackgroundUnknownMsg", m.received[1])
	}
}

// Criterion #106: without WithTheme nothing is delivered and nothing is
// queried, and a model that is not Themeable is untouched.
func TestNoWithThemeDeliversNothing(t *testing.T) {
	write, read, wait := bgRun(t, themedModel{recorderModel: recorderModel{quitAfter: 2}})
	write("a")
	if got := wait().model.(themedModel).themes; len(got) != 0 {
		t.Fatalf("delivered %d themes without WithTheme", len(got))
	}
	if strings.Contains(string(read()), "]11;?") {
		t.Error("background query sent without WithTheme")
	}

	write2, _, wait2 := bgRun(t, recorderModel{quitAfter: 2}, WithTheme(testAuto))
	write2("\x1b]11;rgb:ffff/ffff/ffff\x07")
	if got := wait2().model.(recorderModel).received; len(got) != 2 {
		t.Fatalf("a non-Themeable model got %d messages, want 2", len(got))
	}
}

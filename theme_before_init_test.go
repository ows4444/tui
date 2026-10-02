package tui

import (
	"io"
	"strings"
	"testing"

	"github.com/ows4444/tui/theme"
)

// initThemeModel reports, from Init, the themes it had been given by then.
type initThemeModel struct {
	themes  []theme.Theme
	sawInit []theme.Theme // set by Update from the Init Cmd's message
}

type initSawMsg struct{ themes []theme.Theme }

func (m initThemeModel) Init() Cmd {
	seen := append([]theme.Theme(nil), m.themes...)
	return func() Msg { return initSawMsg{seen} }
}

func (m initThemeModel) Update(msg Msg) (Model, Cmd) {
	if s, ok := msg.(initSawMsg); ok {
		m.sawInit = s.themes
		return m, Quit()
	}
	return m, nil
}

func (m initThemeModel) View() string { return "v" }

func (m initThemeModel) SetTheme(t theme.Theme) Model {
	m.themes = append(m.themes, t)
	return m
}

// #22: with WithTheme and a Themeable model, SetTheme runs before Init.
func TestThemeIsSetBeforeInit(t *testing.T) {
	pr, pw := io.Pipe() // an input that stays open until the model quits
	t.Cleanup(func() { pw.Close() })
	p := NewProgram(initThemeModel{}, WithTheme(testAuto),
		WithInput(pr), WithOutput(&strings.Builder{}))
	final, err := p.Run()
	if err != nil {
		t.Fatal(err)
	}
	got := final.(initThemeModel).sawInit
	if len(got) != 1 || got[0] != testDark {
		t.Fatalf("Init saw themes %v, want exactly [Dark]", got)
	}
}

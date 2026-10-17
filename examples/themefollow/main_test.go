package main

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/theme"
)

func TestSetThemeChangesTheFrame(t *testing.T) {
	pair := theme.Pair(theme.DraculaTheme())
	var dark, light tui.Model = newModel().SetTheme(pair.Dark), newModel().SetTheme(pair.Light)
	if dark.View() == light.View() {
		t.Fatal("the dark and light themes drew the same frame")
	}
	if pair.Light == pair.Dark {
		t.Fatal("Pair returned the same theme for both backgrounds")
	}
}

func TestQuitKeys(t *testing.T) {
	for _, k := range []tui.Key{{Type: tui.KeyCtrlC}, {Type: tui.KeyRunes, Text: "q"}} {
		_, cmd := newModel().Update(k)
		if cmd == nil {
			t.Errorf("%v did not quit", k)
		}
	}
}

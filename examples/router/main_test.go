package main

import (
	"os"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/picker"
)

// selectMenuItem drives the menu picker's cursor to label (moving Up or
// Down as needed, since picker.Model's cursor doesn't wrap) and confirms it
// with Enter, running the returned Cmd and feeding the resulting
// picker.SelectedMsg back into Update the way Program's real event loop
// would (see examples/table/main_test.go for the same pattern).
func selectMenuItem(t *testing.T, m model, label string) model {
	t.Helper()

	target := -1
	for i, item := range m.menu.Items {
		if item.Label == label {
			target = i
			break
		}
	}
	if target == -1 {
		t.Fatalf("no menu item labelled %q", label)
	}

	for m.menu.Cursor() != target {
		dir := tui.KeyDown
		if m.menu.Cursor() > target {
			dir = tui.KeyUp
		}
		next, _ := m.Update(tui.Key{Type: dir})
		m = next.(model)
	}

	next, cmd := m.Update(tui.Key{Type: tui.KeyEnter})
	m = next.(model)
	if cmd == nil {
		t.Fatalf("selecting %q should return a Cmd delivering SelectedMsg", label)
	}
	msg := cmd()
	if _, ok := msg.(picker.SelectedMsg); !ok {
		t.Fatalf("expected picker.SelectedMsg, got %T", msg)
	}
	next, _ = m.Update(msg)
	return next.(model)
}

// model has an internal screen-selecting field with at least 3 distinct
// top-level screens, each with its own sub-model (criterion #575).
func TestModelHasThreeScreensEachWithOwnSubModel(t *testing.T) {
	m := initialModel()

	if m.screen != screenMenu {
		t.Fatalf("initial screen = %v, want screenMenu", m.screen)
	}

	// Three distinct, named screen values.
	screens := map[screen]bool{screenMenu: true, screenSettings: true, screenAbout: true}
	if len(screens) != 3 {
		t.Fatalf("expected 3 distinct screen values, got %d", len(screens))
	}

	// Each screen delegates to its own sub-model/state: screenMenu owns a
	// picker.Model, screenSettings owns a textinput.Model.
	if len(m.menu.Items) != 3 {
		t.Fatalf("menu should have 3 items (Settings/About/Quit), got %d", len(m.menu.Items))
	}
	var _ = m.settings // textinput.Model, screenSettings' own sub-state
}

// Selecting "Settings" from the menu switches m.screen, and View shows only
// the Settings screen's content — no menu content bleeding through
// (criterion #576).
func TestMenuSelectionSwitchesScreenAndViewShowsOnlyThatScreen(t *testing.T) {
	m := initialModel()

	menuView := ansi.StripANSI(m.View())
	if !strings.Contains(menuView, "Settings") || !strings.Contains(menuView, "About") {
		t.Fatalf("menu screen should list Settings/About/Quit, got:\n%s", menuView)
	}

	m = selectMenuItem(t, m, "Settings")

	if m.screen != screenSettings {
		t.Fatalf("screen after selecting Settings = %v, want screenSettings", m.screen)
	}

	settingsView := ansi.StripANSI(m.View())
	if !strings.Contains(settingsView, "Display name") {
		t.Fatalf("settings screen should show the text input prompt, got:\n%s", settingsView)
	}
	if strings.Contains(settingsView, "About") || strings.Contains(settingsView, "Quit") {
		t.Fatalf("settings screen should not blend in menu content, got:\n%s", settingsView)
	}

	// Switch to About too, and confirm it's exclusively About's content.
	m.screen = screenMenu
	m = selectMenuItem(t, m, "About")
	if m.screen != screenAbout {
		t.Fatalf("screen after selecting About = %v, want screenAbout", m.screen)
	}
	aboutView := ansi.StripANSI(m.View())
	if !strings.Contains(aboutView, "independent") || !strings.Contains(aboutView, "wizard.Model") {
		t.Fatalf("about screen should describe the pattern, got:\n%s", aboutView)
	}
	if strings.Contains(aboutView, "Display name") {
		t.Fatalf("about screen should not blend in settings content, got:\n%s", aboutView)
	}
}

// Typed text in the Settings screen survives switching to About and back,
// because m.settings lives as a field on model rather than being
// reconstructed on each screen switch (criterion #577).
func TestSettingsStatePersistsAcrossScreenSwitches(t *testing.T) {
	m := initialModel()
	m = selectMenuItem(t, m, "Settings")

	for _, r := range "Ada" {
		next, cmd := m.Update(tui.Key{Type: tui.KeyRunes, Text: string([]rune{r})})
		m = next.(model)
		if cmd != nil {
			// textinput's blink Cmd, if any, is irrelevant here.
			_ = cmd
		}
	}
	if got := m.settings.Value(); got != "Ada" {
		t.Fatalf("settings value after typing = %q, want %q", got, "Ada")
	}

	// Leave Settings for About.
	next, _ := m.Update(tui.Key{Type: tui.KeyEsc})
	m = next.(model)
	if m.screen != screenMenu {
		t.Fatalf("Esc from settings should return to menu, got %v", m.screen)
	}
	m = selectMenuItem(t, m, "About")
	if m.screen != screenAbout {
		t.Fatalf("screen = %v, want screenAbout", m.screen)
	}

	// Come back to Settings: the typed value must still be there.
	next, _ = m.Update(tui.Key{Type: tui.KeyEsc})
	m = next.(model)
	m = selectMenuItem(t, m, "Settings")

	if got := m.settings.Value(); got != "Ada" {
		t.Fatalf("settings value after round trip = %q, want %q (state should persist)", got, "Ada")
	}
}

// main.go's top-of-file doc comment names this the canonical reference for
// switching between top-level screens and explicitly distinguishes it from
// wizard.Model's linear within-one-flow step navigation (criterion #578).
func TestDocCommentNamesPatternAndDistinguishesFromWizard(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("reading main.go: %v", err)
	}
	text := string(src)

	if !strings.Contains(text, "canonical reference") {
		t.Error("doc comment should state this is the canonical reference for the pattern")
	}
	if !strings.Contains(text, "top-level screens") {
		t.Error("doc comment should describe switching between top-level screens")
	}
	if !strings.Contains(text, "wizard.Model") {
		t.Error("doc comment should explicitly reference wizard.Model")
	}
	if !strings.Contains(text, "within-one-flow") && !strings.Contains(text, "linear") {
		t.Error("doc comment should distinguish this from wizard.Model's linear within-one-flow step navigation")
	}
}

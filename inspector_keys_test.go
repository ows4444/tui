package tui

import "testing"

// Criterion #76: a zero field takes that pane's default key, and a zero
// InspectorKeys turns on all four.
func TestInspectorKeysZeroFieldsUseTheDefaults(t *testing.T) {
	p := NewProgram(staticModel{}, WithInspector(InspectorKeys{}))
	if p.inspectorKey != DefaultInspectorKey || p.outlineKey != DefaultInspectorLayoutKey ||
		p.msgKey != DefaultInspectorMessagesKey || p.modelKey != DefaultInspectorModelKey {
		t.Errorf("keys = %q %q %q %q, want the four defaults", p.inspectorKey, p.outlineKey, p.msgKey, p.modelKey)
	}
	if DefaultInspectorKey != "f12" || DefaultInspectorLayoutKey != "f11" || DefaultInspectorMessagesKey != "f10" || DefaultInspectorModelKey != "f9" {
		t.Error("a default key changed")
	}
	// A field set to a key replaces only that pane's default; InspectorOff leaves one out.
	p = NewProgram(staticModel{}, WithInspector(InspectorKeys{Panel: "ctrl+i", Layout: InspectorOff}))
	if p.inspectorKey != "ctrl+i" || p.outlineKey != "" || p.msgKey != DefaultInspectorMessagesKey || p.modelKey != DefaultInspectorModelKey {
		t.Errorf("keys = %q %q %q %q", p.inspectorKey, p.outlineKey, p.msgKey, p.modelKey)
	}
	// Without the option nothing is on.
	p = NewProgram(staticModel{})
	if p.inspectorKey != "" || p.outlineKey != "" || p.msgKey != "" || p.modelKey != "" {
		t.Error("an inspector pane is on without WithInspector")
	}
}

// Criterion #77: each pane toggles on the key given for it, and only that key.
func TestInspectorKeysTogglePanesOnTheirOwnKeys(t *testing.T) {
	p := NewProgram(staticModel{view: "x"}, WithInspector(InspectorKeys{Panel: "f1", Layout: "f2", Messages: "f3", Model: "f4"}))
	for key, on := range map[string]func() bool{
		"f1": func() bool { return p.inspectorOn },
		"f2": func() bool { return p.outlineOn },
		"f3": func() bool { return p.msgOn },
		"f4": func() bool { return p.modelOn },
	} {
		var typ KeyType
		switch key {
		case "f1":
			typ = KeyF1
		case "f2":
			typ = KeyF2
		case "f3":
			typ = KeyF3
		case "f4":
			typ = KeyF4
		}
		if on() {
			t.Fatalf("%s: pane on before its key", key)
		}
		if !p.inspectorKeyPressed(Key{Type: typ}) || !on() {
			t.Errorf("%s did not turn its pane on", key)
		}
		if !p.inspectorKeyPressed(Key{Type: typ}) || on() {
			t.Errorf("%s did not turn its pane off again", key)
		}
	}
	if p.inspectorKeyPressed(Key{Type: KeyF5}) {
		t.Error("an unrelated key was consumed")
	}
}
